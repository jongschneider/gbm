package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestHelpOverlay_NewHelpOverlay(t *testing.T) {
	overlay := NewHelpOverlay()

	assert.NotNil(t, overlay)
	assert.NotEmpty(t, overlay.Groups())

	// Verify default groups exist
	groupNames := make([]string, len(overlay.Groups()))
	for i, g := range overlay.Groups() {
		groupNames[i] = g.Name
	}
	assert.Contains(t, groupNames, "Sidebar")
	assert.Contains(t, groupNames, "Content Pane")
	assert.Contains(t, groupNames, "Table Forms (FileCopy, Worktrees)")
	assert.Contains(t, groupNames, "Global")
	assert.Contains(t, groupNames, "Modals")
}

func TestHelpOverlay_WithTheme(t *testing.T) {
	overlay := NewHelpOverlay()
	theme := DefaultTheme()

	result := overlay.WithTheme(theme)

	assert.Same(t, overlay, result) // Chainable
	assert.NotNil(t, overlay.theme)
}

func TestHelpOverlay_WithTheme_Nil(t *testing.T) {
	overlay := NewHelpOverlay()
	originalTheme := overlay.theme

	overlay.WithTheme(nil)

	assert.Same(t, originalTheme, overlay.theme) // Unchanged
}

func TestHelpOverlay_WithWidth(t *testing.T) {
	overlay := NewHelpOverlay()

	result := overlay.WithWidth(80)

	assert.Same(t, overlay, result) // Chainable
	assert.Equal(t, 80, overlay.width)
}

func TestHelpOverlay_WithHeight(t *testing.T) {
	overlay := NewHelpOverlay()

	result := overlay.WithHeight(40)

	assert.Same(t, overlay, result) // Chainable
	assert.Equal(t, 40, overlay.height)
}

func TestHelpOverlay_WithGroups(t *testing.T) {
	overlay := NewHelpOverlay()
	customGroups := []ShortcutGroup{
		{
			Name: "Custom",
			Shortcuts: []Shortcut{
				{Key: "x", Description: "Test action"},
			},
		},
	}

	result := overlay.WithGroups(customGroups)

	assert.Same(t, overlay, result) // Chainable
	assert.Len(t, overlay.Groups(), 1)
	assert.Equal(t, "Custom", overlay.Groups()[0].Name)
}

func TestHelpOverlay_Init(t *testing.T) {
	overlay := NewHelpOverlay()

	cmd := overlay.Init()

	assert.Nil(t, cmd)
}

func TestHelpOverlay_Update_DismissOnEsc(t *testing.T) {
	overlay := NewHelpOverlay()

	_, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyEscape})

	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(HelpOverlayDismissedMsg)
	assert.True(t, ok)
}

func TestHelpOverlay_Update_DismissOnQuestionMark(t *testing.T) {
	overlay := NewHelpOverlay()

	_, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(HelpOverlayDismissedMsg)
	assert.True(t, ok)
}

func TestHelpOverlay_Update_DismissOnEnter(t *testing.T) {
	overlay := NewHelpOverlay()

	_, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(HelpOverlayDismissedMsg)
	assert.True(t, ok)
}

func TestHelpOverlay_Update_IgnoresOtherKeys(t *testing.T) {
	overlay := NewHelpOverlay()

	_, cmd := overlay.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	assert.Nil(t, cmd)
}

func TestHelpOverlay_Update_WindowSize(t *testing.T) {
	overlay := NewHelpOverlay()

	_, _ = overlay.Update(tea.WindowSizeMsg{Width: 100, Height: 50})

	assert.Equal(t, 100, overlay.width)
	assert.Equal(t, 50, overlay.height)
}

func TestHelpOverlay_View_ContainsTitle(t *testing.T) {
	overlay := NewHelpOverlay()

	view := overlay.View()

	assert.Contains(t, view, "Help - Keyboard Shortcuts")
}

func TestHelpOverlay_View_ContainsGroupNames(t *testing.T) {
	overlay := NewHelpOverlay()

	view := overlay.View()

	assert.Contains(t, view, "Sidebar")
	assert.Contains(t, view, "Content Pane")
	assert.Contains(t, view, "Table Forms (FileCopy, Worktrees)")
	assert.Contains(t, view, "Global")
	assert.Contains(t, view, "Modals")
}

func TestHelpOverlay_SaveShortcutInGlobalGroup(t *testing.T) {
	overlay := NewHelpOverlay()

	// Find the Global and Sidebar groups
	var globalGroup *ShortcutGroup
	var sidebarGroup *ShortcutGroup
	for i, g := range overlay.Groups() {
		switch g.Name {
		case "Global":
			globalGroup = &overlay.Groups()[i]
		case "Sidebar":
			sidebarGroup = &overlay.Groups()[i]
		}
	}

	// Ctrl+S should be in Global group
	assert.NotNil(t, globalGroup, "Global group should exist")
	globalKeys := make([]string, len(globalGroup.Shortcuts))
	for i, s := range globalGroup.Shortcuts {
		globalKeys[i] = s.Key
	}
	assert.Contains(t, globalKeys, "Ctrl+S", "Ctrl+S save shortcut should be in Global group")
	assert.Contains(t, globalKeys, "Ctrl+C", "Ctrl+C quit shortcut should be in Global group")

	// q should be in Sidebar group (not Global)
	assert.NotNil(t, sidebarGroup, "Sidebar group should exist")
	sidebarKeys := make([]string, len(sidebarGroup.Shortcuts))
	for i, s := range sidebarGroup.Shortcuts {
		sidebarKeys[i] = s.Key
	}
	assert.Contains(t, sidebarKeys, "q", "q quit shortcut should be in Sidebar group")
	assert.NotContains(t, globalKeys, "q", "q should not be in Global group")
}

func TestHelpOverlay_View_ContainsShortcuts(t *testing.T) {
	overlay := NewHelpOverlay()

	view := overlay.View()

	// Check some key shortcuts are present
	assert.Contains(t, view, "j/k")
	assert.Contains(t, view, "Enter")
	assert.Contains(t, view, "Tab")
	assert.Contains(t, view, "Esc")
}

func TestHelpOverlay_View_ContainsDismissHelp(t *testing.T) {
	overlay := NewHelpOverlay()

	view := overlay.View()

	assert.Contains(t, view, "Press Escape")
}

func TestHelpOverlay_View_HasBorder(t *testing.T) {
	overlay := NewHelpOverlay()

	view := overlay.View()

	// Rounded border uses these characters
	assert.True(t, strings.Contains(view, "╭") || strings.Contains(view, "┌"))
}

func TestHelpOverlay_View_RespondsToWidth(t *testing.T) {
	overlay := NewHelpOverlay().WithWidth(30)

	view := overlay.View()

	// View should be constrained by width
	lines := strings.SplitSeq(view, "\n")
	for line := range lines {
		// Allow some margin for border decorations
		assert.LessOrEqual(t, len([]rune(line)), 35)
	}
}
