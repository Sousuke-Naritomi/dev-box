package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
	"github.com/Sousuke-Naritomi/dev-box/internal/registry"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new development environment",
	Long:  `Run an interactive prompt or load dev-box.yaml to generate a development environment.`,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().String("frontend", "", "Frontend framework (react, vue, none)")
	initCmd.Flags().String("backend", "", "Backend framework (node, python, go, php, java, none)")
	initCmd.Flags().String("database", "", "Database (mysql, postgres, none)")
	initCmd.Flags().String("cache", "", "Cache (redis, none)")
	initCmd.Flags().Bool("nginx", false, "Include Nginx reverse proxy")
	initCmd.Flags().Bool("moto", false, "Include Moto (AWS mock server)")
	initCmd.Flags().Bool("monitoring", false, "Include Monitoring (Prometheus + Grafana + Loki)")
	initCmd.Flags().Bool("cicd", false, "Include CI/CD Pipeline")
	initCmd.Flags().String("cloud", "", "Also generate Terraform for a cloud provider (aws, gcp, azure, none)")
	initCmd.Flags().String("config", "", "Path to a dev-box.yaml project configuration")
	initCmd.Flags().Bool("force", false, "Overwrite existing files without prompting")
	initCmd.Flags().Bool("dry-run", false, "Print generated files without writing them")
	initCmd.Flags().String("format", "text", "Output format for dry-run results (text, json)")
	initCmd.Flags().String("output-dir", ".", "Directory to write generated files into")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := loadProjectConfig(app.FS, configPath)
	if err != nil {
		return err
	}
	applyCoreFlags(cmd, &cfg)
	applyAddonFlags(cmd, &cfg)
	applyCloudFlag(cmd, &cfg)
	cfg.Force, _ = cmd.Flags().GetBool("force")

	cloud := string(cfg.Cloud)
	frontend := string(cfg.Frontend)
	backend := string(cfg.Backend)
	database := string(cfg.Database)
	cache := string(cfg.Cache)

	if frontend == "" || backend == "" || database == "" || cache == "" {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("non-interactive environment detected; provide all stack values with flags or --config")
		}

		var fields []huh.Field
		if frontend == "" {
			fields = append(fields, huh.NewSelect[string]().Title("Choose your Frontend Framework").Options(
				huh.NewOption("React (Vite)", "react"),
				huh.NewOption("Vue.js (Vite)", "vue"),
				huh.NewOption("None", "none"),
			).Value(&frontend))
		}
		if backend == "" {
			fields = append(fields, huh.NewSelect[string]().Title("Choose your Backend Language").Options(
				huh.NewOption("Node.js (Express)", "node"),
				huh.NewOption("Python (Flask)", "python"),
				huh.NewOption("Go (net/http)", "go"),
				huh.NewOption("PHP (Apache)", "php"),
				huh.NewOption("Java (HttpServer)", "java"),
				huh.NewOption("None", "none"),
			).Value(&backend))
		}
		if database == "" {
			fields = append(fields, huh.NewSelect[string]().Title("Choose your Database").Options(
				huh.NewOption("MySQL", "mysql"),
				huh.NewOption("PostgreSQL", "postgres"),
				huh.NewOption("None", "none"),
			).Value(&database))
		}
		if cache == "" {
			fields = append(fields, huh.NewSelect[string]().Title("Choose your Cache").Options(
				huh.NewOption("Redis", "redis"),
				huh.NewOption("None", "none"),
			).Value(&cache))
		}

		// A supplied project file is complete and should not re-prompt for addons.
		if configPath == "" && !cmd.Flags().Changed("nginx") {
			fields = append(fields, huh.NewConfirm().Title("Include Nginx Reverse Proxy?").Value(&cfg.Nginx))
		}
		if configPath == "" && !cmd.Flags().Changed("moto") {
			fields = append(fields, huh.NewConfirm().Title("Include Moto? (AWS mock server: S3, SQS, SNS, DynamoDB)").Value(&cfg.Moto))
		}
		if configPath == "" && !cmd.Flags().Changed("monitoring") {
			fields = append(fields, huh.NewConfirm().Title("Include Monitoring? (Prometheus + Grafana + Loki)").Value(&cfg.Monitoring))
		}
		if configPath == "" && !cmd.Flags().Changed("cicd") {
			fields = append(fields, huh.NewConfirm().Title("Include CI/CD Pipeline? (GitHub Actions)").Value(&cfg.CICD))
		}
		if configPath == "" && !cmd.Flags().Changed("cloud") {
			fields = append(fields, huh.NewSelect[string]().Title("Generate Terraform for a cloud provider?").Options(
				huh.NewOption("No", "none"),
				huh.NewOption("AWS", "aws"),
				huh.NewOption("GCP", "gcp"),
				huh.NewOption("Azure", "azure"),
			).Value(&cloud))
		}

		if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
			return fmt.Errorf("form cancelled: %w", err)
		}
		cfg.Frontend = config.FrontendType(frontend)
		cfg.Backend = config.BackendType(backend)
		cfg.Database = config.DatabaseType(database)
		cfg.Cache = config.CacheType(cache)
		cfg.Cloud = config.CloudProvider(cloud)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}
	outputDir, _ := cmd.Flags().GetString("output-dir")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	outputFormat, _ := cmd.Flags().GetString("format")
	return runInitLogic(cmd.Context(), cfg, outputDir, dryRun, outputFormat, cmd.OutOrStdout(), app.FS, app.Gen)
}

type initOutputFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	Perm    string `json:"perm"`
}

type initOutputConfig struct {
	Frontend   config.FrontendType  `json:"frontend"`
	Backend    config.BackendType   `json:"backend"`
	Database   config.DatabaseType  `json:"database"`
	Cache      config.CacheType     `json:"cache"`
	Nginx      bool                 `json:"nginx"`
	Moto       bool                 `json:"moto"`
	Monitoring bool                 `json:"monitoring"`
	CICD       bool                 `json:"cicd"`
	Cloud      config.CloudProvider `json:"cloud"`
}

type initOutput struct {
	Config    initOutputConfig `json:"config"`
	OutputDir string           `json:"outputDir"`
	DryRun    bool             `json:"dryRun"`
	Files     []initOutputFile `json:"files"`
}

func runInitLogic(ctx context.Context, cfg config.ProjectConfig, outputDir string, dryRun bool, outputFormat string, output io.Writer, fileSystem fs.FileSystem, gen generator.Generator) error {
	if outputFormat != "text" && outputFormat != "json" {
		return fmt.Errorf("unsupported output format: %q", outputFormat)
	}
	if outputFormat == "json" && !dryRun {
		return fmt.Errorf("--format json requires --dry-run")
	}

	needsEnv := cfg.Database != config.DatabaseNone || cfg.Monitoring || cfg.Backend != config.BackendNone
	if outputFormat == "text" {
		if err := writeCommandOutput(output, "Generating environment: frontend=%s backend=%s database=%s cache=%s\n", cfg.Frontend, cfg.Backend, cfg.Database, cfg.Cache); err != nil {
			return err
		}
		if runtime.GOOS == "windows" && needsEnv {
			if err := writeCommandOutput(output, "Note: verify the Windows ACL on .env; POSIX mode 0600 does not map directly to NTFS permissions.\n"); err != nil {
				return err
			}
		}
	}

	filesToWrite, needsEnv, err := buildInitFiles(cfg, gen)
	if err != nil {
		return err
	}
	if outputFormat == "json" {
		return writeInitJSONOutput(ctx, fileSystem, outputDir, filesToWrite, cfg, output)
	}

	if err := writeGeneratedFiles(ctx, fileSystem, outputDir, filesToWrite, cfg.Force, dryRun, output); err != nil {
		return err
	}
	if dryRun {
		return nil
	}

	if err := writeCommandOutput(output, "Successfully generated local environment configuration.\n"); err != nil {
		return err
	}
	if needsEnv {
		if err := writeCommandOutput(output, "A secure .env file was generated and added to .gitignore.\n"); err != nil {
			return err
		}
	}
	if cfg.CICD {
		if err := writeCommandOutput(output, "Generated .github/workflows/ci.yml.\n"); err != nil {
			return err
		}
	}
	if cfg.HasCloud() {
		if err := writeCommandOutput(output, "Generated Terraform for %s in terraform/. Copy terraform.tfvars.example to terraform.tfvars, review it, then run terraform init and terraform plan.\n", cfg.Cloud); err != nil {
			return err
		}
	}
	return nil
}

func buildInitFiles(cfg config.ProjectConfig, gen generator.Generator) (map[string]generator.FileDef, bool, error) {
	data, err := generator.ResolveComposeData(cfg)
	if err != nil {
		return nil, false, fmt.Errorf("resolving configuration: %w", err)
	}
	filesToWrite := make(map[string]generator.FileDef)

	projectConfig, err := config.RenderProjectFile(cfg)
	if err != nil {
		return nil, false, err
	}
	filesToWrite["dev-box.yaml"] = generator.FileDef{Content: projectConfig, Perm: 0644}

	composeContent, err := generator.RenderCompose(data)
	if err != nil {
		return nil, false, fmt.Errorf("rendering compose: %w", err)
	}
	filesToWrite["docker-compose.yml"] = generator.FileDef{Content: composeContent, Perm: 0644}

	needsEnv := cfg.Database != config.DatabaseNone || cfg.Monitoring || cfg.Backend != config.BackendNone
	if needsEnv {
		envContent, err := generator.RenderEnvFile(data)
		if err != nil {
			return nil, false, fmt.Errorf("rendering env file: %w", err)
		}
		filesToWrite[".env"] = generator.FileDef{Content: envContent, Perm: 0600}

		envExampleContent, err := generator.RenderEnvExampleFile(data)
		if err != nil {
			return nil, false, fmt.Errorf("rendering env.example file: %w", err)
		}
		filesToWrite[".env.example"] = generator.FileDef{Content: envExampleContent, Perm: 0644}
		filesToWrite[".gitignore"] = generator.FileDef{Content: generator.RenderGitignore(), Perm: 0644}
	}

	backendPort := 0
	if backend, ok := registry.SupportedBackends[cfg.Backend]; ok {
		backendPort = backend.InternalPort
	}
	frontendPort := 0
	if frontend, ok := registry.SupportedFrontends[cfg.Frontend]; ok {
		frontendPort = frontend.InternalPort
	}

	if cfg.Nginx && (cfg.Frontend != config.FrontendNone || cfg.Backend != config.BackendNone) {
		nginxConf, err := generator.GenerateNginxConf(cfg, backendPort, frontendPort)
		if err != nil {
			return nil, false, fmt.Errorf("generating nginx conf: %w", err)
		}
		filesToWrite[filepath.Join("nginx", "nginx.conf")] = generator.FileDef{Content: nginxConf, Perm: 0644}
	}

	if cfg.Monitoring {
		prometheus, err := generator.RenderPrometheusConfig(backendPort)
		if err != nil {
			return nil, false, fmt.Errorf("rendering Prometheus config: %w", err)
		}
		filesToWrite[filepath.Join("monitoring", "prometheus.yml")] = generator.FileDef{Content: prometheus, Perm: 0644}

		datasource, err := generator.RenderGrafanaDatasource()
		if err != nil {
			return nil, false, fmt.Errorf("rendering Grafana datasource: %w", err)
		}
		filesToWrite[filepath.Join("monitoring", "grafana", "provisioning", "datasources", "prometheus.yml")] = generator.FileDef{Content: datasource, Perm: 0644}

		alloy, err := generator.RenderAlloyConfig()
		if err != nil {
			return nil, false, fmt.Errorf("rendering Alloy config: %w", err)
		}
		filesToWrite[filepath.Join("monitoring", "config.alloy")] = generator.FileDef{Content: alloy, Perm: 0644}
	}

	if cfg.CICD {
		workflow, err := generator.GenerateCICD(cfg, backendPort, frontendPort)
		if err != nil {
			return nil, false, fmt.Errorf("generating CI workflow: %w", err)
		}
		filesToWrite[filepath.Join(".github", "workflows", "ci.yml")] = generator.FileDef{Content: workflow, Perm: 0644}
	}

	scaffoldFiles, err := gen.GenerateScaffold(cfg)
	if err != nil {
		return nil, false, fmt.Errorf("generating scaffold: %w", err)
	}
	for path, content := range scaffoldFiles {
		filesToWrite[path] = content
	}

	if cfg.HasCloud() {
		terraformFiles, err := gen.GenerateTerraform(string(cfg.Cloud), cfg)
		if err != nil {
			return nil, false, fmt.Errorf("generating terraform: %w", err)
		}
		for name, content := range terraformFiles {
			filesToWrite[filepath.Join("terraform", name)] = content
		}
	}
	return filesToWrite, needsEnv, nil
}

func normalizedCloud(cloud config.CloudProvider) config.CloudProvider {
	if cloud == "" {
		return config.CloudNone
	}
	return cloud
}

func writeInitJSONOutput(ctx context.Context, fileSystem fs.FileSystem, outputDir string, files map[string]generator.FileDef, cfg config.ProjectConfig, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	prepared, _, err := prepareGeneratedFiles(fileSystem, outputDir, files)
	if err != nil {
		return err
	}

	paths := make([]string, 0, len(prepared))
	for path := range prepared {
		paths = append(paths, path)
	}
	sort.Strings(paths)

	response := initOutput{
		Config: initOutputConfig{
			Frontend:   cfg.Frontend,
			Backend:    cfg.Backend,
			Database:   cfg.Database,
			Cache:      cfg.Cache,
			Nginx:      cfg.Nginx,
			Moto:       cfg.Moto,
			Monitoring: cfg.Monitoring,
			CICD:       cfg.CICD,
			Cloud:      normalizedCloud(cfg.Cloud),
		},
		OutputDir: outputDir,
		DryRun:    true,
		Files:     make([]initOutputFile, 0, len(paths)),
	}
	for _, path := range paths {
		response.Files = append(response.Files, initOutputFile{
			Path:    filepath.ToSlash(path),
			Content: string(prepared[path].Data),
			Perm:    fmt.Sprintf("%#o", prepared[path].Perm),
		})
	}

	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(response)
}
