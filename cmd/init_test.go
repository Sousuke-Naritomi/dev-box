package cmd

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
)

func TestRunInitLogic_Success(t *testing.T) {
	mockFS := fs.NewMockFileSystem()
	cfg := config.ProjectConfig{
		Frontend: "react",
		Backend:  "go",
		Database: "postgres",
		Cache:    "redis",
		Nginx:    true,
		Force:    true,
	}

	err := runInitLogic(context.Background(), cfg, mockFS)
	if err != nil {
		t.Fatalf("runInitLogic failed: %v", err)
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
	cmd := initCmd
	// Set valid backend but invalid frontend
	cmd.Flags().Set("frontend", "invalid-frontend")
	cmd.Flags().Set("backend", "go")
	cmd.Flags().Set("database", "mysql")
	cmd.Flags().Set("cache", "redis")

	err := runInit(cmd, []string{})
	if err == nil {
		t.Fatal("expected validation error for invalid frontend")
	}
	if !strings.Contains(err.Error(), "unsupported frontend") {
		t.Errorf("expected 'unsupported frontend' error, got %v", err)
	}
}

func TestInitCmdValidation_InvalidBackend(t *testing.T) {
	cmd := initCmd
	cmd.Flags().Set("frontend", "react")
	cmd.Flags().Set("backend", "invalid-backend")
	cmd.Flags().Set("database", "mysql")
	cmd.Flags().Set("cache", "redis")

	err := runInit(cmd, []string{})
	if err == nil {
		t.Fatal("expected validation error for invalid backend")
	}
	if !strings.Contains(err.Error(), "unsupported backend") {
		t.Errorf("expected 'unsupported backend' error, got %v", err)
	}
}
