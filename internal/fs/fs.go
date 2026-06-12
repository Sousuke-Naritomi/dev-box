package fs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type File struct {
	Data []byte
	Perm os.FileMode
}

// FileSystem defines operations for interacting with files.
type FileSystem interface {
	WriteFile(path string, data []byte, perm os.FileMode) error
	WriteFiles(files map[string]File) error
	ReadFile(path string) ([]byte, error)
	MkdirAll(path string, perm os.FileMode) error
	Exists(path string) bool
	IsSymlink(path string) (bool, error)
}

// OSFileSystem implements FileSystem using the standard os package.
type OSFileSystem struct{}

func NewOSFileSystem() *OSFileSystem {
	return &OSFileSystem{}
}

func (fs *OSFileSystem) WriteFile(path string, data []byte, perm os.FileMode) error {
	return fs.WriteFiles(map[string]File{path: {Data: data, Perm: perm}})
}

func (fs *OSFileSystem) WriteFiles(files map[string]File) error {
	paths := make([]string, 0, len(files))
	for path := range files {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	type stagedFile struct {
		target string
		stage  string
		backup string
		perm   os.FileMode
	}
	staged := make([]stagedFile, 0, len(paths))

	cleanup := func() {
		for _, file := range staged {
			if file.stage != "" {
				_ = os.Remove(file.stage)
			}
			if file.backup != "" {
				_ = os.Remove(file.backup)
			}
		}
	}
	defer cleanup()

	for _, path := range paths {
		isLink, err := fs.IsSymlink(path)
		if err != nil {
			return fmt.Errorf("checking %s: %w", path, err)
		}
		if isLink {
			return fmt.Errorf("refusing to overwrite symbolic link: %s", path)
		}

		dir := filepath.Dir(path)
		if err := fs.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", path, err)
		}
		tmp, err := os.CreateTemp(dir, ".dev-box-stage-*")
		if err != nil {
			return fmt.Errorf("creating temporary file for %s: %w", path, err)
		}
		stagePath := tmp.Name()
		if _, err := tmp.Write(files[path].Data); err != nil {
			_ = tmp.Close()
			_ = os.Remove(stagePath)
			return fmt.Errorf("writing temporary file for %s: %w", path, err)
		}
		if err := tmp.Chmod(files[path].Perm); err != nil {
			_ = tmp.Close()
			_ = os.Remove(stagePath)
			return fmt.Errorf("setting permissions for %s: %w", path, err)
		}
		if err := tmp.Sync(); err != nil {
			_ = tmp.Close()
			_ = os.Remove(stagePath)
			return fmt.Errorf("syncing temporary file for %s: %w", path, err)
		}
		if err := tmp.Close(); err != nil {
			_ = os.Remove(stagePath)
			return fmt.Errorf("closing temporary file for %s: %w", path, err)
		}
		staged = append(staged, stagedFile{target: path, stage: stagePath, perm: files[path].Perm})
	}

	restoreBackups := func() {
		for i := len(staged) - 1; i >= 0; i-- {
			if staged[i].backup != "" {
				_ = os.Remove(staged[i].target)
				_ = os.Rename(staged[i].backup, staged[i].target)
				staged[i].backup = ""
			}
		}
	}

	for i := range staged {
		file := &staged[i]
		if fs.Exists(file.target) {
			backup, err := os.CreateTemp(filepath.Dir(file.target), ".dev-box-backup-*")
			if err != nil {
				restoreBackups()
				return fmt.Errorf("creating backup path for %s: %w", file.target, err)
			}
			backupPath := backup.Name()
			if err := backup.Close(); err != nil {
				_ = os.Remove(backupPath)
				restoreBackups()
				return fmt.Errorf("closing backup path for %s: %w", file.target, err)
			}
			if err := os.Remove(backupPath); err != nil {
				restoreBackups()
				return fmt.Errorf("preparing backup path for %s: %w", file.target, err)
			}
			if err := os.Rename(file.target, backupPath); err != nil {
				restoreBackups()
				return fmt.Errorf("backing up %s: %w", file.target, err)
			}
			file.backup = backupPath
		}
	}

	committed := 0
	rollback := func() {
		for i := 0; i < committed; i++ {
			_ = os.Remove(staged[i].target)
		}
		restoreBackups()
	}

	for i := range staged {
		file := &staged[i]
		if err := os.Rename(file.stage, file.target); err != nil {
			rollback()
			return fmt.Errorf("replacing %s: %w", file.target, err)
		}
		file.stage = ""
		committed++
	}

	for i := range staged {
		file := &staged[i]
		if err := os.Chmod(file.target, file.perm); err != nil {
			rollback()
			return fmt.Errorf("setting final permissions for %s: %w", file.target, err)
		}
	}

	return nil
}

func (fs *OSFileSystem) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (fs *OSFileSystem) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (fs *OSFileSystem) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (fs *OSFileSystem) IsSymlink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}
