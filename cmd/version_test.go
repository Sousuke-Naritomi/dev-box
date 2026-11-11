package cmd

import (
	"bytes"
	"context"
	"runtime/debug"
	"testing"
)

func buildInfo(mainVersion string, settings map[string]string) *debug.BuildInfo {
	info := &debug.BuildInfo{Main: debug.Module{Version: mainVersion}}
	for key, value := range settings {
		info.Settings = append(info.Settings, debug.BuildSetting{Key: key, Value: value})
	}
	return info
}

func TestResolvedVersion(t *testing.T) {
	tests := []struct {
		name     string
		injected string
		info     *debug.BuildInfo
		want     string
	}{
		{"release build keeps the injected version", "1.4.0", buildInfo("v9.9.9", nil), "1.4.0"},
		{"go install of a tagged release", "dev", buildInfo("v1.2.3", nil), "v1.2.3"},
		{"go install of an untagged commit", "dev", buildInfo("v0.0.0-20261106081119-a50ec0b1c2d3", nil), "v0.0.0-20261106081119-a50ec0b1c2d3"},
		{"checkout build shows the commit", "dev", buildInfo("(devel)", map[string]string{"vcs.revision": "a50ec0b1c2d3e4f5"}), "dev+a50ec0b"},
		{"uncommitted changes are marked", "dev", buildInfo("(devel)", map[string]string{"vcs.revision": "a50ec0b1c2d3e4f5", "vcs.modified": "true"}), "dev+a50ec0b-dirty"},
		{"a short revision is kept whole", "dev", buildInfo("", map[string]string{"vcs.revision": "abc"}), "dev+abc"},
		{"no build information", "dev", nil, "dev"},
		{"build information without a commit", "dev", buildInfo("(devel)", nil), "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolvedVersion(tt.injected, tt.info); got != tt.want {
				t.Errorf("resolvedVersion(%q) = %q, want %q", tt.injected, got, tt.want)
			}
		})
	}
}

func TestVersionCommandPrintsTheResolvedVersion(t *testing.T) {
	original := displayVersion
	displayVersion = "v1.2.3"
	t.Cleanup(func() { displayVersion = original })

	var out, errOut bytes.Buffer
	rootCmd.SetOut(&out)
	rootCmd.SetErr(&errOut)
	rootCmd.SetArgs([]string{"version"})
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("version failed: %v\n%s", err, errOut.String())
	}
	if want := "dev-box v1.2.3\ntemplate schema " + TemplateVersion + "\n"; out.String() != want {
		t.Errorf("output = %q, want %q", out.String(), want)
	}
	if rootCmd.Version == "" {
		t.Error("rootCmd.Version must be set so --version works")
	}
}
