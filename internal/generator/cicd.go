package generator

import (
	"bytes"
	"fmt"
	"text/template"
)

// GenerateCICD returns the GitHub Actions CI/CD workflow YAML content.
func GenerateCICD(backendPort int) (string, error) {
	tmpl, err := template.ParseFS(FS, "templates/ci.yml.tmpl")
	if err != nil {
		return "", fmt.Errorf("parsing cicd template: %w", err)
	}

	var buf bytes.Buffer
	data := struct{ BackendPort int }{BackendPort: backendPort}
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("executing cicd template: %w", err)
	}

	return buf.String(), nil
}
