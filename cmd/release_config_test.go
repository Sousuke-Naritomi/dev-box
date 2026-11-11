package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// GoReleaser injects the release version through a linker flag that names the
// variable by its import path. If the module or the variable is renamed the flag
// silently stops working and every release prints "dev", so pin it here.
func TestGoReleaserInjectsTheVersionVariable(t *testing.T) {
	goMod, err := os.ReadFile(filepath.Join("..", "go.mod"))
	if err != nil {
		t.Skipf("go.mod not available: %v", err)
	}
	module := regexp.MustCompile(`(?m)^module (\S+)`).FindSubmatch(goMod)
	if module == nil {
		t.Fatal("no module directive in go.mod")
	}

	data, err := os.ReadFile(filepath.Join("..", ".goreleaser.yaml"))
	if err != nil {
		t.Skipf(".goreleaser.yaml not available: %v", err)
	}
	var cfg struct {
		Version int `yaml:"version"`
		Builds  []struct {
			Ldflags []string `yaml:"ldflags"`
			Flags   []string `yaml:"flags"`
		} `yaml:"builds"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Version != 2 {
		t.Errorf("version = %d; current GoReleaser only reads version 2 files", cfg.Version)
	}
	if len(cfg.Builds) == 0 {
		t.Fatal("no builds configured")
	}
	want := "-X " + string(module[1]) + "/cmd.Version={{.Version}}"
	for _, build := range cfg.Builds {
		if !strings.Contains(strings.Join(build.Ldflags, " "), want) {
			t.Errorf("ldflags %v should contain %q", build.Ldflags, want)
		}
		if !strings.Contains(strings.Join(build.Flags, " "), "-trimpath") {
			t.Errorf("builds should use -trimpath so they do not embed local paths, flags = %v", build.Flags)
		}
	}
	if Version != "dev" {
		t.Errorf("Version must default to dev in source and be set only at release time, got %q", Version)
	}
}
