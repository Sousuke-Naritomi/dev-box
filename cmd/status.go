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
		dockerCmd, err := getDockerCmd()
		if err != nil {
			return err
		}

		fmt.Println("📊 Checking environment status...")

		c := exec.Command(dockerCmd, "compose", "ps")
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
