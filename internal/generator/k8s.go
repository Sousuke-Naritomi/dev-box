package generator

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/registry"
)

type k8sService struct {
	Name            string
	Image           string
	Port            int
	Replicas        int
	RunAsNonRoot    bool
	HasDBSecret     bool
	HealthCmd       []string
	VolumeMountPath string
	EnvPasswordKey  string
	ServiceType     string
}

// GenerateK8sManifests generates Kubernetes Deployment, Service, and PVC manifests
// for the selected stack components. Returns a map of filename to FileDef content.
func GenerateK8sManifests(cfg config.ProjectConfig) (map[string]FileDef, error) {
	manifests := make(map[string]FileDef)

	nsContent, err := FS.ReadFile("templates/k8s/namespace.yaml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to read k8s namespace template: %w", err)
	}
	manifests["namespace.yaml"] = FileDef{Content: string(nsContent), Perm: 0644}

	netPolContent, err := FS.ReadFile("templates/k8s/networkpolicy.yaml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to read k8s networkpolicy template: %w", err)
	}
	manifests["networkpolicy.yaml"] = FileDef{Content: string(netPolContent), Perm: 0644}

	var services []k8sService

	if f, ok := registry.SupportedFrontends[cfg.Frontend]; ok {
		services = append(services, k8sService{
			Name: "frontend", Image: f.Image, Port: f.InternalPort,
			Replicas: 2, RunAsNonRoot: true, ServiceType: "LoadBalancer",
		})
	}

	if b, ok := registry.SupportedBackends[cfg.Backend]; ok {
		services = append(services, k8sService{
			Name: "api", Image: b.Image, Port: b.InternalPort,
			Replicas: 2, RunAsNonRoot: true, ServiceType: "ClusterIP",
		})
	}

	switch cfg.Database {
	case config.DatabaseMySQL:
		services = append(services, k8sService{
			Name: "db", Image: "mysql:" + config.VersionMySQL, Port: 3306,
			Replicas: 1, RunAsNonRoot: false, HasDBSecret: true,
			HealthCmd:       []string{"mysqladmin", "ping", "-h", "localhost"},
			VolumeMountPath: "/var/lib/mysql", EnvPasswordKey: "MYSQL_ROOT_PASSWORD",
			ServiceType: "ClusterIP",
		})
	case config.DatabasePostgres:
		services = append(services, k8sService{
			Name: "db", Image: "postgres:" + config.VersionPostgres, Port: 5432,
			Replicas: 1, RunAsNonRoot: false, HasDBSecret: true,
			HealthCmd:       []string{"pg_isready", "-U", "postgres"},
			VolumeMountPath: "/var/lib/postgresql/data", EnvPasswordKey: "POSTGRES_PASSWORD",
			ServiceType: "ClusterIP",
		})
	}

	if cfg.Cache == config.CacheRedis {
		services = append(services, k8sService{
			Name: "cache", Image: "redis:" + config.VersionRedis, Port: 6379,
			Replicas: 1, RunAsNonRoot: true, HasDBSecret: false,
			HealthCmd: []string{"redis-cli", "ping"}, ServiceType: "ClusterIP",
		})
	}

	k8sTmpl, err := template.ParseFS(FS, "templates/k8s/deployment.yaml.tmpl")
	if err != nil {
		return nil, fmt.Errorf("failed to parse k8s deployment template: %w", err)
	}

	for _, svc := range services {
		var buf bytes.Buffer
		if err := k8sTmpl.Execute(&buf, svc); err != nil {
			return nil, fmt.Errorf("failed to render k8s manifest for %s: %w", svc.Name, err)
		}
		manifests[svc.Name+".yaml"] = FileDef{Content: buf.String(), Perm: 0644}
	}

	return manifests, nil
}
