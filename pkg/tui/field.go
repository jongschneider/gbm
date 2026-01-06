// Package tui provides reusable terminal UI components for wizard-style forms.
package tui

import tea "github.com/charmbracelet/bubbletea"

// Field defines the interface for form components in a wizard.
// Each Field is a Bubble Tea Model with additional lifecycle and state methods.
type Field interface {
	// Bubble Tea Model methods
	Init() tea.Cmd
	Update(tea.Msg) (Field, tea.Cmd)
	View() string

	// Lifecycle methods
	Focus() tea.Cmd
	Blur() tea.Cmd

	// State methods
	IsComplete() bool
	IsCancelled() bool
	Error() error
	Skip() bool

	// Configuration methods
	WithTheme(*Theme) Field
	WithWidth(int) Field
	WithHeight(int) Field

	// Accessor methods
	GetKey() string
	GetValue() any
}

// NextStepMsg signals that the current field is complete and the wizard should advance.
type NextStepMsg struct{}

// CancelMsg signals that the user wants to cancel the wizard.
type CancelMsg struct{}

// PrevStepMsg signals that the user wants to go back to the previous step.
type PrevStepMsg struct{}
