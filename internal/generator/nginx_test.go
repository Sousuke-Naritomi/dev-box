package generator

import (
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func TestGenerateNginxConf_FrontendOnlySkipsBackend(t *testing.T) {
	cfg := config.ProjectConfig{
		Frontend: config.FrontendReact,
		Backend:  config.BackendNone,
	}

	result, err := GenerateNginxConf(cfg, 0, 5173)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "zone=api_limit") {
		t.Fatal("frontend-only nginx config must not allocate an unused api rate-limit zone")
	}
	if !strings.Contains(result, "zone=general_limit") {
		t.Fatal("frontend-only nginx config should allocate the frontend rate-limit zone")
	}
	if strings.Contains(result, "api:0") {
		t.Fatal("frontend-only nginx config must not point to api:0")
	}
	if strings.Contains(result, "location /api/") {
		t.Fatal("frontend-only nginx config must not include API routing")
	}
	if !strings.Contains(result, "upstream frontend") {
		t.Fatal("expected frontend upstream")
	}
	if !strings.Contains(result, "server frontend:5173;") {
		t.Fatal("expected frontend upstream to use the configured frontend port")
	}
}

func TestGenerateNginxConf_BackendOnlySkipsFrontend(t *testing.T) {
	cfg := config.ProjectConfig{
		Frontend: config.FrontendNone,
		Backend:  config.BackendNode,
	}

	result, err := GenerateNginxConf(cfg, 3000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result, "zone=api_limit") {
		t.Fatal("backend-only nginx config should allocate the API rate-limit zone")
	}
	if strings.Contains(result, "zone=general_limit") {
		t.Fatal("backend-only nginx config must not allocate an unused frontend rate-limit zone")
	}
	if !strings.Contains(result, "server api:3000;") {
		t.Fatal("expected backend upstream")
	}
	if strings.Contains(result, "upstream frontend") {
		t.Fatal("backend-only nginx config must not include frontend upstream")
	}
}

func TestGenerateNginxConf_NoServicesSkipsUnusedRateLimitZones(t *testing.T) {
	cfg := config.ProjectConfig{
		Frontend: config.FrontendNone,
		Backend:  config.BackendNone,
	}

	result, err := GenerateNginxConf(cfg, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(result, "limit_req_zone") {
		t.Fatal("nginx config without proxied services must not allocate unused rate-limit zones")
	}
}
