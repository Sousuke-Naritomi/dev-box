package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Sousuke-Naritomi/dev-box/internal/config"
	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Generate Kubernetes manifests for your environment",
	Long:  `Run an interactive prompt to generate Kubernetes Deployment, Service, PVC, and NetworkPolicy YAML files.`,
	RunE:  runK8s,
}

func init() {
	k8sCmd.Flags().String("frontend", "", "Frontend framework (react, vue, none)")
	k8sCmd.Flags().String("backend", "", "Backend framework (node, python, go, php, java, none)")
	k8sCmd.Flags().String("database", "", "Database (mysql, postgres, none)")
	k8sCmd.Flags().String("cache", "", "Cache (redis, none)")
	k8sCmd.Flags().Bool("force", false, "Overwrite existing files without prompting")
	k8sCmd.Flags().Bool("dry-run", false, "Print generated files without writing them")
	k8sCmd.Flags().String("output-dir", "k8s-manifests", "Directory to write Kubernetes manifests into")
	rootCmd.AddCommand(k8sCmd)
}

func runK8s(cmd *cobra.Command, args []string) error {
	frontend, _ := cmd.Flags().GetString("frontend")
	backend, _ := cmd.Flags().GetString("backend")
	database, _ := cmd.Flags().GetString("database")
	cache, _ := cmd.Flags().GetString("cache")

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
	if len(fields) > 0 {
		if err := huh.NewForm(huh.NewGroup(fields...)).Run(); err != nil {
			return fmt.Errorf("form cancelled: %w", err)
		}
	}

	cfg := config.ProjectConfig{
		Frontend: config.FrontendType(frontend),
		Backend:  config.BackendType(backend),
		Database: config.DatabaseType(database),
		Cache:    config.CacheType(cache),
	}
	if err := cfg.Validate(); err != nil {
		return err
	}

	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	fmt.Fprintln(cmd.OutOrStdout(), "\nGenerating Kubernetes manifests...")
	if err := runK8sLogic(cmd.Context(), cfg, outputDir, force, dryRun, cmd.OutOrStdout(), app.FS, app.Gen); err != nil {
		return err
	}
	if !dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "Generated Kubernetes manifests in %s.\n", outputDir)
		if cfg.Database != config.DatabaseNone {
			fmt.Fprintln(cmd.OutOrStdout(), "Create a Secret named 'db-secret' before applying the database manifest.")
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
