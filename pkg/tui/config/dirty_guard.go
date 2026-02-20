package config

import (
	"fmt"
	"gbm/pkg/tui"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Quit guard ---.

// handleQuit checks for unsaved changes before quitting. If the dirty tracker
// reports no changes, the TUI exits immediately. Otherwise, it transitions to
// the quit confirmation overlay.
func (m *ConfigModel) handleQuit() (tea.Model, tea.Cmd) {
	if !m.dirty.IsDirty() {
		return m, tea.Quit
	}
	m.state = StateQuitConfirm
	return m, nil
}

// handleQuitConfirmKey processes key presses in the quit confirmation overlay.
//
//   - s:   save and quit (triggers the save flow with quitAfterSave=true)
//   - d:   discard changes and quit immediately
//   - esc: cancel and return to browsing
func (m *ConfigModel) handleQuitConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "s":
		// Save & Quit: trigger the save flow.
		m.state = StateBrowsing
		return m.startSave(true)

	case msg.String() == "d":
		// Discard: quit without saving.
		return m, tea.Quit

	case key.Matches(msg, m.confirmKeys.Cancel):
		// Cancel: return to browsing.
		m.state = StateBrowsing
		return m, nil
	}

	return m, nil
}

// viewQuitConfirm renders the quit confirmation overlay showing a list of
// dirty fields and three action options.
func (m *ConfigModel) viewQuitConfirm() string {
	return renderDirtyOverlay(
		m.theme,
		m.width, m.height,
		"Unsaved Changes",
		m.dirty.DirtyKeys(),
		[]overlayOption{
			{key: "s", label: "Save & Quit"},
			{key: "d", label: "Discard"},
			{key: "esc", label: "Cancel"},
		},
	)
}

// --- Single-field reset ---.

// handleResetField initiates a single-field reset. If the focused field is not
// dirty, this is a no-op. Otherwise, it stores the key and transitions to the
// inline reset confirmation state.
func (m *ConfigModel) handleResetField() (tea.Model, tea.Cmd) {
	if !m.dirty.IsDirty() {
		return m, nil
	}

	if m.focusedFieldKey == "" {
		return m, nil
	}

	if !m.dirty.IsKeyDirty(m.focusedFieldKey) {
		return m, nil
	}

	m.resetKey = m.focusedFieldKey
	m.state = StateResetConfirm
	return m, nil
}

// handleResetConfirmKey processes y/n input for single-field reset.
//
//   - y: reset the field to its original value
//   - n/esc: cancel
func (m *ConfigModel) handleResetConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.confirmKeys.Confirm):
		// y: reset the field.
		resetKey := m.resetKey
		m.dirty.ResetKey(resetKey)
		m.resetFieldRows(resetKey)
		m.resetKey = ""
		m.state = StateBrowsing
		return m, nil

	case key.Matches(msg, m.confirmKeys.Deny), key.Matches(msg, m.confirmKeys.Cancel):
		// n or esc: cancel.
		m.resetKey = ""
		m.state = StateBrowsing
		return m, nil
	}

	return m, nil
}

// --- Reset all ---.

// handleResetAll opens the reset-all confirmation overlay. If nothing is dirty,
// this is a no-op.
func (m *ConfigModel) handleResetAll() (tea.Model, tea.Cmd) {
	if !m.dirty.IsDirty() {
		return m, nil
	}
	m.state = StateResetAllConfirm
	return m, nil
}

// handleResetAllConfirmKey processes key presses in the reset-all overlay.
//
//   - y: reset all fields to their original values
//   - esc: cancel
func (m *ConfigModel) handleResetAllConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.confirmKeys.Confirm):
		m.dirty.ResetAll()
		m.resetAllFieldRows()
		m.state = StateBrowsing
		return m, nil

	case key.Matches(msg, m.confirmKeys.Deny), key.Matches(msg, m.confirmKeys.Cancel):
		m.state = StateBrowsing
		return m, nil
	}

	return m, nil
}

// viewResetAllConfirm renders the reset-all confirmation overlay.
func (m *ConfigModel) viewResetAllConfirm() string {
	return renderDirtyOverlay(
		m.theme,
		m.width, m.height,
		"Reset All Fields",
		m.dirty.DirtyKeys(),
		[]overlayOption{
			{key: "y", label: "Reset"},
			{key: "esc", label: "Cancel"},
		},
	)
}

// --- Shared overlay rendering ---.

// overlayOption represents a key-label pair for overlay action hints.
type overlayOption struct {
	key   string
	label string
}

// renderDirtyOverlay renders a centered modal overlay that lists dirty fields
// and shows action options at the bottom. Used by both quit confirmation and
// reset-all confirmation.
func renderDirtyOverlay(
	theme *tui.Theme,
	width, height int,
	title string,
	dirtyKeys []string,
	options []overlayOption,
) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.Highlight)

	bodyStyle := lipgloss.NewStyle().
		Foreground(theme.Muted)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.Accent)

	fieldStyle := lipgloss.NewStyle().
		Foreground(theme.ErrorAccent)

	// Build the dirty field list.
	var fieldLines []string
	labels := dirtyKeysToLabels(dirtyKeys)
	for _, label := range labels {
		fieldLines = append(fieldLines, fieldStyle.Render("  - "+label))
	}

	// Build option hints.
	var hints []string
	for _, opt := range options {
		hints = append(hints, hintStyle.Render(opt.key)+bodyStyle.Render(" "+opt.label))
	}

	content := titleStyle.Render(title) + "\n\n" +
		bodyStyle.Render(fmt.Sprintf("%d field(s) modified:", len(dirtyKeys))) + "\n" +
		strings.Join(fieldLines, "\n") + "\n\n" +
		strings.Join(hints, bodyStyle.Render("  "))

	innerWidth := max(width-4, 30)
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.Highlight).
		Padding(1, 2).
		Width(innerWidth)

	box := boxStyle.Render(content)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// dirtyKeysToLabels resolves a list of dot-path keys to their human-readable
// labels using the section field definitions. Keys that are not found in any
// section are included as-is.
func dirtyKeysToLabels(keys []string) []string {
	// Build a key -> label lookup from all section fields.
	lookup := make(map[string]string)
	for _, fields := range [][]FieldMeta{generalFields, jiraFields, fileCopyAutoFields} {
		for _, f := range fields {
			lookup[f.Key] = f.Label
		}
	}

	labels := make([]string, 0, len(keys))
	for _, k := range keys {
		if label, ok := lookup[k]; ok {
			labels = append(labels, label)
		} else {
			labels = append(labels, k)
		}
	}
	return labels
}
