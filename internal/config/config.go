// Package config defines the configuration types used throughout
// the dev-box CLI tool for environment generation.
package config

// ProjectConfig holds all user selections for environment generation.
// Each field corresponds to a prompt in the interactive setup wizard.
type ProjectConfig struct {
	Frontend   string // "react", "vue", "none"
	Backend    string // "node", "python", "go", "php", "java", "none"
	Database   string // "mysql", "postgres", "none"
	Cache      string // "redis", "none"
	Nginx      bool
	LocalStack bool
	Monitoring bool
	CICD       bool
}

// BackendInternalPort returns the internal container port for the selected backend.
// Returns 0 if no backend is selected or the backend type is unrecognized.
func (c *ProjectConfig) BackendInternalPort() int {
	switch c.Backend {
	case "node":
		return 3000
	case "python":
		return 8000
	case "go", "java":
		return 8080
	case "php":
		return 80
	default:
		return 0
	}
}
