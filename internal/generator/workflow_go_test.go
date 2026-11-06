package generator

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// setup-go reads "go 1.26.0" from go.mod as exactly 1.26.0, an early patch release
// whose standard library carries vulnerabilities that were fixed in later
// patches (govulncheck found some). The workflows therefore ask for the latest
// patch of the go.mod minor version instead.
func TestWorkflowsUseTheLatestPatchOfTheModulesGoVersion(t *testing.T) {
	goMod, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Skipf("go.mod not available: %v", err)
	}
	m := regexp.MustCompile(`(?m)^go (\d+)\.(\d+)`).FindSubmatch(goMod)
	if m == nil {
		t.Fatal("no go directive in go.mod")
	}
	want := string(m[1]) + "." + string(m[2])

	workflows, err := filepath.Glob(filepath.Join("..", "..", ".github", "workflows", "*.yml"))
	if err != nil || len(workflows) == 0 {
		t.Skipf("workflows not available: %v", err)
	}
	checked := 0
	for _, path := range workflows {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflowFile
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		for job, def := range wf.Jobs {
			for _, step := range def.Steps {
				if !strings.HasPrefix(step.Uses, "actions/setup-go@") {
					continue
				}
				checked++
				where := filepath.Base(path) + " job " + job
				if _, file := step.With["go-version-file"]; file {
					t.Errorf("%s uses go-version-file, which pins the exact patch release in go.mod", where)
				}
				if got, _ := step.With["go-version"].(string); got != want {
					t.Errorf("%s: go-version = %v, want the go.mod minor version %q", where, step.With["go-version"], want)
				}
				if latest, _ := step.With["check-latest"].(bool); !latest {
					t.Errorf("%s: check-latest must be true so the newest patch release is used", where)
				}
			}
		}
	}
	if checked == 0 {
		t.Error("expected workflows to use actions/setup-go")
	}
}
