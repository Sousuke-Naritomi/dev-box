package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/port"
	"github.com/Sousuke-Naritomi/dev-box/internal/registry"
	"github.com/Sousuke-Naritomi/dev-box/internal/security"
)

// ComposeData holds resolved configuration for docker-compose template rendering.
type ComposeData struct {
	Config          config.ProjectConfig
	Frontend        *registry.FrontendFramework
	Backend         *registry.BackendFramework
	DBPort          int
	CachePort       int
	FrontendPort    int
	BackendPort     int
	LocalStackPort  int
	NginxPort       int
	PrometheusPort  int
	GrafanaPort     int
	LokiPort        int
	DBPassword      string
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

	nextPort := 13000
	allocate := func() (int, error) {
		p, err := port.FindAvailable(nextPort)
		if err != nil {
			return 0, err
		}
		nextPort = p + 1
		return p, nil
	}

	var err error

	if cfg.Database == "mysql" || cfg.Database == "postgres" {
		if data.DBPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating DB port: %w", err)
		}
		data.DBPassword, err = security.GeneratePassword(20)
		if err != nil {
			return nil, fmt.Errorf("generating DB password: %w", err)
		}
	}

	if cfg.Cache == "redis" {
		if data.CachePort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating cache port: %w", err)
		}
	}

	if data.Frontend != nil {
		if data.FrontendPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating frontend port: %w", err)
		}
	}

	if data.Backend != nil {
		if data.BackendPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating backend port: %w", err)
		}
	}

	if cfg.LocalStack {
		if data.LocalStackPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating LocalStack port: %w", err)
		}
	}

	if cfg.Nginx {
		if data.NginxPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating Nginx port: %w", err)
		}
	}

	if cfg.Monitoring {
		if data.PrometheusPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating Prometheus port: %w", err)
		}
		if data.GrafanaPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating Grafana port: %w", err)
		}
		if data.LokiPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating Loki port: %w", err)
		}
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
func RenderPrometheusConfig() (string, error) {
	b, err := FS.ReadFile("templates/prometheus.yml.tmpl")
	return string(b), err
}

// RenderPromtailConfig returns the promtail-config.yml configuration content.
func RenderPromtailConfig() (string, error) {
	b, err := FS.ReadFile("templates/promtail-config.yml.tmpl")
	return string(b), err
}
