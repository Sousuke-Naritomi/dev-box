package generator

import (
	"bytes"
	"fmt"
	"text/template"
)

const k8sDeploymentTmpl = `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}-deployment
  namespace: dev-box-ns
  labels:
    app: {{ .Name }}
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
    spec:
      {{- if .RunAsNonRoot }}
      securityContext:
        runAsNonRoot: true
      {{- end }}
      containers:
        - name: {{ .Name }}
          image: {{ .Image }}
          ports:
            - containerPort: {{ .Port }}
          resources:
            requests:
              cpu: "100m"
              memory: "128Mi"
            limits:
              cpu: "500m"
              memory: "256Mi"
          {{- if .HasDBSecret }}
          env:
            - name: {{ .EnvPasswordKey }}
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: password
          {{- end }}
          {{- if .HealthCmd }}
          livenessProbe:
            exec:
              command:
                {{- range .HealthCmd }}
                - "{{ . }}"
                {{- end }}
            initialDelaySeconds: 30
            periodSeconds: 10
          readinessProbe:
            exec:
              command:
                {{- range .HealthCmd }}
                - "{{ . }}"
                {{- end }}
            initialDelaySeconds: 5
            periodSeconds: 5
          volumeMounts:
            - name: {{ .Name }}-storage
              mountPath: {{ .VolumeMountPath }}
          {{- else }}
          livenessProbe:
            httpGet:
              path: /health
              port: {{ .Port }}
            initialDelaySeconds: 15
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: {{ .Port }}
            initialDelaySeconds: 5
            periodSeconds: 5
          {{- end }}
      {{- if .VolumeMountPath }}
      volumes:
        - name: {{ .Name }}-storage
          persistentVolumeClaim:
            claimName: {{ .Name }}-pvc
      {{- end }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}-service
  namespace: dev-box-ns
spec:
  selector:
    app: {{ .Name }}
  ports:
    - protocol: TCP
      port: {{ .Port }}
      targetPort: {{ .Port }}
  type: {{ .ServiceType }}
{{- if .VolumeMountPath }}
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: {{ .Name }}-pvc
  namespace: dev-box-ns
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
{{- end }}
`

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

var k8sTmpl = template.Must(template.New("k8s").Parse(k8sDeploymentTmpl))

// GenerateK8sManifests generates Kubernetes Deployment, Service, and PVC manifests
// for the selected stack components. Returns a map of filename to YAML content.
func GenerateK8sManifests(frontend, backend, database, cache string) (map[string]string, error) {
	manifests := make(map[string]string)

	manifests["namespace.yaml"] = `apiVersion: v1
kind: Namespace
metadata:
  name: dev-box-ns
`

	manifests["networkpolicy.yaml"] = `apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-ingress
  namespace: dev-box-ns
spec:
  podSelector: {}
  policyTypes:
    - Ingress
`

	var services []k8sService

	if frontend == "react" || frontend == "vue" {
		services = append(services, k8sService{
			Name: "frontend", Image: "node:20-alpine", Port: 5173,
			Replicas: 2, RunAsNonRoot: true, ServiceType: "LoadBalancer",
		})
	}

	switch backend {
	case "node":
		services = append(services, k8sService{
			Name: "api", Image: "node:20-alpine", Port: 3000,
			Replicas: 2, RunAsNonRoot: true, ServiceType: "ClusterIP",
		})
	case "python":
		services = append(services, k8sService{
			Name: "api", Image: "python:3.12-slim", Port: 8000,
			Replicas: 2, RunAsNonRoot: true, ServiceType: "ClusterIP",
		})
	case "go":
		services = append(services, k8sService{
			Name: "api", Image: "golang:1.22-alpine", Port: 8080,
			Replicas: 2, RunAsNonRoot: true, ServiceType: "ClusterIP",
		})
	case "php":
		services = append(services, k8sService{
			Name: "api", Image: "php:8.3-apache", Port: 80,
			Replicas: 2, RunAsNonRoot: true, ServiceType: "ClusterIP",
		})
	case "java":
		services = append(services, k8sService{
			Name: "api", Image: "eclipse-temurin:17-jdk", Port: 8080,
			Replicas: 2, RunAsNonRoot: true, ServiceType: "ClusterIP",
		})
	}

	switch database {
	case "mysql":
		services = append(services, k8sService{
			Name: "db", Image: "mysql:8.0", Port: 3306,
			Replicas: 1, RunAsNonRoot: false, HasDBSecret: true,
			HealthCmd: []string{"mysqladmin", "ping", "-h", "localhost"},
			VolumeMountPath: "/var/lib/mysql", EnvPasswordKey: "MYSQL_ROOT_PASSWORD",
			ServiceType: "ClusterIP",
		})
	case "postgres":
		services = append(services, k8sService{
			Name: "db", Image: "postgres:16-alpine", Port: 5432,
			Replicas: 1, RunAsNonRoot: false, HasDBSecret: true,
			HealthCmd: []string{"pg_isready", "-U", "postgres"},
			VolumeMountPath: "/var/lib/postgresql/data", EnvPasswordKey: "POSTGRES_PASSWORD",
			ServiceType: "ClusterIP",
		})
	}

	if cache == "redis" {
		services = append(services, k8sService{
			Name: "cache", Image: "redis:7-alpine", Port: 6379,
			Replicas: 1, RunAsNonRoot: true, HasDBSecret: false,
			HealthCmd: []string{"redis-cli", "ping"}, ServiceType: "ClusterIP",
		})
	}

	for _, svc := range services {
		var buf bytes.Buffer
		if err := k8sTmpl.Execute(&buf, svc); err != nil {
			return nil, fmt.Errorf("failed to render k8s manifest for %s: %w", svc.Name, err)
		}
		manifests[svc.Name+".yaml"] = buf.String()
	}

	return manifests, nil
}
