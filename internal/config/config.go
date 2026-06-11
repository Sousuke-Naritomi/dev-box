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
