package cmd

import (
	"bytes"
	"context"
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
