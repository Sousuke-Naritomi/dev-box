package config

import (
	"fmt"
)

type FrontendType string
type BackendType string
type DatabaseType string
type CacheType string

const (
	FrontendReact FrontendType = "react"
	FrontendVue   FrontendType = "vue"
	FrontendNone  FrontendType = "none"

	BackendNode   BackendType = "node"
	BackendPython BackendType = "python"
	BackendGo     BackendType = "go"
	BackendPHP    BackendType = "php"
	BackendJava   BackendType = "java"
	BackendNone   BackendType = "none"

	DatabaseMySQL    DatabaseType = "mysql"
	DatabasePostgres DatabaseType = "postgres"
	DatabaseNone     DatabaseType = "none"

	CacheRedis CacheType = "redis"
	CacheNone  CacheType = "none"
)

// ProjectConfig holds all user selections for environment generation.
// Each field corresponds to a prompt in the interactive setup wizard.
type ProjectConfig struct {
	Frontend   FrontendType `yaml:"frontend"`
	Backend    BackendType  `yaml:"backend"`
	Database   DatabaseType `yaml:"database"`
	Cache      CacheType    `yaml:"cache"`
	Nginx      bool         `yaml:"nginx"`
	LocalStack bool         `yaml:"localstack"`
	Monitoring bool         `yaml:"monitoring"`
	CICD       bool         `yaml:"cicd"`
	Force      bool         `yaml:"-"` // Skip overwrite prompts; never persisted.
}

// Validate checks if the configuration values are valid.
func (c *ProjectConfig) Validate() error {
	if err := c.ValidateSelection(); err != nil {
		return err
	}

	if !c.HasWorkload() && !c.LocalStack && !c.Monitoring {
		return fmt.Errorf("at least one service must be selected")
	}
	if c.Nginx && c.Frontend == FrontendNone && c.Backend == BackendNone {
		return fmt.Errorf("nginx requires a frontend or backend service")
	}

	return nil
}

// ValidateSelection checks the core stack values without considering addons.
func (c *ProjectConfig) ValidateSelection() error {
	switch c.Frontend {
	case FrontendReact, FrontendVue, FrontendNone:
		// valid
	default:
		return fmt.Errorf("unsupported frontend: %q", c.Frontend)
	}

	switch c.Backend {
	case BackendNode, BackendPython, BackendGo, BackendPHP, BackendJava, BackendNone:
		// valid
	default:
		return fmt.Errorf("unsupported backend: %q", c.Backend)
	}

	switch c.Database {
	case DatabaseMySQL, DatabasePostgres, DatabaseNone:
		// valid
	default:
		return fmt.Errorf("unsupported database: %q", c.Database)
	}

	switch c.Cache {
	case CacheRedis, CacheNone:
		// valid
	default:
		return fmt.Errorf("unsupported cache: %q", c.Cache)
	}

	return nil
}

// HasWorkload reports whether the configuration contains a core application,
// database, or cache service.
func (c ProjectConfig) HasWorkload() bool {
	return c.Frontend != FrontendNone ||
		c.Backend != BackendNone ||
		c.Database != DatabaseNone ||
		c.Cache != CacheNone
}
