package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/term"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
	"github.com/Sousuke-Naritomi/dev-box/internal/registry"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new development environment",
	Long:  `Run an interactive prompt to configure and generate your development environment.`,
	RunE:  runInit,
}

func init() {
	initCmd.Flags().String("frontend", "", "Frontend framework (react, vue, none)")
	initCmd.Flags().String("backend", "", "Backend framework (node, python, go, php, java, none)")
	initCmd.Flags().String("database", "", "Database (mysql, postgres, none)")
	initCmd.Flags().String("cache", "", "Cache (redis, none)")
	initCmd.Flags().Bool("nginx", false, "Include Nginx reverse proxy")
	initCmd.Flags().Bool("localstack", false, "Include LocalStack")
	initCmd.Flags().Bool("monitoring", false, "Include Monitoring (Prometheus + Grafana + Loki)")
	initCmd.Flags().Bool("cicd", false, "Include CI/CD Pipeline")
	initCmd.Flags().Bool("force", false, "Overwrite existing files without prompting")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg := config.ProjectConfig{}

	frontendStr, _ := cmd.Flags().GetString("frontend")
	backendStr, _ := cmd.Flags().GetString("backend")
	databaseStr, _ := cmd.Flags().GetString("database")
	cacheStr, _ := cmd.Flags().GetString("cache")

	cfg.Frontend = config.FrontendType(frontendStr)
	cfg.Backend = config.BackendType(backendStr)
	cfg.Database = config.DatabaseType(databaseStr)
	cfg.Cache = config.CacheType(cacheStr)
	cfg.Nginx, _ = cmd.Flags().GetBool("nginx")
	cfg.LocalStack, _ = cmd.Flags().GetBool("localstack")
	cfg.Monitoring, _ = cmd.Flags().GetBool("monitoring")
	cfg.CICD, _ = cmd.Flags().GetBool("cicd")
	cfg.Force, _ = cmd.Flags().GetBool("force")

	if cfg.Frontend == "" || cfg.Backend == "" || cfg.Database == "" || cfg.Cache == "" {
		isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
		if !isTerminal && !cfg.Force {
			return fmt.Errorf("non-interactive environment detected. Please provide all required flags or use --force")
		}

		var fields []huh.Field

		if frontendStr == "" {
			fields = append(fields, huh.NewSelect[string]().
				Title("Choose your Frontend Framework").
				Options(
					huh.NewOption("React (Vite)", "react"),
					huh.NewOption("Vue.js (Vite)", "vue"),
					huh.NewOption("None", "none"),
				).Value(&frontendStr))
		}
		if backendStr == "" {
			fields = append(fields, huh.NewSelect[string]().
				Title("Choose your Backend Language").
				Options(
					huh.NewOption("Node.js (Express)", "node"),
					huh.NewOption("Python (Flask)", "python"),
					huh.NewOption("Go (net/http)", "go"),
					huh.NewOption("PHP (Apache)", "php"),
					huh.NewOption("Java (HttpServer)", "java"),
					huh.NewOption("None", "none"),
				).Value(&backendStr))
		}
		if databaseStr == "" {
			fields = append(fields, huh.NewSelect[string]().
				Title("Choose your Database").
				Options(
					huh.NewOption("MySQL", "mysql"),
					huh.NewOption("PostgreSQL", "postgres"),
					huh.NewOption("None", "none"),
				).Value(&databaseStr))
		}
		if cacheStr == "" {
			fields = append(fields, huh.NewSelect[string]().
				Title("Choose your Cache").
				Options(
					huh.NewOption("Redis", "redis"),
					huh.NewOption("None", "none"),
				).Value(&cacheStr))
		}

		if !cmd.Flags().Changed("nginx") {
			fields = append(fields, huh.NewConfirm().Title("Include Nginx Reverse Proxy?").Value(&cfg.Nginx))
		}
		if !cmd.Flags().Changed("localstack") {
			fields = append(fields, huh.NewConfirm().Title("Include LocalStack? (AWS Local Emulator)").Value(&cfg.LocalStack))
		}
		if !cmd.Flags().Changed("monitoring") {
			fields = append(fields, huh.NewConfirm().Title("Include Monitoring? (Prometheus + Grafana + Loki)").Value(&cfg.Monitoring))
		}
		if !cmd.Flags().Changed("cicd") {
			fields = append(fields, huh.NewConfirm().Title("Include CI/CD Pipeline? (GitHub Actions)").Value(&cfg.CICD))
		}

		if len(fields) > 0 {
			form := huh.NewForm(huh.NewGroup(fields...))
			if err := form.Run(); err != nil {
				return fmt.Errorf("form cancelled: %w", err)
			}
		}

		cfg.Frontend = config.FrontendType(frontendStr)
		cfg.Backend = config.BackendType(backendStr)
		cfg.Database = config.DatabaseType(databaseStr)
		cfg.Cache = config.CacheType(cacheStr)
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	return runInitLogic(cmd.Context(), cfg, app.FS, app.Gen)
}

func runInitLogic(ctx context.Context, cfg config.ProjectConfig, fileSystem fs.FileSystem, gen generator.Generator) error {
	fmt.Printf("\n✨ Generating environment for: Frontend=%s, Backend=%s, Database=%s\n", cfg.Frontend, cfg.Backend, cfg.Database)

	// Resolve ports and passwords
	data, err := generator.ResolveComposeData(cfg)
	if err != nil {
		return fmt.Errorf("resolving configuration: %w", err)
	}

	filesToWrite := make(map[string]generator.FileDef)

	// Generate docker-compose.yml
	composeContent, err := generator.RenderCompose(data)
	if err != nil {
		return fmt.Errorf("rendering compose: %w", err)
	}
	filesToWrite["docker-compose.yml"] = generator.FileDef{Content: composeContent, Perm: 0644}

	// Generate .env and .gitignore if needed
	needsEnv := (cfg.Database != "none" && cfg.Database != "") || cfg.Monitoring || (cfg.Backend != "none" && cfg.Backend != "")
	if needsEnv {
		envContent, err := generator.RenderEnvFile(data)
		if err != nil {
			return fmt.Errorf("rendering env file: %w", err)
		}
		filesToWrite[".env"] = generator.FileDef{Content: envContent, Perm: 0600}

		envExampleContent, err := generator.RenderEnvExampleFile(data)
		if err != nil {
			return fmt.Errorf("rendering env.example file: %w", err)
		}
		filesToWrite[".env.example"] = generator.FileDef{Content: envExampleContent, Perm: 0644}

		filesToWrite[".gitignore"] = generator.FileDef{Content: generator.RenderGitignore(), Perm: 0644}
	}

	backendInternalPort := 0
	if b, ok := registry.SupportedBackends[cfg.Backend]; ok {
		backendInternalPort = b.InternalPort
	}

	// Generate nginx config
	if cfg.Nginx && (cfg.Frontend != "none" || cfg.Backend != "none") {
		nginxConf, err := generator.GenerateNginxConf(cfg, backendInternalPort)
		if err != nil {
			return fmt.Errorf("generating nginx conf: %w", err)
		}
		filesToWrite[filepath.Join("nginx", "nginx.conf")] = generator.FileDef{Content: nginxConf, Perm: 0644}
	}

	// Generate monitoring configs
	if cfg.Monitoring {
		promConfig, err := generator.RenderPrometheusConfig()
		if err != nil {
			return err
		}
		filesToWrite[filepath.Join("monitoring", "prometheus.yml")] = generator.FileDef{Content: promConfig, Perm: 0644}

		promtailConfig, err := generator.RenderPromtailConfig()
		if err != nil {
			return err
		}
		filesToWrite[filepath.Join("monitoring", "promtail-config.yml")] = generator.FileDef{Content: promtailConfig, Perm: 0644}
	}

	// Generate CI/CD
	if cfg.CICD {
		cicdContent, err := generator.GenerateCICD(backendInternalPort)
		if err != nil {
			return err
		}
		filesToWrite[filepath.Join(".github", "workflows", "ci.yml")] = generator.FileDef{Content: cicdContent, Perm: 0644}
	}

	// Generate scaffold files
	scaffoldFiles, err := gen.GenerateScaffold(cfg)
	if err != nil {
		return fmt.Errorf("generating scaffold: %w", err)
	}
	for path, content := range scaffoldFiles {
		filesToWrite[path] = content
	}

	var existingFiles []string
	for path := range filesToWrite {
		if fileSystem.Exists(path) {
			existingFiles = append(existingFiles, path)
		}
	}

	if len(existingFiles) > 0 && !cfg.Force {
		isTerminal := term.IsTerminal(int(os.Stdin.Fd()))
		if !isTerminal {
			return fmt.Errorf("non-interactive environment detected and existing files found; use --force to overwrite")
		}

		var overwrite bool
		err := huh.NewConfirm().
			Title("Existing files found. Overwrite? [y/N]").
			Value(&overwrite).
			Run()
		if err != nil {
			return fmt.Errorf("prompt cancelled: %w", err)
		}
		if !overwrite {
			fmt.Println("❌ File generation aborted.")
			return nil
		}
	}

	for path, fd := range filesToWrite {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := fileSystem.WriteFile(path, []byte(fd.Content), fd.Perm); err != nil {
			return fmt.Errorf("writing %s: %w", path, err)
		}
	}

	fmt.Println("✅ Successfully generated local environment configuration!")
	if needsEnv {
		fmt.Println("🔒 A secure .env file was generated and added to .gitignore.")
	}
	if cfg.CICD {
		fmt.Println("✅ Generated .github/workflows/ci.yml")
	}
	return nil
}
