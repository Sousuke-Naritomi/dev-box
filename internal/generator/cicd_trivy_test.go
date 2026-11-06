package generator

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

type workflowStep struct {
	Name string            `yaml:"name"`
	Uses string            `yaml:"uses"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
	With map[string]any    `yaml:"with"`
}

type workflowFile struct {
	Jobs map[string]struct {
		Steps []workflowStep `yaml:"steps"`
	} `yaml:"jobs"`
}

func securityScanSteps(t *testing.T) []workflowStep {
	t.Helper()
	for _, cfg := range []config.ProjectConfig{
		{Frontend: config.FrontendNone, Backend: config.BackendGo},
		{Frontend: config.FrontendReact, Backend: config.BackendNone},
		{Frontend: config.FrontendNone, Backend: config.BackendNone, Database: config.DatabasePostgres},
	} {
		rendered, err := GenerateCICD(cfg, 8080, 5173)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflowFile
		if err := yaml.Unmarshal([]byte(rendered), &wf); err != nil {
			t.Fatalf("generated CI is not valid YAML: %v\n%s", err, rendered)
		}
		if len(wf.Jobs["security-scan"].Steps) == 0 {
			t.Fatalf("no security-scan job in:\n%s", rendered)
		}
		if cfg.Backend == config.BackendGo {
			return wf.Jobs["security-scan"].Steps
		}
	}
	return nil
}

func stepIndex(steps []workflowStep, name string) int {
	for i, step := range steps {
		if step.Name == name {
			return i
		}
	}
	return -1
}

func TestGeneratedCIInstallsTrivyWithoutTheActionInstaller(t *testing.T) {
	steps := securityScanSteps(t)
	install := stepIndex(steps, "Install Trivy")
	scan := stepIndex(steps, "Run Trivy vulnerability scanner")
	if install < 0 || scan < 0 {
		t.Fatalf("expected Install Trivy and Run Trivy vulnerability scanner steps, got %+v", steps)
	}
	if install > scan {
		t.Error("Trivy must be installed before the scan step runs")
	}
	if skip, _ := steps[scan].With["skip-setup-trivy"].(bool); !skip {
		t.Errorf("the action must be told to skip its own installer, with = %v", steps[scan].With)
	}
	if !strings.HasPrefix(steps[scan].Uses, "aquasecurity/trivy-action@v") {
		t.Errorf("the scan should still use the pinned action, got %q", steps[scan].Uses)
	}
	for key, want := range map[string]any{"scan-type": "fs", "severity": "CRITICAL,HIGH", "exit-code": "1"} {
		if steps[scan].With[key] != want {
			t.Errorf("scan setting %s = %v, want %v (the policy must not change)", key, steps[scan].With[key], want)
		}
	}

	step := steps[install]
	if step.Env["TRIVY_VERSION"] != config.VersionTrivy {
		t.Errorf("TRIVY_VERSION = %q, want %q", step.Env["TRIVY_VERSION"], config.VersionTrivy)
	}
	script := step.Run
	for _, want := range []string{
		"https://github.com/aquasecurity/trivy/releases/download/v${TRIVY_VERSION}",
		"sha256sum -c -",
		`>> "${GITHUB_PATH}"`,
	} {
		if !strings.Contains(script, want) {
			t.Errorf("install script should contain %q:\n%s", want, script)
		}
	}
	for _, banned := range []string{"get.trivy.dev", "sudo"} {
		if strings.Contains(script, banned) {
			t.Errorf("install script must not contain %q", banned)
		}
	}
	if regexp.MustCompile(`\|\s*(ba)?sh(\s|$)`).MatchString(script) {
		t.Error("install script must not pipe a download into a shell")
	}
	// Downloads must land outside the workspace: the scan covers ".", and the
	// archive would otherwise be scanned too.
	cd := strings.Index(script, `cd "${RUNNER_TEMP}"`)
	firstCurl := strings.Index(script, "curl ")
	if cd < 0 || firstCurl < 0 || cd > firstCurl {
		t.Errorf("the script must change into RUNNER_TEMP before downloading:\n%s", script)
	}
}

func TestRepositoryCIInstallsTheSameTrivyAsGeneratedProjects(t *testing.T) {
	workflows, err := filepath.Glob(filepath.Join("..", "..", ".github", "workflows", "*.yml"))
	if err != nil || len(workflows) == 0 {
		t.Skipf("repository workflows not available: %v", err)
	}
	found := 0
	for _, path := range workflows {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var wf workflowFile
		if err := yaml.Unmarshal(data, &wf); err != nil {
			t.Fatalf("%s is not valid YAML: %v", path, err)
		}
		for name, job := range wf.Jobs {
			installs, scans := 0, 0
			for _, step := range job.Steps {
				if step.Name == "Install Trivy" {
					installs++
					found++
					if step.Env["TRIVY_VERSION"] != config.VersionTrivy {
						t.Errorf("%s job %s installs Trivy %q but generated projects install %q; update all together", filepath.Base(path), name, step.Env["TRIVY_VERSION"], config.VersionTrivy)
					}
				}
				if strings.HasPrefix(step.Uses, "aquasecurity/trivy-action@") {
					scans++
					if skip, _ := step.With["skip-setup-trivy"].(bool); !skip {
						t.Errorf("%s job %s runs trivy-action without skip-setup-trivy", filepath.Base(path), name)
					}
				}
			}
			if scans > 0 && installs == 0 {
				t.Errorf("%s job %s scans with Trivy but never installs it", filepath.Base(path), name)
			}
		}
	}
	if found == 0 {
		t.Error("the repository CI should install Trivy itself")
	}
}

func buildTarGz(t *testing.T, name, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// TestTrivyInstallStepVerifiesChecksum runs the generated install script with a
// stub curl, so the security property is exercised rather than just asserted:
// an archive that does not match the published checksum must never reach PATH.
func TestTrivyInstallStepVerifiesChecksum(t *testing.T) {
	if runtime.GOOS == "windows" && os.Getenv("MSYSTEM") == "" {
		t.Skip("needs a POSIX shell; run from Git Bash or Linux")
	}
	for _, tool := range []string{"bash", "tar", "sha256sum", "grep"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available", tool)
		}
	}

	steps := securityScanSteps(t)
	step := steps[stepIndex(steps, "Install Trivy")]
	version := step.Env["TRIVY_VERSION"]
	archiveName := fmt.Sprintf("trivy_%s_Linux-64bit.tar.gz", version)
	checksumsName := fmt.Sprintf("trivy_%s_checksums.txt", version)

	genuine := buildTarGz(t, "trivy", "#!/bin/sh\necho genuine trivy\n")
	tampered := buildTarGz(t, "trivy", "#!/bin/sh\necho malicious\n")
	other := "0000000000000000000000000000000000000000000000000000000000000000"

	tests := []struct {
		name      string
		served    []byte
		checksums string
		wantOK    bool
	}{
		{
			name:      "matching checksum installs the binary",
			served:    genuine,
			checksums: fmt.Sprintf("%s  trivy_%s_Linux-ARM64.tar.gz\n%s  %s\n", other, version, sha256Hex(genuine), archiveName),
			wantOK:    true,
		},
		{
			name:      "archive swapped after the checksum was published",
			served:    tampered,
			checksums: fmt.Sprintf("%s  %s\n", sha256Hex(genuine), archiveName),
		},
		{
			name:      "wrong checksum is rejected",
			served:    genuine,
			checksums: fmt.Sprintf("%s  %s\n", other, archiveName),
		},
		{
			name:      "archive missing from the checksum list is rejected",
			served:    genuine,
			checksums: fmt.Sprintf("%s  trivy_%s_Linux-ARM64.tar.gz\n", sha256Hex(genuine), version),
		},
		{
			name:      "a longer name that only ends the same way does not satisfy the check",
			served:    tampered,
			checksums: fmt.Sprintf("%s  prefix-%s\n", sha256Hex(tampered), archiveName),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			slash := func(parts ...string) string {
				return filepath.ToSlash(filepath.Join(append([]string{root}, parts...)...))
			}
			for _, dir := range []string{"assets", "bin", "runner"} {
				if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			files := map[string][]byte{
				filepath.Join("assets", archiveName):   tt.served,
				filepath.Join("assets", checksumsName): []byte(tt.checksums),
				filepath.Join("bin", "curl"): []byte("#!/bin/sh\n" +
					"for url; do :; done\n" +
					"echo \"$url\" >> \"$ASSETS/urls.log\"\n" +
					"cp \"$ASSETS/${url##*/}\" \"./${url##*/}\"\n"),
				"install.sh":  []byte(step.Run),
				"github_path": nil,
			}
			for name, content := range files {
				mode := os.FileMode(0o644)
				if name == filepath.Join("bin", "curl") {
					mode = 0o755
				}
				if err := os.WriteFile(filepath.Join(root, name), content, mode); err != nil {
					t.Fatal(err)
				}
			}

			cmd := exec.Command("bash", "--noprofile", "--norc", "-eo", "pipefail", slash("install.sh"))
			cmd.Env = append(os.Environ(),
				"PATH="+filepath.Join(root, "bin")+string(os.PathListSeparator)+os.Getenv("PATH"),
				"TRIVY_VERSION="+version,
				"RUNNER_TEMP="+slash("runner"),
				"GITHUB_PATH="+slash("github_path"),
				"ASSETS="+slash("assets"),
			)
			output, err := cmd.CombinedOutput()

			githubPath, _ := os.ReadFile(filepath.Join(root, "github_path"))
			_, binaryErr := os.Stat(filepath.Join(root, "runner", "trivy-bin", "trivy"))
			installed := binaryErr == nil && strings.Contains(string(githubPath), "trivy-bin")

			if tt.wantOK {
				if err != nil {
					t.Fatalf("install failed: %v\n%s", err, output)
				}
				if !installed {
					t.Fatalf("expected the binary on PATH, github_path=%q err=%v", githubPath, binaryErr)
				}
				urls, _ := os.ReadFile(filepath.Join(root, "assets", "urls.log"))
				for _, url := range strings.Fields(string(urls)) {
					if !strings.HasPrefix(url, "https://github.com/aquasecurity/trivy/releases/download/v"+version+"/") {
						t.Errorf("unexpected download URL %s", url)
					}
				}
				return
			}
			if err == nil {
				t.Fatalf("the install step must fail, output:\n%s", output)
			}
			if strings.TrimSpace(string(githubPath)) != "" {
				t.Errorf("a rejected archive must never be added to PATH, got %q", githubPath)
			}
			if binaryErr == nil {
				t.Error("a rejected archive must never be extracted")
			}
		})
	}
}
