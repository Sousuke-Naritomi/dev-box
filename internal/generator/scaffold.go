package generator

import (
	"fmt"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

// GenerateScaffold generates boilerplate source code files for the selected
// frontend and backend. Returns a map of relative filepath to file content.
func GenerateScaffold(cfg config.ProjectConfig) (map[string]string, error) {
	files := make(map[string]string)

	switch cfg.Frontend {
	case "react":
		scaffoldReact(files)
	case "vue":
		scaffoldVue(files)
	case "none", "":
		// no-op
	default:
		return nil, fmt.Errorf("unsupported frontend: %q", cfg.Frontend)
	}

	switch cfg.Backend {
	case "node":
		scaffoldNode(files)
	case "python":
		scaffoldPython(files)
	case "go":
		scaffoldGo(files)
	case "php":
		scaffoldPHP(files)
	case "java":
		scaffoldJava(files)
	case "none", "":
		// no-op
	default:
		return nil, fmt.Errorf("unsupported backend: %q", cfg.Backend)
	}

	return files, nil
}

func scaffoldReact(files map[string]string) {
	files["frontend/package.json"] = `{
  "name": "dev-box-frontend",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": { "dev": "vite", "build": "vite build" },
  "dependencies": { "react": "^18.2.0", "react-dom": "^18.2.0" },
  "devDependencies": { "@vitejs/plugin-react": "^4.2.0", "vite": "^5.0.0" }
}
`
	files["frontend/index.html"] = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" /><title>Dev Box</title></head>
<body><div id="root"></div><script type="module" src="/src/main.jsx"></script></body>
</html>
`
	files["frontend/src/main.jsx"] = `import React from 'react';
import ReactDOM from 'react-dom/client';
ReactDOM.createRoot(document.getElementById('root')).render(<h1>Dev Box (React)</h1>);
`
	files["frontend/vite.config.js"] = `import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
});
`
	files["frontend/Dockerfile"] = `FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
`
}

func scaffoldVue(files map[string]string) {
	files["frontend/package.json"] = `{
  "name": "dev-box-frontend",
  "private": true,
  "version": "0.1.0",
  "type": "module",
  "scripts": { "dev": "vite", "build": "vite build" },
  "dependencies": { "vue": "^3.4.0" },
  "devDependencies": { "@vitejs/plugin-vue": "^5.0.0", "vite": "^5.0.0" }
}
`
	files["frontend/index.html"] = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" /><title>Dev Box</title></head>
<body><div id="app"></div><script type="module" src="/src/main.js"></script></body>
</html>
`
	files["frontend/src/main.js"] = `import { createApp } from 'vue';
const App = { template: '<h1>Dev Box (Vue)</h1>' };
createApp(App).mount('#app');
`
	files["frontend/vite.config.js"] = `import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig({
  plugins: [vue()],
});
`
	files["frontend/Dockerfile"] = `FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
CMD ["npm", "run", "dev", "--", "--host", "0.0.0.0"]
`
}

func scaffoldNode(files map[string]string) {
	files["backend/package.json"] = `{
  "name": "dev-box-api",
  "version": "0.1.0",
  "scripts": { "start": "node main.js" },
  "dependencies": { "express": "^4.18.0" }
}
`
	files["backend/main.js"] = `const express = require('express');
const app = express();
const PORT = process.env.BACKEND_PORT || 3000;

app.use(express.json());
app.get('/', (req, res) => res.json({ message: 'Hello from dev-box API (Node.js)' }));
app.get('/health', (req, res) => res.json({ status: 'ok' }));
app.listen(PORT, '0.0.0.0', () => console.log('Server running on port ' + PORT));
`
	files["backend/Dockerfile"] = `FROM node:20-alpine
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
CMD ["npm", "start"]
`
}

func scaffoldPython(files map[string]string) {
	files["backend/requirements.txt"] = "flask==3.0.*\ngunicorn==21.*\n"
	files["backend/main.py"] = `"""Dev-box API server (Python/Flask)."""
import os
from flask import Flask, jsonify

app = Flask(__name__)

@app.route("/")
def index():
    return jsonify({"message": "Hello from dev-box API (Python)"})

@app.route("/health")
def health():
    return jsonify({"status": "ok"})

if __name__ == "__main__":
    port = int(os.environ.get("BACKEND_PORT", 8000))
    app.run(host="0.0.0.0", port=port, debug=True)
`
	files["backend/Dockerfile"] = `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
CMD ["python", "main.py"]
`
}

func scaffoldGo(files map[string]string) {
	files["backend/go.mod"] = "module github.com/Sousuke-Naritomi/dev-box-api\n\ngo 1.22\n"
	files["backend/main.go"] = `package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Hello from dev-box API (Go)"})
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})
	log.Printf("Server starting on :%s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
`
	files["backend/Dockerfile"] = `FROM golang:1.22-alpine
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
RUN go build -o main .
CMD ["./main"]
`
}

func scaffoldPHP(files map[string]string) {
	files["backend/index.php"] = `<?php
header('Content-Type: application/json');
$path = parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH);
switch ($path) {
    case '/': echo json_encode(['message' => 'Hello from dev-box API (PHP)']); break;
    case '/health': echo json_encode(['status' => 'ok']); break;
    default: http_response_code(404); echo json_encode(['error' => 'Not Found']); break;
}
`
	files["backend/Dockerfile"] = `FROM php:8.3-apache
COPY . /var/www/html/
EXPOSE 80
`
}

func scaffoldJava(files map[string]string) {
	files["backend/Main.java"] = `import com.sun.net.httpserver.HttpServer;
import com.sun.net.httpserver.HttpExchange;
import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;

public class Main {
    public static void main(String[] args) throws IOException {
        String portStr = System.getenv("BACKEND_PORT");
        int port = (portStr != null && !portStr.isEmpty()) ? Integer.parseInt(portStr) : 8080;

        HttpServer server = HttpServer.create(new InetSocketAddress("0.0.0.0", port), 0);
        server.createContext("/", exchange -> sendJson(exchange, "{\"message\": \"Hello from dev-box API (Java)\"}"));
        server.createContext("/health", exchange -> sendJson(exchange, "{\"status\": \"ok\"}"));
        server.start();
        System.out.println("Server running on port " + port);
    }

    private static void sendJson(HttpExchange exchange, String json) throws IOException {
        exchange.getResponseHeaders().set("Content-Type", "application/json");
        byte[] bytes = json.getBytes();
        exchange.sendResponseHeaders(200, bytes.length);
        try (OutputStream os = exchange.getResponseBody()) { os.write(bytes); }
    }
}
`
}
