package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the development environment",
	Long:  `Runs docker compose down to stop and remove the containers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dockerCmd, err := getDockerCmd(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Println("Stopping the development environment...")
		stop := exec.CommandContext(cmd.Context(), dockerCmd, "compose", "down")
		stop.Stdout = os.Stdout
		stop.Stderr = os.Stderr
		if err := stop.Run(); err != nil {
			return fmt.Errorf("stopping containers: %w", err)
		}

		fmt.Println("Environment has been stopped.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
