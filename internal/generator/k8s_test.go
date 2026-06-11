package generator

import (
	"strings"
	"testing"
)

func TestGenerateK8sManifests_Frontend(t *testing.T) {
	manifests, err := GenerateK8sManifests("react", "none", "none", "none")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	content, ok := manifests["frontend.yaml"]
	if !ok {
		t.Fatal("expected frontend.yaml in manifests")
	}
	checks := []string{"resources:", "limits:", "requests:", "livenessProbe:", "readinessProbe:", "runAsNonRoot: true"}
	for _, check := range checks {
		if !strings.Contains(content, check) {
			t.Errorf("expected frontend manifest to contain %q", check)
		}
	}
}

func TestGenerateK8sManifests_DBWithPVC(t *testing.T) {
	tests := []struct {
		name     string
		database string
		image    string
		pvcPath  string
	}{
		{"MySQL", "mysql", "mysql:8.0", "/var/lib/mysql"},
		{"Postgres", "postgres", "postgres:16-alpine", "/var/lib/postgresql/data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifests, err := GenerateK8sManifests("none", "none", tt.database, "none")
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
				if !strings.Contains(content, check) {
					t.Errorf("expected db manifest to contain %q", check)
				}
			}
			if strings.Contains(content, "runAsNonRoot: true") {
				t.Error("DB should NOT have runAsNonRoot: true")
			}
		})
	}
}

func TestGenerateK8sManifests_AllBackends(t *testing.T) {
	backends := []string{"node", "python", "go", "php", "java"}
	for _, backend := range backends {
		t.Run(backend, func(t *testing.T) {
			manifests, err := GenerateK8sManifests("none", backend, "none", "none")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			content, ok := manifests["api.yaml"]
			if !ok {
				t.Fatal("expected api.yaml in manifests")
			}
			if !strings.Contains(content, "resources:") {
				t.Error("expected manifest to contain resource limits")
			}
			if !strings.Contains(content, "livenessProbe:") {
				t.Error("expected manifest to contain livenessProbe")
			}
		})
	}
}
