package cmd

import (
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/spf13/cobra"
)

func TestLoadProjectConfig(t *testing.T) {
	fileSystem := fs.NewMockFileSystem()
	fileSystem.Files["dev-box.yaml"] = fs.MockFile{Data: []byte(
		"version: 1\nfrontend: react\nbackend: go\ndatabase: postgres\ncache: redis\nnginx: true\n",
	)}
	cfg, err := loadProjectConfig(fileSystem, "dev-box.yaml")
	if err != nil {
		t.Fatalf("loadProjectConfig failed: %v", err)
	}
	if cfg.Frontend != config.FrontendReact || cfg.Backend != config.BackendGo || !cfg.Nginx {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestExplicitFlagsOverrideProjectConfig(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("frontend", "", "")
	cmd.Flags().String("backend", "", "")
	cmd.Flags().String("database", "", "")
	cmd.Flags().String("cache", "", "")
	cmd.Flags().Bool("nginx", false, "")
	cmd.Flags().Bool("localstack", false, "")
	cmd.Flags().Bool("monitoring", false, "")
	cmd.Flags().Bool("cicd", false, "")
	if err := cmd.Flags().Parse([]string{"--backend=python", "--nginx=false"}); err != nil {
		t.Fatalf("parsing flags: %v", err)
	}

	cfg := config.ProjectConfig{Frontend: config.FrontendReact, Backend: config.BackendGo, Database: config.DatabasePostgres, Cache: config.CacheRedis, Nginx: true}
	applyCoreFlags(cmd, &cfg)
	applyAddonFlags(cmd, &cfg)
	if cfg.Backend != config.BackendPython || cfg.Frontend != config.FrontendReact || cfg.Nginx {
		t.Fatalf("explicit flags did not override configuration correctly: %#v", cfg)
	}
}
