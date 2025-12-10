package git

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
)

type Service struct{}

func NewService() *Service {
	if _, err := exec.LookPath("git"); err != nil {
		log.Fatal("git command not found in PATH - please install git")
	}
	return &Service{}
}

// runCommand executes a command or prints it if in dry-run mode
func (s *Service) runCommand(cmd *exec.Cmd, dryRun bool) ([]byte, error) {
	cmdStr := formatCommand(cmd)

	if dryRun {
		fmt.Printf("[DRY RUN] %s\n", cmdStr)
		return nil, nil
	}

	return cmd.CombinedOutput()
}

// formatCommand formats a command for display
func formatCommand(cmd *exec.Cmd) string {
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
