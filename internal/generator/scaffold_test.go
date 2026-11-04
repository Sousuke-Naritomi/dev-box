package generator

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
)

func scaffoldFile(t *testing.T, backend config.BackendType, path string) string {
	t.Helper()
	files, err := GenerateScaffold(config.ProjectConfig{Frontend: config.FrontendNone, Backend: backend})
	if err != nil {
		t.Fatal(err)
	}
	file, ok := files[path]
	if !ok {
		t.Fatalf("scaffold for %s did not produce %s", backend, path)
	}
	return file.Content
}

func TestJavaRuntimeStageMatchesBuildStage(t *testing.T) {
	dockerfile := scaffoldFile(t, config.BackendJava, "backend/Dockerfile")
	froms := regexp.MustCompile(`(?m)^FROM eclipse-temurin:(\S+)`).FindAllStringSubmatch(dockerfile, -1)
	if len(froms) != 2 {
		t.Fatalf("expected a build and a runtime stage, got %v\n%s", froms, dockerfile)
	}
	build, runtime := froms[0][1], froms[1][1]
	if build != config.VersionJava {
		t.Errorf("build stage uses %q, want %q", build, config.VersionJava)
	}
	major := func(tag string) string { return strings.SplitN(tag, "-", 2)[0] }
	if major(runtime) != major(build) {
		t.Errorf("classes compiled by JDK %s cannot run on JRE %s", build, runtime)
	}
	if !strings.HasSuffix(runtime, "-jre") {
		t.Errorf("runtime stage should be a JRE image, got %q", runtime)
	}
}

func TestGoModDirectiveMatchesBuilderImage(t *testing.T) {
	goMod := scaffoldFile(t, config.BackendGo, "backend/go.mod")
	m := regexp.MustCompile(`(?m)^go (\S+)$`).FindStringSubmatch(goMod)
	if m == nil {
		t.Fatalf("no go directive in:\n%s", goMod)
	}
	if want := strings.SplitN(config.VersionGo, "-", 2)[0]; m[1] != want {
		t.Errorf("go directive = %q, want %q (derived from VersionGo %q)", m[1], want, config.VersionGo)
	}
}

func TestGoScaffoldModuleIsNeutral(t *testing.T) {
	goMod := scaffoldFile(t, config.BackendGo, "backend/go.mod")
	if strings.Contains(goMod, "Sousuke") || strings.Contains(goMod, "github.com/") {
		t.Errorf("generated projects must not inherit the tool author's module path:\n%s", goMod)
	}
}

func TestDerivedVersionHelpers(t *testing.T) {
	jre := map[string]string{"17-jdk": "17-jre", "21-jdk": "21-jre", "25-jdk-alpine": "25-jre-alpine", "21": "21"}
	for in, want := range jre {
		if got := javaRuntimeVersion(in); got != want {
			t.Errorf("javaRuntimeVersion(%q) = %q, want %q", in, got, want)
		}
	}
	goVersions := map[string]string{"1.22-alpine": "1.22", "1.26-alpine": "1.26", "1.26": "1.26", "1.26.1-alpine": "1.26.1"}
	for in, want := range goVersions {
		if got := goModVersion(in); got != want {
			t.Errorf("goModVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
