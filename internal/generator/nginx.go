package generator

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

const nginxTemplate = `worker_processes auto;

events {
    worker_connections 1024;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;
    sendfile      on;
    keepalive_timeout 65;

    # Rate limiting
    limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
    limit_req_zone $binary_remote_addr zone=general_limit:10m rate=30r/s;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_min_length 256;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml image/svg+xml;

    upstream backend {
        server api:{{ .BackendPort }};
    }
{{- if .HasFrontend }}

    upstream frontend {
        server frontend:5173;
    }
{{- end }}

    server {
        listen 80;
        server_name localhost;

        add_header X-Frame-Options "SAMEORIGIN" always;
        add_header X-Content-Type-Options "nosniff" always;
{{- if .HasFrontend }}

        location / {
            limit_req zone=general_limit burst=20 nodelay;
            proxy_pass http://frontend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_http_version 1.1;
        }
{{- end }}

        location /api/ {
            limit_req zone=api_limit burst=10 nodelay;
            proxy_pass http://backend/;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_http_version 1.1;
        }

        location /health {
            access_log off;
            return 200 "OK\n";
            add_header Content-Type text/plain;
        }
    }
}
`

type nginxData struct {
	BackendPort int
	HasFrontend bool
}

var nginxTmpl = template.Must(template.New("nginx").Parse(nginxTemplate))

// GenerateNginxConf generates an Nginx configuration with rate limiting,
// gzip compression, WebSocket support, and proper proxy headers.
func GenerateNginxConf(cfg config.ProjectConfig, backendPort int) (string, error) {
	data := nginxData{
		BackendPort: backendPort,
		HasFrontend: cfg.Frontend != "none" && cfg.Frontend != "",
	}

	var buf bytes.Buffer
	if err := nginxTmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute nginx template: %w", err)
	}

	return buf.String(), nil
}
