package cmd

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

func TestMergeGitignorePreservesExistingEntries(t *testing.T) {
	got := mergeGitignore("custom/\n.env\n", ".env\nnode_modules/\n")
	for _, entry := range []string{"custom/", ".env", "node_modules/"} {
		if !strings.Contains(got, entry) {
			t.Fatalf("expected merged gitignore to contain %q: %s", entry, got)
		}
	}
	if strings.Count(got, ".env") != 1 {
		t.Fatalf("expected .env only once: %s", got)
	}
}

func TestWriteGeneratedFilesRejectsSymlink(t *testing.T) {
	fileSystem := fs.NewMockFileSystem()
	fileSystem.Symlinks["secret"] = true
	err := writeGeneratedFiles(context.Background(), fileSystem, ".", map[string]generator.FileDef{
		"secret": {Content: "value", Perm: 0600},
	}, true, false, &bytes.Buffer{})
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("expected symbolic link rejection, got %v", err)
	}
}

func TestWriteGeneratedFilesRejectsSymlinkedParentDirectory(t *testing.T) {
	tests := []struct {
		name      string
		outputDir string
		link      string
		files     map[string]generator.FileDef
	}{
		{
			name:      "directory directly under output dir",
			outputDir: "out",
			link:      filepath.Join("out", "backend"),
			files:     map[string]generator.FileDef{"backend/Dockerfile": {Content: "FROM scratch", Perm: 0644}},
		},
		{
			name:      "intermediate directory several levels deep",
			outputDir: "out",
			link:      filepath.Join("out", "frontend", "src"),
			files:     map[string]generator.FileDef{"frontend/src/components/App.jsx": {Content: "x", Perm: 0644}},
		},
		{
			name:      "current directory as output dir",
			outputDir: ".",
			link:      "monitoring",
			files:     map[string]generator.FileDef{"monitoring/prometheus.yml": {Content: "x", Perm: 0644}},
		},
	}
	for _, tt := range tests {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dryRun=%v", tt.name, dryRun), func(t *testing.T) {
				fileSystem := fs.NewMockFileSystem()
				fileSystem.Symlinks[tt.link] = true

				err := writeGeneratedFiles(context.Background(), fileSystem, tt.outputDir, tt.files, true, dryRun, &bytes.Buffer{})
				if err == nil || !strings.Contains(err.Error(), "symbolic link") {
					t.Fatalf("expected symbolic link rejection, got %v", err)
				}
				if len(fileSystem.Files) != 0 {
					t.Fatalf("nothing may be written through a symlinked directory, wrote %v", fileSystem.Files)
				}
			})
		}
	}
}

func TestWriteGeneratedFilesWritesNestedFilesWhenNoSymlinks(t *testing.T) {
	fileSystem := fs.NewMockFileSystem()
	err := writeGeneratedFiles(context.Background(), fileSystem, "out", map[string]generator.FileDef{
		"backend/Dockerfile":          {Content: "FROM scratch", Perm: 0644},
		"frontend/src/components/App": {Content: "x", Perm: 0644},
		"dev-box.yaml":                {Content: "version: 1\n", Perm: 0644},
	}, false, false, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		filepath.Join("out", "backend", "Dockerfile"),
		filepath.Join("out", "frontend", "src", "components", "App"),
		filepath.Join("out", "dev-box.yaml"),
	} {
		if !fileSystem.Exists(want) {
			t.Errorf("expected %s to be written", want)
		}
	}
}

func TestWriteGeneratedFilesDryRunDoesNotWrite(t *testing.T) {
	fileSystem := fs.NewMockFileSystem()
	var output bytes.Buffer
	err := writeGeneratedFiles(context.Background(), fileSystem, ".", map[string]generator.FileDef{
		"example.txt": {Content: "hello\n", Perm: 0644},
	}, false, true, &output)
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if fileSystem.Exists("example.txt") {
		t.Fatal("dry run must not write files")
	}
	if !strings.Contains(output.String(), "hello") {
		t.Fatalf("expected generated content in dry-run output: %s", output.String())
	}
}
