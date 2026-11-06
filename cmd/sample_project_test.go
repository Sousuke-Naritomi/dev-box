package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

// The repository root doubles as a sample project generated from dev-box.yaml
// (docker-compose.yml, backend/, frontend/, k8s-manifests/). This keeps it from
// drifting away from what the generator produces, which is how it ended up on
// end-of-life image tags before.
func TestRootSampleMatchesGeneratorOutput(t *testing.T) {
	root := ".."
	data, err := os.ReadFile(filepath.Join(root, "dev-box.yaml"))
	if err != nil {
		t.Skipf("sample project not available: %v", err)
	}
	cfg, err := config.ParseProjectFile(data)
	if err != nil {
		t.Fatal(err)
	}
	gen := &generator.DefaultGenerator{}
	files, _, err := buildInitFiles(cfg, gen)
	if err != nil {
		t.Fatal(err)
	}
	manifests, err := gen.GenerateK8sManifests(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// .env holds random secrets and .gitignore is the repository's own.
	notCompared := map[string]bool{".env": true, ".gitignore": true}
	normalize := func(s string) string { return strings.ReplaceAll(s, "\r\n", "\n") }

	var stale []string
	check := func(rel, want string) {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
		switch {
		case err != nil:
			stale = append(stale, rel+" (missing)")
		case normalize(string(got)) != normalize(want):
			stale = append(stale, rel)
		}
	}
	for path, file := range files {
		if rel := filepath.ToSlash(path); !notCompared[rel] {
			check(rel, file.Content)
		}
	}
	for name, file := range manifests {
		check("k8s-manifests/"+name, file.Content)
	}

	if len(stale) > 0 {
		sort.Strings(stale)
		t.Fatalf("the sample project no longer matches the generator:\n  %s\n\n"+
			"Regenerate it and copy the differing files (not .env or .gitignore):\n"+
			"  dev-box init --config dev-box.yaml --force --output-dir <temp dir>\n"+
			"  dev-box k8s  --config dev-box.yaml --force --output-dir k8s-manifests",
			strings.Join(stale, "\n  "))
	}
}
