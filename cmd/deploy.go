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

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Generate Terraform files for cloud deployment",
	Long: `Generate Terraform configurations for AWS, GCP, or Azure.

With --config, the instance size and tags follow the stack in dev-box.yaml, and
the provider defaults to its cloud setting. Without a provider, an interactive
prompt asks for one.`,
	RunE: runDeploy,
}

func init() {
	deployCmd.Flags().String("provider", "", "Cloud provider to deploy to (aws, gcp, azure); defaults to the cloud setting in --config")
	deployCmd.Flags().String("config", "", "Path to a dev-box.yaml project configuration")
	deployCmd.Flags().Bool("force", false, "Overwrite existing files without prompting")
	deployCmd.Flags().Bool("dry-run", false, "Print generated files without writing them")
	deployCmd.Flags().String("output-dir", "terraform", "Directory to write Terraform files into")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := loadProjectConfig(app.FS, configPath)
	if err != nil {
		return err
	}

	provider, _ := cmd.Flags().GetString("provider")
	if provider == "" && cfg.HasCloud() {
		provider = string(cfg.Cloud)
	}
	if provider == "" {
		if !term.IsTerminal(int(os.Stdin.Fd())) {
			return fmt.Errorf("non-interactive environment detected; provide --provider or set cloud in --config")
		}
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
	if err := writeCommandOutput(cmd.OutOrStdout(), "\nGenerating Terraform configuration for %s...\n", provider); err != nil {
		return err
	}
	if err := runDeployLogic(cmd.Context(), provider, cfg, outputDir, force, dryRun, cmd.OutOrStdout(), app.FS, app.Gen); err != nil {
		return err
	}
	if !dryRun {
		if err := writeCommandOutput(cmd.OutOrStdout(), "Successfully generated Terraform files in %s.\n", outputDir); err != nil {
			return err
		}
	}
	return nil
}

func runDeployLogic(ctx context.Context, provider string, cfg config.ProjectConfig, outputDir string, force, dryRun bool, output io.Writer, fileSystem fs.FileSystem, gen generator.Generator) error {
	files, err := gen.GenerateTerraform(provider, cfg)
	if err != nil {
		return fmt.Errorf("generating terraform: %w", err)
	}
	return writeGeneratedFiles(ctx, fileSystem, outputDir, files, force, dryRun, output)
}
