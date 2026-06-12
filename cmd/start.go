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
		dockerCmd, err := getDockerCmd(cmd.Context())
		if err != nil {
			return err
		}

		fmt.Println("🚀 Starting the development environment...")

		c := exec.CommandContext(cmd.Context(), dockerCmd, "compose", "up", "-d")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			return fmt.Errorf("starting containers: %w", err)
		}

		fmt.Println("✅ Environment is up and running!")
		fmt.Println("📡 Assigned Ports:")

		// Run `docker compose ps` to show the dynamically allocated ports
		psCmd := exec.CommandContext(cmd.Context(), dockerCmd, "compose", "ps")
		psCmd.Stdout = os.Stdout
		psCmd.Stderr = os.Stderr
		_ = psCmd.Run()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
