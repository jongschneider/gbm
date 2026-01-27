// Package testutil provides testing utilities for Bubble Tea TUI components.
package testutil

import (
	"gbm/pkg/tui"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// NewTestWizard creates a new Wizard for testing with the given steps and context.
// It initializes the Wizard with the provided configuration and returns it ready for testing.
func NewTestWizard(steps []tui.Step, ctx *tui.Context) *tui.Wizard {
	return tui.NewWizard(steps, ctx)
}

// UpdateWithKeyMsg is a test helper that updates a Bubble Tea model with a key message.
// It returns the updated model and any command returned by the Update method.
func UpdateWithKeyMsg(model tea.Model, keyStr string) (tea.Model, tea.Cmd) {
	keyMsg := StringToKeyMsg(keyStr)
	return model.Update(keyMsg)
}

// UpdateWithMsg is a test helper that updates a Bubble Tea model with any message.
// It returns the updated model and any command returned by the Update method.
func UpdateWithMsg(model tea.Model, msg tea.Msg) (tea.Model, tea.Cmd) {
	return model.Update(msg)
}

// GetViewAsString renders the model's view and returns it as a string.
// This is a convenience function for asserting on the rendered output.
func GetViewAsString(model tea.Model) string {
	return model.View()
}

// WaitForTimeout simulates waiting with a timeout.
// This is useful for testing async operations or delays.
func WaitForTimeout(duration time.Duration) <-chan time.Time {
	return time.After(duration)
}

// ViewContains checks if the rendered view contains the given substring.
// Useful for verifying that specific UI elements are present.
func ViewContains(model tea.Model, substring string) bool {
	view := model.View()
	return strings.Contains(view, substring)
}

// SendKeySequence sends a sequence of key messages to a model.
// It converts strings like "enter", "up", "down" to their corresponding tea.KeyMsg types
// and applies them in sequence.
func SendKeySequence(model tea.Model, keyStrings ...string) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	for _, keyStr := range keyStrings {
		keyMsg := StringToKeyMsg(keyStr)
		model, cmd = model.Update(keyMsg)
	}
	return model, cmd
}

// StringToKeyMsg converts a key string to a tea.KeyMsg.
// Supports standard key names like "enter", "up", "down", "left", "right", etc.
func StringToKeyMsg(keyStr string) tea.KeyMsg {
	switch strings.ToLower(keyStr) {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "backspace":
		return tea.KeyMsg{Type: tea.KeyBackspace}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "ctrl+c":
		return tea.KeyMsg{Type: tea.KeyCtrlC}
	case "ctrl+d":
		return tea.KeyMsg{Type: tea.KeyCtrlD}
	default:
		// Assume it's a regular character
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(keyStr)}
	}
}

// GetRenderedOutput retrieves the current rendered output from the model.
// This is an alias for GetViewAsString for consistency with common testing patterns.
func GetRenderedOutput(model tea.Model) string {
	return model.View()
}
