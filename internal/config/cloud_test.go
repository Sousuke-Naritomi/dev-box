package config

import (
	"strings"
	"testing"
)

func baseConfig() ProjectConfig {
	return ProjectConfig{Frontend: FrontendReact, Backend: BackendGo, Database: DatabasePostgres, Cache: CacheRedis}
}

func TestValidateCloud(t *testing.T) {
	tests := []struct {
		cloud   CloudProvider
		wantErr bool
	}{
		{"", false},
		{CloudNone, false},
		{CloudAWS, false},
		{CloudGCP, false},
		{CloudAzure, false},
		{"digitalocean", true},
		{"AWS", true},
		{" aws", true},
	}
	for _, tt := range tests {
		t.Run(string(tt.cloud), func(t *testing.T) {
			cfg := baseConfig()
			cfg.Cloud = tt.cloud
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "cloud provider") {
				t.Errorf("error should name the problem: %v", err)
			}
		})
	}
}

func TestHasCloud(t *testing.T) {
	for cloud, want := range map[CloudProvider]bool{"": false, CloudNone: false, CloudAWS: true, CloudGCP: true, CloudAzure: true} {
		if got := (ProjectConfig{Cloud: cloud}).HasCloud(); got != want {
			t.Errorf("HasCloud(%q) = %v, want %v", cloud, got, want)
		}
	}
}

func TestProjectFileWithoutCloudStillParses(t *testing.T) {
	cfg, err := ParseProjectFile([]byte("version: 1\nfrontend: react\nbackend: go\ndatabase: postgres\ncache: redis\nnginx: false\nlocalstack: false\nmonitoring: false\ncicd: false\n"))
	if err != nil {
		t.Fatalf("project files written before the cloud option must keep working: %v", err)
	}
	if cfg.HasCloud() || cfg.Cloud != CloudNone {
		t.Errorf("missing cloud must be normalized to none, got %q", cfg.Cloud)
	}
}

func TestProjectFileParsesCloud(t *testing.T) {
	cfg, err := ParseProjectFile([]byte("version: 1\nfrontend: none\nbackend: go\ndatabase: none\ncache: none\ncloud: gcp\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Cloud != CloudGCP {
		t.Fatalf("cloud = %q, want gcp", cfg.Cloud)
	}
}

func TestProjectFileRejectsUnknownCloud(t *testing.T) {
	_, err := ParseProjectFile([]byte("version: 1\nfrontend: none\nbackend: go\ndatabase: none\ncache: none\ncloud: heroku\n"))
	if err == nil || !strings.Contains(err.Error(), "cloud provider") {
		t.Fatalf("expected an unsupported cloud provider error, got %v", err)
	}
}

func TestRenderProjectFileAlwaysStatesCloud(t *testing.T) {
	for cloud, want := range map[CloudProvider]string{"": "cloud: none\n", CloudNone: "cloud: none\n", CloudAWS: "cloud: aws\n"} {
		cfg := baseConfig()
		cfg.Cloud = cloud
		out, err := RenderProjectFile(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out, want) {
			t.Errorf("cloud %q: expected %q in\n%s", cloud, want, out)
		}
	}
}

func TestProjectFileRoundTripPreservesCloud(t *testing.T) {
	for _, cloud := range []CloudProvider{CloudNone, CloudAWS, CloudGCP, CloudAzure} {
		cfg := baseConfig()
		cfg.Cloud = cloud
		rendered, err := RenderProjectFile(cfg)
		if err != nil {
			t.Fatal(err)
		}
		parsed, err := ParseProjectFile([]byte(rendered))
		if err != nil {
			t.Fatalf("%s: %v\n%s", cloud, err, rendered)
		}
		if parsed != cfg {
			t.Errorf("round trip changed the config: got %+v, want %+v", parsed, cfg)
		}
	}
}
