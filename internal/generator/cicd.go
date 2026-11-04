package generator

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

type cicdData struct {
	HasBackend   bool
	BackendPort  int
	HasFrontend  bool
	FrontendPort int
	TrivyVersion string
}

// GenerateCICD returns the GitHub Actions CI/CD workflow YAML content.
func GenerateCICD(cfg config.ProjectConfig, backendPort, frontendPort int) (string, error) {
	tmpl, err := template.ParseFS(FS, "templates/ci.yml.tmpl")
	if err != nil {
		return "", fmt.Errorf("parsing cicd template: %w", err)
	}

	var buf bytes.Buffer
	data := cicdData{
		HasBackend:   cfg.Backend != config.BackendNone && cfg.Backend != "",
		BackendPort:  backendPort,
		HasFrontend:  cfg.Frontend != config.FrontendNone && cfg.Frontend != "",
		FrontendPort: frontendPort,
		TrivyVersion: config.VersionTrivy,
	}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing cicd template: %w", err)
	}

	return buf.String(), nil
}
