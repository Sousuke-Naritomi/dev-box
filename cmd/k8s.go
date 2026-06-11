package cmd

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Generate Kubernetes manifests for your environment",
	Long:  `Run an interactive prompt to generate Kubernetes Deployment and Service YAML files.`,
	Run: func(cmd *cobra.Command, args []string) {
		var frontend string
		var backend string
		var database string
		var cache string

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Generate K8s manifests for Frontend?").
					Options(
						huh.NewOption("Yes", "yes"),
						huh.NewOption("None", "none"),
					).
					Value(&frontend),

				huh.NewSelect[string]().
					Title("Generate K8s manifests for Backend?").
					Options(
						huh.NewOption("Yes", "yes"),
						huh.NewOption("None", "none"),
					).
					Value(&backend),

				huh.NewSelect[string]().
					Title("Generate K8s manifests for Database?").
					Options(
						huh.NewOption("MySQL", "mysql"),
						huh.NewOption("PostgreSQL", "postgres"),
						huh.NewOption("None", "none"),
					).
					Value(&database),

				huh.NewSelect[string]().
					Title("Generate K8s manifests for Cache?").
					Options(
						huh.NewOption("Redis", "redis"),
						huh.NewOption("None", "none"),
					).
					Value(&cache),
			),
		)

		err := form.Run()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("\n✨ Generating Kubernetes manifests in k8s-manifests/ directory...")
		generateK8sManifests(frontend, backend, database, cache)
	},
}

func init() {
	rootCmd.AddCommand(k8sCmd)
}

func generateK8sManifests(frontend, backend, database, cache string) {
	outDir := "k8s-manifests"
	os.MkdirAll(outDir, 0755)

	if database == "mysql" {
		dbDeployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: mysql-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: mysql
  template:
    metadata:
      labels:
        app: mysql
    spec:
      containers:
      - name: mysql
        image: mysql:8.0
        env:
        - name: MYSQL_ROOT_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        ports:
        - containerPort: 3306
---
apiVersion: v1
kind: Service
metadata:
  name: mysql-service
spec:
  selector:
    app: mysql
  ports:
    - protocol: TCP
      port: 3306
      targetPort: 3306
`
		os.WriteFile(filepath.Join(outDir, "mysql.yaml"), []byte(dbDeployment), 0644)
	} else if database == "postgres" {
		dbDeployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: postgres-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: postgres
  template:
    metadata:
      labels:
        app: postgres
    spec:
      containers:
      - name: postgres
        image: postgres:15
        env:
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        ports:
        - containerPort: 5432
---
apiVersion: v1
kind: Service
metadata:
  name: postgres-service
spec:
  selector:
    app: postgres
  ports:
    - protocol: TCP
      port: 5432
      targetPort: 5432
`
		os.WriteFile(filepath.Join(outDir, "postgres.yaml"), []byte(dbDeployment), 0644)
	}

	if cache == "redis" {
		redisDeployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: redis-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: redis
  template:
    metadata:
      labels:
        app: redis
    spec:
      containers:
      - name: redis
        image: redis:alpine
        ports:
        - containerPort: 6379
---
apiVersion: v1
kind: Service
metadata:
  name: redis-service
spec:
  selector:
    app: redis
  ports:
    - protocol: TCP
      port: 6379
      targetPort: 6379
`
		os.WriteFile(filepath.Join(outDir, "redis.yaml"), []byte(redisDeployment), 0644)
	}

	if frontend != "none" {
		frontendDeployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: frontend
  template:
    metadata:
      labels:
        app: frontend
    spec:
      containers:
      - name: frontend
        image: my-frontend:latest
        ports:
        - containerPort: 80
---
apiVersion: v1
kind: Service
metadata:
  name: frontend-service
spec:
  type: LoadBalancer
  selector:
    app: frontend
  ports:
    - protocol: TCP
      port: 80
      targetPort: 80
`
		os.WriteFile(filepath.Join(outDir, "frontend.yaml"), []byte(frontendDeployment), 0644)
	}

	if backend != "none" {
		backendDeployment := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-deployment
spec:
  replicas: 2
  selector:
    matchLabels:
      app: backend
  template:
    metadata:
      labels:
        app: backend
    spec:
      containers:
      - name: backend
        image: my-backend:latest
        ports:
        - containerPort: 8080
---
apiVersion: v1
kind: Service
metadata:
  name: backend-service
spec:
  selector:
    app: backend
  ports:
    - protocol: TCP
      port: 8080
      targetPort: 8080
`
		os.WriteFile(filepath.Join(outDir, "backend.yaml"), []byte(backendDeployment), 0644)
	}

	fmt.Println("✅ Successfully generated Kubernetes manifests in 'k8s-manifests' directory!")
	fmt.Println("💡 Note: You will need to create a Kubernetes Secret named 'db-secret' for database passwords before applying.")
}
