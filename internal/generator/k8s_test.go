package generator

import (
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func TestGenerateK8sManifests_Frontend(t *testing.T) {
	cfg := config.ProjectConfig{Frontend: config.FrontendReact, Backend: "none", Database: "none", Cache: "none"}
	manifests, err := GenerateK8sManifests(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, ok := manifests["frontend.yaml"]
	if !ok {
		t.Fatal("expected frontend.yaml in manifests")
	}
	checks := []string{"resources:", "limits:", "requests:", "livenessProbe:", "readinessProbe:", "runAsNonRoot: true"}
	for _, check := range checks {
		if !strings.Contains(content.Content, check) {
			t.Errorf("expected frontend manifest to contain %q", check)
		}
	}
	if !strings.Contains(content.Content, "image: dev-box-frontend:latest") {
		t.Error("expected frontend manifest to use the generated app image")
	}
	if !strings.Contains(content.Content, "imagePullPolicy: IfNotPresent") {
		t.Error("expected frontend manifest to support locally built images")
	}
	if !strings.Contains(content.Content, "path: /") {
		t.Error("expected frontend probe to target /")
	}
	if strings.Contains(content.Content, "path: /health") {
		t.Error("frontend probe should not assume a /health endpoint")
	}
}

func TestGenerateK8sManifests_DBWithPVC(t *testing.T) {
	tests := []struct {
		name     string
		database string
		image    string
		pvcPath  string
	}{
		{"MySQL", "mysql", "mysql:" + config.VersionMySQL, "/var/lib/mysql"},
		{"Postgres", "postgres", "postgres:" + config.VersionPostgres, "/var/lib/postgresql/data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.ProjectConfig{Frontend: "none", Backend: "none", Database: config.DatabaseType(tt.database), Cache: "none"}
			manifests, err := GenerateK8sManifests(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			content, ok := manifests["db.yaml"]
			if !ok {
				t.Fatal("expected db.yaml in manifests")
			}
			checks := []string{
				"PersistentVolumeClaim",
				tt.pvcPath,
				"resources:",
				"secretKeyRef:",
				tt.image,
			}
			for _, check := range checks {
				if !strings.Contains(content.Content, check) {
					t.Errorf("expected db manifest to contain %q", check)
				}
			}
			if strings.Contains(content.Content, "runAsNonRoot: true") {
				t.Error("DB should NOT have runAsNonRoot: true")
			}
		})
	}
}

func TestGenerateK8sManifests_AllBackends(t *testing.T) {
	backends := []string{"node", "python", "go", "php", "java"}
	for _, backend := range backends {
		t.Run(backend, func(t *testing.T) {
			cfg := config.ProjectConfig{Frontend: "none", Backend: config.BackendType(backend), Database: "none", Cache: "none"}
			manifests, err := GenerateK8sManifests(cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			content, ok := manifests["api.yaml"]
			if !ok {
				t.Fatal("expected api.yaml in manifests")
			}
			if !strings.Contains(content.Content, "resources:") {
				t.Error("expected manifest to contain resource limits")
			}
			if !strings.Contains(content.Content, "livenessProbe:") {
				t.Error("expected manifest to contain livenessProbe")
			}
			if !strings.Contains(content.Content, "image: dev-box-api:latest") {
				t.Error("expected backend manifest to use the generated app image")
			}
			if !strings.Contains(content.Content, "path: /health") {
				t.Error("expected backend probe to target /health")
			}
			if !strings.Contains(content.Content, "runAsUser:") || !strings.Contains(content.Content, "allowPrivilegeEscalation: false") {
				t.Error("expected backend manifest to enforce non-root container security")
			}
		})
	}
}
