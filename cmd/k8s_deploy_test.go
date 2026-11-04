package cmd

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"golang.org/x/term"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

func skipIfStdinIsTerminal(t *testing.T) {
	t.Helper()
	if term.IsTerminal(int(os.Stdin.Fd())) {
		t.Skip("stdin is a terminal; this test exercises the non-interactive path")
	}
}

func resetFlags(t *testing.T, flags *pflag.FlagSet) {
	t.Helper()
	t.Cleanup(func() {
		flags.VisitAll(func(f *pflag.Flag) {
			_ = f.Value.Set(f.DefValue)
			f.Changed = false
		})
	})
}

func fullStack() config.ProjectConfig {
	return config.ProjectConfig{
		Frontend: config.FrontendReact,
		Backend:  config.BackendGo,
		Database: config.DatabasePostgres,
		Cache:    config.CacheRedis,
	}
}

func TestRunK8sLogicGeneratesManifestsForSelectedServices(t *testing.T) {
	tests := []struct {
		name    string
		cfg     config.ProjectConfig
		want    []string
		notWant []string
	}{
		{
			name: "full stack",
			cfg:  fullStack(),
			want: []string{"namespace.yaml", "networkpolicy.yaml", "frontend.yaml", "api.yaml", "db.yaml", "cache.yaml"},
		},
		{
			name:    "backend only",
			cfg:     config.ProjectConfig{Frontend: config.FrontendNone, Backend: config.BackendNode, Database: config.DatabaseNone, Cache: config.CacheNone},
			want:    []string{"namespace.yaml", "api.yaml"},
			notWant: []string{"frontend.yaml", "db.yaml", "cache.yaml"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileSystem := fs.NewMockFileSystem()
			err := runK8sLogic(context.Background(), tt.cfg, "k8s", true, false, &bytes.Buffer{}, fileSystem, &generator.DefaultGenerator{})
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range tt.want {
				if !fileSystem.Exists(filepath.Join("k8s", name)) {
					t.Errorf("expected %s to be generated", name)
				}
			}
			for _, name := range tt.notWant {
				if fileSystem.Exists(filepath.Join("k8s", name)) {
					t.Errorf("%s must not be generated for this stack", name)
				}
			}
		})
	}
}

func TestRunK8sLogicDryRunWritesNothingAndPrintsManifests(t *testing.T) {
	fileSystem := fs.NewMockFileSystem()
	var out bytes.Buffer
	if err := runK8sLogic(context.Background(), fullStack(), "k8s", false, true, &out, fileSystem, &generator.DefaultGenerator{}); err != nil {
		t.Fatal(err)
	}
	if len(fileSystem.Files) != 0 {
		t.Fatalf("dry run wrote files: %v", fileSystem.Files)
	}
	if !strings.Contains(out.String(), "kind: Deployment") {
		t.Fatalf("dry run output should include the manifests:\n%s", out.String())
	}
}

func TestRunK8sLogicPropagatesGeneratorErrors(t *testing.T) {
	gen := &MockGenerator{K8sFunc: func(config.ProjectConfig) (map[string]generator.FileDef, error) {
		return nil, errors.New("boom")
	}}
	err := runK8sLogic(context.Background(), fullStack(), "k8s", true, false, &bytes.Buffer{}, fs.NewMockFileSystem(), gen)
	if err == nil || !strings.Contains(err.Error(), "generating k8s manifests") || !strings.Contains(err.Error(), "boom") {
		t.Fatalf("expected wrapped generator error, got %v", err)
	}
}

func TestRunDeployLogicGeneratesTerraformForEveryProvider(t *testing.T) {
	for _, provider := range []string{"aws", "gcp", "azure"} {
		t.Run(provider, func(t *testing.T) {
			fileSystem := fs.NewMockFileSystem()
			if err := runDeployLogic(context.Background(), provider, config.ProjectConfig{}, "tf", true, false, &bytes.Buffer{}, fileSystem, &generator.DefaultGenerator{}); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"main.tf", "variables.tf", "terraform.tfvars.example"} {
				if !fileSystem.Exists(filepath.Join("tf", name)) {
					t.Errorf("expected %s", name)
				}
			}
		})
	}
}

func TestRunDeployLogicRejectsUnknownProvider(t *testing.T) {
	fileSystem := fs.NewMockFileSystem()
	err := runDeployLogic(context.Background(), "digitalocean", config.ProjectConfig{}, "tf", true, false, &bytes.Buffer{}, fileSystem, &generator.DefaultGenerator{})
	if err == nil || !strings.Contains(err.Error(), "generating terraform") {
		t.Fatalf("expected an error for an unsupported provider, got %v", err)
	}
	if len(fileSystem.Files) != 0 {
		t.Fatalf("nothing may be written for an unsupported provider: %v", fileSystem.Files)
	}
}

func TestGeneratedFilesProtectExistingFilesWithoutForce(t *testing.T) {
	skipIfStdinIsTerminal(t)

	fileSystem := fs.NewMockFileSystem()
	fileSystem.Files[filepath.Join("k8s", "namespace.yaml")] = fs.MockFile{Data: []byte("precious"), Perm: 0o644}

	var out bytes.Buffer
	err := runK8sLogic(context.Background(), fullStack(), "k8s", false, false, &out, fileSystem, &generator.DefaultGenerator{})
	if err == nil || !strings.Contains(err.Error(), "--force") {
		t.Fatalf("expected a refusal that mentions --force, got %v", err)
	}
	if !strings.Contains(out.String(), filepath.Join("k8s", "namespace.yaml")) {
		t.Errorf("the overwrite list should name the conflicting file:\n%s", out.String())
	}
	if got := string(fileSystem.Files[filepath.Join("k8s", "namespace.yaml")].Data); got != "precious" {
		t.Errorf("existing file was modified: %q", got)
	}
	if len(fileSystem.Files) != 1 {
		t.Errorf("no other files may be written when the run is refused: %v", fileSystem.Files)
	}
}

func TestK8sCommandNeedsAllStackValuesWhenNonInteractive(t *testing.T) {
	skipIfStdinIsTerminal(t)
	resetFlags(t, k8sCmd.Flags())

	app.FS = fs.NewMockFileSystem()
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"k8s", "--frontend=react"})

	err := rootCmd.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "non-interactive") {
		t.Fatalf("expected a non-interactive error, got %v", err)
	}
}

func TestK8sCommandRejectsEmptySelection(t *testing.T) {
	resetFlags(t, k8sCmd.Flags())

	app.FS = fs.NewMockFileSystem()
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"k8s", "--frontend=none", "--backend=none", "--database=none", "--cache=none"})

	err := rootCmd.ExecuteContext(context.Background())
	if err == nil || !strings.Contains(err.Error(), "at least one") {
		t.Fatalf("expected an error for a stack without workloads, got %v", err)
	}
}

func TestK8sCommandPrintsSecretReminderWhenDatabaseSelected(t *testing.T) {
	resetFlags(t, k8sCmd.Flags())

	mockFS := fs.NewMockFileSystem()
	app.FS = mockFS
	app.Gen = &generator.DefaultGenerator{}
	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"k8s", "--frontend=none", "--backend=go", "--database=postgres", "--cache=none", "--force", "--output-dir=manifests"})

	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(out.String(), "db-secret") {
		t.Errorf("users must be reminded to create db-secret:\n%s", out.String())
	}
	if !mockFS.Exists(filepath.Join("manifests", "db.yaml")) {
		t.Error("db.yaml should be generated")
	}
}
