package cmd

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new development environment",
	Long:  `Run an interactive prompt to configure your docker-compose environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		var frontend string
		var backend string
		var database string
		var cache string
		var monitoring bool
		var cicd bool

		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose your Frontend Framework").
					Options(
						huh.NewOption("React (Vite)", "react"),
						huh.NewOption("Vue.js (Vite)", "vue"),
						huh.NewOption("None", "none"),
					).
					Value(&frontend),

				huh.NewSelect[string]().
					Title("Choose your Backend Language").
					Options(
						huh.NewOption("Node.js (Express)", "node"),
						huh.NewOption("Python (FastAPI)", "python"),
						huh.NewOption("Go (Gin)", "go"),
						huh.NewOption("PHP (Apache/Plain)", "php"),
						huh.NewOption("Java (Spring Boot)", "java"),
						huh.NewOption("None", "none"),
					).
					Value(&backend),

				huh.NewSelect[string]().
					Title("Choose your Database").
					Options(
						huh.NewOption("MySQL", "mysql"),
						huh.NewOption("PostgreSQL", "postgres"),
						huh.NewOption("None", "none"),
					).
					Value(&database),

				huh.NewSelect[string]().
					Title("Choose your Cache/Message Broker").
					Options(
						huh.NewOption("Redis", "redis"),
						huh.NewOption("None", "none"),
					).
					Value(&cache),
				
				huh.NewConfirm().
					Title("Include Monitoring? (Prometheus + Grafana)").
					Value(&monitoring),

				huh.NewConfirm().
					Title("Include CI/CD Pipeline? (GitHub Actions)").
					Value(&cicd),
			),
		)

		err := form.Run()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\n✨ Generating environment for: Frontend=%s, Backend=%s, Database=%s, Cache=%s\n", frontend, backend, database, cache)
		
		generateDockerCompose(frontend, backend, database, cache, monitoring, cicd)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func getAvailablePort(port int) int {
	for {
		addr := fmt.Sprintf(":%d", port)
		l, err := net.Listen("tcp", addr)
		if err == nil {
			l.Close()
			return port
		}
		fmt.Printf("⚠️  Port %d is in use, trying port %d...\n", port, port+1)
		port++
	}
}

func generateDockerCompose(frontend, backend, database, cache string, monitoring bool, cicd bool) {
	content := "version: '3.8'\nservices:\n"
	
	if database == "mysql" {
		port := getAvailablePort(3306)
		content += fmt.Sprintf(`  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: devdb
    ports:
      - "%d:3306"
`, port)
	} else if database == "postgres" {
		port := getAvailablePort(5432)
		content += fmt.Sprintf(`  db:
    image: postgres:15
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: devdb
    ports:
      - "%d:5432"
`, port)
	}

	if cache == "redis" {
		port := getAvailablePort(6379)
		content += fmt.Sprintf(`  redis:
    image: redis:alpine
    ports:
      - "%d:6379"
`, port)
	}

	if frontend == "react" || frontend == "vue" {
		port := getAvailablePort(5173) // Vite default port
		content += fmt.Sprintf(`  frontend:
    image: node:20-alpine
    working_dir: /app
    volumes:
      - ./frontend:/app
    command: sh -c "npm install && npm run dev -- --host 0.0.0.0"
    ports:
      - "%d:5173"
`, port)
		os.MkdirAll("frontend", 0755)
		os.WriteFile("frontend/package.json", []byte(`{
  "name": "dev-box-frontend",
  "private": true,
  "version": "0.0.0",
  "type": "module",
  "scripts": { "dev": "vite" },
  "dependencies": {},
  "devDependencies": { "vite": "^5.0.0" }
}`), 0644)
		os.WriteFile("frontend/index.html", []byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
<body>
  <h1>Hello from dev-box %s Environment!</h1>
</body>
</html>`, frontend)), 0644)
	}

	if backend == "node" {
		port := getAvailablePort(3000)
		content += fmt.Sprintf(`  api:
    image: node:20-alpine
    working_dir: /app
    volumes:
      - ./backend:/app
    command: sh -c "npm install && npm start"
    ports:
      - "%d:3000"
`, port)
		os.MkdirAll("backend", 0755)
		os.WriteFile("backend/package.json", []byte(`{
  "name": "dev-box-node",
  "version": "1.0.0",
  "scripts": { "start": "node index.js" },
  "dependencies": { "express": "^4.18.2" }
}`), 0644)
		os.WriteFile("backend/index.js", []byte(`const express = require('express');
const app = express();
app.get('/', (req, res) => res.send('Hello from dev-box Node.js Environment!'));
app.listen(3000, () => console.log('Server running on port 3000'));
`), 0644)

	} else if backend == "python" {
		port := getAvailablePort(8000)
		content += fmt.Sprintf(`  api:
    image: python:3.11-slim
    working_dir: /app
    volumes:
      - ./backend:/app
    command: sh -c "pip install -r requirements.txt && uvicorn main:app --host 0.0.0.0 --port 8000 --reload"
    ports:
      - "%d:8000"
`, port)
		os.MkdirAll("backend", 0755)
		os.WriteFile("backend/requirements.txt", []byte("fastapi\nuvicorn\n"), 0644)
		os.WriteFile("backend/main.py", []byte(`from fastapi import FastAPI
app = FastAPI()

@app.get("/")
def read_root():
    return {"message": "Hello from dev-box Python Environment!"}
`), 0644)

	} else if backend == "go" {
		port := getAvailablePort(8080)
		content += fmt.Sprintf(`  api:
    image: golang:1.22-alpine
    working_dir: /app
    volumes:
      - ./backend:/app
    command: sh -c "go run main.go"
    ports:
      - "%d:8080"
`, port)
		os.MkdirAll("backend", 0755)
		os.WriteFile("backend/go.mod", []byte("module dev-box-go-api\n\ngo 1.22\n"), 0644)
		os.WriteFile("backend/main.go", []byte(`package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from dev-box Go Environment!")
	})
	http.ListenAndServe(":8080", nil)
}
`), 0644)

	} else if backend == "php" {
		port := getAvailablePort(80)
		content += fmt.Sprintf(`  api:
    image: php:8.2-apache
    volumes:
      - ./backend:/var/www/html
    ports:
      - "%d:80"
`, port)
		os.MkdirAll("backend", 0755)
		os.WriteFile("backend/index.php", []byte(`<?php
echo "Hello from dev-box PHP Environment!";
?>
`), 0644)

	} else if backend == "java" {
		port := getAvailablePort(8080)
		content += fmt.Sprintf(`  api:
    image: maven:3.9-eclipse-temurin-17
    working_dir: /app
    volumes:
      - ./backend:/app
    command: tail -f /dev/null
    ports:
      - "%d:8080"
`, port)
		os.MkdirAll("backend", 0755)
	}

	if monitoring {
		promPort := getAvailablePort(9090)
		grafPort := getAvailablePort(3001)
		content += fmt.Sprintf(`  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "%d:9090"
  grafana:
    image: grafana/grafana:latest
    ports:
      - "%d:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    depends_on:
      - prometheus
`, promPort, grafPort)

		os.WriteFile("prometheus.yml", []byte(`global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
`), 0644)
	}

	err := os.WriteFile("docker-compose.yml", []byte(content), 0644)
	if err != nil {
		fmt.Println("❌ Error writing docker-compose.yml:", err)
		return
	}

	if cicd {
		err := os.MkdirAll(filepath.Join(".github", "workflows"), 0755)
		if err == nil {
			workflowContent := `name: CI/CD Pipeline

on:
  push:
    branches: [ "main" ]
  pull_request:
    branches: [ "main" ]

jobs:
  build-and-test:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3
    
    - name: Set up Docker Buildx
      uses: docker/setup-buildx-action@v2
      
    - name: Build and run services
      run: docker-compose up -d
      
    - name: Test services
      run: |
        sleep 10
        docker ps
        
    - name: Tear down
      run: docker-compose down
`
			os.WriteFile(filepath.Join(".github", "workflows", "deploy.yml"), []byte(workflowContent), 0644)
			fmt.Println("✅ Successfully generated .github/workflows/deploy.yml CI/CD pipeline!")
		}
	}

	fmt.Println("✅ Successfully generated local environment configuration!")
}
