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

		fmt.Println("Starting the development environment...")
		start := exec.CommandContext(cmd.Context(), dockerCmd, "compose", "up", "-d")
		start.Stdout = os.Stdout
		start.Stderr = os.Stderr
		if err := start.Run(); err != nil {
			return fmt.Errorf("starting containers: %w", err)
		}

		fmt.Println("Environment is up and running.")
		fmt.Println("Assigned ports:")
		status := exec.CommandContext(cmd.Context(), dockerCmd, "compose", "ps")
		status.Stdout = os.Stdout
		status.Stderr = os.Stderr
		_ = status.Run()
		return nil
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
