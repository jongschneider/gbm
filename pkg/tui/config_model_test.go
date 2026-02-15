package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestConfigModel_Create(t *testing.T) {
	testCases := []struct {
		expect func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "creates with sidebar and theme",
			expect: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.sidebar)
				assert.NotNil(t, m.theme)
			},
		},
		{
			name: "creates with form cache",
			expect: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.formCache)
			},
		},
		{
			name: "starts with sidebar focused",
			expect: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, SidebarFocused, m.paneFocus)
			},
		},
		{
			name: "creates with empty state by default",
			expect: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.state)
				assert.Empty(t, m.state.DefaultBranch)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(DefaultTheme())
			tc.expect(t, m)
		})
	}
}

func TestConfigModel_WithInitialState(t *testing.T) {
	state := &ConfigState{
		DefaultBranch: "develop",
		WorktreesDir:  "wt",
	}

	m := NewConfigModel(DefaultTheme(), WithInitialState(state))

	assert.Equal(t, "develop", m.GetState().DefaultBranch)
	assert.Equal(t, "wt", m.GetState().WorktreesDir)
}

func TestConfigModel_WithFormFactory(t *testing.T) {
	factoryCalled := false
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		factoryCalled = true
		return &configTestMockModel{}
	}

	// Factory is called during construction to create initial form
	_ = NewConfigModel(DefaultTheme(), WithFormFactory(factory))

	assert.True(t, factoryCalled)
}

func TestConfigModel_SidebarSelection_FocusesContent(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	assert.Equal(t, SidebarFocused, m.paneFocus)

	// Simulate sidebar selection (Enter key)
	m.Update(SidebarSelectionMsg{Section: "Basics"})

	// Focus should move to content
	assert.Equal(t, ContentFocused, m.paneFocus)
}

func TestConfigModel_SidebarSelectionChanged_UpdatesPreview(t *testing.T) {
	callCount := 0
	var lastSection string
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		callCount++
		lastSection = section
		return &configTestMockModel{}
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	initialCallCount := callCount

	// Simulate selection changed (up/down navigation)
	m.Update(SidebarSelectionChangedMsg{Section: "JIRA"})

	// Factory should be called for new section
	assert.Greater(t, callCount, initialCallCount)
	assert.Equal(t, "JIRA", lastSection)
	// Focus should remain on sidebar
	assert.Equal(t, SidebarFocused, m.paneFocus)
}

func TestConfigModel_BackBoundary_FocusesSidebar(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))

	// Focus content
	m.Update(SidebarSelectionMsg{Section: "Basics"})
	assert.Equal(t, ContentFocused, m.paneFocus)

	// Back should focus sidebar
	m.Update(BackBoundaryMsg{})
	assert.Equal(t, SidebarFocused, m.paneFocus)
}

func TestConfigModel_FocusTransitions_LKey(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	assert.Equal(t, SidebarFocused, m.paneFocus)

	// 'l' should focus content
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	assert.Equal(t, ContentFocused, m.paneFocus)
}

func TestConfigModel_FocusTransitions_RightKey(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	assert.Equal(t, SidebarFocused, m.paneFocus)

	// Right arrow should focus content
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	assert.Equal(t, ContentFocused, m.paneFocus)
}

func TestConfigModel_FocusTransitions_HKey(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Focus content
	assert.Equal(t, ContentFocused, m.paneFocus)

	// 'h' should focus sidebar
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.Equal(t, SidebarFocused, m.paneFocus)
}

func TestConfigModel_FocusTransitions_LeftKey(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Focus content
	assert.Equal(t, ContentFocused, m.paneFocus)

	// Left arrow should focus sidebar
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, SidebarFocused, m.paneFocus)
}

func TestConfigModel_DirtyState(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	assert.False(t, m.IsDirty())

	// Manually mark as dirty
	m.state.dirty = true
	assert.True(t, m.IsDirty())
}

func TestConfigModel_DirtyIndicator(t *testing.T) {
	testCases := []struct {
		name   string
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel)
	}{
		{
			name: "insert mode typing marks dirty",
			setup: func(m *ConfigModel) {
				// Focus content pane with insert-mode-capable form
				m.Update(SidebarSelectionMsg{Section: "Basics"})
				// Simulate form entering insert mode
				mock := m.currentForm.(*configTestInsertModeMockModel)
				mock.insertMode = true
				// Type a character while in insert mode
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.True(t, m.IsDirty(), "typing in insert mode should mark dirty")
			},
		},
		{
			name: "space in normal mode marks dirty for confirm toggle",
			setup: func(m *ConfigModel) {
				// Focus content pane
				m.Update(SidebarSelectionMsg{Section: "Basics"})
				// Simulate a Confirm field being focused
				mock := m.currentForm.(*configTestInsertModeMockModel)
				mock.confirmFocused = true
				// Press space in normal mode (toggles Confirm fields)
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.True(t, m.IsDirty(), "space in normal mode should mark dirty")
			},
		},
		{
			name: "navigation keys do not mark dirty",
			setup: func(m *ConfigModel) {
				// Focus content pane
				m.Update(SidebarSelectionMsg{Section: "Basics"})
				// Navigate with j/k (should not dirty)
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
				m.Update(tea.KeyMsg{Type: tea.KeyTab})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.False(t, m.IsDirty(), "navigation keys should not mark dirty")
			},
		},
		{
			name: "esc in insert mode does not mark dirty",
			setup: func(m *ConfigModel) {
				// Focus content pane with insert-mode-capable form
				m.Update(SidebarSelectionMsg{Section: "Basics"})
				mock := m.currentForm.(*configTestInsertModeMockModel)
				mock.insertMode = true
				// Press Esc to exit insert mode
				m.Update(tea.KeyMsg{Type: tea.KeyEsc})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.False(t, m.IsDirty(), "esc should not mark dirty")
			},
		},
		{
			name: "save clears dirty flag",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
				// Save from sidebar
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.False(t, m.IsDirty(), "save should clear dirty flag")
			},
		},
		{
			name: "reset clears dirty flag",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
				// Reset from sidebar
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.False(t, m.IsDirty(), "reset should clear dirty flag")
			},
		},
		{
			name: "footer shows modified when dirty",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
				m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "[modified]")
			},
		},
		{
			name: "footer does not show modified when clean",
			setup: func(m *ConfigModel) {
				m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.NotContains(t, view, "[modified]")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockForm := &configTestInsertModeMockModel{}
			factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
				return mockForm
			}
			onSave := func(state *ConfigState) error { return nil }
			onReset := func() (*ConfigState, error) {
				return &ConfigState{}, nil
			}

			m := NewConfigModel(DefaultTheme(),
				WithFormFactory(factory),
				WithOnSave(onSave),
				WithOnReset(onReset),
			)

			tc.setup(m)
			tc.assert(t, m)
		})
	}
}

func TestConfigModel_Reset(t *testing.T) {
	resetCalled := false
	onReset := func() (*ConfigState, error) {
		resetCalled = true
		return &ConfigState{DefaultBranch: "main"}, nil
	}

	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return &configTestMockModel{}
	}

	m := NewConfigModel(DefaultTheme(), WithOnReset(onReset), WithFormFactory(factory))
	m.state.DefaultBranch = "develop"
	m.state.dirty = true

	// Pre-populate cache by visiting JIRA section
	m.Update(SidebarSelectionChangedMsg{Section: "JIRA"})
	assert.NotEmpty(t, m.GetFormCache())

	// Simulate 'r' key at sidebar level
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.True(t, resetCalled)
	assert.Equal(t, "main", m.state.DefaultBranch)
	// Form cache must be cleared so forms are recreated with fresh state
	assert.Len(t, m.GetFormCache(), 1, "cache should only contain the recreated current form")
}

func TestConfigModel_Reset_InvalidatesFormCache(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func(m *ConfigModel)
		assert      func(t *testing.T, m *ConfigModel)
		assertError func(t *testing.T, err error)
	}{
		{
			name: "clears all cached forms on reset",
			setup: func(m *ConfigModel) {
				// Visit multiple sections to populate cache
				m.Update(SidebarSelectionChangedMsg{Section: "JIRA"})
				m.Update(SidebarSelectionChangedMsg{Section: "Worktrees"})
				m.Update(SidebarSelectionChangedMsg{Section: "Basics"})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// After reset, only the current section's form should exist (recreated fresh)
				assert.Len(t, m.GetFormCache(), 1)
				assert.NotNil(t, m.GetCurrentForm())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "recreates current form with new state",
			setup: func(m *ConfigModel) {
				m.state.DefaultBranch = "edited-value"
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, "main", m.GetState().DefaultBranch)
				assert.NotNil(t, m.GetCurrentForm(), "current form should be recreated")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "previously cached forms are not returned after reset",
			setup: func(m *ConfigModel) {
				// Visit JIRA to cache it
				m.Update(SidebarSelectionChangedMsg{Section: "JIRA"})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// After reset the JIRA form should not be in cache
				_, hasCachedJira := m.GetFormCache()["JIRA"]
				assert.False(t, hasCachedJira, "JIRA form should not survive reset")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factoryCallCount := 0
			factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
				factoryCallCount++
				return &configTestMockModel{}
			}
			onReset := func() (*ConfigState, error) {
				return &ConfigState{DefaultBranch: "main"}, nil
			}

			m := NewConfigModel(DefaultTheme(), WithOnReset(onReset), WithFormFactory(factory))

			tc.setup(m)

			// Reset
			_, cmd := m.handleReset()
			tc.assertError(t, nil) // onReset does not return error in these cases
			_ = cmd

			tc.assert(t, m)
		})
	}
}

func TestConfigModel_Save(t *testing.T) {
	saveCalled := false
	var savedState *ConfigState
	onSave := func(state *ConfigState) error {
		saveCalled = true
		savedState = state
		return nil
	}

	m := NewConfigModel(DefaultTheme(), WithOnSave(onSave))
	m.state.DefaultBranch = "develop"
	m.state.dirty = true

	// Simulate 's' key at sidebar level
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	assert.True(t, saveCalled)
	assert.Equal(t, "develop", savedState.DefaultBranch)
	assert.False(t, m.state.dirty) // dirty flag should be cleared
}

func TestConfigModel_Accessors(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	assert.Equal(t, m.sidebar, m.GetSidebar())
	assert.Equal(t, m.theme, m.GetTheme())
	assert.Equal(t, m.paneFocus, m.GetPaneFocus())
	assert.NotNil(t, m.GetState())
}

func TestConfigModel_GetCurrentForm(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))

	// Should have a current form (created during init for first section)
	assert.NotNil(t, m.GetCurrentForm())
}

func TestConfigModel_WindowSize(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	m.Update(sizeMsg)

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
	// Sidebar gets ~25% of width (max 30)
	assert.LessOrEqual(t, m.sidebar.width, 30)
}

func TestConfigModel_View(t *testing.T) {
	m := NewConfigModel(DefaultTheme())
	// Send WindowSizeMsg to initialize viewports
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	view := m.View()

	// Should render sidebar
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Basics")
}

func TestConfigModel_View_TwoPaneLayout(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	// Send WindowSizeMsg to initialize viewports
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})

	view := m.View()

	// Should contain sidebar content
	assert.Contains(t, view, "Basics")
	// Should contain form content
	assert.Contains(t, view, "mock")
}

func TestConfigModel_Help_ShowsOnQuestionMark(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	// Press '?' at sidebar level
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	assert.NotNil(t, m.helpOverlay)
}

func TestConfigModel_Help_DismissesOnEsc(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	// Show help
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.NotNil(t, m.helpOverlay)

	// Dismiss with Esc
	m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	assert.Nil(t, m.helpOverlay)
}

func TestConfigModel_Help_DismissesOnQuestionMark(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	// Show help
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.NotNil(t, m.helpOverlay)

	// Dismiss with '?' again
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.Nil(t, m.helpOverlay)
}

func TestConfigModel_Help_DismissesOnEnter(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	// Show help
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	assert.NotNil(t, m.helpOverlay)

	// Dismiss with Enter
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, m.helpOverlay)
}

func TestConfigModel_Help_ViewShowsOverlay(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	// Show help
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})

	view := m.View()
	assert.Contains(t, view, "Help - Keyboard Shortcuts")
	assert.Contains(t, view, "Navigation")
}

func TestConfigModel_View_ContainsHelpHint(t *testing.T) {
	m := NewConfigModel(DefaultTheme())
	// Send WindowSizeMsg to initialize viewports
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	view := m.View()

	// Footer should mention '?' for help
	assert.Contains(t, view, "?=help")
}

func TestConfigModel_FormCache(t *testing.T) {
	callCount := 0
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		callCount++
		return &configTestMockModel{}
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	initialCount := callCount

	// Navigate to JIRA section
	m.Update(SidebarSelectionChangedMsg{Section: "JIRA"})
	afterJira := callCount

	// Navigate back to Basics
	m.Update(SidebarSelectionChangedMsg{Section: "Basics"})
	afterBasics := callCount

	// Navigate back to JIRA - should use cache
	m.Update(SidebarSelectionChangedMsg{Section: "JIRA"})
	afterJira2 := callCount

	// Factory was called for initial Basics, then for JIRA
	assert.Greater(t, afterJira, initialCount)
	// Factory was not called again for Basics (already cached)
	assert.Equal(t, afterJira, afterBasics)
	// Factory was not called again for JIRA (already cached)
	assert.Equal(t, afterBasics, afterJira2)
}

func TestConfigModel_SidebarFocusState(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))

	// Initially sidebar is focused
	assert.True(t, m.sidebar.IsFocused())

	// Focus content
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	assert.False(t, m.sidebar.IsFocused())

	// Focus sidebar
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.True(t, m.sidebar.IsFocused())
}

func TestConfigModel_ContentFocused_DelegatesKeysToForm(t *testing.T) {
	testCases := []struct {
		name   string
		key    tea.KeyMsg
		assert func(t *testing.T, m *ConfigModel, mock *configTestTrackingMockModel)
	}{
		{
			name: "s key delegates to form instead of global save",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}},
			assert: func(t *testing.T, m *ConfigModel, mock *configTestTrackingMockModel) {
				t.Helper()
				assert.Contains(t, mock.receivedKeys, "s", "form should receive 's' key")
			},
		},
		{
			name: "q key delegates to form instead of quitting",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}},
			assert: func(t *testing.T, m *ConfigModel, mock *configTestTrackingMockModel) {
				t.Helper()
				assert.Contains(t, mock.receivedKeys, "q", "form should receive 'q' key")
			},
		},
		{
			name: "? key delegates to form instead of showing global help",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}},
			assert: func(t *testing.T, m *ConfigModel, mock *configTestTrackingMockModel) {
				t.Helper()
				assert.Contains(t, mock.receivedKeys, "?", "form should receive '?' key")
				assert.Nil(t, m.helpOverlay, "ConfigModel help overlay should not be shown")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saveCalled := false
			onSave := func(state *ConfigState) error {
				saveCalled = true
				return nil
			}

			mockForm := &configTestTrackingMockModel{}
			factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
				return mockForm
			}

			m := NewConfigModel(DefaultTheme(), WithFormFactory(factory), WithOnSave(onSave))

			// Focus content pane
			m.Update(SidebarSelectionMsg{Section: "Basics"})
			assert.Equal(t, ContentFocused, m.paneFocus)

			// Send the key
			m.Update(tc.key)

			// Global save should NOT have been called
			assert.False(t, saveCalled, "global save should not be called when content is focused")

			tc.assert(t, m, mockForm)
		})
	}
}

func TestConfigModel_SidebarFocused_HandlesKeysGlobally(t *testing.T) {
	testCases := []struct {
		name   string
		key    tea.KeyMsg
		assert func(t *testing.T, m *ConfigModel, saveCalled bool)
	}{
		{
			name: "s key triggers global save from sidebar",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}},
			assert: func(t *testing.T, m *ConfigModel, saveCalled bool) {
				t.Helper()
				assert.True(t, saveCalled, "global save should be called from sidebar")
			},
		},
		{
			name: "? key shows global help from sidebar",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}},
			assert: func(t *testing.T, m *ConfigModel, saveCalled bool) {
				t.Helper()
				assert.NotNil(t, m.helpOverlay, "help overlay should be shown from sidebar")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saveCalled := false
			onSave := func(state *ConfigState) error {
				saveCalled = true
				return nil
			}

			m := NewConfigModel(DefaultTheme(), WithOnSave(onSave))
			assert.Equal(t, SidebarFocused, m.paneFocus)

			m.Update(tc.key)

			tc.assert(t, m, saveCalled)
		})
	}
}

func TestConfigModel_CtrlC_AlwaysQuits(t *testing.T) {
	testCases := []struct {
		name  string
		setup func(m *ConfigModel)
	}{
		{
			name:  "quits from sidebar",
			setup: func(m *ConfigModel) {},
		},
		{
			name: "quits from content pane",
			setup: func(m *ConfigModel) {
				m.Update(SidebarSelectionMsg{Section: "Basics"})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockForm := &configTestMockModel{}
			factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
				return mockForm
			}

			m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
			tc.setup(m)

			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

			// Should produce a quit command
			assert.NotNil(t, cmd, "ctrl+c should produce a quit command")
		})
	}
}

func TestConfigModel_QuitConfirmation(t *testing.T) {
	testCases := []struct {
		name string
		// action returns the cmd from the final Update call being tested
		action func(m *ConfigModel) tea.Cmd
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
	}{
		{
			name: "q quits immediately when not dirty",
			action: func(m *ConfigModel) tea.Cmd {
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.NotNil(t, cmd, "should produce a quit command")
				assert.False(t, m.ShowConfirmQuit(), "should not show confirmation")
			},
		},
		{
			name: "q shows confirmation when dirty",
			action: func(m *ConfigModel) tea.Cmd {
				m.state.MarkDirty()
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.True(t, m.ShowConfirmQuit(), "should show quit confirmation")
				assert.NotNil(t, m.discardField, "discard field should be created")
			},
		},
		{
			name: "confirmation y quits",
			action: func(m *ConfigModel) tea.Cmd {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.NotNil(t, cmd, "should produce a quit command after confirmation")
			},
		},
		{
			name: "confirmation n returns to sidebar",
			action: func(m *ConfigModel) tea.Cmd {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.False(t, m.ShowConfirmQuit(), "confirmation should be dismissed")
				assert.Nil(t, m.discardField, "discard field should be cleared")
			},
		},
		{
			name: "confirmation enter with Yes selected quits",
			action: func(m *ConfigModel) tea.Cmd {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.NotNil(t, cmd, "should produce a quit command")
			},
		},
		{
			name: "confirmation enter with No selected returns to sidebar",
			action: func(m *ConfigModel) tea.Cmd {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
				// Move selection to No
				m.Update(tea.KeyMsg{Type: tea.KeyRight})
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.False(t, m.ShowConfirmQuit(), "confirmation should be dismissed")
				assert.Nil(t, m.discardField, "discard field should be cleared")
			},
		},
		{
			name: "view shows confirmation dialog when active",
			action: func(m *ConfigModel) tea.Cmd {
				m.state.MarkDirty()
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Discard unsaved changes?")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(DefaultTheme())
			assert.Equal(t, SidebarFocused, m.paneFocus)

			cmd := tc.action(m)
			tc.assert(t, m, cmd)
		})
	}
}

// configTestMockModel is a simple mock tea.Model for testing.
type configTestMockModel struct{}

func (m *configTestMockModel) Init() tea.Cmd                       { return nil }
func (m *configTestMockModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *configTestMockModel) View() string                        { return "mock" }
func (m *configTestMockModel) Focus() tea.Cmd                      { return nil }
func (m *configTestMockModel) Blur() tea.Cmd                       { return nil }

// configTestTrackingMockModel tracks which key messages it receives.
type configTestTrackingMockModel struct {
	receivedKeys []string
}

func (m *configTestTrackingMockModel) Init() tea.Cmd { return nil }
func (m *configTestTrackingMockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		m.receivedKeys = append(m.receivedKeys, keyMsg.String())
	}
	return m, nil
}
func (m *configTestTrackingMockModel) View() string   { return "mock" }
func (m *configTestTrackingMockModel) Focus() tea.Cmd { return nil }
func (m *configTestTrackingMockModel) Blur() tea.Cmd  { return nil }

// configTestInsertModeMockModel simulates a form that supports insert mode
// and reports whether a confirm field is focused.
type configTestInsertModeMockModel struct {
	insertMode     bool
	confirmFocused bool
}

func (m *configTestInsertModeMockModel) Init() tea.Cmd { return nil }
func (m *configTestInsertModeMockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyEsc {
			m.insertMode = false
		}
	}
	return m, nil
}
func (m *configTestInsertModeMockModel) View() string              { return "mock" }
func (m *configTestInsertModeMockModel) Focus() tea.Cmd            { return nil }
func (m *configTestInsertModeMockModel) Blur() tea.Cmd             { return nil }
func (m *configTestInsertModeMockModel) InInsertMode() bool        { return m.insertMode }
func (m *configTestInsertModeMockModel) ConfirmFieldFocused() bool { return m.confirmFocused }
