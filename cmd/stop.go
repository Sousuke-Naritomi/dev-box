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
	Long:  `Runs docker-compose down to stop and remove the containers.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🛑 Stopping the development environment...")
		
		c := exec.Command("docker-compose", "down")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		
		err := c.Run()
		if err != nil {
			fmt.Println("❌ Error stopping containers:", err)
			return
		}
		
		fmt.Println("✅ Environment has been stopped.")
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
