package generator

import (
	"io"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"gopkg.in/yaml.v3"
)

func TestGeneratedK8sAppRunsAsMatchingNonRootUser(t *testing.T) {
	cfg := config.ProjectConfig{Frontend: config.FrontendReact, Backend: config.BackendGo, Database: config.DatabaseNone, Cache: config.CacheNone}
	manifests, err := GenerateK8sManifests(cfg)
	if err != nil {
		t.Fatalf("GenerateK8sManifests failed: %v", err)
	}

	for _, name := range []string{"frontend.yaml", "api.yaml"} {
		var document map[string]any
		decoder := yaml.NewDecoder(strings.NewReader(manifests[name].Content))
		if err := decoder.Decode(&document); err != nil {
			t.Fatalf("decode %s: %v", name, err)
		}
		spec := document["spec"].(map[string]any)["template"].(map[string]any)["spec"].(map[string]any)
		container := spec["containers"].([]any)[0].(map[string]any)
		securityContext := container["securityContext"].(map[string]any)
		if securityContext["runAsUser"].(int) <= 0 {
			t.Fatalf("%s must set a non-root runAsUser", name)
		}
		if securityContext["allowPrivilegeEscalation"] != false {
			t.Fatalf("%s must disable privilege escalation", name)
		}
		capabilities := securityContext["capabilities"].(map[string]any)
		if capabilities["drop"].([]any)[0] != "ALL" {
			t.Fatalf("%s must drop all capabilities", name)
		}
	}

	if !strings.Contains(manifests["frontend.yaml"].Content, "containerPort: 8080") {
		t.Fatal("frontend manifest must target the production image port")
	}
}

func TestGeneratedNetworkPolicyMatchesSelectedStack(t *testing.T) {
	cfg := config.ProjectConfig{Frontend: config.FrontendNone, Backend: config.BackendGo, Database: config.DatabasePostgres, Cache: config.CacheRedis}
	manifests, err := GenerateK8sManifests(cfg)
	if err != nil {
		t.Fatalf("GenerateK8sManifests failed: %v", err)
	}
	policy := manifests["networkpolicy.yaml"].Content
	if strings.Contains(policy, "allow-public-to-frontend") || strings.Contains(policy, "allow-frontend-to-api") {
		t.Fatal("frontend policies must not be generated without a frontend")
	}
	for _, expected := range []string{"port: 53", "port: 5432", "port: 6379", "allow-api-egress-to-db", "allow-api-egress-to-cache"} {
		if !strings.Contains(policy, expected) {
			t.Fatalf("expected network policy to contain %q", expected)
		}
	}

	decoder := yaml.NewDecoder(strings.NewReader(policy))
	for {
		var document map[string]any
		err := decoder.Decode(&document)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("network policy is invalid YAML: %v", err)
		}
	}
}

func TestGeneratedDockerfilesDeclareUsers(t *testing.T) {
	cfg := config.ProjectConfig{Frontend: config.FrontendReact, Backend: config.BackendGo}
	files, err := GenerateScaffold(cfg)
	if err != nil {
		t.Fatalf("GenerateScaffold failed: %v", err)
	}
	for _, path := range []string{"frontend/Dockerfile", "backend/Dockerfile"} {
		if !strings.Contains(files[path].Content, "USER ") {
			t.Fatalf("%s must declare a non-root USER", path)
		}
	}
	for _, path := range []string{"frontend/.dockerignore", "backend/.dockerignore"} {
		if _, ok := files[path]; !ok {
			t.Fatalf("expected %s to be generated", path)
		}
	}
	for _, path := range []string{"frontend/package-lock.json", "frontend/src/main.jsx", "backend/main.go"} {
		file, ok := files[path]
		if !ok {
			t.Fatalf("expected %s to be generated", path)
		}
		if file.Perm != 0644 {
			t.Fatalf("expected %s to be writable with mode 0644, got %o", path, file.Perm)
		}
	}
	if !strings.Contains(files["frontend/Dockerfile"].Content, "npm run build") {
		t.Fatal("frontend Dockerfile must build static assets")
	}
}

func TestEveryBackendDockerfileDeclaresNonRootUser(t *testing.T) {
	for _, backend := range []config.BackendType{
		config.BackendNode,
		config.BackendPython,
		config.BackendGo,
		config.BackendPHP,
		config.BackendJava,
	} {
		t.Run(string(backend), func(t *testing.T) {
			files, err := GenerateScaffold(config.ProjectConfig{Frontend: config.FrontendNone, Backend: backend})
			if err != nil {
				t.Fatalf("GenerateScaffold failed: %v", err)
			}
			dockerfile := files["backend/Dockerfile"].Content
			if !strings.Contains(dockerfile, "USER ") {
				t.Fatalf("backend %s Dockerfile must declare USER", backend)
			}
		})
	}
}
