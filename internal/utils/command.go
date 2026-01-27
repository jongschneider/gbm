package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// FormatCommand formats an exec.Cmd for display
// Handles working directory and environment variables (like GIT_DIR).
func FormatCommand(cmd *exec.Cmd) string {
	parts := []string{cmd.Path}
	parts = append(parts, cmd.Args[1:]...)

	// Add working directory if set
	if cmd.Dir != "" {
		return fmt.Sprintf("(cd %s && %s)", cmd.Dir, strings.Join(parts, " "))
	}

	// Add git-dir if set in env
	for _, env := range cmd.Env {
		if after, ok := strings.CutPrefix(env, "GIT_DIR="); ok {
			return fmt.Sprintf("GIT_DIR=%s %s", after, strings.Join(parts, " "))
		}
	}

	return strings.Join(parts, " ")
}
