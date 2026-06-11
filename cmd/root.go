package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/Sousuke-Naritomi/dev-box/internal/fs"
	"github.com/Sousuke-Naritomi/dev-box/internal/generator"
	"github.com/spf13/cobra"
)

type App struct {
	FS  fs.FileSystem
	Gen generator.Generator
}

var app = &App{
	FS:  fs.NewOSFileSystem(),
	Gen: &generator.DefaultGenerator{},
}

var rootCmd = &cobra.Command{
	Use:   "dev-box",
	Short: "dev-box is a CLI tool to quickly spin up development environments",
	Long:  `A fast and flexible CLI tool built in Go that generates docker-compose, Kubernetes, and Terraform configurations.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

// Execute runs the root command.
func Execute() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
