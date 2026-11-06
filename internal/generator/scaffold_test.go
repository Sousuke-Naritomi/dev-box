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

// A server without ReadHeaderTimeout lets a slow client hold a connection open
// indefinitely (gosec G114); the sample should not teach that pattern.
func TestGoScaffoldServerSetsAHeaderTimeout(t *testing.T) {
	source := scaffoldFile(t, config.BackendGo, "backend/main.go")
	if !strings.Contains(source, "ReadHeaderTimeout") {
		t.Errorf("the Go sample must configure http.Server.ReadHeaderTimeout:\n%s", source)
	}
	if strings.Contains(source, "http.ListenAndServe(") {
		t.Error("use an explicit http.Server instead of the package-level ListenAndServe, which has no timeouts")
	}
}

// The Flask development server is not meant to serve traffic, so the production
// image runs the app under gunicorn. The development Compose service keeps
// `python main.py` for fast restarts.
func TestPythonImageRunsGunicornNotTheDevelopmentServer(t *testing.T) {
	dockerfile := scaffoldFile(t, config.BackendPython, "backend/Dockerfile")

	var cmd string
	for _, line := range strings.Split(dockerfile, "\n") {
		if strings.HasPrefix(line, "CMD ") {
			cmd = line
		}
	}
	if cmd == "" {
		t.Fatalf("no CMD in:\n%s", dockerfile)
	}
	if strings.Contains(cmd, "main.py") {
		t.Errorf("the image must not start the Flask development server: %s", cmd)
	}
	for _, want := range []string{"gunicorn", "exec ", "0.0.0.0:${BACKEND_PORT}", "main:app"} {
		if !strings.Contains(cmd, want) {
			t.Errorf("CMD should contain %q (exec makes gunicorn PID 1 so it receives stop signals; BACKEND_PORT keeps the port configurable): %s", want, cmd)
		}
	}
	if !strings.Contains(dockerfile, "USER 10001:10001") {
		t.Error("the image must keep running as a non-root user")
	}

	files, err := GenerateScaffold(config.ProjectConfig{Frontend: config.FrontendNone, Backend: config.BackendPython})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(files["backend/requirements.txt"].Content, "gunicorn") {
		t.Error("gunicorn must be installed in the image")
	}
}
