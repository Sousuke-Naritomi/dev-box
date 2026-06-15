package cmd

import (
	"fmt"
	"io"
)

func writeCommandOutput(output io.Writer, format string, args ...any) error {
	if _, err := fmt.Fprintf(output, format, args...); err != nil {
		return fmt.Errorf("writing command output: %w", err)
	}
	return nil
}
