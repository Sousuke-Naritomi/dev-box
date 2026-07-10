package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type stackConfig struct {
	Frontend   string `json:"frontend"`
	Backend    string `json:"backend"`
	Database   string `json:"database"`
	Cache      string `json:"cache"`
	Nginx      bool   `json:"nginx"`
	LocalStack bool   `json:"localstack"`
	Monitoring bool   `json:"monitoring"`
	CICD       bool   `json:"cicd"`
	Project    string `json:"project"`
}

type generatedFile struct {
	Path    string `json:"path"`
	Kind    string `json:"kind"`
	Content string `json:"content"`
	Lines   int    `json:"lines"`
	Bytes   int    `json:"bytes"`
}

type previewResponse struct {
	Summary string          `json:"summary"`
	Command []string        `json:"command"`
	Files   []generatedFile `json:"files"`
}

type generateResponse struct {
	Path      string `json:"path"`
	FileCount int    `json:"fileCount"`
	Message   string `json:"message"`
}

var (
	allowedFrontends = map[string]bool{"react": true, "vue": true, "none": true}
	allowedBackends  = map[string]bool{"node": true, "python": true, "go": true, "php": true, "java": true, "none": true}
	allowedDatabases = map[string]bool{"mysql": true, "postgres": true, "none": true}
	allowedCaches    = map[string]bool{"redis": true, "none": true}
	projectNameRE    = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{1,39}$`)
)

func main() {
	port := os.Getenv("BACKEND_PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleRoot)
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/metrics", handleMetrics)
	mux.HandleFunc("/api/options", handleOptions)
	mux.HandleFunc("/api/preview", handlePreview)
	mux.HandleFunc("/api/generate", handleGenerate)

	handler := withCORS(mux)
	log.Printf("DevBox Studio API listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"service": "devbox-studio-api"})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	_, _ = fmt.Fprint(w, "# HELP devbox_studio_up Whether the Studio API process is running.\n# TYPE devbox_studio_up gauge\ndevbox_studio_up 1\n")
}

func handleOptions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"frontends": []string{"react", "vue", "none"},
		"backends":  []string{"go", "node", "python", "php", "java", "none"},
		"databases": []string{"postgres", "mysql", "none"},
		"caches":    []string{"redis", "none"},
		"addons":    []string{"nginx", "localstack", "monitoring", "cicd"},
	})
}

func handlePreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := readConfig(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()

	result, err := runDevBoxPreview(ctx, cfg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	cfg, err := readConfig(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if cfg.Project == "" {
		cfg.Project = "studio-sample"
	}
	if !projectNameRE.MatchString(cfg.Project) {
		http.Error(w, "project must be 2-40 characters and contain only letters, numbers, underscores, or hyphens", http.StatusBadRequest)
		return
	}

	outputRoot, err := generatedOutputRoot()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	outputDir := filepath.Join(outputRoot, cfg.Project)
	cleanRoot, cleanOutput, err := containedPaths(outputRoot, outputDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	if err := runDevBoxGenerate(ctx, cfg, cleanOutput); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	count, err := countFiles(cleanOutput)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, generateResponse{
		Path:      cleanOutput,
		FileCount: count,
		Message:   fmt.Sprintf("Generated %s inside %s", cfg.Project, cleanRoot),
	})
}

func readConfig(r *http.Request) (stackConfig, error) {
	defer r.Body.Close()
	var cfg stackConfig
	if err := json.NewDecoder(r.Body).Decode(&cfg); err != nil {
		return cfg, fmt.Errorf("invalid JSON body")
	}
	if cfg.Frontend == "" {
		cfg.Frontend = "react"
	}
	if cfg.Backend == "" {
		cfg.Backend = "go"
	}
	if cfg.Database == "" {
		cfg.Database = "none"
	}
	if cfg.Cache == "" {
		cfg.Cache = "none"
	}
	if !allowedFrontends[cfg.Frontend] {
		return cfg, fmt.Errorf("unsupported frontend: %s", cfg.Frontend)
	}
	if !allowedBackends[cfg.Backend] {
		return cfg, fmt.Errorf("unsupported backend: %s", cfg.Backend)
	}
	if !allowedDatabases[cfg.Database] {
		return cfg, fmt.Errorf("unsupported database: %s", cfg.Database)
	}
	if !allowedCaches[cfg.Cache] {
		return cfg, fmt.Errorf("unsupported cache: %s", cfg.Cache)
	}
	if cfg.Frontend == "none" && cfg.Backend == "none" && cfg.Database == "none" && cfg.Cache == "none" && !cfg.LocalStack && !cfg.Monitoring {
		return cfg, fmt.Errorf("select at least one workload or addon")
	}
	if cfg.Nginx && cfg.Frontend == "none" && cfg.Backend == "none" {
		return cfg, fmt.Errorf("nginx requires a frontend or backend")
	}
	return cfg, nil
}

func runDevBoxPreview(ctx context.Context, cfg stackConfig) (previewResponse, error) {
	outputName := "studio-preview"
	args := devBoxArgs(cfg, true, outputName)
	raw, err := runDevBox(ctx, args, "")
	if err != nil {
		return previewResponse{}, err
	}
	files := parseDryRunFiles(raw, outputName)
	if len(files) == 0 {
		return previewResponse{}, errors.New("dev-box returned no files to preview")
	}
	return previewResponse{
		Summary: summarize(cfg, len(files)),
		Command: append([]string{"dev-box"}, args...),
		Files:   files,
	}, nil
}

func runDevBoxGenerate(ctx context.Context, cfg stackConfig, outputDir string) error {
	args := devBoxArgs(cfg, false, outputDir)
	_, err := runDevBox(ctx, args, outputDir)
	return err
}

func devBoxArgs(cfg stackConfig, dryRun bool, outputDir string) []string {
	args := []string{
		"init",
		"--frontend", cfg.Frontend,
		"--backend", cfg.Backend,
		"--database", cfg.Database,
		"--cache", cfg.Cache,
		"--force",
		"--output-dir", outputDir,
	}
	if cfg.Nginx {
		args = append(args, "--nginx")
	}
	if cfg.LocalStack {
		args = append(args, "--localstack")
	}
	if cfg.Monitoring {
		args = append(args, "--monitoring")
	}
	if cfg.CICD {
		args = append(args, "--cicd")
	}
	if dryRun {
		args = append(args, "--dry-run")
	}
	return args
}

func runDevBox(ctx context.Context, devBoxArgs []string, outputDir string) (string, error) {
	sourceDir, err := devBoxSourceDir()
	if err != nil {
		return "", err
	}

	args := append([]string{"run", "."}, devBoxArgs...)
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = sourceDir
	cmd.Env = os.Environ()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("dev-box failed: %s", msg)
	}
	if outputDir != "" {
		log.Printf("dev-box generated output at %s", outputDir)
	}
	return stdout.String(), nil
}

func devBoxSourceDir() (string, error) {
	candidates := []string{}
	if env := os.Getenv("DEV_BOX_SOURCE_DIR"); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates,
		`C:\Job hunting\dev-box`,
		filepath.Join("..", "..", "dev-box"),
	)

	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		if info, err := os.Stat(filepath.Join(clean, "go.mod")); err == nil && !info.IsDir() {
			return clean, nil
		}
	}
	return "", errors.New("dev-box source directory was not found; set DEV_BOX_SOURCE_DIR")
}

func generatedOutputRoot() (string, error) {
	if env := os.Getenv("DEV_BOX_OUTPUT_ROOT"); env != "" {
		return filepath.Abs(env)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(wd, "..", "generated"))
}

func containedPaths(root, target string) (string, string, error) {
	cleanRoot, err := filepath.Abs(root)
	if err != nil {
		return "", "", err
	}
	cleanTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", err
	}
	rel, err := filepath.Rel(cleanRoot, cleanTarget)
	if err != nil {
		return "", "", err
	}
	if rel == "." || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", "", errors.New("output path escapes the generated project directory")
	}
	return cleanRoot, cleanTarget, nil
}

func parseDryRunFiles(raw, outputName string) []generatedFile {
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Buffer(make([]byte, 1024), 1024*1024*8)

	var files []generatedFile
	var currentPath string
	var content strings.Builder
	flush := func() {
		if currentPath == "" {
			return
		}
		text := strings.TrimSuffix(content.String(), "\n")
		text = maskSensitiveFile(currentPath, text)
		files = append(files, generatedFile{
			Path:    normalizePreviewPath(currentPath, outputName),
			Kind:    classifyFile(currentPath),
			Content: text,
			Lines:   lineCount(text),
			Bytes:   len([]byte(text)),
		})
		content.Reset()
	}

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "--- ") && strings.HasSuffix(line, " ---") {
			flush()
			currentPath = strings.TrimSuffix(strings.TrimPrefix(line, "--- "), " ---")
			continue
		}
		if currentPath != "" {
			content.WriteString(line)
			content.WriteByte('\n')
		}
	}
	flush()

	sort.Slice(files, func(i, j int) bool {
		if files[i].Kind == files[j].Kind {
			return files[i].Path < files[j].Path
		}
		return files[i].Kind < files[j].Kind
	})
	return files
}

func normalizePreviewPath(path, outputName string) string {
	path = filepath.ToSlash(path)
	prefix := strings.TrimSuffix(filepath.ToSlash(outputName), "/") + "/"
	return strings.TrimPrefix(path, prefix)
}

func maskSensitiveFile(path, content string) string {
	if filepath.Base(path) != ".env" {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		key, _, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		upper := strings.ToUpper(strings.TrimSpace(key))
		if strings.Contains(upper, "PASSWORD") || strings.Contains(upper, "SECRET") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "KEY") {
			lines[i] = key + "='generated-secret'"
		}
	}
	return strings.Join(lines, "\n")
}

func classifyFile(path string) string {
	slash := filepath.ToSlash(path)
	switch {
	case strings.Contains(slash, "docker-compose"):
		return "Compose"
	case strings.Contains(slash, ".github/workflows"):
		return "CI"
	case strings.Contains(slash, "k8s") || strings.Contains(slash, "terraform"):
		return "Infrastructure"
	case strings.Contains(slash, "frontend"):
		return "Frontend"
	case strings.Contains(slash, "backend"):
		return "Backend"
	case strings.Contains(slash, "monitoring") || strings.Contains(slash, "nginx"):
		return "Operations"
	case strings.HasSuffix(slash, ".env") || strings.HasSuffix(slash, ".env.example"):
		return "Environment"
	default:
		return "Project"
	}
}

func summarize(cfg stackConfig, fileCount int) string {
	addons := []string{}
	if cfg.Nginx {
		addons = append(addons, "Nginx")
	}
	if cfg.LocalStack {
		addons = append(addons, "LocalStack")
	}
	if cfg.Monitoring {
		addons = append(addons, "Monitoring")
	}
	if cfg.CICD {
		addons = append(addons, "CI")
	}
	if len(addons) == 0 {
		addons = append(addons, "no addons")
	}
	return fmt.Sprintf("%s frontend, %s backend, %s database, %s cache, %s: %d files", cfg.Frontend, cfg.Backend, cfg.Database, cfg.Cache, strings.Join(addons, ", "), fileCount)
}

func lineCount(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

func countFiles(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	return count, err
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
