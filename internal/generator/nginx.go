package generator

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

type nginxData struct {
	BackendPort int
	HasFrontend bool
}

// GenerateNginxConf generates an Nginx configuration with rate limiting,
// gzip compression, WebSocket support, and proper proxy headers.
func GenerateNginxConf(cfg config.ProjectConfig, backendPort int) (string, error) {
	data := nginxData{
		BackendPort: backendPort,
		HasFrontend: cfg.Frontend != "none" && cfg.Frontend != "",
	}

	tmpl, err := template.ParseFS(FS, "templates/nginx.conf.tmpl")
	if err != nil {
		return "", fmt.Errorf("parsing nginx template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing nginx template: %w", err)
	}

	return buf.String(), nil
}
