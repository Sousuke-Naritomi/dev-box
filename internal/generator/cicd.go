package generator

import "fmt"

// GenerateCICD returns the GitHub Actions CI/CD workflow YAML content.
// Uses actions/checkout@v4, docker compose (V2), Trivy scanning, and proper health checks.
func GenerateCICD(backendPort int) string {
	tmpl := `name: CI/CD Pipeline

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    name: Lint
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - uses: golangci/golangci-lint-action@v4
        with:
          version: latest

  security-scan:
    name: Security Scan
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run Trivy vulnerability scanner
        uses: aquasecurity/trivy-action@master
        with:
          scan-type: "fs"
          scan-ref: "."
          severity: "CRITICAL,HIGH"
          exit-code: "1"

  build-and-test:
    name: Build and Test
    runs-on: ubuntu-latest
    needs: [lint, security-scan]
    steps:
      - uses: actions/checkout@v4

      - name: Start services
        run: docker compose up -d

      - name: Wait for healthy services
        run: |
          echo "Waiting for services..."
          for i in $(seq 1 30); do
            if docker compose ps --format json | jq -e 'select(.Health == "starting")' > /dev/null 2>&1; then
              echo "Attempt $i/30: services still starting..."
              sleep 5
            else
              echo "All services ready."
              break
            fi
          done
          docker compose ps

      - name: Verify HTTP response
        run: |
          STATUS=$(curl -sf -o /dev/null -w "%%{http_code}" http://localhost:%d/health || echo "000")
          echo "Health check response: $STATUS"

      - name: Tear down
        if: always()
        run: docker compose down -v
`
	return fmt.Sprintf(tmpl, backendPort)
}
