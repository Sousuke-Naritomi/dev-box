package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

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
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
