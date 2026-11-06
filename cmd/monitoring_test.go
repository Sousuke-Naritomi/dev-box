package cmd

import (
	"path/filepath"
	"strings"
	"testing"

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
