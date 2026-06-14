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
	data.DBPassword = "testpass"
	result, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var parsed struct {
		Services map[string]struct {
			Image       string   `yaml:"image"`
			Restart     string   `yaml:"restart"`
			Ports       []string `yaml:"ports"`
			Healthcheck struct {
				Test []string `yaml:"test"`
			} `yaml:"healthcheck"`
		} `yaml:"services"`
		Volumes map[string]interface{} `yaml:"volumes"`
	}

	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("generated compose is not valid YAML: %v\n%s", err, result)
	}

	db, ok := parsed.Services["db"]
	if !ok {
		t.Fatal("expected 'db' service in compose")
	}

	if !strings.HasPrefix(db.Image, "mysql:") {
		t.Errorf("expected 'mysql' image, got %s", db.Image)
	}

	if db.Restart != "unless-stopped" {
		t.Errorf("expected restart 'unless-stopped', got %s", db.Restart)
	}

	if len(db.Ports) == 0 || db.Ports[0] != "127.0.0.1:0:3306" {
		t.Errorf("expected localhost-only port mapping, got %v", db.Ports)
	}

	if len(db.Healthcheck.Test) == 0 || db.Healthcheck.Test[1] != "mysqladmin" {
		t.Errorf("expected healthcheck with mysqladmin, got %v", db.Healthcheck.Test)
	}

	if _, ok := parsed.Volumes["db-data"]; !ok {
		t.Errorf("expected volume 'db-data'")
	}
}

func TestRenderCompose_Postgres(t *testing.T) {
	cfg := config.ProjectConfig{Database: "postgres", Frontend: "none", Backend: "none", Cache: "none"}
	data, _ := ResolveComposeData(cfg)
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

func TestRenderCompose_PHPUsesFrontController(t *testing.T) {
	cfg := config.ProjectConfig{Backend: config.BackendPHP, Frontend: config.FrontendNone, Database: config.DatabaseNone, Cache: config.CacheNone}
	data, err := ResolveComposeData(cfg)
	if err != nil {
		t.Fatalf("ResolveComposeData failed: %v", err)
	}
	result, err := RenderCompose(data)
	if err != nil {
		t.Fatalf("RenderCompose failed: %v", err)
	}
	if !strings.Contains(result, "php -S 0.0.0.0:80 index.php") {
		t.Fatal("PHP Compose service must route all requests through index.php")
	}
	var parsed struct {
		Volumes map[string]any `yaml:"volumes"`
	}
	if err := yaml.Unmarshal([]byte(result), &parsed); err != nil {
		t.Fatalf("generated PHP Compose is invalid YAML: %v", err)
	}
	if parsed.Volumes != nil {
		t.Fatalf("PHP-only Compose must omit an empty volumes section: %#v", parsed.Volumes)
	}
}

func TestResolveComposeDataDetectsNamedVolumes(t *testing.T) {
	tests := []config.ProjectConfig{
		{Frontend: config.FrontendReact, Backend: config.BackendNone, Database: config.DatabaseNone, Cache: config.CacheNone},
		{Frontend: config.FrontendNone, Backend: config.BackendNode, Database: config.DatabaseNone, Cache: config.CacheNone},
		{Frontend: config.FrontendNone, Backend: config.BackendGo, Database: config.DatabaseNone, Cache: config.CacheNone},
		{Frontend: config.FrontendNone, Backend: config.BackendNone, Database: config.DatabaseNone, Cache: config.CacheNone, LocalStack: true},
		{Frontend: config.FrontendNone, Backend: config.BackendNone, Database: config.DatabaseNone, Cache: config.CacheNone, Monitoring: true},
	}
	for _, cfg := range tests {
		data, err := ResolveComposeData(cfg)
		if err != nil {
			t.Fatalf("ResolveComposeData failed: %v", err)
		}
		if !data.HasVolumes {
			t.Fatalf("expected named volumes for config %#v", cfg)
		}
	}
}

func TestRenderEnvFile_MySQL(t *testing.T) {
	cfg := config.ProjectConfig{Database: "mysql", Backend: "node", Frontend: "none", Cache: "none"}
	data, _ := ResolveComposeData(cfg)
	data.DBPassword = "secretpass"
	data.DBRootPassword = "rootsecret"
	result, err := RenderEnvFile(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "MYSQL_ROOT_PASSWORD='rootsecret'") {
		t.Error("expected env file to contain MYSQL_ROOT_PASSWORD with root password")
	}
	if !strings.Contains(result, "MYSQL_PASSWORD='secretpass'") {
		t.Error("expected env file to contain MYSQL_PASSWORD with app password")
	}
	if result == "" || strings.Contains(result, "MYSQL_ROOT_PASSWORD=secretpass") {
		t.Error("root password must differ from app password")
	}
}

func TestRenderGitignore(t *testing.T) {
	result := RenderGitignore()
	if !strings.Contains(result, ".env") {
		t.Error("expected gitignore to contain .env")
	}
}
