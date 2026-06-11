package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the development environment",
	Long:  `Runs docker compose up -d to spin up the containers in the background.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := exec.LookPath("docker"); err != nil {
			return fmt.Errorf("Docker is not installed. Please install Docker: https://docs.docker.com/get-docker/")
		}

		fmt.Println("🚀 Starting the development environment...")

		c := exec.Command("docker", "compose", "up", "-d")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			return fmt.Errorf("starting containers: %w", err)
		}

		fmt.Println("✅ Environment is up and running!")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
