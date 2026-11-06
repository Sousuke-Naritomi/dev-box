package generator

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func renderCompose(t *testing.T, cfg config.ProjectConfig) (string, map[string]any) {
	t.Helper()
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

func service(t *testing.T, doc map[string]any, name string) map[string]any {
	t.Helper()
	svc, ok := doc["services"].(map[string]any)[name].(map[string]any)
	if !ok {
		t.Fatalf("expected a %s service, got %v", name, doc["services"])
	}
	return svc
}

func TestMotoServiceIsHardenedAndHealthChecked(t *testing.T) {
	_, doc := renderCompose(t, config.ProjectConfig{Frontend: config.FrontendNone, Backend: config.BackendGo, Database: config.DatabaseNone, Cache: config.CacheNone, Moto: true})
	moto := service(t, doc, "moto")

	if moto["image"] != "motoserver/moto:"+config.VersionMoto {
		t.Errorf("image = %v", moto["image"])
	}
	if moto["user"] != "10001:10001" {
		t.Errorf("Moto runs as a non-root user, got %v", moto["user"])
	}
	ports, _ := moto["ports"].([]any)
	if len(ports) != 1 || ports[0] != "127.0.0.1:0:5000" {
		t.Errorf("Moto must be published on loopback only, got %v", ports)
	}
	health, _ := moto["healthcheck"].(map[string]any)
	if health == nil || !strings.Contains(strings.Join(toStrings(health["test"]), " "), "/moto-api/") {
		t.Errorf("Moto needs a healthcheck on /moto-api/ so dependants can wait for it, got %v", health)
	}
	if _, hasVolumes := moto["volumes"]; hasVolumes {
		t.Error("Moto keeps its state in memory and needs no volume")
	}
}

func toStrings(v any) []string {
	var out []string
	items, _ := v.([]any)
	for _, item := range items {
		out = append(out, item.(string))
	}
	return out
}

func TestAPIReceivesTheAWSEndpointAndWaitsForMoto(t *testing.T) {
	cfg := config.ProjectConfig{Frontend: config.FrontendNone, Backend: config.BackendGo, Database: config.DatabasePostgres, Cache: config.CacheNone, Moto: true}
	_, doc := renderCompose(t, cfg)
	api := service(t, doc, "api")

	env := toStrings(api["environment"])
	for _, want := range []string{"AWS_ENDPOINT_URL=http://moto:5000", "AWS_DEFAULT_REGION=ap-northeast-1", "AWS_ACCESS_KEY_ID=test", "AWS_SECRET_ACCESS_KEY=test"} {
		found := false
		for _, e := range env {
			found = found || e == want
		}
		if !found {
			t.Errorf("api should receive %s, got %v", want, env)
		}
	}

	deps, _ := api["depends_on"].(map[string]any)
	for _, name := range []string{"db", "moto"} {
		dep, _ := deps[name].(map[string]any)
		if dep["condition"] != "service_healthy" {
			t.Errorf("api should wait for %s to be healthy, depends_on = %v", name, deps)
		}
	}
}

func TestAPIDoesNotWaitForMotoWhenItIsNotSelected(t *testing.T) {
	rendered, doc := renderCompose(t, config.ProjectConfig{Frontend: config.FrontendNone, Backend: config.BackendGo, Database: config.DatabaseNone, Cache: config.CacheNone})
	api := service(t, doc, "api")
	if _, ok := api["environment"]; ok {
		t.Errorf("no AWS settings without Moto: %v", api["environment"])
	}
	if _, ok := api["depends_on"]; ok {
		t.Errorf("api has nothing to wait for here: %v", api["depends_on"])
	}
	if strings.Contains(rendered, "moto") {
		t.Errorf("Moto must not appear when it is not selected:\n%s", rendered)
	}
}

func TestMotoOnlyProjectIsValidCompose(t *testing.T) {
	rendered, doc := renderCompose(t, config.ProjectConfig{Frontend: config.FrontendNone, Backend: config.BackendNone, Database: config.DatabaseNone, Cache: config.CacheNone, Moto: true})
	if _, ok := doc["services"].(map[string]any)["moto"]; !ok {
		t.Fatalf("expected the moto service:\n%s", rendered)
	}
	if _, ok := doc["volumes"]; ok {
		t.Errorf("Moto alone declares no named volumes, so the volumes section must be omitted:\n%s", rendered)
	}
}

func TestNoLocalStackIsGeneratedAnymore(t *testing.T) {
	cfg := config.ProjectConfig{Frontend: config.FrontendReact, Backend: config.BackendNode, Database: config.DatabaseMySQL, Cache: config.CacheRedis, Nginx: true, Moto: true, Monitoring: true}
	rendered, _ := renderCompose(t, cfg)
	if strings.Contains(strings.ToLower(rendered), "localstack") {
		t.Errorf("LocalStack was replaced by Moto:\n%s", rendered)
	}
}
