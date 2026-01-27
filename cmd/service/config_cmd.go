package service

import (
	"fmt"
	"gbm/pkg/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// newConfigCommand creates the 'gbm config' command that launches the interactive TUI.
func newConfigCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage GBM configuration interactively",
		Long: `Launch an interactive terminal UI for managing .gbm/config.yaml.

The config TUI provides a user-friendly interface to configure GBM settings including:
- Basic settings (default branch, worktrees directory)
- JIRA integration
- File copy rules
- Worktree definitions

All changes are saved to .gbm/config.yaml when you confirm.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigTUI(svc)
		},
	}

	return cmd
}

// runConfigTUI launches the config TUI.
func runConfigTUI(svc *Service) error {
	// Create the root config model
	theme := tui.DefaultTheme()
	model := tui.NewConfigModel(theme)

	// Create and run the program
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Handle result
	_ = finalModel
	return nil
}
