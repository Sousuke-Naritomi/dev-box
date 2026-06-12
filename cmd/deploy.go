package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
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
	deployCmd.Flags().Bool("force", false, "Overwrite existing files without prompting")
	deployCmd.Flags().Bool("dry-run", false, "Print generated files without writing them")
	deployCmd.Flags().String("output-dir", "terraform", "Directory to write Terraform files into")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	provider, _ := cmd.Flags().GetString("provider")
	if provider == "" {
		form := huh.NewForm(huh.NewGroup(
			huh.NewSelect[string]().
				Title("Choose your Cloud Provider").
				Options(
					huh.NewOption("AWS (Amazon Web Services)", "aws"),
					huh.NewOption("GCP (Google Cloud Platform)", "gcp"),
					huh.NewOption("Azure (Microsoft Azure)", "azure"),
				).
				Value(&provider),
		))
		if err := form.Run(); err != nil {
			return fmt.Errorf("form cancelled: %w", err)
		}
	}

	force, _ := cmd.Flags().GetBool("force")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	outputDir, _ := cmd.Flags().GetString("output-dir")
	fmt.Fprintf(cmd.OutOrStdout(), "\nGenerating Terraform configuration for %s...\n", provider)
	if err := runDeployLogic(cmd.Context(), provider, outputDir, force, dryRun, cmd.OutOrStdout(), app.FS, app.Gen); err != nil {
		return err
	}
	if !dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "Successfully generated Terraform files in %s.\n", outputDir)
	}
	return nil
}

func runDeployLogic(ctx context.Context, provider, outputDir string, force, dryRun bool, output io.Writer, fileSystem fs.FileSystem, gen generator.Generator) error {
	files, err := gen.GenerateTerraform(provider)
	if err != nil {
		return fmt.Errorf("generating terraform: %w", err)
	}
	return writeGeneratedFiles(ctx, fileSystem, outputDir, files, force, dryRun, output)
}
