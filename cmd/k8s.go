package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

var k8sCmd = &cobra.Command{
	Use:   "k8s",
	Short: "Generate Kubernetes manifests for your environment",
	Long:  `Run an interactive prompt to generate Kubernetes Deployment, Service, and PVC YAML files.`,
	RunE:  runK8s,
}

func init() {
	k8sCmd.Flags().String("frontend", "", "Frontend framework (react, vue, none)")
	k8sCmd.Flags().String("backend", "", "Backend framework (node, python, go, php, java, none)")
	k8sCmd.Flags().String("database", "", "Database (mysql, postgres, none)")
	k8sCmd.Flags().String("cache", "", "Cache (redis, none)")
	rootCmd.AddCommand(k8sCmd)
}

func runK8s(cmd *cobra.Command, args []string) error {
	frontend, _ := cmd.Flags().GetString("frontend")
	backend, _ := cmd.Flags().GetString("backend")
	database, _ := cmd.Flags().GetString("database")
	cache, _ := cmd.Flags().GetString("cache")

	if frontend == "" || backend == "" || database == "" || cache == "" {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().Title("Frontend for K8s?").
					Options(huh.NewOption("React", "react"), huh.NewOption("Vue", "vue"), huh.NewOption("None", "none")).Value(&frontend),
				huh.NewSelect[string]().Title("Backend for K8s?").
					Options(huh.NewOption("Yes", "yes"), huh.NewOption("None", "none")).Value(&backend),
				huh.NewSelect[string]().Title("Database for K8s?").
					Options(huh.NewOption("MySQL", "mysql"), huh.NewOption("PostgreSQL", "postgres"), huh.NewOption("None", "none")).Value(&database),
				huh.NewSelect[string]().Title("Cache for K8s?").
					Options(huh.NewOption("Redis", "redis"), huh.NewOption("None", "none")).Value(&cache),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("form cancelled: %w", err)
		}
	}

	fmt.Println("\n✨ Generating Kubernetes manifests...")

	manifests, err := generator.GenerateK8sManifests(frontend, backend, database, cache)
	if err != nil {
		return fmt.Errorf("generating k8s manifests: %w", err)
	}

	outDir := "k8s-manifests"
	for name, content := range manifests {
		if err := writeFile(filepath.Join(outDir, name), []byte(content)); err != nil {
			return err
		}
	}

	fmt.Println("✅ Generated Kubernetes manifests in k8s-manifests/ directory!")
	fmt.Println("💡 Create a Secret named 'db-secret' for database passwords before applying.")
	return nil
}
