package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type runnerCall struct {
	args      []string
	outputDir string
}

func stubRunner(t *testing.T, fn func(args []string, outputDir string) (string, error)) *[]runnerCall {
	t.Helper()
	calls := &[]runnerCall{}
	original := runDevBoxFn
	runDevBoxFn = func(_ context.Context, args []string, outputDir string) (string, error) {
		*calls = append(*calls, runnerCall{args: args, outputDir: outputDir})
		return fn(args, outputDir)
	}
	t.Cleanup(func() { runDevBoxFn = original })
	return calls
}

func postJSON(handler http.Handler, path, body string, mutate func(*http.Request)) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Host = "127.0.0.1:5173"
	req.Header.Set("Content-Type", "application/json")
	if mutate != nil {
		mutate(req)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestIsLoopbackHost(t *testing.T) {
	tests := map[string]bool{
		"localhost":              true,
		"LOCALHOST:5173":         true,
		"127.0.0.1":              true,
		"127.0.0.1:8080":         true,
		"[::1]:8080":             true,
		"::1":                    true,
		"evil.example":           false,
		"127.0.0.1.evil.example": false,
		"10.0.0.1:8080":          false,
		"0.0.0.0":                false,
		"":                       false,
	}
	for host, want := range tests {
		if got := isLoopbackHost(host); got != want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestLocalOnlyMiddleware(t *testing.T) {
	handler := newHandler()
	tests := []struct {
		name   string
		host   string
		origin string
		want   int
	}{
		{"loopback without origin", "127.0.0.1:8080", "", http.StatusOK},
		{"vite proxy origin", "127.0.0.1:5173", "http://127.0.0.1:5173", http.StatusOK},
		{"localhost origin", "localhost:5173", "http://localhost:5173", http.StatusOK},
		{"foreign host (DNS rebinding)", "evil.example", "", http.StatusForbidden},
		{"foreign origin (CSRF)", "127.0.0.1:8080", "https://evil.example", http.StatusForbidden},
		{"null origin", "127.0.0.1:8080", "null", http.StatusForbidden},
		{"lookalike origin", "127.0.0.1:8080", "http://127.0.0.1.evil.example", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/health", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestNoWildcardCORSHeaders(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/options", nil)
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("Access-Control-Allow-Origin = %q, want empty", got)
	}
}

func TestPreviewRejectsInvalidRequestsWithoutRunningDevBox(t *testing.T) {
	calls := stubRunner(t, func([]string, string) (string, error) { return `{"files":[]}`, nil })
	handler := newHandler()

	tests := []struct {
		name   string
		body   string
		mutate func(*http.Request)
		want   int
	}{
		{"text/plain content type", `{"frontend":"react"}`, func(r *http.Request) { r.Header.Set("Content-Type", "text/plain") }, http.StatusBadRequest},
		{"missing content type", `{"frontend":"react"}`, func(r *http.Request) { r.Header.Del("Content-Type") }, http.StatusBadRequest},
		{"malformed JSON", `{`, nil, http.StatusBadRequest},
		{"unknown frontend", `{"frontend":"angular"}`, nil, http.StatusBadRequest},
		{"unknown backend", `{"backend":"ruby"}`, nil, http.StatusBadRequest},
		{"unknown database", `{"database":"oracle"}`, nil, http.StatusBadRequest},
		{"unknown cache", `{"cache":"memcached"}`, nil, http.StatusBadRequest},
		{"empty stack", `{"frontend":"none","backend":"none","database":"none","cache":"none"}`, nil, http.StatusBadRequest},
		{"nginx without workload", `{"frontend":"none","backend":"none","database":"postgres","nginx":true}`, nil, http.StatusBadRequest},
		{"oversized body", `{"project":"` + strings.Repeat("a", maxRequestBytes) + `"}`, nil, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := postJSON(handler, "/api/preview", tt.body, tt.mutate)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (body: %s)", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
	if len(*calls) != 0 {
		t.Fatalf("dev-box was invoked %d times for invalid requests", len(*calls))
	}
}

func TestPreviewRejectsNonPost(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/preview", nil)
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestPreviewMasksSecretsClassifiesAndSortsFiles(t *testing.T) {
	calls := stubRunner(t, func([]string, string) (string, error) {
		return `{"files":[
			{"path":"studio-preview/.env","content":"DB_PASSWORD=hunter2\nPORT=3000\nGRAFANA_SECRET=s3cret"},
			{"path":"studio-preview/backend/main.go","content":"package main\n"},
			{"path":"studio-preview/docker-compose.yml","content":"services:\n  api: {}\n"}
		]}`, nil
	})

	rec := postJSON(newHandler(), "/api/preview", `{"frontend":"none","backend":"go","database":"postgres"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var resp previewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}

	var paths []string
	for _, f := range resp.Files {
		paths = append(paths, f.Path)
	}
	if want := []string{"docker-compose.yml", "backend/main.go", ".env"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}

	env := resp.Files[2]
	if env.Kind != "環境変数" {
		t.Errorf(".env kind = %q", env.Kind)
	}
	if strings.Contains(env.Content, "hunter2") || strings.Contains(env.Content, "s3cret") {
		t.Errorf("secrets leaked into preview: %q", env.Content)
	}
	if !strings.Contains(env.Content, "PORT=3000") {
		t.Errorf("non-secret value was altered: %q", env.Content)
	}
	if env.Lines != 3 || env.Bytes != len(env.Content) {
		t.Errorf("lines/bytes = %d/%d", env.Lines, env.Bytes)
	}

	if len(*calls) != 1 {
		t.Fatalf("expected 1 dev-box call, got %d", len(*calls))
	}
	args := (*calls)[0].args
	for _, want := range []string{"--dry-run", "--format", "json", "--force"} {
		if !contains(args, want) {
			t.Errorf("args %v missing %s", args, want)
		}
	}
}

func TestPreviewEmptyDevBoxOutputIsBadGateway(t *testing.T) {
	stubRunner(t, func([]string, string) (string, error) { return `{"files":[]}`, nil })
	rec := postJSON(newHandler(), "/api/preview", `{}`, nil)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
}

func TestGenerateRejectsUnsafeProjectNames(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DEV_BOX_OUTPUT_ROOT", root)
	calls := stubRunner(t, func([]string, string) (string, error) { return "", nil })

	names := []string{"../escape", "..", ".", "a/b", `a\b`, "a", "-leading", "_leading", "has space", "semi;colon", "$(whoami)", strings.Repeat("a", 41), "日本語"}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			body, _ := json.Marshal(map[string]any{"frontend": "react", "project": name})
			rec := postJSON(newHandler(), "/api/generate", string(body), nil)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
			}
		})
	}
	if len(*calls) != 0 {
		t.Fatalf("dev-box was invoked for rejected names: %v", *calls)
	}
}

func TestGenerateWritesInsideOutputRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DEV_BOX_OUTPUT_ROOT", root)
	stubRunner(t, func(_ []string, outputDir string) (string, error) {
		if err := os.MkdirAll(filepath.Join(outputDir, "backend"), 0o755); err != nil {
			return "", err
		}
		for _, name := range []string{"dev-box.yaml", filepath.Join("backend", "main.go")} {
			if err := os.WriteFile(filepath.Join(outputDir, name), []byte("x"), 0o644); err != nil {
				return "", err
			}
		}
		return "", nil
	})

	rec := postJSON(newHandler(), "/api/generate", `{"frontend":"none","backend":"go","project":"my-app_1"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	var resp generateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.FileCount != 2 {
		t.Errorf("fileCount = %d, want 2", resp.FileCount)
	}
	wantPath := filepath.Join(root, "my-app_1")
	if resp.Path != wantPath {
		t.Errorf("path = %q, want %q", resp.Path, wantPath)
	}
}

func TestGenerateDefaultsProjectName(t *testing.T) {
	root := t.TempDir()
	t.Setenv("DEV_BOX_OUTPUT_ROOT", root)
	calls := stubRunner(t, func(_ []string, outputDir string) (string, error) {
		return "", os.MkdirAll(outputDir, 0o755)
	})
	rec := postJSON(newHandler(), "/api/generate", `{"backend":"go"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if got := (*calls)[0].outputDir; got != filepath.Join(root, "studio-sample") {
		t.Fatalf("outputDir = %q", got)
	}
}

func TestContainedPaths(t *testing.T) {
	root := t.TempDir()
	tests := []struct {
		name    string
		target  string
		wantErr bool
	}{
		{"child", filepath.Join(root, "app"), false},
		{"nested child", filepath.Join(root, "a", "b"), false},
		{"dot-dot prefixed name is not traversal", filepath.Join(root, "..cache"), false},
		{"root itself", root, true},
		{"parent", filepath.Dir(root), true},
		{"sibling via traversal", filepath.Join(root, "..", "other"), true},
		{"traversal back inside", filepath.Join(root, "a", "..", "b"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := containedPaths(root, tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDevBoxSourceDir(t *testing.T) {
	t.Run("explicit directory wins", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("DEV_BOX_SOURCE_DIR", dir)
		got, err := devBoxSourceDir()
		if err != nil {
			t.Fatal(err)
		}
		if got != filepath.Clean(dir) {
			t.Fatalf("got %q, want %q", got, dir)
		}
	})

	t.Run("falls back to the repository root without hardcoded paths", func(t *testing.T) {
		t.Setenv("DEV_BOX_SOURCE_DIR", "")
		got, err := devBoxSourceDir()
		if err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(got, "go.mod"))
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(string(data), "module github.com/Sousuke-Naritomi/dev-box\n") {
			t.Fatalf("resolved %q which is not the dev-box module:\n%s", got, data)
		}
	})
}

func TestMaskSensitiveFile(t *testing.T) {
	in := "DB_PASSWORD=a\nAPI_TOKEN=b\nSIGNING_KEY=c\nAPP_SECRET=d\nPORT=8080\nNO_EQUALS_LINE"
	got := maskSensitiveFile(".env", in)
	for _, leaked := range []string{"=a", "=b", "=c", "=d"} {
		if strings.Contains(got, leaked+"\n") || strings.HasSuffix(got, leaked) {
			t.Errorf("secret survived masking (%s): %q", leaked, got)
		}
	}
	for _, kept := range []string{"PORT=8080", "NO_EQUALS_LINE"} {
		if !strings.Contains(got, kept) {
			t.Errorf("%q was altered: %q", kept, got)
		}
	}
	if untouched := maskSensitiveFile("backend/config.txt", in); untouched != in {
		t.Errorf("non-.env file was modified")
	}
}

func TestClassifyFile(t *testing.T) {
	tests := map[string]string{
		"docker-compose.yml":        "Compose設定",
		".github/workflows/ci.yml":  "CI",
		"k8s/api.yaml":              "インフラ",
		"terraform/main.tf":         "インフラ",
		"frontend/src/App.jsx":      "フロントエンド",
		"backend/main.go":           "バックエンド",
		"monitoring/prometheus.yml": "運用設定",
		"nginx/nginx.conf":          "運用設定",
		".env":                      "環境変数",
		".env.example":              "環境変数",
		"dev-box.yaml":              "プロジェクト",
	}
	for path, want := range tests {
		if got := classifyFile(path); got != want {
			t.Errorf("classifyFile(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestNormalizePreviewPathAndLineCount(t *testing.T) {
	if got := normalizePreviewPath("studio-preview/backend/main.go", "studio-preview"); got != "backend/main.go" {
		t.Errorf("normalizePreviewPath = %q", got)
	}
	if got := normalizePreviewPath("other/main.go", "studio-preview"); got != "other/main.go" {
		t.Errorf("normalizePreviewPath should keep foreign prefixes, got %q", got)
	}
	for in, want := range map[string]int{"": 0, "a": 1, "a\n": 2, "a\nb": 2, "a\nb\nc": 3} {
		if got := lineCount(in); got != want {
			t.Errorf("lineCount(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestDevBoxArgs(t *testing.T) {
	cfg := stackConfig{Frontend: "vue", Backend: "node", Database: "mysql", Cache: "redis", Nginx: true, Monitoring: true, CICD: true}
	got := devBoxArgs(cfg, true, "out")
	want := []string{
		"init", "--frontend", "vue", "--backend", "node", "--database", "mysql", "--cache", "redis",
		"--force", "--output-dir", "out", "--nginx", "--monitoring", "--cicd", "--dry-run", "--format", "json",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("args = %v\nwant   %v", got, want)
	}
	if contains(devBoxArgs(cfg, false, "out"), "--dry-run") {
		t.Fatal("--dry-run must only be set for previews")
	}
}

func TestDevBoxArgsPassesCloudOnlyWhenSelected(t *testing.T) {
	for _, cloud := range []string{"", "none"} {
		if contains(devBoxArgs(stackConfig{Frontend: "none", Backend: "go", Cloud: cloud}, false, "out"), "--cloud") {
			t.Errorf("cloud %q must not add --cloud", cloud)
		}
	}
	got := devBoxArgs(stackConfig{Frontend: "none", Backend: "go", CICD: true, Cloud: "aws"}, true, "out")
	idx := -1
	for i, arg := range got {
		if arg == "--cloud" {
			idx = i
		}
	}
	if idx < 0 || got[idx+1] != "aws" {
		t.Fatalf("expected --cloud aws in %v", got)
	}
}

func TestOptionsAndHealth(t *testing.T) {
	handler := newHandler()
	for _, path := range []string{"/health", "/", "/metrics", "/api/options"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Host = "127.0.0.1:8080"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("GET %s = %d", path, rec.Code)
		}
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestCloudSelectionIsValidatedDefaultedAndForwarded(t *testing.T) {
	calls := stubRunner(t, func([]string, string) (string, error) {
		return `{"files":[{"path":"studio-preview/terraform/main.tf","content":"terraform {}\n"},{"path":"studio-preview/dev-box.yaml","content":"cloud: aws\n"}]}`, nil
	})
	handler := newHandler()

	for _, bad := range []string{"heroku", "AWS", "aws ", "azure;rm"} {
		body, _ := json.Marshal(map[string]any{"backend": "go", "cloud": bad})
		if rec := postJSON(handler, "/api/preview", string(body), nil); rec.Code != http.StatusBadRequest {
			t.Errorf("cloud %q: status = %d, want %d", bad, rec.Code, http.StatusBadRequest)
		}
	}
	if len(*calls) != 0 {
		t.Fatalf("dev-box must not run for an invalid cloud: %v", *calls)
	}

	rec := postJSON(handler, "/api/preview", `{"backend":"go","cloud":"aws"}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body: %s", rec.Code, rec.Body.String())
	}
	if args := (*calls)[0].args; !contains(args, "--cloud") || !contains(args, "aws") {
		t.Errorf("cloud was not forwarded to dev-box: %v", args)
	}
	var resp previewResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Summary, "Terraform (AWS)") {
		t.Errorf("summary should mention Terraform: %q", resp.Summary)
	}
	for _, f := range resp.Files {
		if f.Path == "terraform/main.tf" && f.Kind != "インフラ" {
			t.Errorf("terraform files belong to the infra group, got %q", f.Kind)
		}
	}

	*calls = nil
	if rec := postJSON(handler, "/api/preview", `{"backend":"go"}`, nil); rec.Code != http.StatusOK {
		t.Fatalf("a request without cloud must still work: %d", rec.Code)
	}
	if contains((*calls)[0].args, "--cloud") {
		t.Errorf("an omitted cloud must default to none: %v", (*calls)[0].args)
	}
}

func TestOptionsListTheClouds(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/options", nil)
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, req)
	var got map[string][]string
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got["clouds"], []string{"none", "aws", "gcp", "azure"}) {
		t.Errorf("clouds = %v", got["clouds"])
	}
}

func TestMotoIsOfferedAndForwarded(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/options", nil)
	req.Host = "127.0.0.1:8080"
	rec := httptest.NewRecorder()
	newHandler().ServeHTTP(rec, req)
	var opts map[string][]string
	if err := json.Unmarshal(rec.Body.Bytes(), &opts); err != nil {
		t.Fatal(err)
	}
	if !contains(opts["addons"], "moto") || contains(opts["addons"], "localstack") {
		t.Errorf("addons = %v, want moto and not localstack", opts["addons"])
	}

	args := devBoxArgs(stackConfig{Frontend: "none", Backend: "go", Moto: true}, false, "out")
	if !contains(args, "--moto") || contains(args, "--localstack") {
		t.Errorf("args = %v", args)
	}
	if !strings.Contains(summarize(stackConfig{Frontend: "none", Backend: "go", Moto: true}, 5), "Moto") {
		t.Error("the preview summary should mention Moto")
	}
}

func TestMotoAloneIsAValidSelection(t *testing.T) {
	stubRunner(t, func([]string, string) (string, error) {
		return `{"files":[{"path":"studio-preview/docker-compose.yml","content":"services: {}\n"}]}`, nil
	})
	rec := postJSON(newHandler(), "/api/preview", `{"frontend":"none","backend":"none","database":"none","cache":"none","moto":true}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("Moto alone is a workload; status = %d, body: %s", rec.Code, rec.Body.String())
	}
}
