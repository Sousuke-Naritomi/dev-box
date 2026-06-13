package generator

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/registry"
	"gopkg.in/yaml.v3"
)

type workloadDocument struct {
	Kind string `yaml:"kind"`
	Spec struct {
		Ports []struct {
			Port int `yaml:"port"`
		} `yaml:"ports"`
		Template struct {
			Spec struct {
				Containers []struct {
					Ports []struct {
						ContainerPort int `yaml:"containerPort"`
					} `yaml:"ports"`
					SecurityContext struct {
						RunAsUser int64 `yaml:"runAsUser"`
					} `yaml:"securityContext"`
				} `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	} `yaml:"spec"`
}

func TestBackendImageAndKubernetesRuntimeContract(t *testing.T) {
	for backendType, backend := range registry.SupportedBackends {
		t.Run(string(backendType), func(t *testing.T) {
			cfg := config.ProjectConfig{Frontend: config.FrontendNone, Backend: backendType, Database: config.DatabaseNone, Cache: config.CacheNone}
			files, err := GenerateScaffold(cfg)
			if err != nil {
				t.Fatalf("GenerateScaffold failed: %v", err)
			}
			dockerfile := files["backend/Dockerfile"].Content
			if !strings.Contains(dockerfile, fmt.Sprintf("EXPOSE %d", backend.ImagePort)) {
				t.Fatalf("Dockerfile must expose registry image port %d:\n%s", backend.ImagePort, dockerfile)
			}

			manifests, err := GenerateK8sManifests(cfg)
			if err != nil {
				t.Fatalf("GenerateK8sManifests failed: %v", err)
			}
			decoder := yaml.NewDecoder(strings.NewReader(manifests["api.yaml"].Content))
			var deployment workloadDocument
			if err := decoder.Decode(&deployment); err != nil {
				t.Fatalf("decode Deployment: %v", err)
			}
			var service workloadDocument
			if err := decoder.Decode(&service); err != nil {
				t.Fatalf("decode Service: %v", err)
			}
			container := deployment.Spec.Template.Spec.Containers[0]
			if container.Ports[0].ContainerPort != backend.ImagePort {
				t.Fatalf("container port %d does not match image port %d", container.Ports[0].ContainerPort, backend.ImagePort)
			}
			if service.Spec.Ports[0].Port != backend.ImagePort {
				t.Fatalf("service port %d does not match image port %d", service.Spec.Ports[0].Port, backend.ImagePort)
			}
			if container.SecurityContext.RunAsUser != backend.RunAsUser {
				t.Fatalf("runAsUser %d does not match registry user %d", container.SecurityContext.RunAsUser, backend.RunAsUser)
			}
		})
	}
}
