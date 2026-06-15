package cmd

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Generate Kubernetes manifests for your environment",
	Long:  `Generate Kubernetes Deployment, Service, PVC, and NetworkPolicy YAML files from flags or dev-box.yaml.`,
	RunE:  runK8s,
}

func init() {
	k8sCmd.Flags().String("frontend", "", "Frontend framework (react, vue, none)")
	k8sCmd.Flags().String("backend", "", "Backend framework (node, python, go, php, java, none)")
	k8sCmd.Flags().String("database", "", "Database (mysql, postgres, none)")
	k8sCmd.Flags().String("cache", "", "Cache (redis, none)")
	k8sCmd.Flags().String("config", "", "Path to a dev-box.yaml project configuration")
	k8sCmd.Flags().Bool("force", false, "Overwrite existing files without prompting")
	k8sCmd.Flags().Bool("dry-run", false, "Print generated files without writing them")
	k8sCmd.Flags().String("output-dir", "k8s-manifests", "Directory to write Kubernetes manifests into")
	rootCmd.AddCommand(k8sCmd)
}

func runK8s(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := loadProjectConfig(app.FS, configPath)
	if err != nil {
		return err
	}
	applyCoreFlags(cmd, &cfg)

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
			fields = append(fields, huh.NewSelect[string]().Title("Frontend for K8s?").Options(
				huh.NewOption("React", "react"), huh.NewOption("Vue", "vue"), huh.NewOption("None", "none"),
			).Value(&frontend))
		}
		if backend == "" {
			fields = append(fields, huh.NewSelect[string]().Title("Backend for K8s?").Options(
				huh.NewOption("Node.js (Express)", "node"), huh.NewOption("Python (Flask)", "python"),
				huh.NewOption("Go (net/http)", "go"), huh.NewOption("PHP (Apache)", "php"),
				huh.NewOption("Java (HttpServer)", "java"), huh.NewOption("None", "none"),
			).Value(&backend))
		}
		if database == "" {
			fields = append(fields, huh.NewSelect[string]().Title("Database for K8s?").Options(
				huh.NewOption("MySQL", "mysql"), huh.NewOption("PostgreSQL", "postgres"), huh.NewOption("None", "none"),
			).Value(&database))
		}
		if cache == "" {
			fields = append(fields, huh.NewSelect[string]().Title("Cache for K8s?").Options(
				huh.NewOption("Redis", "redis"), huh.NewOption("None", "none"),
			).Value(&cache))
		}
		if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
			return fmt.Errorf("form cancelled: %w", err)
		}
		cfg.Frontend = config.FrontendType(frontend)
		cfg.Backend = config.BackendType(backend)
		cfg.Database = config.DatabaseType(database)
		cfg.Cache = config.CacheType(cache)
	}

	if err := cfg.ValidateSelection(); err != nil {
		return err
	}
	if !cfg.HasWorkload() {
		return fmt.Errorf("kubernetes generation requires at least one frontend, backend, database, or cache service")
	}

	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	if err := writeCommandOutput(cmd.OutOrStdout(), "Generating Kubernetes manifests...\n"); err != nil {
		return err
	}
	if err := runK8sLogic(cmd.Context(), cfg, outputDir, force, dryRun, cmd.OutOrStdout(), app.FS, app.Gen); err != nil {
		return err
	}
	if !dryRun {
		if err := writeCommandOutput(cmd.OutOrStdout(), "Generated Kubernetes manifests in %s.\n", outputDir); err != nil {
			return err
		}
		if cfg.Database != config.DatabaseNone {
			if err := writeCommandOutput(cmd.OutOrStdout(), "Create a Secret named 'db-secret' before applying the database manifest.\n"); err != nil {
				return err
			}
		}
	}
	return nil
}

func runK8sLogic(ctx context.Context, cfg config.ProjectConfig, outputDir string, force, dryRun bool, output io.Writer, fileSystem fs.FileSystem, gen generator.Generator) error {
	manifests, err := gen.GenerateK8sManifests(cfg)
	if err != nil {
		return fmt.Errorf("generating k8s manifests: %w", err)
	}
	return writeGeneratedFiles(ctx, fileSystem, outputDir, manifests, force, dryRun, output)
}
