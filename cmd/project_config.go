package cmd

import (
	"fmt"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/spf13/cobra"
)

func loadProjectConfig(fileSystem fs.FileSystem, path string) (config.ProjectConfig, error) {
	if path == "" {
		return config.ProjectConfig{}, nil
	}
	data, err := fileSystem.ReadFile(path)
	if err != nil {
		return config.ProjectConfig{}, fmt.Errorf("reading project configuration %s: %w", path, err)
	}
	cfg, err := config.ParseProjectFile(data)
	if err != nil {
		return config.ProjectConfig{}, fmt.Errorf("loading %s: %w", path, err)
	}
	return cfg, nil
}

func applyCoreFlags(cmd *cobra.Command, cfg *config.ProjectConfig) {
	if cmd.Flags().Changed("frontend") {
		value, _ := cmd.Flags().GetString("frontend")
		cfg.Frontend = config.FrontendType(value)
	}
	if cmd.Flags().Changed("backend") {
		value, _ := cmd.Flags().GetString("backend")
		cfg.Backend = config.BackendType(value)
	}
	if cmd.Flags().Changed("database") {
		value, _ := cmd.Flags().GetString("database")
		cfg.Database = config.DatabaseType(value)
	}
	if cmd.Flags().Changed("cache") {
		value, _ := cmd.Flags().GetString("cache")
		cfg.Cache = config.CacheType(value)
	}
}

func applyAddonFlags(cmd *cobra.Command, cfg *config.ProjectConfig) {
	if cmd.Flags().Changed("nginx") {
		cfg.Nginx, _ = cmd.Flags().GetBool("nginx")
	}
	if cmd.Flags().Changed("localstack") {
		cfg.LocalStack, _ = cmd.Flags().GetBool("localstack")
	}
	if cmd.Flags().Changed("monitoring") {
		cfg.Monitoring, _ = cmd.Flags().GetBool("monitoring")
	}
	if cmd.Flags().Changed("cicd") {
		cfg.CICD, _ = cmd.Flags().GetBool("cicd")
	}
}
