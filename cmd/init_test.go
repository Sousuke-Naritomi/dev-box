package cmd

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

// MockGenerator is a mock implementation of generator.Generator for testing.
type MockGenerator struct {
	ScaffoldFunc func(cfg config.ProjectConfig) (map[string]generator.FileDef, error)
	TfFunc       func(provider string) (map[string]generator.FileDef, error)
	K8sFunc      func(frontend, backend, database, cache string) (map[string]generator.FileDef, error)
}

func (m *MockGenerator) GenerateScaffold(cfg config.ProjectConfig) (map[string]generator.FileDef, error) {
	if m.ScaffoldFunc != nil {
		return m.ScaffoldFunc(cfg)
	}
	return map[string]generator.FileDef{}, nil
}

func (m *MockGenerator) GenerateTerraform(provider string) (map[string]generator.FileDef, error) {
	if m.TfFunc != nil {
		return m.TfFunc(provider)
	}
	return map[string]generator.FileDef{}, nil
}

func (m *MockGenerator) GenerateK8sManifests(frontend, backend, database, cache string) (map[string]generator.FileDef, error) {
	if m.K8sFunc != nil {
		return m.K8sFunc(frontend, backend, database, cache)
	}
	return map[string]generator.FileDef{}, nil
}

func TestInitCmd_E2E(t *testing.T) {
	mockFS := fs.NewMockFileSystem()
	mockGen := &MockGenerator{
		ScaffoldFunc: func(cfg config.ProjectConfig) (map[string]generator.FileDef, error) {
			return map[string]generator.FileDef{
				"frontend/package.json": {Content: "{}", Perm: 0644},
				"backend/main.go":       {Content: "package main", Perm: 0644},
			}, nil
		},
	}

	app.FS = mockFS
	app.Gen = mockGen

	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{
		"init",
		"--frontend=react",
		"--backend=go",
		"--database=postgres",
		"--cache=redis",
		"--nginx=true",
		"--force=true",
	})

	err := rootCmd.ExecuteContext(context.Background())
	if err != nil {
		t.Fatalf("cmd.ExecuteContext failed: %v\nstderr: %s", err, errBuf.String())
	}

	expectedFiles := []string{
		"docker-compose.yml",
		".env",
		".gitignore",
		filepath.Join("nginx", "nginx.conf"),
		"frontend/package.json",
		"backend/main.go",
	}

	for _, ef := range expectedFiles {
		if !mockFS.Exists(ef) {
			t.Errorf("expected file %q to be generated", ef)
		}
	}
}

func TestInitCmdValidation_InvalidFrontend(t *testing.T) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{
		"init",
		"--frontend=invalid-frontend",
		"--backend=go",
		"--database=mysql",
		"--cache=redis",
	})

	err := rootCmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected validation error for invalid frontend")
	}
	if !strings.Contains(err.Error(), "unsupported frontend") {
		t.Errorf("expected 'unsupported frontend' error, got %v", err)
	}
}

func TestInitCmdValidation_InvalidBackend(t *testing.T) {
	outBuf := new(bytes.Buffer)
	errBuf := new(bytes.Buffer)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{
		"init",
		"--frontend=react",
		"--backend=invalid-backend",
		"--database=mysql",
		"--cache=redis",
	})

	err := rootCmd.ExecuteContext(context.Background())
	if err == nil {
		t.Fatal("expected validation error for invalid backend")
	}
	if !strings.Contains(err.Error(), "unsupported backend") {
		t.Errorf("expected 'unsupported backend' error, got %v", err)
	}
}
