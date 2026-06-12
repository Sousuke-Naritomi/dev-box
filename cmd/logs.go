package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [service]",
	Args:  cobra.MaximumNArgs(1),
	Short: "View logs from the development environment",
	Long:  `Runs docker compose logs -f to stream logs from containers.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		dockerCmd, err := getDockerCmd(cmd.Context())
		if err != nil {
			return err
		}

		execArgs := []string{"compose", "logs", "-f"}
		if len(args) > 0 {
			execArgs = append(execArgs, args...)
		}

		c := exec.CommandContext(cmd.Context(), dockerCmd, execArgs...)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr

		if err := c.Run(); err != nil {
			return fmt.Errorf("getting logs: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logsCmd)
}
