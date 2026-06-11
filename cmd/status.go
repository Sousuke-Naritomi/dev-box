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
		if _, err := exec.LookPath("docker"); err != nil {
			return fmt.Errorf("Docker is not installed. Please install Docker: https://docs.docker.com/get-docker/")
		}

		fmt.Println("📊 Checking environment status...")

		c := exec.Command("docker", "compose", "ps")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			return fmt.Errorf("getting status: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
