// Package port provides utilities for finding available TCP ports
// on the local machine.
package port

import (
	"fmt"
	"net"
)

// MaxPort is the highest valid TCP port number.
const MaxPort = 65535

// FindAvailable attempts to find an available TCP port starting from startPort.
// It tries each port sequentially up to MaxPort.
//
// Note: This function is subject to a TOCTOU (Time-Of-Check-Time-Of-Use) race
// condition. The port may become unavailable between the time it is checked here
// and when it is actually bound by Docker or another process. This is acceptable
// for dev-box because the ports are used in generated configuration files, and
// Docker will report a clear error if a port conflict occurs at runtime.
func FindAvailable(startPort int) (int, error) {
	if startPort < 1 || startPort > MaxPort {
		return 0, fmt.Errorf("start port %d is out of valid range (1-%d)", startPort, MaxPort)
	}

	for port := startPort; port <= MaxPort; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		if err := ln.Close(); err != nil {
			return 0, fmt.Errorf("failed to close listener on port %d: %w", port, err)
		}
		return port, nil
	}

	return 0, fmt.Errorf("no available port found in range %d-%d", startPort, MaxPort)
}
