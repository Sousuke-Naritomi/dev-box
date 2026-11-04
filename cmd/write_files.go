package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

func prepareGeneratedFiles(fileSystem fs.FileSystem, outputDir string, files map[string]generator.FileDef) (map[string]fs.File, []string, error) {
	prepared := make(map[string]fs.File, len(files))
	existing := make([]string, 0)

	for relativePath, file := range files {
		target := generatedTargetPath(outputDir, relativePath)
		if err := rejectSymlinkedParents(fileSystem, outputDir, relativePath); err != nil {
			return nil, nil, err
		}
		isLink, err := fileSystem.IsSymlink(target)
		if err != nil {
			return nil, nil, fmt.Errorf("checking %s: %w", target, err)
		}
		if isLink {
			return nil, nil, fmt.Errorf("refusing to overwrite symbolic link: %s", target)
		}
		if fileSystem.Exists(target) {
			existing = append(existing, target)
		}

		content := file.Content
		if filepath.Base(relativePath) == ".gitignore" && fileSystem.Exists(target) {
			current, err := fileSystem.ReadFile(target)
			if err != nil {
				return nil, nil, fmt.Errorf("reading %s: %w", target, err)
			}
			content = mergeGitignore(string(current), content)
		}
		prepared[target] = fs.File{Data: []byte(content), Perm: file.Perm}
	}

	sort.Strings(existing)
	return prepared, existing, nil
}

// rejectSymlinkedParents refuses paths that traverse a symlinked directory below
// outputDir, which would redirect writes outside the output tree. outputDir
// itself is the caller's choice and may legitimately be a link.
func rejectSymlinkedParents(fileSystem fs.FileSystem, outputDir, relativePath string) error {
	dir := filepath.Dir(relativePath)
	if dir == "." {
		return nil
	}
	current := outputDir
	for _, part := range strings.Split(filepath.ToSlash(dir), "/") {
		current = filepath.Join(current, part)
		isLink, err := fileSystem.IsSymlink(current)
		if err != nil {
			return fmt.Errorf("checking %s: %w", current, err)
		}
		if isLink {
			return fmt.Errorf("refusing to write through symbolic link: %s", current)
		}
	}
	return nil
}

func generatedTargetPath(outputDir, relativePath string) string {
	if outputDir == "" || outputDir == "." {
		return relativePath
	}
	return filepath.Join(outputDir, relativePath)
}

func writeGeneratedFiles(ctx context.Context, fileSystem fs.FileSystem, outputDir string, files map[string]generator.FileDef, force, dryRun bool, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	prepared, existing, err := prepareGeneratedFiles(fileSystem, outputDir, files)
	if err != nil {
		return err
	}
	if dryRun {
		paths := make([]string, 0, len(prepared))
		for path := range prepared {
			paths = append(paths, path)
		}
		sort.Strings(paths)
		for _, path := range paths {
			if _, err := fmt.Fprintf(output, "--- %s ---\n%s", path, prepared[path].Data); err != nil {
				return fmt.Errorf("writing dry-run output: %w", err)
			}
		}
		return nil
	}
	if len(existing) > 0 && !force {
		if err := writeCommandOutput(output, "Existing files would be overwritten:\n  %s\n", strings.Join(existing, "\n  ")); err != nil {
			return err
		}
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("existing files found in a non-interactive environment; use --force to overwrite")
		}
		var overwrite bool
		if err := huh.NewConfirm().Title("Overwrite these files? [y/N]").Value(&overwrite).Run(); err != nil {
			return fmt.Errorf("prompt cancelled: %w", err)
		}
		if !overwrite {
			return fmt.Errorf("file generation cancelled")
		}
	}
	if err := fileSystem.WriteFiles(prepared); err != nil {
		return fmt.Errorf("writing generated files: %w", err)
	}
	return nil
}

func mergeGitignore(current, required string) string {
	lines := strings.Split(strings.ReplaceAll(current, "\r\n", "\n"), "\n")
	seen := make(map[string]bool, len(lines))
	for _, line := range lines {
		seen[strings.TrimSpace(line)] = true
	}
	for _, line := range strings.Split(required, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !seen[line] {
			lines = append(lines, line)
			seen[line] = true
		}
	}
	return strings.TrimRight(strings.Join(lines, "\n"), "\n") + "\n"
}
