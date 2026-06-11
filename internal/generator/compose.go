// Package generator provides functions for generating Docker Compose,
// Kubernetes, Terraform, CI/CD, Nginx, and scaffold configurations.
package generator

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/port"
	"github.com/Sousuke-Naritomi/dev-box/internal/security"
)

// ComposeData holds resolved configuration for docker-compose template rendering.
type ComposeData struct {
	Config          config.ProjectConfig
	DBPort          int
	CachePort       int
	FrontendPort    int
	BackendPort     int
	LocalStackPort  int
	NginxPort       int
	PrometheusPort  int
	GrafanaPort     int
	LokiPort        int
	BackendInternal int
	DBPassword      string
	GrafanaPassword string
	HasVolumes      bool
}

// ResolveComposeData resolves ports and generates passwords based on the project config.
func ResolveComposeData(cfg config.ProjectConfig) (*ComposeData, error) {
	data := &ComposeData{
		Config:          cfg,
		BackendInternal: cfg.BackendInternalPort(),
		HasVolumes:      cfg.Database != "none" && cfg.Database != "",
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

	if cfg.Frontend == "react" || cfg.Frontend == "vue" {
		if data.FrontendPort, err = allocate(); err != nil {
			return nil, fmt.Errorf("allocating frontend port: %w", err)
		}
	}

	if cfg.Backend != "none" && cfg.Backend != "" {
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

const composeTemplate = `services:
{{- if and (ne .Config.Database "none") (ne .Config.Database "") }}

  db:
    image: {{ if eq .Config.Database "mysql" }}mysql:8.0{{ else }}postgres:16-alpine{{ end }}
    restart: unless-stopped
    env_file:
      - .env
    ports:
      - "{{ .DBPort }}:{{ if eq .Config.Database "mysql" }}3306{{ else }}5432{{ end }}"
    volumes:
      - db-data:{{ if eq .Config.Database "mysql" }}/var/lib/mysql{{ else }}/var/lib/postgresql/data{{ end }}
    healthcheck:
      test: {{ if eq .Config.Database "mysql" }}["CMD", "mysqladmin", "ping", "-h", "localhost"]{{ else }}["CMD-SHELL", "pg_isready -U postgres"]{{ end }}
      interval: 10s
      timeout: 5s
      retries: 5
      start_period: 30s
    networks:
      - dev-network
{{- end }}

{{- if eq .Config.Cache "redis" }}

  cache:
    image: redis:7-alpine
    restart: unless-stopped
    ports:
      - "{{ .CachePort }}:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - dev-network
{{- end }}

{{- if and (ne .Config.Frontend "none") (ne .Config.Frontend "") }}

  frontend:
    image: node:20-alpine
    restart: unless-stopped
    working_dir: /app
    volumes:
      - ./frontend:/app
      - frontend-node-modules:/app/node_modules
    command: sh -c "npm install && npm run dev -- --host 0.0.0.0"
    ports:
      - "{{ .FrontendPort }}:5173"
    networks:
      - dev-network
{{- end }}

{{- if and (ne .Config.Backend "none") (ne .Config.Backend "") }}

  api:
    {{- if eq .Config.Backend "node" }}
    image: node:20-alpine
    working_dir: /app
    volumes:
      - ./backend:/app
      - backend-node-modules:/app/node_modules
    command: sh -c "npm install && npm start"
    {{- else if eq .Config.Backend "python" }}
    image: python:3.12-slim
    working_dir: /app
    volumes:
      - ./backend:/app
    command: sh -c "pip install -r requirements.txt && python main.py"
    {{- else if eq .Config.Backend "go" }}
    image: golang:1.22-alpine
    working_dir: /app
    volumes:
      - ./backend:/app
      - go-mod-cache:/go/pkg/mod
    command: sh -c "go run main.go"
    {{- else if eq .Config.Backend "php" }}
    image: php:8.3-apache
    volumes:
      - ./backend:/var/www/html
    {{- else if eq .Config.Backend "java" }}
    image: eclipse-temurin:17-jdk
    working_dir: /app
    volumes:
      - ./backend:/app
    command: sh -c "javac Main.java && java Main"
    {{- end }}
    restart: unless-stopped
    ports:
      - "{{ .BackendPort }}:{{ .BackendInternal }}"
    {{- if and (ne .Config.Database "none") (ne .Config.Database "") }}
    depends_on:
      db:
        condition: service_healthy
    {{- end }}
    env_file:
      - .env
    networks:
      - dev-network
{{- end }}

{{- if .Config.Nginx }}

  nginx:
    image: nginx:alpine
    restart: unless-stopped
    ports:
      - "{{ .NginxPort }}:80"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
    depends_on:
      {{- if and (ne .Config.Backend "none") (ne .Config.Backend "") }}
      - api
      {{- end }}
      {{- if and (ne .Config.Frontend "none") (ne .Config.Frontend "") }}
      - frontend
      {{- end }}
    networks:
      - dev-network
{{- end }}

{{- if .Config.LocalStack }}

  localstack:
    image: localstack/localstack:latest
    restart: unless-stopped
    ports:
      - "{{ .LocalStackPort }}:4566"
    environment:
      - SERVICES=s3,sqs,sns,dynamodb
      - DEFAULT_REGION=ap-northeast-1
    volumes:
      - localstack-data:/var/lib/localstack
    networks:
      - dev-network
{{- end }}

{{- if .Config.Monitoring }}

  prometheus:
    image: prom/prometheus:latest
    restart: unless-stopped
    ports:
      - "{{ .PrometheusPort }}:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro
      - prometheus-data:/prometheus
    networks:
      - dev-network

  grafana:
    image: grafana/grafana:latest
    restart: unless-stopped
    ports:
      - "{{ .GrafanaPort }}:3000"
    env_file:
      - .env
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    volumes:
      - grafana-data:/var/lib/grafana
    depends_on:
      - prometheus
      - loki
    networks:
      - dev-network

  loki:
    image: grafana/loki:latest
    restart: unless-stopped
    ports:
      - "{{ .LokiPort }}:3100"
    volumes:
      - loki-data:/loki
    networks:
      - dev-network

  promtail:
    image: grafana/promtail:latest
    restart: unless-stopped
    volumes:
      - /var/log:/var/log:ro
      - ./monitoring/promtail-config.yml:/etc/promtail/config.yml:ro
    depends_on:
      - loki
    networks:
      - dev-network
{{- end }}

volumes:
{{- if .HasVolumes }}
  db-data:
{{- end }}
{{- if and (ne .Config.Frontend "none") (ne .Config.Frontend "") }}
  frontend-node-modules:
{{- end }}
{{- if eq .Config.Backend "node" }}
  backend-node-modules:
{{- end }}
{{- if eq .Config.Backend "go" }}
  go-mod-cache:
{{- end }}
{{- if .Config.LocalStack }}
  localstack-data:
{{- end }}
{{- if .Config.Monitoring }}
  prometheus-data:
  grafana-data:
  loki-data:
{{- end }}

networks:
  dev-network:
    driver: bridge
`

var composeTmpl = template.Must(template.New("compose").Parse(composeTemplate))

// RenderCompose renders the docker-compose.yml content using text/template.
func RenderCompose(data *ComposeData) (string, error) {
	var buf bytes.Buffer
	if err := composeTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute compose template: %w", err)
	}

	return buf.String(), nil
}

const envTemplate = `# Generated by dev-box — do not commit this file
{{- if eq .Config.Database "mysql" }}
MYSQL_ROOT_PASSWORD={{ .DBPassword }}
MYSQL_DATABASE=devbox
MYSQL_USER=devuser
MYSQL_PASSWORD={{ .DBPassword }}
DB_HOST=db
DB_PORT=3306
{{- end }}
{{- if eq .Config.Database "postgres" }}
POSTGRES_PASSWORD={{ .DBPassword }}
POSTGRES_DB=devbox
POSTGRES_USER=devuser
DB_HOST=db
DB_PORT=5432
{{- end }}
{{- if eq .Config.Cache "redis" }}
REDIS_HOST=cache
REDIS_PORT=6379
{{- end }}
{{- if and (ne .Config.Backend "none") (ne .Config.Backend "") }}
BACKEND_PORT={{ .BackendInternal }}
{{- end }}
{{- if .GrafanaPassword }}
GRAFANA_PASSWORD={{ .GrafanaPassword }}
{{- end }}
`

var envTmpl = template.Must(template.New("env").Parse(envTemplate))

// RenderEnvFile renders the .env file content using text/template.
func RenderEnvFile(data *ComposeData) (string, error) {
	var buf bytes.Buffer
	if err := envTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute env template: %w", err)
	}

	return strings.TrimSpace(buf.String()) + "\n", nil
}

// RenderGitignore returns .gitignore content appropriate for a dev-box project.
func RenderGitignore() string {
	return `.env
node_modules/
__pycache__/
*.pyc
.vscode/
.idea/
*.class
target/
vendor/
dist/
build/
tmp/
`
}

// RenderPrometheusConfig returns the prometheus.yml configuration content.
func RenderPrometheusConfig() string {
	return `global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: "prometheus"
    static_configs:
      - targets: ["localhost:9090"]
`
}

// RenderPromtailConfig returns the promtail-config.yml configuration content.
func RenderPromtailConfig() string {
	return `server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: system
    static_configs:
      - targets:
          - localhost
        labels:
          job: varlogs
          __path__: /var/log/*.log
`
}
