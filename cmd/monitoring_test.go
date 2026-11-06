package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

func TestInitWritesAlloyConfigAndNoPromtail(t *testing.T) {
	cfg := fullStack()
	cfg.Monitoring = true
	files, _, err := buildInitFiles(cfg, &generator.DefaultGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		filepath.Join("monitoring", "config.alloy"),
		filepath.Join("monitoring", "prometheus.yml"),
		filepath.Join("monitoring", "grafana", "provisioning", "datasources", "prometheus.yml"),
	} {
		if _, ok := files[want]; !ok {
			t.Errorf("expected %s", want)
		}
	}
	for path := range files {
		if strings.Contains(strings.ToLower(path), "promtail") {
			t.Errorf("Promtail files must not be generated: %s", path)
		}
	}
	if !strings.Contains(files[filepath.Join("monitoring", "grafana", "provisioning", "datasources", "prometheus.yml")].Content, "type: loki") {
		t.Error("the Loki datasource must be provisioned so collected logs can be queried")
	}

	cfg.Monitoring = false
	files, _, err = buildInitFiles(cfg, &generator.DefaultGenerator{})
	if err != nil {
		t.Fatal(err)
	}
	for path := range files {
		if strings.HasPrefix(path, "monitoring") {
			t.Errorf("monitoring files must only be generated when monitoring is enabled: %s", path)
		}
	}
}

func TestInitWarnsThatLocalStackIsUnmaintained(t *testing.T) {
	var with, without bytes.Buffer
	cfg := fullStack()
	cfg.LocalStack = true
	if err := runInitLogic(context.Background(), cfg, "p", true, "text", &with, fs.NewMockFileSystem(), &generator.DefaultGenerator{}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"LocalStack is pinned to " + config.VersionLocalStack, "no longer maintained", "LOCALSTACK_AUTH_TOKEN"} {
		if !strings.Contains(with.String(), want) {
			t.Errorf("expected the LocalStack note to contain %q:\n%s", want, with.String())
		}
	}

	cfg.LocalStack = false
	if err := runInitLogic(context.Background(), cfg, "p", true, "text", &without, fs.NewMockFileSystem(), &generator.DefaultGenerator{}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(without.String(), "LocalStack") {
		t.Errorf("the note must only appear when LocalStack is selected:\n%s", without.String())
	}

	var asJSON bytes.Buffer
	cfg.LocalStack = true
	if err := runInitLogic(context.Background(), cfg, "p", true, "json", &asJSON, fs.NewMockFileSystem(), &generator.DefaultGenerator{}); err != nil {
		t.Fatal(err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(asJSON.Bytes(), &parsed); err != nil {
		t.Errorf("JSON output must stay machine readable, the note must not leak into it: %v", err)
	}
}
