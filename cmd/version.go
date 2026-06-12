package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version         = "dev"
	TemplateVersion = "1"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI and template versions",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Fprintf(cmd.OutOrStdout(), "dev-box %s\ntemplate schema %s\n", Version, TemplateVersion)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
