package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new development environment",
	Long:  `Run an interactive prompt to configure your docker-compose environment.`,
	Run: func(cmd *cobra.Command, args []string) {
		var backend string
		var database string
		var cache string

		// ここで美しい対話フォームを作成します
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose your Backend Language").
					Options(
						huh.NewOption("Node.js (Express)", "node"),
						huh.NewOption("Python (FastAPI)", "python"),
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
			),
		)

		err := form.Run()
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("\n✨ Great! We are generating the docker-compose.yml for: Backend=%s, Database=%s, Cache=%s\n", backend, database, cache)

		generateDockerCompose(backend, database, cache)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

// ユーザーの回答をもとにファイルを生成する関数
func generateDockerCompose(backend, database, cache string) {
	content := "version: '3.8'\nservices:\n"

	if database == "mysql" {
		content += `  db:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: rootpassword
      MYSQL_DATABASE: devdb
    ports:
      - "3306:3306"
`
	} else if database == "postgres" {
		content += `  db:
    image: postgres:15
    environment:
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
      POSTGRES_DB: devdb
    ports:
      - "5432:5432"
`
	}

	if cache == "redis" {
		content += `  redis:
    image: redis:alpine
    ports:
      - "6379:6379"
`
	}

	if backend == "node" {
		content += `  api:
    image: node:18-alpine
    working_dir: /app
    volumes:
      - .:/app
    command: sh -c "npm install && npm start"
    ports:
      - "3000:3000"
`
		// アプリの雛形も一緒に作ってあげます
		os.WriteFile("package.json", []byte(`{
  "name": "dev-box-node",
  "version": "1.0.0",
  "scripts": { "start": "node index.js" },
  "dependencies": { "express": "^4.18.2" }
}`), 0644)
		os.WriteFile("index.js", []byte(`const express = require('express');
const app = express();
app.get('/', (req, res) => res.send('Hello from dev-box Node.js Environment!'));
app.listen(3000, () => console.log('Server running on port 3000'));
`), 0644)

	} else if backend == "python" {
		content += `  api:
    image: python:3.11-slim
    working_dir: /app
    volumes:
      - .:/app
    command: sh -c "pip install -r requirements.txt && uvicorn main:app --host 0.0.0.0 --port 8000 --reload"
    ports:
      - "8000:8000"
`
		os.WriteFile("requirements.txt", []byte("fastapi\nuvicorn\n"), 0644)
		os.WriteFile("main.py", []byte(`from fastapi import FastAPI
app = FastAPI()

@app.get("/")
def read_root():
    return {"message": "Hello from dev-box Python Environment!"}
`), 0644)
	}

	err := os.WriteFile("docker-compose.yml", []byte(content), 0644)
	if err != nil {
		fmt.Println("Error writing docker-compose.yml:", err)
		return
	}
	fmt.Println("✅ Successfully generated docker-compose.yml and necessary source files!")
}
