package generator

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"gopkg.in/yaml.v3"
)

func TestAllCoreStackCombinationsGenerateValidYAML(t *testing.T) {
	frontends := []config.FrontendType{config.FrontendNone, config.FrontendReact, config.FrontendVue}
	backends := []config.BackendType{config.BackendNone, config.BackendNode, config.BackendPython, config.BackendGo, config.BackendPHP, config.BackendJava}
	databases := []config.DatabaseType{config.DatabaseNone, config.DatabaseMySQL, config.DatabasePostgres}
	caches := []config.CacheType{config.CacheNone, config.CacheRedis}

	for _, frontend := range frontends {
		for _, backend := range backends {
			for _, database := range databases {
				for _, cache := range caches {
					cfg := config.ProjectConfig{Frontend: frontend, Backend: backend, Database: database, Cache: cache}
					if !cfg.HasWorkload() {
						continue
					}
					name := fmt.Sprintf("%s_%s_%s_%s", frontend, backend, database, cache)
					t.Run(name, func(t *testing.T) {
						if err := cfg.Validate(); err != nil {
							t.Fatalf("Validate failed: %v", err)
						}
						data, err := ResolveComposeData(cfg)
						if err != nil {
							t.Fatalf("ResolveComposeData failed: %v", err)
						}
						compose, err := RenderCompose(data)
						if err != nil {
							t.Fatalf("RenderCompose failed: %v", err)
						}
						var composeDocument map[string]any
						if err := yaml.Unmarshal([]byte(compose), &composeDocument); err != nil {
							t.Fatalf("invalid Compose YAML: %v", err)
						}
						if _, err := GenerateScaffold(cfg); err != nil {
							t.Fatalf("GenerateScaffold failed: %v", err)
						}
						manifests, err := GenerateK8sManifests(cfg)
						if err != nil {
							t.Fatalf("GenerateK8sManifests failed: %v", err)
						}
						for path, manifest := range manifests {
							decoder := yaml.NewDecoder(strings.NewReader(manifest.Content))
							for document := 0; ; document++ {
								var value map[string]any
								err := decoder.Decode(&value)
								if err == io.EOF {
									break
								}
								if err != nil {
									t.Fatalf("invalid Kubernetes YAML in %s document %d: %v", path, document, err)
								}
							}
						}
					})
				}
			}
		}
	}
}
