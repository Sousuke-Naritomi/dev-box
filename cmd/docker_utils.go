package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// getDockerCmd determines the appropriate docker command (docker or podman)
// and checks if it's available. It also provides a helpful error message if
// the docker daemon is unreachable (e.g. requires sudo).
func getDockerCmd(ctx context.Context) (string, error) {
	cmd := os.Getenv("DOCKER_CMD")
	if cmd == "" {
		if _, err := exec.LookPath("docker"); err == nil {
			cmd = "docker"
		} else if _, err := exec.LookPath("podman"); err == nil {
			cmd = "podman"
		} else {
			return "", fmt.Errorf("neither 'docker' nor 'podman' could be found in PATH. Please install Docker or Podman, or set DOCKER_CMD")
		}
	} else {
		if _, err := exec.LookPath(cmd); err != nil {
			return "", fmt.Errorf("command %q specified by DOCKER_CMD not found in PATH", cmd)
		}
	}

	// Simple check to see if we can connect to the daemon.
	checkCmd := exec.CommandContext(ctx, cmd, "info")
	out, err := checkCmd.CombinedOutput()
	if err != nil {
		outStr := strings.ToLower(string(out))
		if strings.Contains(outStr, "permission denied") || strings.Contains(outStr, "connect: ") {
			return "", fmt.Errorf("cannot connect to %s daemon. You might need to start the daemon, use 'sudo', or add your user to the docker group.\nError details: %s", cmd, strings.TrimSpace(string(out)))
		}
		// Some other error, let it pass and fail during actual compose up/down if needed
	}

	return cmd, nil
}
