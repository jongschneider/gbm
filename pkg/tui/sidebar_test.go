package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSidebar_Create(t *testing.T) {
	testCases := []struct {
		expect func(t *testing.T, sb *Sidebar)
		name   string
	}{
		{
			name: "new sidebar has 4 sections",
			expect: func(t *testing.T, sb *Sidebar) {
				assert.Len(t, sb.sections, 4)
				assert.Equal(t, "Basics", sb.sections[0].Name)
				assert.Equal(t, "JIRA", sb.sections[1].Name)
				assert.Equal(t, "FileCopy", sb.sections[2].Name)
				assert.Equal(t, "Worktrees", sb.sections[3].Name)
			},
		},
		{
			name: "basics section is expanded by default",
			expect: func(t *testing.T, sb *Sidebar) {
				assert.True(t, sb.sections[0].Expanded)
			},
		},
		{
			name: "focused index starts at 0",
			expect: func(t *testing.T, sb *Sidebar) {
				assert.Equal(t, 0, sb.focusedIdx)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sb := NewSidebar(DefaultTheme())
			tc.expect(t, sb)
		})
	}
}

func TestSidebar_Navigation(t *testing.T) {
	testCases := []struct {
		setupFunc func(*Sidebar)
		name      string
		input     tea.KeyMsg
		expected  int
	}{
		{
			name:     "down arrow moves focus down",
			input:    tea.KeyMsg{Type: tea.KeyDown},
			expected: 1,
			setupFunc: func(sb *Sidebar) {
				sb.focusedIdx = 0
			},
		},
		{
			name:     "up arrow moves focus up",
			input:    tea.KeyMsg{Type: tea.KeyUp},
			expected: 1,
			setupFunc: func(sb *Sidebar) {
				sb.focusedIdx = 2
			},
		},
		{
			name:     "down arrow at bottom stays at end",
			input:    tea.KeyMsg{Type: tea.KeyDown},
			expected: 3,
			setupFunc: func(sb *Sidebar) {
				sb.focusedIdx = 3
			},
		},
		{
			name:     "up arrow at top stays at 0",
			input:    tea.KeyMsg{Type: tea.KeyUp},
			expected: 0,
			setupFunc: func(sb *Sidebar) {
				sb.focusedIdx = 0
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sb := NewSidebar(DefaultTheme())
			if tc.setupFunc != nil {
				tc.setupFunc(sb)
			}
			sb.Update(tc.input)
			assert.Equal(t, tc.expected, sb.focusedIdx)
		})
	}
}

func TestSidebar_ExpandCollapse(t *testing.T) {
	testCases := []struct {
		setupFunc func(*Sidebar)
		name      string
		input     tea.KeyMsg
		expected  bool
	}{
		{
			name:     "right arrow expands focused section",
			input:    tea.KeyMsg{Type: tea.KeyRight},
			expected: true,
			setupFunc: func(sb *Sidebar) {
				sb.focusedIdx = 1
				sb.sections[1].Expanded = false
			},
		},
		{
			name:     "left arrow collapses focused section",
			input:    tea.KeyMsg{Type: tea.KeyLeft},
			expected: false,
			setupFunc: func(sb *Sidebar) {
				sb.focusedIdx = 0
				sb.sections[0].Expanded = true
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sb := NewSidebar(DefaultTheme())
			if tc.setupFunc != nil {
				tc.setupFunc(sb)
			}
			sb.Update(tc.input)
			assert.Equal(t, tc.expected, sb.sections[sb.focusedIdx].Expanded)
		})
	}
}

func TestSidebar_SetError(t *testing.T) {
	sb := NewSidebar(DefaultTheme())

	// Set error on JIRA section
	sb.SetError("JIRA", true)
	assert.True(t, sb.sections[1].HasError)
	assert.True(t, sb.hasErrors["JIRA"])

	// Clear error
	sb.SetError("JIRA", false)
	assert.False(t, sb.sections[1].HasError)
	assert.False(t, sb.hasErrors["JIRA"])
}

func TestSidebar_FocusedSection(t *testing.T) {
	sb := NewSidebar(DefaultTheme())

	assert.Equal(t, "Basics", sb.FocusedSection())

	sb.focusedIdx = 2
	assert.Equal(t, "FileCopy", sb.FocusedSection())
}

func TestSidebar_View(t *testing.T) {
	sb := NewSidebar(DefaultTheme())
	view := sb.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Basics")
	assert.Contains(t, view, "JIRA")
	assert.Contains(t, view, "FileCopy")
	assert.Contains(t, view, "Worktrees")
}

func TestSidebar_ViewIndicators(t *testing.T) {
	sb := NewSidebar(DefaultTheme())
	sb.sections[0].Expanded = true
	sb.sections[1].Expanded = false

	view := sb.View()

	// Basics should have ▾ (expanded)
	assert.Contains(t, view, "▾")
	// JIRA should have ▸ (collapsed)
	assert.Contains(t, view, "▸")
}

func TestSidebar_ErrorBadges(t *testing.T) {
	sb := NewSidebar(DefaultTheme())
	sb.SetError("JIRA", true)

	view := sb.View()

	// Should contain error indicator
	assert.Contains(t, view, "⚠")
}
