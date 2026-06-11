package generator

import (
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func TestGenerateCICD_BackendUsesDynamicComposePort(t *testing.T) {
	cfg := config.ProjectConfig{
		Frontend: config.FrontendNone,
		Backend:  config.BackendNode,
	}

	result, err := GenerateCICD(cfg, 3000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	checks := []string{
		"docker compose config",
		"aquasecurity/trivy-action@0.32.0",
		"docker compose port api 3000",
		"http://localhost:${API_PORT}/health",
	}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Errorf("expected generated CI to contain %q", check)
		}
	}
	if strings.Contains(result, "http://localhost:3000/health") {
		t.Fatal("generated CI must not assume the host port matches the container port")
	}
	if strings.Contains(result, "go-version-file") {
		t.Fatal("generated CI must not assume a Go module exists")
	}
	if strings.Contains(result, "trivy-action@master") {
		t.Fatal("generated CI must pin the Trivy action instead of tracking master")
	}
}

func TestGenerateCICD_FrontendOnlyUsesFrontendProbe(t *testing.T) {
	cfg := config.ProjectConfig{
		Frontend: config.FrontendReact,
		Backend:  config.BackendNone,
	}

	result, err := GenerateCICD(cfg, 0, 5173)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "docker compose port frontend 5173") {
		t.Fatal("expected generated CI to inspect the frontend host port")
	}
	if strings.Contains(result, "docker compose port api") {
		t.Fatal("frontend-only CI must not inspect the api service")
	}
}
