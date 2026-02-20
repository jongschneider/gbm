package config

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// --- Empty section state ---

// EmptyState tracks whether a section is in its "not configured" state and
// provides the placeholder message and default-population behavior.
type EmptyState struct {
	defaults []FieldMeta
	isEmpty  bool
}

// NewEmptyState creates an EmptyState. If isEmpty is true, the section
// renders a placeholder instead of fields.
func NewEmptyState(isEmpty bool, defaults []FieldMeta) *EmptyState {
	return &EmptyState{
		isEmpty:  isEmpty,
		defaults: defaults,
	}
}

// IsEmpty reports whether the section is in its empty (not configured) state.
func (es *EmptyState) IsEmpty() bool {
	return es.isEmpty
}

// SetEmpty sets the empty state.
func (es *EmptyState) SetEmpty(empty bool) {
	es.isEmpty = empty
}

// Defaults returns the default fields to populate when the user presses `e`
// on an empty section.
func (es *EmptyState) Defaults() []FieldMeta {
	return es.defaults
}

// PlaceholderText returns the message shown in an empty section.
func PlaceholderText() string {
	return "not configured, press e to set up"
}

// RenderEmptySection renders the empty-section placeholder centered in the
// content area.
func RenderEmptySection(width, height int, theme *lipgloss.Style) string {
	msg := PlaceholderText()

	style := lipgloss.NewStyle().
		Italic(true).
		Width(width).
		Align(lipgloss.Center)

	if theme != nil {
		style = *theme
	}

	// Center vertically.
	padding := ""
	if height > 1 {
		topPad := height / 2
		padding = strings.Repeat("\n", topPad)
	}
	return padding + style.Render(msg)
}

// --- SectionModel empty state methods ---

// EmptyState returns the section's empty state, or nil if not configured.
func (s *SectionModel) EmptyState() *EmptyState {
	return s.emptyState
}

// IsEmpty reports whether the section is in its empty (not configured) state.
func (s *SectionModel) IsEmpty() bool {
	return s.emptyState != nil && s.emptyState.IsEmpty()
}

// PopulateDefaults transitions the section out of the empty state by replacing
// the field list with the defaults from the EmptyState. Returns true if the
// section was empty and defaults were populated; false otherwise.
func (s *SectionModel) PopulateDefaults() bool {
	if s.emptyState == nil || !s.emptyState.IsEmpty() {
		return false
	}
	s.emptyState.SetEmpty(false)
	s.fields = s.emptyState.Defaults()
	s.buildRows()
	s.focusFirst()
	return true
}

// ViewEmpty renders the empty-section placeholder. This is used by View()
// when the section is in its empty state.
func (s *SectionModel) ViewEmpty() string {
	msg := PlaceholderText()
	style := lipgloss.NewStyle().
		Foreground(s.theme.Muted).
		Italic(true).
		Width(s.effectiveWidth()).
		Align(lipgloss.Center)

	vpHeight := s.viewportHeight
	padding := ""
	if vpHeight > 1 {
		topPad := vpHeight / 2
		padding = strings.Repeat("\n", topPad)
	}

	lines := padding + style.Render(msg)
	// Pad remaining lines to fill viewport.
	current := strings.Count(lines, "\n") + 1
	for current < vpHeight {
		lines += "\n"
		current++
	}
	return lines
}

// --- Enabled toggle / section removal ---

// SectionEnabled tracks the enabled state for optional sections like JIRA.
// When enabled is toggled off and the config is saved, the section is removed
// from the YAML (lossy operation).
type SectionEnabled struct {
	key     string // dot-path key for the enabled flag (e.g., "jira.enabled")
	enabled bool
}

// NewSectionEnabled creates a SectionEnabled tracker.
func NewSectionEnabled(key string, enabled bool) *SectionEnabled {
	return &SectionEnabled{
		key:     key,
		enabled: enabled,
	}
}

// IsEnabled reports whether the section is enabled.
func (se *SectionEnabled) IsEnabled() bool {
	return se.enabled
}

// Toggle flips the enabled state.
func (se *SectionEnabled) Toggle() {
	se.enabled = !se.enabled
}

// Key returns the dot-path key for this toggle.
func (se *SectionEnabled) Key() string {
	return se.key
}

// VisibleFieldCount returns 0 when disabled (remaining fields are hidden),
// or fieldCount when enabled.
func (se *SectionEnabled) VisibleFieldCount(fieldCount int) int {
	if !se.enabled {
		return 0
	}
	return fieldCount
}

// --- Corrupt config state ---

// CorruptConfigState holds the error message from a YAML parse failure.
// When active, the TUI shows an error banner with the parse error and
// allows the user to open $EDITOR to fix the file.
type CorruptConfigState struct {
	parseError string
	filePath   string
}

// NewCorruptConfigState creates a CorruptConfigState with the given parse
// error and config file path.
func NewCorruptConfigState(parseError string, filePath string) *CorruptConfigState {
	return &CorruptConfigState{
		parseError: parseError,
		filePath:   filePath,
	}
}

// ParseError returns the YAML parse error message.
func (cs *CorruptConfigState) ParseError() string {
	return cs.parseError
}

// FilePath returns the path to the corrupt config file.
func (cs *CorruptConfigState) FilePath() string {
	return cs.filePath
}

// editorReloadMsg is sent after the external editor closes so the model
// can attempt to reload the config file.
type editorReloadMsg struct {
	err error
}

// openEditorCmd returns a tea.Cmd that opens the config file in the user's
// $EDITOR (falling back to "vi"). After the editor exits, it sends an
// editorReloadMsg.
func openEditorCmd(filePath string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, filePath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return editorReloadMsg{err: err}
	})
}

// RenderCorruptConfig renders the corrupt config error banner. It shows the
// parse error message and instructions to press `e` to open $EDITOR.
func RenderCorruptConfig(width, height int, parseError string, theme CorruptConfigTheme) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(theme.ErrorAccent)

	bodyStyle := lipgloss.NewStyle().
		Foreground(theme.Muted)

	hintStyle := lipgloss.NewStyle().
		Foreground(theme.Accent)

	errorStyle := lipgloss.NewStyle().
		Foreground(theme.ErrorAccent)

	content := titleStyle.Render("Corrupt Config File") + "\n\n" +
		errorStyle.Render(parseError) + "\n\n" +
		hintStyle.Render("e") + bodyStyle.Render(" open in $EDITOR  ") +
		hintStyle.Render("q") + bodyStyle.Render("/") +
		hintStyle.Render("esc") + bodyStyle.Render(" quit")

	innerWidth := max(width-4, 30)
	boxStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(theme.ErrorAccent).
		Padding(1, 2).
		Width(innerWidth)

	box := boxStyle.Render(content)

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(box)
}

// CorruptConfigTheme holds the color values needed to render the corrupt
// config overlay. This avoids depending on the full tui.Theme type.
type CorruptConfigTheme struct {
	ErrorAccent lipgloss.AdaptiveColor
	Muted       lipgloss.AdaptiveColor
	Accent      lipgloss.AdaptiveColor
}

// --- ConfigModel integration ---

// handleCorruptConfigKey processes key presses in the corrupt config state.
// `e` opens $EDITOR, `q`/esc quits.
func (m *ConfigModel) handleCorruptConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "e":
		if m.corruptConfig != nil {
			return m, openEditorCmd(m.corruptConfig.FilePath())
		}
		return m, nil
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	}
	return m, nil
}

// handleEditorReload processes the result of the external editor closing.
// It attempts to reload the config file. On success, it clears the corrupt
// state and transitions to browsing. On failure, it updates the error message.
func (m *ConfigModel) handleEditorReload(msg editorReloadMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		// Editor failed to run -- keep the corrupt state with updated message.
		if m.corruptConfig != nil {
			m.corruptConfig = NewCorruptConfigState(
				fmt.Sprintf("editor error: %s", msg.err),
				m.corruptConfig.FilePath(),
			)
		}
		return m, nil
	}

	// Attempt to reload the config file.
	if m.corruptConfig == nil {
		m.state = StateBrowsing
		return m, nil
	}

	cf, err := LoadConfigFile(m.corruptConfig.FilePath())
	if err != nil {
		// Still corrupt -- update the error message.
		m.corruptConfig = NewCorruptConfigState(
			err.Error(),
			m.corruptConfig.FilePath(),
		)
		return m, nil
	}

	// Config is now valid. Clear corrupt state and transition to browsing.
	m.root = cf.Root
	m.modTime = cf.ModTime
	m.corruptConfig = nil
	m.state = StateBrowsing

	return m, m.SetFlash("config reloaded")
}

// viewCorruptConfig renders the corrupt config error banner.
func (m *ConfigModel) viewCorruptConfig() string {
	parseError := "unknown error"
	if m.corruptConfig != nil {
		parseError = m.corruptConfig.ParseError()
	}

	return RenderCorruptConfig(m.width, m.height, parseError, CorruptConfigTheme{
		ErrorAccent: m.theme.ErrorAccent,
		Muted:       m.theme.Muted,
		Accent:      m.theme.Accent,
	})
}

// SetCorruptConfig puts the model into corrupt config state with the given
// parse error and file path.
func (m *ConfigModel) SetCorruptConfig(parseError string, filePath string) {
	m.corruptConfig = NewCorruptConfigState(parseError, filePath)
	m.state = StateCorruptConfig
}

// CorruptConfig returns the corrupt config state, or nil if not in that state.
func (m *ConfigModel) CorruptConfig() *CorruptConfigState {
	return m.corruptConfig
}

// --- Terminal too small (supplemental) ---

// IsTerminalTooSmall reports whether the current terminal dimensions are
// below the minimum required by the Config TUI.
func (m *ConfigModel) IsTerminalTooSmall() bool {
	return m.width < MinTermWidth || m.height < MinTermHeight
}

// TooSmallMessage returns the message shown when the terminal is too small.
func TooSmallMessage(width, height int) string {
	return fmt.Sprintf(
		"Terminal too small (%dx%d). Minimum: %dx%d.",
		width, height, MinTermWidth, MinTermHeight,
	)
}
