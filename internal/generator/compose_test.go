package generator

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func TestRenderCompose_NoVersion(t *testing.T) {
	cfg := config.ProjectConfig{
		Frontend: "none", Backend: "none", Database: "none", Cache: "none",
	}
	data, _ := ResolveComposeData(cfg)
	result, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "version:") {
		t.Error("compose output should not contain deprecated 'version' field")
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("generated compose is not valid YAML: %v\n%s", err, result)
	}
}

func TestRenderCompose_MySQL(t *testing.T) {
	cfg := config.ProjectConfig{Database: "mysql", Frontend: "none", Backend: "none", Cache: "none"}
	data, _ := ResolveComposeData(cfg)
	data.DBPort = 3306
	data.DBPassword = "testpass"
	result, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checks := []string{
		"image: mysql:8.0",
		"restart: unless-stopped",
		"healthcheck:",
		"mysqladmin",
		"db-data:/var/lib/mysql",
		"dev-network",
		"volumes:",
		"networks:",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected compose to contain %q", check)
		}
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("generated compose is not valid YAML: %v\n%s", err, result)
	}
}

func TestRenderCompose_Postgres(t *testing.T) {
	cfg := config.ProjectConfig{Database: "postgres", Frontend: "none", Backend: "none", Cache: "none"}
	data, _ := ResolveComposeData(cfg)
	data.DBPort = 5432
	data.DBPassword = "testpass"
	result, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	checks := []string{
		"postgres:16-alpine",
		"pg_isready",
		"db-data:/var/lib/postgresql/data",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected compose to contain %q", check)
		}
	}
}

func TestRenderCompose_FullStack(t *testing.T) {
	cfg := config.ProjectConfig{
		Frontend: "react", Backend: "node", Database: "mysql", Cache: "redis",
		Nginx: true, LocalStack: true, Monitoring: true,
	}
	data, _ := ResolveComposeData(cfg)
	data.DBPort = 3306
	data.CachePort = 6379
	data.FrontendPort = 5173
	data.BackendPort = 3000
	data.NginxPort = 80
	data.LocalStackPort = 4566
	data.PrometheusPort = 9090
	data.GrafanaPort = 3001
	data.LokiPort = 3100
	data.DBPassword = "test"
	data.GrafanaPassword = "graftest"
	result, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	services := []string{"db:", "cache:", "frontend:", "api:", "nginx:", "localstack:", "prometheus:", "grafana:", "loki:", "promtail:"}
	for _, svc := range services {
		if !strings.Contains(result, svc) {
			t.Errorf("expected compose to contain service %q", svc)
		}
	}

	if !strings.Contains(result, "condition: service_healthy") {
		t.Error("expected api to depend on db with service_healthy condition")
	}
}

func TestRenderCompose_BackendDependsOnDB(t *testing.T) {
	cfg := config.ProjectConfig{Database: "postgres", Backend: "go", Frontend: "none", Cache: "none"}
	data, _ := ResolveComposeData(cfg)
	data.DBPort = 5432
	data.BackendPort = 8080
	data.DBPassword = "test"
	result, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "condition: service_healthy") {
		t.Error("expected api to have depends_on with service_healthy condition when DB is present")
	}
}

func TestRenderCompose_JavaBackend(t *testing.T) {
	cfg := config.ProjectConfig{Backend: "java", Frontend: "none", Database: "none", Cache: "none"}
	data, _ := ResolveComposeData(cfg)
	data.BackendPort = 8080
	result, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "eclipse-temurin:17-jdk") {
		t.Error("expected java backend to use eclipse-temurin image")
	}
	if strings.Contains(result, "tail -f /dev/null") {
		t.Error("java backend must not use 'tail -f /dev/null'")
	}
	if !strings.Contains(result, "javac Main.java") {
		t.Error("expected java backend to compile and run Main.java")
	}
}

func TestRenderEnvFile_MySQL(t *testing.T) {
	cfg := config.ProjectConfig{Database: "mysql", Backend: "node", Frontend: "none", Cache: "none"}
	data, _ := ResolveComposeData(cfg)
	data.DBPassword = "secretpass"
	result, err := RenderEnvFile(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "MYSQL_ROOT_PASSWORD=secretpass") {
		t.Error("expected env file to contain MYSQL_ROOT_PASSWORD")
	}
	if !strings.Contains(result, "BACKEND_PORT=3000") {
		t.Error("expected env file to contain BACKEND_PORT")
	}
}

func TestRenderGitignore(t *testing.T) {
	result := RenderGitignore()
	if !strings.Contains(result, ".env") {
		t.Error("expected gitignore to contain .env")
	}
}
