package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Generate Terraform files for cloud deployment",
	Long:  `Run an interactive prompt to generate Terraform configurations for AWS, GCP, or Azure.`,
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().String("provider", "", "Cloud provider to deploy to (aws, gcp, azure)")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	provider, _ := cmd.Flags().GetString("provider")
	if provider == "" {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Choose your Cloud Provider").
					Options(
						huh.NewOption("AWS (Amazon Web Services)", "aws"),
						huh.NewOption("GCP (Google Cloud Platform)", "gcp"),
						huh.NewOption("Azure (Microsoft Azure)", "azure"),
					).Value(&provider),
			),
		)

		if err := form.Run(); err != nil {
			return fmt.Errorf("form cancelled: %w", err)
		}
	}

	fmt.Printf("\n☁️  Generating Terraform configuration for %s...\n", provider)

	files, err := generator.GenerateTerraform(provider)
	if err != nil {
		return fmt.Errorf("generating terraform: %w", err)
	}

	outDir := "terraform"
	for name, content := range files {
		if err := writeFile(filepath.Join(outDir, name), []byte(content)); err != nil {
			return err
		}
	}

	fmt.Println("✅ Successfully generated Terraform files in terraform/ directory!")
	fmt.Println("👉 Next steps: cd terraform && terraform init && terraform plan")
	return nil
}
