package utils

import (
	"fmt"
	"os"
)

// MkdirAll creates a directory and all necessary parents, with dry-run support.
func MkdirAll(path string, dryRun bool) error {
	if dryRun {
		fmt.Printf("[DRY RUN] mkdir -p %s\n", path)
		return nil
	}

	err := os.MkdirAll(path, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}
