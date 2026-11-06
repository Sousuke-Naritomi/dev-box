package config

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"
	"time"
)

// pinnedComponent is a pinned version whose support status endoflife.date tracks.
type pinnedComponent struct {
	Constant string
	Version  string
	Product  string
}

var trackedComponents = []pinnedComponent{
	{"VersionNode", VersionNode, "nodejs"},
	{"VersionPython", VersionPython, "python"},
	{"VersionGo", VersionGo, "go"},
	{"VersionPHP", VersionPHP, "php"},
	{"VersionJava", VersionJava, "eclipse-temurin"},
	{"VersionMySQL", VersionMySQL, "mysql"},
	{"VersionPostgres", VersionPostgres, "postgresql"},
	{"VersionRedis", VersionRedis, "redis"},
	{"VersionNginx", VersionNginx, "nginx"},
	{"VersionGrafana", VersionGrafana, "grafana"},
	{"VersionLoki", VersionLoki, "grafana-loki"},
}

// untrackedComponents are pinned versions that endoflife.date has no data for.
// Each needs a reason, and a person to look at it when bumping.
var untrackedComponents = map[string]string{
	"VersionMoto":       "releases often and is not on endoflife.date; bump it by hand (see versions.go)",
	"VersionPrometheus": "not on endoflife.date",
	"VersionAlloy":      "not on endoflife.date",
	"VersionTrivy":      "a CI tool, pinned together with trivy-action",
}

// releaseLine turns an image tag or version into the release line that
// endoflife.date lists: "24-alpine" -> "24", "3.12-slim" -> "3.12",
// "v3.16.0" -> "3.16", "13.3.0" -> "13.3".
func releaseLine(version string) string {
	v := strings.TrimPrefix(version, "v")
	v, _, _ = strings.Cut(v, "-")
	parts := strings.Split(v, ".")
	if len(parts) > 2 {
		parts = parts[:2]
	}
	return strings.Join(parts, ".")
}

// supportedUntil reports whether a release line is still supported for at least
// margin. eol is endoflife.date's "eol" field: false (no end date announced),
// true (already ended) or a date.
func supportedUntil(eol any, now time.Time, margin time.Duration) (bool, string) {
	switch v := eol.(type) {
	case bool:
		if v {
			return false, "already end of life"
		}
		return true, "no end of life announced"
	case string:
		date, err := time.Parse("2006-01-02", v)
		if err != nil {
			return false, "unreadable end-of-life date " + v
		}
		if date.Before(now.Add(margin)) {
			return false, "end of life " + v
		}
		return true, "supported until " + v
	default:
		return false, "unexpected end-of-life value"
	}
}

func TestReleaseLine(t *testing.T) {
	tests := map[string]string{
		"24-alpine": "24", "3.12-slim": "3.12", "1.27-alpine": "1.27", "8.3-apache": "8.3",
		"17-jdk": "17", "8.4": "8.4", "7-alpine": "7", "1.30-alpine": "1.30",
		"v3.16.0": "3.16", "13.3.0": "13.3", "3.7.9": "3.7",
	}
	for in, want := range tests {
		if got := releaseLine(in); got != want {
			t.Errorf("releaseLine(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSupportedUntil(t *testing.T) {
	now := time.Date(2026, 11, 6, 0, 0, 0, 0, time.UTC)
	margin := 45 * 24 * time.Hour
	tests := []struct {
		name string
		eol  any
		want bool
	}{
		{"no end announced", false, true},
		{"already ended", true, false},
		{"far in the future", "2028-04-30", true},
		{"already passed", "2026-04-30", false},
		{"inside the margin", "2026-12-01", false},
		{"just outside the margin", "2026-12-25", true},
		{"unreadable", "soon", false},
		{"unexpected type", 42, false},
	}
	for _, tt := range tests {
		if got, detail := supportedUntil(tt.eol, now, margin); got != tt.want {
			t.Errorf("%s: supportedUntil(%v) = %v (%s), want %v", tt.name, tt.eol, got, detail, tt.want)
		}
	}
}

// Adding a pinned version must come with a decision: track its support status,
// or record why that is not possible.
func TestEveryPinnedVersionIsTrackedOrExplained(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "versions.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	var declared []string
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			for _, name := range spec.(*ast.ValueSpec).Names {
				if strings.HasPrefix(name.Name, "Version") {
					declared = append(declared, name.Name)
				}
			}
		}
	}
	known := map[string]bool{}
	for _, c := range trackedComponents {
		known[c.Constant] = true
		if c.Product == "" || c.Version == "" {
			t.Errorf("%s needs a product and a version", c.Constant)
		}
	}
	for name, reason := range untrackedComponents {
		if known[name] {
			t.Errorf("%s is both tracked and exempt", name)
		}
		if strings.TrimSpace(reason) == "" {
			t.Errorf("%s is exempt without a reason", name)
		}
		known[name] = true
	}
	sort.Strings(declared)
	for _, name := range declared {
		if !known[name] {
			t.Errorf("%s is pinned in versions.go but neither tracked for end of life nor listed as exempt (see eol_test.go)", name)
		}
	}
	for name := range known {
		found := false
		for _, d := range declared {
			found = found || d == name
		}
		if !found {
			t.Errorf("%s is listed in eol_test.go but no longer declared in versions.go", name)
		}
	}
}
