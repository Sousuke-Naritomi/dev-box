package generator

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/registry"
)

// GenerateScaffold generates boilerplate source code files for the selected
// frontend and backend. Returns a map of relative filepath to file content.
func GenerateScaffold(cfg config.ProjectConfig) (map[string]FileDef, error) {
	files := make(map[string]FileDef)

	templateData := struct {
		VersionNode   string
		VersionPython string
		VersionGo     string
		VersionPHP    string
		VersionJava   string
		VersionNginx  string
	}{
		VersionNode:   config.VersionNode,
		VersionPython: config.VersionPython,
		VersionGo:     config.VersionGo,
		VersionPHP:    config.VersionPHP,
		VersionJava:   config.VersionJava,
		VersionNginx:  config.VersionNginx,
	}

	if frontend, ok := registry.SupportedFrontends[cfg.Frontend]; ok {
		if err := copyScaffoldFiles(frontend.TemplateDir, "frontend", templateData, files); err != nil {
			return nil, fmt.Errorf("scaffolding frontend: %w", err)
		}
	} else if cfg.Frontend != "none" && cfg.Frontend != "" {
		return nil, fmt.Errorf("unsupported frontend: %q", cfg.Frontend)
	}

	if backend, ok := registry.SupportedBackends[cfg.Backend]; ok {
		if err := copyScaffoldFiles(backend.TemplateDir, "backend", templateData, files); err != nil {
			return nil, fmt.Errorf("scaffolding backend: %w", err)
		}
	} else if cfg.Backend != "none" && cfg.Backend != "" {
		return nil, fmt.Errorf("unsupported backend: %q", cfg.Backend)
	}

	return files, nil
}

func copyScaffoldFiles(templateDir string, destPrefix string, data any, out map[string]FileDef) error {
	return fs.WalkDir(FS, templateDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		const mode = 0644

		contentBytes, err := fs.ReadFile(FS, path)
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(templateDir, path)
		if err != nil {
			return err
		}

		relPath = filepath.ToSlash(relPath)
		destPath := filepath.Join(destPrefix, relPath)
		destPath = filepath.ToSlash(destPath)

		if strings.HasSuffix(destPath, ".tmpl") {
			destPath = strings.TrimSuffix(destPath, ".tmpl")
			tmpl, err := template.New(destPath).Parse(string(contentBytes))
			if err != nil {
				return err
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, data); err != nil {
				return err
			}
			out[destPath] = FileDef{Content: buf.String(), Perm: mode}
		} else {
			out[destPath] = FileDef{Content: string(contentBytes), Perm: mode}
		}

		return nil
	})
}
