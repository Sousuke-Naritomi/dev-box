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
	Long:  `Runs docker-compose up -d to spin up the containers in the background.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("🚀 Starting the development environment...")
		
		c := exec.Command("docker-compose", "up", "-d")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		
		err := c.Run()
		if err != nil {
			fmt.Println("❌ Error starting containers:", err)
			return
		}
		
		fmt.Println("✅ Environment is up and running!")
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
