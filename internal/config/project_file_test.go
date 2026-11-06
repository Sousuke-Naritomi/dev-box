package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestProjectFileRoundTrip(t *testing.T) {
	want := ProjectConfig{
		Frontend: FrontendReact, Backend: BackendGo, Database: DatabasePostgres, Cache: CacheRedis,
		Nginx: true, Moto: true, Monitoring: true, CICD: true, Cloud: CloudNone, Force: true,
	}
	encoded, err := RenderProjectFile(want)
	if err != nil {
		t.Fatalf("RenderProjectFile failed: %v", err)
	}
	if strings.Contains(encoded, "force:") {
		t.Fatal("runtime-only force option must not be persisted")
	}
	got, err := ParseProjectFile([]byte(encoded))
	if err != nil {
		t.Fatalf("ParseProjectFile failed: %v", err)
	}
	want.Force = false
	if got != want {
		t.Fatalf("round trip mismatch:\nwant: %#v\n got: %#v", want, got)
	}
}

func TestProjectFileRejectsUnknownField(t *testing.T) {
	data := []byte("version: 1\nfrontend: react\nbackend: go\ndatabase: none\ncache: none\nmonotoring: true\n")
	if _, err := ParseProjectFile(data); err == nil || !strings.Contains(err.Error(), "field monotoring not found") {
		t.Fatalf("expected unknown field error, got %v", err)
	}
}

func TestProjectFileRejectsUnsupportedVersion(t *testing.T) {
	data := []byte("version: 99\nfrontend: react\nbackend: none\ndatabase: none\ncache: none\n")
	if _, err := ParseProjectFile(data); err == nil || !strings.Contains(err.Error(), "unsupported project configuration version") {
		t.Fatalf("expected version error, got %v", err)
	}
}

func TestProjectFileRejectsMultipleDocuments(t *testing.T) {
	data := []byte("version: 1\nfrontend: react\nbackend: none\ndatabase: none\ncache: none\n---\nextra: value\n")
	if _, err := ParseProjectFile(data); err == nil || !strings.Contains(err.Error(), "exactly one YAML document") {
		t.Fatalf("expected multiple document error, got %v", err)
	}
}

func TestRepositoryConfigurationExamplesAreValid(t *testing.T) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve test file path")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))

	projectData, err := os.ReadFile(filepath.Join(repositoryRoot, "dev-box.yaml"))
	if err != nil {
		t.Fatalf("reading repository dev-box.yaml: %v", err)
	}
	if _, err := ParseProjectFile(projectData); err != nil {
		t.Fatalf("repository dev-box.yaml is invalid: %v", err)
	}

	workflowData, err := os.ReadFile(filepath.Join(repositoryRoot, ".github", "workflows", "ci.yml"))
	if err != nil {
		t.Fatalf("reading repository workflow: %v", err)
	}
	var workflow yaml.Node
	if err := yaml.Unmarshal(workflowData, &workflow); err != nil {
		t.Fatalf("repository workflow is invalid YAML: %v", err)
	}
}
