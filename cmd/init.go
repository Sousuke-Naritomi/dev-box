package cmd

import (
	"crypto/rand"
	"encoding/hex"
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
		var localstack bool
		var monitoring bool
		var nginxProxy bool
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
					Title("Include Nginx Reverse Proxy? (API Gateway)").
					Value(&nginxProxy),

				huh.NewConfirm().
					Title("Include LocalStack? (AWS Local Emulator)").
					Value(&localstack),

				huh.NewConfirm().
					Title("Include Monitoring? (Prometheus + Grafana + Loki)").
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

		fmt.Printf("\n✨ Generating environment for: Frontend=%s, Backend=%s, Database=%s, Cache=%s, Proxy=%v, LocalStack=%v\n", frontend, backend, database, cache, nginxProxy, localstack)
		
		generateDockerCompose(frontend, backend, database, cache, monitoring, cicd, localstack, nginxProxy)
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

func generatePassword(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "fallback_Secure_Password_123!"
	}
	return hex.EncodeToString(bytes)
}

func generateDockerCompose(frontend, backend, database, cache string, monitoring bool, cicd bool, localstack bool, nginxProxy bool) {
	content := "version: '3.8'\nservices:\n"
	envContent := ""
	
	if database == "mysql" {
		port := getAvailablePort(3306)
		dbPassword := generatePassword(8)
		envContent += fmt.Sprintf("MYSQL_ROOT_PASSWORD=%s\n", dbPassword)

		content += fmt.Sprintf(`  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: devdb
    ports:
      - "%d:3306"
`, port)
	} else if database == "postgres" {
		port := getAvailablePort(5432)
		dbPassword := generatePassword(8)
		envContent += fmt.Sprintf("POSTGRES_PASSWORD=%s\n", dbPassword)

		content += fmt.Sprintf(`  db:
    image: postgres:15
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
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

	if localstack {
		port := getAvailablePort(4566)
		content += fmt.Sprintf(`  localstack:
    image: localstack/localstack:latest
    ports:
      - "%d:4566"
    environment:
      - SERVICES=s3,dynamodb,sqs,sns
      - DEFAULT_REGION=ap-northeast-1
      - AWS_ACCESS_KEY_ID=test
      - AWS_SECRET_ACCESS_KEY=test
`, port)
	}

	backendInternalPort := ""

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
		backendInternalPort = "3000"
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
		backendInternalPort = "8000"
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
		backendInternalPort = "8080"
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
		backendInternalPort = "80"
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
		backendInternalPort = "8080"
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

	if nginxProxy && (frontend != "none" || backend != "none") {
		nginxPort := getAvailablePort(80)
		content += fmt.Sprintf(`  nginx:
    image: nginx:alpine
    ports:
      - "%d:80"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
    depends_on:
`, nginxPort)
		if frontend != "none" {
			content += "      - frontend\n"
		}
		if backend != "none" {
			content += "      - api\n"
		}

		nginxConf := `events {
    worker_connections 1024;
}
http {
    server {
        listen 80;
`
		if frontend != "none" {
			nginxConf += `        location / {
            proxy_pass http://frontend:5173;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
`
		}
		if backend != "none" {
			nginxConf += fmt.Sprintf(`        location /api/ {
            proxy_pass http://api:%s/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
`, backendInternalPort)
		}
		nginxConf += `    }
}
`
		os.WriteFile("nginx.conf", []byte(nginxConf), 0644)
	}

	if monitoring {
		promPort := getAvailablePort(9090)
		grafPort := getAvailablePort(3001)
		lokiPort := getAvailablePort(3100)

		grafPassword := generatePassword(8)
		envContent += fmt.Sprintf("GRAFANA_PASSWORD=%s\n", grafPassword)

		content += fmt.Sprintf(`  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "%d:9090"
  loki:
    image: grafana/loki:latest
    ports:
      - "%d:3100"
    command: -config.file=/etc/loki/local-config.yaml
  promtail:
    image: grafana/promtail:latest
    volumes:
      - /var/log:/var/log
      - ./promtail-config.yml:/etc/promtail/config.yml
    command: -config.file=/etc/promtail/config.yml
    depends_on:
      - loki
  grafana:
    image: grafana/grafana:latest
    ports:
      - "%d:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    depends_on:
      - prometheus
      - loki
`, promPort, lokiPort, grafPort)

		os.WriteFile("prometheus.yml", []byte(`global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'prometheus'
    static_configs:
      - targets: ['localhost:9090']
`), 0644)

		os.WriteFile("promtail-config.yml", []byte(`server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: system
    static_configs:
    - targets:
        - localhost
      labels:
        job: varlogs
        __path__: /var/log/*log
`), 0644)
	}

	err := os.WriteFile("docker-compose.yml", []byte(content), 0644)
	if err != nil {
		fmt.Println("❌ Error writing docker-compose.yml:", err)
		return
	}

	if envContent != "" {
		os.WriteFile(".env", []byte(envContent), 0644)
		f, err := os.OpenFile(".gitignore", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.WriteString("\n# Ignore env files containing secrets\n.env\n")
			f.Close()
		}
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
	if envContent != "" {
		fmt.Println("🔒 Security Note: A secure .env file was generated and added to .gitignore.")
	}
}
