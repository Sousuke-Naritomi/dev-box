package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
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
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg := config.ProjectConfig{}

	cfg.Frontend, _ = cmd.Flags().GetString("frontend")
	cfg.Backend, _ = cmd.Flags().GetString("backend")
	cfg.Database, _ = cmd.Flags().GetString("database")
	cfg.Cache, _ = cmd.Flags().GetString("cache")
	cfg.Nginx, _ = cmd.Flags().GetBool("nginx")
	cfg.LocalStack, _ = cmd.Flags().GetBool("localstack")
	cfg.Monitoring, _ = cmd.Flags().GetBool("monitoring")
	cfg.CICD, _ = cmd.Flags().GetBool("cicd")

	if cfg.Frontend == "" || cfg.Backend == "" || cfg.Database == "" || cfg.Cache == "" {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose your Frontend Framework").
					Options(
						huh.NewOption("React (Vite)", "react"),
						huh.NewOption("Vue.js (Vite)", "vue"),
						huh.NewOption("None", "none"),
					).Value(&cfg.Frontend),

				huh.NewSelect[string]().
					Title("Choose your Backend Language").
					Options(
						huh.NewOption("Node.js (Express)", "node"),
						huh.NewOption("Python (Flask)", "python"),
						huh.NewOption("Go (net/http)", "go"),
						huh.NewOption("PHP (Apache)", "php"),
						huh.NewOption("Java (HttpServer)", "java"),
						huh.NewOption("None", "none"),
					).Value(&cfg.Backend),

				huh.NewSelect[string]().
					Title("Choose your Database").
					Options(
						huh.NewOption("MySQL", "mysql"),
						huh.NewOption("PostgreSQL", "postgres"),
						huh.NewOption("None", "none"),
					).Value(&cfg.Database),

				huh.NewSelect[string]().
					Title("Choose your Cache").
					Options(
						huh.NewOption("Redis", "redis"),
						huh.NewOption("None", "none"),
					).Value(&cfg.Cache),

				huh.NewConfirm().Title("Include Nginx Reverse Proxy?").Value(&cfg.Nginx),
				huh.NewConfirm().Title("Include LocalStack? (AWS Local Emulator)").Value(&cfg.LocalStack),
				huh.NewConfirm().Title("Include Monitoring? (Prometheus + Grafana + Loki)").Value(&cfg.Monitoring),
				huh.NewConfirm().Title("Include CI/CD Pipeline? (GitHub Actions)").Value(&cfg.CICD),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("form cancelled: %w", err)
		}
	}

	fmt.Printf("\n✨ Generating environment for: Frontend=%s, Backend=%s, Database=%s\n", cfg.Frontend, cfg.Backend, cfg.Database)

	// Resolve ports and passwords
	data, err := generator.ResolveComposeData(cfg)
	if err != nil {
		return fmt.Errorf("resolving configuration: %w", err)
	}

	// Generate and write docker-compose.yml
	composeContent, err := generator.RenderCompose(data)
	if err != nil {
		return fmt.Errorf("rendering compose: %w", err)
	}
	if err := writeFile("docker-compose.yml", []byte(composeContent)); err != nil {
		return err
	}

	// Generate .env and .gitignore if needed
	needsEnv := (cfg.Database != "none" && cfg.Database != "") || cfg.Monitoring || (cfg.Backend != "none" && cfg.Backend != "")
	if needsEnv {
		envContent, err := generator.RenderEnvFile(data)
		if err != nil {
			return fmt.Errorf("rendering env file: %w", err)
		}
		if err := writeFile(".env", []byte(envContent)); err != nil {
			return err
		}
		if err := writeFile(".gitignore", []byte(generator.RenderGitignore())); err != nil {
			return err
		}
	}

	// Generate nginx config
	if cfg.Nginx && (cfg.Frontend != "none" || cfg.Backend != "none") {
		nginxConf, err := generator.GenerateNginxConf(cfg, cfg.BackendInternalPort())
		if err != nil {
			return fmt.Errorf("generating nginx conf: %w", err)
		}
		if err := writeFile(filepath.Join("nginx", "nginx.conf"), []byte(nginxConf)); err != nil {
			return err
		}
	}

	// Generate monitoring configs
	if cfg.Monitoring {
		if err := writeFile(filepath.Join("monitoring", "prometheus.yml"), []byte(generator.RenderPrometheusConfig())); err != nil {
			return err
		}
		if err := writeFile(filepath.Join("monitoring", "promtail-config.yml"), []byte(generator.RenderPromtailConfig())); err != nil {
			return err
		}
	}

	// Generate CI/CD
	if cfg.CICD {
		cicdContent := generator.GenerateCICD(cfg.BackendInternalPort())
		if err := writeFile(filepath.Join(".github", "workflows", "ci.yml"), []byte(cicdContent)); err != nil {
			return err
		}
		fmt.Println("✅ Generated .github/workflows/ci.yml")
	}

	// Generate scaffold files
	scaffoldFiles, err := generator.GenerateScaffold(cfg)
	if err != nil {
		return fmt.Errorf("generating scaffold: %w", err)
	}
	for path, content := range scaffoldFiles {
		if err := writeFile(path, []byte(content)); err != nil {
			return err
		}
	}

	fmt.Println("✅ Successfully generated local environment configuration!")
	if needsEnv {
		fmt.Println("🔒 A secure .env file was generated and added to .gitignore.")
	}
	return nil
}

// writeFile creates the parent directory if needed and writes content to the file.
func writeFile(path string, content []byte) error {
	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}
	if err := os.WriteFile(path, content, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", path, err)
	}
	return nil
}
