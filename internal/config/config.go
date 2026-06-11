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
	Frontend   FrontendType
	Backend    BackendType
	Database   DatabaseType
	Cache      CacheType
	Nginx      bool
	LocalStack bool
	Monitoring bool
	CICD       bool
	Force      bool // Skip overwrite prompts
}

// Validate checks if the configuration values are valid.
func (c *ProjectConfig) Validate() error {
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
