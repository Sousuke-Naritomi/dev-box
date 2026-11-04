package fs

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

// snapshot captures every file under root (relative path -> content), so a
// comparison also catches leftover temporary files.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func writeAll(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func injectRenameFailure(t *testing.T, shouldFail func(oldpath, newpath string) bool) {
	t.Helper()
	original := renameFile
	renameFile = func(oldpath, newpath string) error {
		if shouldFail(oldpath, newpath) {
			return errors.New("injected rename failure")
		}
		return original(oldpath, newpath)
	}
	t.Cleanup(func() { renameFile = original })
}

func requireSnapshot(t *testing.T, dir string, want map[string]string) {
	t.Helper()
	if got := snapshot(t, dir); !reflect.DeepEqual(got, want) {
		t.Fatalf("directory changed after failed write\n got: %v\nwant: %v", got, want)
	}
}

func TestWriteFilesRollsBackWhenCommitRenameFails(t *testing.T) {
	dir := t.TempDir()
	before := map[string]string{"a.txt": "old-a", "b.txt": "old-b"}
	writeAll(t, dir, before)

	injectRenameFailure(t, func(oldpath, newpath string) bool {
		return strings.Contains(filepath.Base(oldpath), ".dev-box-stage-") && filepath.Base(newpath) == "b.txt"
	})

	err := NewOSFileSystem().WriteFiles(map[string]File{
		filepath.Join(dir, "a.txt"): {Data: []byte("new-a"), Perm: 0o644},
		filepath.Join(dir, "b.txt"): {Data: []byte("new-b"), Perm: 0o644},
		filepath.Join(dir, "c.txt"): {Data: []byte("new-c"), Perm: 0o644},
	})
	if err == nil || !strings.Contains(err.Error(), "replacing") {
		t.Fatalf("expected commit failure, got %v", err)
	}
	requireSnapshot(t, dir, before)
}

func TestWriteFilesRollsBackWhenFinalChmodFails(t *testing.T) {
	dir := t.TempDir()
	before := map[string]string{"a.txt": "old-a", "b.txt": "old-b"}
	writeAll(t, dir, before)

	original := chmodFile
	chmodFile = func(name string, mode os.FileMode) error {
		if filepath.Base(name) == "b.txt" {
			return errors.New("injected chmod failure")
		}
		return original(name, mode)
	}
	t.Cleanup(func() { chmodFile = original })

	err := NewOSFileSystem().WriteFiles(map[string]File{
		filepath.Join(dir, "a.txt"): {Data: []byte("new-a"), Perm: 0o644},
		filepath.Join(dir, "b.txt"): {Data: []byte("new-b"), Perm: 0o644},
		filepath.Join(dir, "c.txt"): {Data: []byte("new-c"), Perm: 0o644},
	})
	if err == nil || !strings.Contains(err.Error(), "final permissions") {
		t.Fatalf("expected chmod failure, got %v", err)
	}
	requireSnapshot(t, dir, before)
}

func TestWriteFilesRestoresOriginalsWhenBackupFails(t *testing.T) {
	dir := t.TempDir()
	before := map[string]string{"a.txt": "old-a", "b.txt": "old-b"}
	writeAll(t, dir, before)

	injectRenameFailure(t, func(oldpath, newpath string) bool {
		return filepath.Base(oldpath) == "b.txt" && strings.Contains(filepath.Base(newpath), ".dev-box-backup-")
	})

	err := NewOSFileSystem().WriteFiles(map[string]File{
		filepath.Join(dir, "a.txt"): {Data: []byte("new-a"), Perm: 0o644},
		filepath.Join(dir, "b.txt"): {Data: []byte("new-b"), Perm: 0o644},
	})
	if err == nil || !strings.Contains(err.Error(), "backing up") {
		t.Fatalf("expected backup failure, got %v", err)
	}
	requireSnapshot(t, dir, before)
}

func TestWriteFilesStagingFailureTouchesNothingAndCleansTemporaries(t *testing.T) {
	dir := t.TempDir()
	before := map[string]string{"a.txt": "old-a", "sub": "i am a file, not a directory"}
	writeAll(t, dir, before)

	err := NewOSFileSystem().WriteFiles(map[string]File{
		filepath.Join(dir, "a.txt"):        {Data: []byte("new-a"), Perm: 0o644},
		filepath.Join(dir, "sub", "x.txt"): {Data: []byte("unreachable"), Perm: 0o644},
	})
	if err == nil {
		t.Fatal("expected failure because sub is a file")
	}
	requireSnapshot(t, dir, before)
}

func TestWriteFilesSuccessLeavesNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	writeAll(t, dir, map[string]string{"a.txt": "old-a"})

	err := NewOSFileSystem().WriteFiles(map[string]File{
		filepath.Join(dir, "a.txt"):               {Data: []byte("new-a"), Perm: 0o644},
		filepath.Join(dir, "deep", "er", "b.txt"): {Data: []byte("new-b"), Perm: 0o600},
	})
	if err != nil {
		t.Fatal(err)
	}
	requireSnapshot(t, dir, map[string]string{"a.txt": "new-a", "deep/er/b.txt": "new-b"})
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dir, "deep", "er", "b.txt"))
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode = %o, want 600", info.Mode().Perm())
		}
	}
}

func TestOSFileSystemBasicOperations(t *testing.T) {
	dir := t.TempDir()
	fileSystem := NewOSFileSystem()
	path := filepath.Join(dir, "nested", "file.txt")

	if fileSystem.Exists(path) {
		t.Fatal("file must not exist yet")
	}
	if isLink, err := fileSystem.IsSymlink(path); err != nil || isLink {
		t.Fatalf("IsSymlink(missing) = %v, %v; want false, nil", isLink, err)
	}
	if err := fileSystem.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !fileSystem.Exists(path) {
		t.Fatal("file must exist after WriteFile")
	}
	data, err := fileSystem.ReadFile(path)
	if err != nil || string(data) != "hello" {
		t.Fatalf("ReadFile = %q, %v", data, err)
	}
	if isLink, err := fileSystem.IsSymlink(path); err != nil || isLink {
		t.Fatalf("IsSymlink(regular file) = %v, %v; want false, nil", isLink, err)
	}
	if _, err := fileSystem.ReadFile(filepath.Join(dir, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("ReadFile(missing) err = %v", err)
	}
}

func TestIsSymlinkDetectsLinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	if isLink, err := NewOSFileSystem().IsSymlink(link); err != nil || !isLink {
		t.Fatalf("IsSymlink(link) = %v, %v; want true, nil", isLink, err)
	}
}

func TestWriteFilesReplacesExistingFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secret.env")
	if err := os.WriteFile(path, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	fileSystem := NewOSFileSystem()
	if err := fileSystem.WriteFiles(map[string]File{
		path: {Data: []byte("new"), Perm: 0600},
	}); err != nil {
		t.Fatalf("WriteFiles failed: %v", err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "new" {
		t.Fatalf("expected replacement content, got %q", content)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("expected mode 0600, got %o", info.Mode().Perm())
		}
	}
}

func TestWriteFilesRejectsSymbolicLink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}

	err := NewOSFileSystem().WriteFiles(map[string]File{
		link: {Data: []byte("replacement"), Perm: 0600},
	})
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symbolic link rejection, got %v", err)
	}
}
