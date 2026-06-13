package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the status of the development environment",
	Long:  `Runs docker compose ps to display the status of containers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dockerCmd, err := getDockerCmd(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Println("Checking environment status...")
		status := exec.CommandContext(cmd.Context(), dockerCmd, "compose", "ps")
		status.Stdout = os.Stdout
		status.Stderr = os.Stderr
		if err := status.Run(); err != nil {
			return fmt.Errorf("getting status: %w", err)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
