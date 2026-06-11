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
		if _, err := exec.LookPath("docker"); err != nil {
			return fmt.Errorf("Docker is not installed. Please install Docker: https://docs.docker.com/get-docker/")
		}

		fmt.Println("🛑 Stopping the development environment...")

		c := exec.Command("docker", "compose", "down")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			return fmt.Errorf("stopping containers: %w", err)
		}

		fmt.Println("✅ Environment has been stopped.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
