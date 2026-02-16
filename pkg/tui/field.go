// Package tui provides reusable terminal UI components for wizard-style forms.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

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

// FocusReporter is an optional interface that forms can implement to report
// the Y position of the currently focused field. This enables auto-scrolling
// in viewport containers.
type FocusReporter interface {
	// FocusedYOffset returns the line number (0-indexed) where the focused
	// field starts in the rendered View() output.
	FocusedYOffset() int
}

// Validator is an optional interface that config forms implement to validate
// their fields. Returns a list of human-readable error strings, one per failed
// validation. An empty slice means the form is valid.
// Defined in pkg/tui to avoid import cycles between pkg/tui and pkg/tui/config.
type Validator interface {
	Validate() []string
}

// Flusher is an optional interface that config forms implement to flush
// current field values into the shared ConfigState. This copies in-progress
// edits without triggering onSave callbacks.
// Defined in pkg/tui to avoid import cycles between pkg/tui and pkg/tui/config.
type Flusher interface {
	FlushToState(state *ConfigState)
}

// NextStepMsg signals that the current field is complete and the wizard should advance.
type NextStepMsg struct{}

// CancelMsg signals that the user wants to cancel the wizard.
type CancelMsg struct{}

// PrevStepMsg signals that the user wants to go back to the previous step.
type PrevStepMsg struct{}

// WorkflowCompleteMsg signals that the wizard has completed all steps.
type WorkflowCompleteMsg struct{}
