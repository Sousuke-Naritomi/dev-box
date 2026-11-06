package generator

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func renderMonitoringCompose(t *testing.T) (string, map[string]any) {
	t.Helper()
	cfg := config.ProjectConfig{
		Frontend: config.FrontendNone, Backend: config.BackendGo, Database: config.DatabaseNone, Cache: config.CacheNone,
		Monitoring: true,
	}
	data, err := ResolveComposeData(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rendered, err := RenderCompose(data)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := yaml.Unmarshal([]byte(rendered), &doc); err != nil {
		t.Fatalf("compose is not valid YAML: %v\n%s", err, rendered)
	}
	return rendered, doc
}

func TestMonitoringShipsLogsWithAlloyInsteadOfPromtail(t *testing.T) {
	rendered, doc := renderMonitoringCompose(t)

	if strings.Contains(strings.ToLower(rendered), "promtail") {
		t.Error("Promtail is end of life and must not be generated")
	}
	services := doc["services"].(map[string]any)
	alloy, ok := services["alloy"].(map[string]any)
	if !ok {
		t.Fatalf("expected an alloy service, got %v", services)
	}
	if alloy["image"] != "grafana/alloy:"+config.VersionAlloy {
		t.Errorf("alloy image = %v", alloy["image"])
	}
	if _, published := alloy["ports"]; published {
		t.Error("Alloy's UI is not needed and must not be published")
	}
	volumes := doc["volumes"].(map[string]any)
	if _, ok := volumes["alloy-data"]; !ok {
		t.Error("Alloy needs a named volume so file positions survive restarts")
	}

	mounts := map[string]bool{}
	for _, v := range alloy["volumes"].([]any) {
		mounts[v.(string)] = true
	}
	for _, want := range []string{"/var/log:/var/log:ro", "./monitoring/config.alloy:/etc/alloy/config.alloy:ro", "alloy-data:/var/lib/alloy/data"} {
		if !mounts[want] {
			t.Errorf("alloy should mount %q, mounts = %v", want, mounts)
		}
	}
}

func TestMonitoringNeverMountsTheDockerSocket(t *testing.T) {
	rendered, _ := renderMonitoringCompose(t)
	if strings.Contains(rendered, "docker.sock") {
		t.Error("mounting the Docker socket gives a container control of the host; collect logs from files instead")
	}
}

func TestMonitoringPublishesOnlyLoopbackPorts(t *testing.T) {
	_, doc := renderMonitoringCompose(t)
	for name, svc := range doc["services"].(map[string]any) {
		ports, _ := svc.(map[string]any)["ports"].([]any)
		for _, p := range ports {
			if !strings.HasPrefix(p.(string), "127.0.0.1:") {
				t.Errorf("service %s publishes %q on a non-loopback address", name, p)
			}
		}
	}
}

func TestGrafanaProvisionsPrometheusAndLoki(t *testing.T) {
	content, err := RenderGrafanaDatasource()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Datasources []struct {
			Name      string `yaml:"name"`
			Type      string `yaml:"type"`
			URL       string `yaml:"url"`
			IsDefault bool   `yaml:"isDefault"`
			Editable  bool   `yaml:"editable"`
		} `yaml:"datasources"`
	}
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatal(err)
	}
	urls := map[string]string{}
	defaults := 0
	for _, ds := range doc.Datasources {
		urls[ds.Type] = ds.URL
		if ds.IsDefault {
			defaults++
		}
		if ds.Editable {
			t.Errorf("datasource %s should be read-only", ds.Name)
		}
	}
	if urls["prometheus"] != "http://prometheus:9090" || urls["loki"] != "http://loki:3100" {
		t.Errorf("datasource URLs = %v; logs collected into Loki must be viewable in Grafana", urls)
	}
	if defaults != 1 {
		t.Errorf("exactly one default datasource expected, got %d", defaults)
	}
}

func TestAlloyConfigShipsHostLogsToLoki(t *testing.T) {
	content, err := RenderAlloyConfig()
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`"__path__" = "/var/log/*.log"`,
		`"job" = "varlogs"`,
		"loki.source.file",
		"forward_to = [loki.write.default.receiver]",
		`url = "http://loki:3100/loki/api/v1/push"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("alloy config should contain %q:\n%s", want, content)
		}
	}
	if opens, closes := strings.Count(content, "{"), strings.Count(content, "}"); opens != closes {
		t.Errorf("unbalanced braces: %d open, %d close", opens, closes)
	}
}

// Prometheus 3 rejects a scrape target whose response has no valid Content-Type.
func TestSampleBackendsServeMetricsWithATextContentType(t *testing.T) {
	sources := map[config.BackendType]string{
		config.BackendGo:     "backend/main.go",
		config.BackendNode:   "backend/main.js",
		config.BackendPython: "backend/main.py",
		config.BackendPHP:    "backend/index.php",
		config.BackendJava:   "backend/Main.java",
	}
	for backend, path := range sources {
		t.Run(string(backend), func(t *testing.T) {
			files, err := GenerateScaffold(config.ProjectConfig{Frontend: config.FrontendNone, Backend: backend})
			if err != nil {
				t.Fatal(err)
			}
			source, ok := files[path]
			if !ok {
				t.Fatalf("scaffold has no %s", path)
			}
			if !strings.Contains(source.Content, "/metrics") || !strings.Contains(source.Content, "text/plain") {
				t.Errorf("%s must serve /metrics with a text/plain Content-Type for Prometheus 3", path)
			}
		})
	}
}
