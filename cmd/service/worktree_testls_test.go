package service

import (
	"bytes"
	"testing"
	"time"

	"gbm/pkg/tui/async"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// TestTestLSInitialRender tests that the model initializes with a table.
func TestTestLSInitialRender(t *testing.T) {
	model := newTestLSModel()
	view := model.View()

	// Check that table header is rendered
	assert.Contains(t, view, "Name")
	assert.Contains(t, view, "Branch")
	assert.Contains(t, view, "Kind")
	assert.Contains(t, view, "Git Status")
}

// TestTestLSPullOperation tests that pressing 'l' triggers a pull operation.
func TestTestLSPullOperation(t *testing.T) {
	model := newTestLSModel()

	// Simulate Init
	model.Init()

	// Simulate pressing 'l' on first row
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(*testlsModel)

	// Check that operation was triggered
	assert.Equal(t, "pull", m.operationStates[0].operation)
}

// TestTestLSPushOperation tests that pressing 'p' triggers a push operation for ad-hoc worktrees.
func TestTestLSPushOperation(t *testing.T) {
	model := newTestLSModel()
	model.Init()

	// First row is "main" which is tracked, so use row 1 instead
	model.table.SetCursor(1) // Move to second row (feature/auth, not tracked)

	// Simulate pressing 'p'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(*testlsModel)

	// Check that operation was triggered
	assert.Equal(t, "push", m.operationStates[1].operation)
}

// TestTestLSDeleteOperation tests that pressing 'd' triggers a delete operation.
func TestTestLSDeleteOperation(t *testing.T) {
	model := newTestLSModel()
	model.Init()

	// Simulate pressing 'd' on first row
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updatedModel, cmd := model.Update(msg)
	m := updatedModel.(*testlsModel)

	// Check that operation was triggered
	assert.NotNil(t, cmd)
	assert.Equal(t, "delete", m.operationStates[0].operation)
}

// TestTestLSQuitCommand tests that pressing 'q' quits the program.
func TestTestLSQuitCommand(t *testing.T) {
	model := newTestLSModel()
	model.Init()

	// Simulate pressing 'q'
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := model.Update(msg)

	// Check that the command is tea.Quit (which returns a Cmd that quits)
	// tea.Quit is a function, so we just check that cmd is not nil
	assert.NotNil(t, cmd)
}

// TestTestLSOperationStateClearing tests that operation state clears after 2 seconds.
func TestTestLSOperationStateClearing(t *testing.T) {
	model := newTestLSModel()
	model.Init()

	// Set up operation state manually
	model.operationStates[0] = operationState{
		operation: "pull",
		result:    "✓ pulled",
		clearAt:   time.Now().Add(-1 * time.Second), // Already expired
	}

	// Simulate clearOperationMsg
	msg := clearOperationMsg{rowIdx: 0}
	updatedModel, _ := model.Update(msg)
	m := updatedModel.(*testlsModel)

	// Check that operation state was cleared
	assert.Equal(t, "", m.operationStates[0].operation)
	assert.Equal(t, "", m.operationStates[0].result)
}

// newTestLSModel creates a fresh testlsModel for testing.
func newTestLSModel() *testlsModel {
	columns := []table.Column{
		{Title: "Name", Width: 30},
		{Title: "Branch", Width: 50},
		{Title: "Kind", Width: 10},
		{Title: "Git Status", Width: 15},
	}

	height := min(len(mockWorktrees)+1, 26)
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(height),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	return &testlsModel{
		table:           t,
		worktrees:       mockWorktrees,
		trackedBranches: trackedBranches,
		delay:           time.Duration(0), // No delay for tests
		asyncStatuses:   make(map[int]*async.Cell[string]),
		asyncOperations: make(map[int]*async.Cell[string]),
		operationStates: make(map[int]operationState),
		messageDump:     &bytes.Buffer{}, // Use buffer for debugging
	}
}
