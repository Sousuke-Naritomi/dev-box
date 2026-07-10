package cmd

import (
	"bytes"
	"context"
	"encoding/json"
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
	K8sFunc      func(cfg config.ProjectConfig) (map[string]generator.FileDef, error)
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

func (m *MockGenerator) GenerateK8sManifests(cfg config.ProjectConfig) (map[string]generator.FileDef, error) {
	if m.K8sFunc != nil {
		return m.K8sFunc(cfg)
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
		"dev-box.yaml",
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

func TestRunInitLogicJSONDryRunOutput(t *testing.T) {
	fileSystem := fs.NewMockFileSystem()
	gen := &MockGenerator{
		ScaffoldFunc: func(cfg config.ProjectConfig) (map[string]generator.FileDef, error) {
			return map[string]generator.FileDef{
				"backend/main.go": {Content: "package main\n", Perm: 0644},
			}, nil
		},
	}
	cfg := config.ProjectConfig{
		Frontend: config.FrontendNone,
		Backend:  config.BackendGo,
		Database: config.DatabaseNone,
		Cache:    config.CacheNone,
	}

	var output bytes.Buffer
	err := runInitLogic(context.Background(), cfg, "sample", true, "json", &output, fileSystem, gen)
	if err != nil {
		t.Fatalf("runInitLogic json failed: %v", err)
	}
	if fileSystem.Exists(filepath.Join("sample", "backend", "main.go")) {
		t.Fatal("json dry-run must not write files")
	}

	var got initOutput
	if err := json.Unmarshal(output.Bytes(), &got); err != nil {
		t.Fatalf("expected valid json output: %v\n%s", err, output.String())
	}
	if !got.DryRun {
		t.Fatal("expected dryRun=true")
	}
	if got.OutputDir != "sample" {
		t.Fatalf("expected outputDir sample, got %q", got.OutputDir)
	}
	if len(got.Files) == 0 {
		t.Fatal("expected generated files in json output")
	}

	foundBackend := false
	for _, file := range got.Files {
		if file.Path == "sample/backend/main.go" {
			foundBackend = true
			if file.Content != "package main\n" {
				t.Fatalf("unexpected backend content: %q", file.Content)
			}
		}
	}
	if !foundBackend {
		t.Fatalf("expected backend/main.go in json output: %#v", got.Files)
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
