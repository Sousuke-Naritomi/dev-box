package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/registry"
	"github.com/Sousuke-Naritomi/dev-box/internal/security"
)

// ComposeData holds resolved configuration for docker-compose template rendering.
type ComposeData struct {
	Config   config.ProjectConfig
	Frontend *registry.FrontendFramework
	Backend  *registry.BackendFramework
	// Ports are now assigned dynamically by Docker Compose (using "0:<internal_port>")
	// Internal ports are read directly from registry Framework definitions.

	DBPassword      string
	DBRootPassword  string
	GrafanaPassword string
	HasVolumes      bool

	// Versions
	VersionNode       string
	VersionPython     string
	VersionGo         string
	VersionPHP        string
	VersionJava       string
	VersionMySQL      string
	VersionPostgres   string
	VersionRedis      string
	VersionNginx      string
	VersionLocalStack string
	VersionPrometheus string
	VersionGrafana    string
	VersionLoki       string
	VersionPromtail   string
}

// ResolveComposeData resolves ports and generates passwords based on the project config.
func ResolveComposeData(cfg config.ProjectConfig) (*ComposeData, error) {
	data := &ComposeData{
		Config:            cfg,
		HasVolumes:        cfg.Database != "none" && cfg.Database != "",
		VersionNode:       config.VersionNode,
		VersionPython:     config.VersionPython,
		VersionGo:         config.VersionGo,
		VersionPHP:        config.VersionPHP,
		VersionJava:       config.VersionJava,
		VersionMySQL:      config.VersionMySQL,
		VersionPostgres:   config.VersionPostgres,
		VersionRedis:      config.VersionRedis,
		VersionNginx:      config.VersionNginx,
		VersionLocalStack: config.VersionLocalStack,
		VersionPrometheus: config.VersionPrometheus,
		VersionGrafana:    config.VersionGrafana,
		VersionLoki:       config.VersionLoki,
		VersionPromtail:   config.VersionPromtail,
	}

	if f, ok := registry.SupportedFrontends[cfg.Frontend]; ok {
		data.Frontend = &f
	}
	if b, ok := registry.SupportedBackends[cfg.Backend]; ok {
		data.Backend = &b
	}

	var err error

	if cfg.Database == config.DatabaseMySQL || cfg.Database == config.DatabasePostgres {
		data.DBPassword, err = security.GeneratePassword(20)
		if err != nil {
			return nil, fmt.Errorf("generating DB password: %w", err)
		}
	}

	if cfg.Database == config.DatabaseMySQL {
		data.DBRootPassword, err = security.GeneratePassword(24)
		if err != nil {
			return nil, fmt.Errorf("generating DB root password: %w", err)
		}
	}

	if cfg.Monitoring {
		data.GrafanaPassword, err = security.GeneratePassword(16)
		if err != nil {
			return nil, fmt.Errorf("generating Grafana password: %w", err)
		}
	}

	return data, nil
}

// RenderCompose renders the docker-compose.yml content using text/template.
func RenderCompose(data *ComposeData) (string, error) {
	tmpl, err := template.ParseFS(FS, "templates/compose.yml.tmpl")
	if err != nil {
		return "", fmt.Errorf("parsing compose template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing compose template: %w", err)
	}

	return buf.String(), nil
}

// RenderEnvFile renders the .env file content using text/template.
func RenderEnvFile(data *ComposeData) (string, error) {
	tmpl, err := template.ParseFS(FS, "templates/env.tmpl")
	if err != nil {
		return "", fmt.Errorf("parsing env template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing env template: %w", err)
	}

	return strings.TrimSpace(buf.String()) + "\n", nil
}

// RenderEnvExampleFile renders the .env.example file content with masked passwords.
func RenderEnvExampleFile(data *ComposeData) (string, error) {
	clone := *data
	if clone.DBPassword != "" {
		clone.DBPassword = "your_db_password"
	}
	if clone.DBRootPassword != "" {
		clone.DBRootPassword = "your_db_root_password"
	}
	if clone.GrafanaPassword != "" {
		clone.GrafanaPassword = "your_grafana_password"
	}
	return RenderEnvFile(&clone)
}

// RenderGitignore returns .gitignore content appropriate for a dev-box project.
func RenderGitignore() string {
	return ".env\nnode_modules/\n__pycache__/\n*.pyc\n.vscode/\n.idea/\n*.class\ntarget/\nvendor/\ndist/\nbuild/\ntmp/\n"
}

// RenderPrometheusConfig returns the prometheus.yml configuration content.
func RenderPrometheusConfig(backendPort int) (string, error) {
	tmpl, err := template.ParseFS(FS, "templates/prometheus.yml.tmpl")
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, struct{ BackendPort int }{BackendPort: backendPort}); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func RenderGrafanaDatasource() (string, error) {
	b, err := FS.ReadFile("templates/grafana-datasource.yml.tmpl")
	return string(b), err
}

// RenderPromtailConfig returns the promtail-config.yml configuration content.
func RenderPromtailConfig() (string, error) {
	b, err := FS.ReadFile("templates/promtail-config.yml.tmpl")
	return string(b), err
}
