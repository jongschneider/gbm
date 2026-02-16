package tui

import (
	"errors"
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

func TestConfigModel_FocusTransitions_HKey_DelegatesToForm(t *testing.T) {
	mockForm := &configTestTrackingMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Focus content
	assert.Equal(t, ContentFocused, m.paneFocus)

	// 'h' is now delegated to the form (no longer intercepted by ConfigModel)
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	assert.Equal(t, ContentFocused, m.paneFocus, "h should be delegated to form, not return to sidebar")
	assert.Contains(t, mockForm.receivedKeys, "h", "form should receive h key")
}

func TestConfigModel_FocusTransitions_LeftKey_DelegatesToForm(t *testing.T) {
	mockForm := &configTestTrackingMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}) // Focus content
	assert.Equal(t, ContentFocused, m.paneFocus)

	// Left arrow is now delegated to the form (no longer intercepted by ConfigModel)
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	assert.Equal(t, ContentFocused, m.paneFocus, "left should be delegated to form, not return to sidebar")
	assert.Contains(t, mockForm.receivedKeys, "left", "form should receive left key")
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
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "save clears dirty flag",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
				// Ctrl+S triggers save flow (flush+validate+confirm), then 'y' confirms
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
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
			mockForm := &configTestMockModel{}
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
		setup       func(m *ConfigModel)
		assert      func(t *testing.T, m *ConfigModel)
		assertError func(t *testing.T, err error)
		name        string
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

	// Ctrl+S triggers save flow (flush+validate+confirm), then 'y' confirms
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

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
	assert.Contains(t, view, "Sidebar")
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

	// BackBoundaryMsg returns to sidebar (forms emit this on Esc)
	m.Update(BackBoundaryMsg{})
	assert.True(t, m.sidebar.IsFocused())
}

func TestConfigModel_ContentFocused_DelegatesKeysToForm(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel, mock *configTestTrackingMockModel)
		name   string
		key    tea.KeyMsg
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
		assert func(t *testing.T, m *ConfigModel)
		name   string
		key    tea.KeyMsg
	}{
		{
			name: "? key shows global help from sidebar",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.helpOverlay, "help overlay should be shown from sidebar")
			},
		},
		{
			name: "s key does not trigger save from sidebar",
			key:  tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.False(t, m.ShowSaveConfirm(), "save confirmation should not be shown from sidebar")
				assert.Nil(t, m.saveConfirmField, "save confirm field should not be created from sidebar")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(DefaultTheme())
			assert.Equal(t, SidebarFocused, m.paneFocus)

			m.Update(tc.key)

			tc.assert(t, m)
		})
	}
}

func TestConfigModel_CtrlS_TriggersSave(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name:  "shows save confirmation from sidebar",
			setup: func(m *ConfigModel) {},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.True(t, m.ShowSaveConfirm(), "should show save confirmation dialog")
				assert.NotNil(t, m.saveConfirmField, "save confirm field should be created")
			},
		},
		{
			name: "shows save confirmation from content pane",
			setup: func(m *ConfigModel) {
				m.Update(SidebarSelectionMsg{Section: "Basics"})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.True(t, m.ShowSaveConfirm(), "should show save confirmation dialog")
				assert.NotNil(t, m.saveConfirmField, "save confirm field should be created")
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

			m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
			tc.assert(t, m)
		})
	}
}

func TestConfigModel_CtrlC_AlwaysQuits(t *testing.T) {
	testCases := []struct {
		setup func(m *ConfigModel)
		name  string
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
		{
			name: "quits from save confirmation dialog",
			setup: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
			},
		},
		{
			name: "quits from help overlay",
			setup: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
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

func TestConfigModel_QuitWhenClean(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
		name   string
	}{
		{
			name:  "q quits from sidebar when clean",
			setup: func(m *ConfigModel) {},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.NotNil(t, cmd, "q should produce a quit command")
				assert.False(t, m.ShowSaveConfirm(), "should not show save dialog")
			},
		},
		{
			name: "q from content pane is delegated to form",
			setup: func(m *ConfigModel) {
				m.Update(SidebarSelectionMsg{Section: "Basics"})
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				// q is now delegated to the form when content is focused
				assert.Equal(t, ContentFocused, m.paneFocus, "should remain on content pane")
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

			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			tc.assert(t, m, cmd)
		})
	}
}

func TestConfigModel_QuitWhenDirty(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel, cmd tea.Cmd)
		name   string
	}{
		{
			name: "q shows save dialog from sidebar when dirty",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.True(t, m.ShowSaveConfirm(), "should show save confirmation dialog")
				assert.NotNil(t, m.saveConfirmField, "save confirm field should be created")
				assert.Equal(t, SaveContextQuit, m.GetSaveConfirmContext())
			},
		},
		{
			name: "q from content pane when dirty is delegated to form",
			setup: func(m *ConfigModel) {
				m.Update(SidebarSelectionMsg{Section: "Basics"})
				m.state.MarkDirty()
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				// q is delegated to form when content is focused, not intercepted by ConfigModel
				assert.False(t, m.ShowSaveConfirm(), "should not show save dialog from content pane")
				assert.Equal(t, ContentFocused, m.paneFocus, "should remain on content pane")
			},
		},
		{
			name: "quit context yes saves and quits",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.True(t, m.ShowSaveConfirm(), "should show save confirmation dialog")

				// Confirm save - in quit context, "Yes" saves and quits
				_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
				assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
				assert.False(t, m.IsDirty(), "dirty flag should be cleared after save")
				assert.NotNil(t, quitCmd, "should produce quit command after save")
			},
		},
		{
			name: "quit context no discards and quits",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.True(t, m.ShowSaveConfirm(), "should show save confirmation dialog")

				// "No" in quit context: discard changes and quit
				_, quitCmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
				assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
				assert.True(t, m.IsDirty(), "dirty flag should remain set (changes discarded)")
				assert.NotNil(t, quitCmd, "should produce quit command")
			},
		},
		{
			name: "quit context esc returns to sidebar without quitting",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
			},
			assert: func(t *testing.T, m *ConfigModel, cmd tea.Cmd) {
				t.Helper()
				assert.True(t, m.ShowSaveConfirm(), "should show save confirmation dialog")

				// Esc cancels the quit dialog
				m.Update(tea.KeyMsg{Type: tea.KeyEscape})
				assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
				assert.True(t, m.IsDirty(), "dirty flag should remain set")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockForm := &configTestMockModel{}
			factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
				return mockForm
			}
			onSave := func(state *ConfigState) error { return nil }

			m := NewConfigModel(DefaultTheme(), WithFormFactory(factory), WithOnSave(onSave))
			tc.setup(m)

			_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			tc.assert(t, m, cmd)
		})
	}
}

func TestConfigModel_CtrlS_SaveFlow(t *testing.T) {
	testCases := []struct {
		action func(m *ConfigModel) tea.Cmd
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "shows save confirmation dialog with save context",
			action: func(m *ConfigModel) tea.Cmd {
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.True(t, m.ShowSaveConfirm(), "save confirmation should be shown")
				assert.NotNil(t, m.saveConfirmField, "save confirm field should be created")
				assert.Equal(t, SaveContextSave, m.GetSaveConfirmContext())
			},
		},
		{
			name: "view shows save configuration title for save context",
			action: func(m *ConfigModel) tea.Cmd {
				_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				return cmd
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Save configuration?")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(DefaultTheme())
			tc.action(m)
			tc.assert(t, m)
		})
	}
}

func TestConfigModel_SaveConfirmation(t *testing.T) {
	testCases := []struct {
		action func(m *ConfigModel, saveCalled *bool)
		assert func(t *testing.T, m *ConfigModel, saveCalled bool)
		name   string
	}{
		{
			name: "y confirms save and writes to disk",
			action: func(m *ConfigModel, saveCalled *bool) {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			},
			assert: func(t *testing.T, m *ConfigModel, saveCalled bool) {
				t.Helper()
				assert.True(t, saveCalled, "onSave should be called")
				assert.False(t, m.IsDirty(), "dirty flag should be cleared")
				assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
			},
		},
		{
			name: "n dismisses without saving in save context",
			action: func(m *ConfigModel, saveCalled *bool) {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
			},
			assert: func(t *testing.T, m *ConfigModel, saveCalled bool) {
				t.Helper()
				assert.False(t, saveCalled, "onSave should not be called")
				assert.True(t, m.IsDirty(), "dirty flag should remain set")
				assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
			},
		},
		{
			name: "enter with Yes selected saves",
			action: func(m *ConfigModel, saveCalled *bool) {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			},
			assert: func(t *testing.T, m *ConfigModel, saveCalled bool) {
				t.Helper()
				assert.True(t, saveCalled, "onSave should be called")
				assert.False(t, m.IsDirty(), "dirty flag should be cleared")
			},
		},
		{
			name: "enter with No selected does not save",
			action: func(m *ConfigModel, saveCalled *bool) {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				// Move selection to No
				m.Update(tea.KeyMsg{Type: tea.KeyRight})
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
			},
			assert: func(t *testing.T, m *ConfigModel, saveCalled bool) {
				t.Helper()
				assert.False(t, saveCalled, "onSave should not be called")
				assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
			},
		},
		{
			name: "esc dismisses dialog without saving",
			action: func(m *ConfigModel, saveCalled *bool) {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyEscape})
			},
			assert: func(t *testing.T, m *ConfigModel, saveCalled bool) {
				t.Helper()
				assert.False(t, saveCalled, "onSave should not be called")
				assert.True(t, m.IsDirty(), "dirty flag should remain set")
				assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
				assert.Nil(t, m.saveConfirmField, "save confirm field should be cleared")
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
			tc.action(m, &saveCalled)
			tc.assert(t, m, saveCalled)
		})
	}
}

func TestConfigModel_SaveError(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "save failure sets error message and keeps dirty flag",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Contains(t, m.GetSaveError(), "permission denied")
				assert.True(t, m.IsDirty(), "dirty flag should remain set on save failure")
				assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
			},
		},
		{
			name: "save failure renders error in footer",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
				m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Save failed:")
				assert.Contains(t, view, "permission denied")
			},
		},
		{
			name: "error clears on next keypress",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
				m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
				// Press any key to clear the error
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Empty(t, m.GetSaveError(), "error should be cleared after keypress")
				view := m.View()
				assert.NotContains(t, view, "Save failed:")
			},
		},
		{
			name:  "successful save does not set error",
			setup: func(m *ConfigModel) {},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Empty(t, m.GetSaveError())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			saveErr := errors.New("permission denied")

			// For the "successful save" case, use a nil error
			onSave := func(state *ConfigState) error {
				return saveErr
			}
			if tc.name == "successful save does not set error" {
				onSave = func(state *ConfigState) error { return nil }
			}

			m := NewConfigModel(DefaultTheme(), WithOnSave(onSave))
			tc.setup(m)
			tc.assert(t, m)
		})
	}
}

func TestConfigModel_SaveConfirmation_RefocusesForm(t *testing.T) {
	testCases := []struct {
		action func(m *ConfigModel)
		name   string
	}{
		{
			name: "esc dismissal re-focuses form",
			action: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyEscape})
			},
		},
		{
			name: "n cancel re-focuses form",
			action: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
			},
		},
		{
			name: "y confirm re-focuses form",
			action: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mockForm := &configTestFocusTrackingMockModel{}
			factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
				return mockForm
			}
			onSave := func(state *ConfigState) error { return nil }

			m := NewConfigModel(DefaultTheme(), WithFormFactory(factory), WithOnSave(onSave))
			// Focus content so the form is the active pane
			m.Update(SidebarSelectionMsg{Section: "Basics"})
			// Reset focus count after the initial focus from focusContent
			mockForm.focusCount = 0

			tc.action(m)

			assert.False(t, m.ShowSaveConfirm(), "dialog should be dismissed")
			assert.Positive(t, mockForm.focusCount, "Focus() should be called after dismissing save dialog")
		})
	}
}

func TestConfigModel_SaveConfirmTitle(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "save context shows Save configuration title",
			setup: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Save configuration?")
			},
		},
		{
			name: "quit context shows Save before quitting title",
			setup: func(m *ConfigModel) {
				m.state.MarkDirty()
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Save changes before quitting?")
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
			tc.assert(t, m)
		})
	}
}

func TestConfigModel_ValidationOverlay(t *testing.T) {
	testCases := []struct {
		setup  func(m *ConfigModel)
		assert func(t *testing.T, m *ConfigModel)
		name   string
	}{
		{
			name: "shows validation errors when form fails validation on Ctrl+S",
			setup: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.GetValidationOverlay(), "validation overlay should be shown")
				assert.False(t, m.ShowSaveConfirm(), "save confirm should not be shown")
			},
		},
		{
			name: "validation overlay view shows errors",
			setup: func(m *ConfigModel) {
				m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Validation Errors")
			},
		},
		{
			name: "validation overlay dismisses on Esc",
			setup: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				// Esc on the validation overlay generates a dismiss msg
				m.Update(tea.KeyMsg{Type: tea.KeyEscape})
				// Process the dismiss msg
				m.Update(ValidationOverlayDismissedMsg{})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Nil(t, m.GetValidationOverlay(), "overlay should be dismissed")
			},
		},
		{
			name: "validation overlay dismisses on Enter",
			setup: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyEnter})
				m.Update(ValidationOverlayDismissedMsg{})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Nil(t, m.GetValidationOverlay(), "overlay should be dismissed")
			},
		},
		{
			name: "b key does not dismiss validation overlay",
			setup: func(m *ConfigModel) {
				m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
				m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.NotNil(t, m.GetValidationOverlay(), "overlay should remain visible")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Use a mock that fails validation
			mockForm := &configTestValidatingMockModel{
				validationErrors: []string{"Field X is required"},
			}
			factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
				return mockForm
			}
			m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
			tc.setup(m)
			tc.assert(t, m)
		})
	}
}

func TestConfigModel_FlushAllForms(t *testing.T) {
	flushed := false
	mockForm := &configTestFlushingMockModel{
		onFlush: func(state *ConfigState) {
			flushed = true
			state.DefaultBranch = "flushed-value"
		},
	}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}
	onSave := func(state *ConfigState) error { return nil }

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory), WithOnSave(onSave))

	// Ctrl+S triggers flush+validate+confirm
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})

	assert.True(t, flushed, "form should have been flushed")
	assert.Equal(t, "flushed-value", m.GetState().DefaultBranch, "flush should update state")
}

func TestConfigModel_CrossSectionValidation(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		forms  map[string]tea.Model
		name   string
	}{
		{
			name: "aggregates errors from multiple sections",
			forms: map[string]tea.Model{
				"Basics": &configTestValidatingMockModel{
					validationErrors: []string{"Default Branch: required"},
				},
				"JIRA": &configTestValidatingMockModel{
					validationErrors: []string{"JIRA Host: required"},
				},
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				overlay := m.GetValidationOverlay()
				assert.NotNil(t, overlay, "validation overlay should be shown")
				errs := overlay.Errors()
				assert.Len(t, errs, 2)
			},
		},
		{
			name: "no overlay when all sections pass",
			forms: map[string]tea.Model{
				"Basics": &configTestValidatingMockModel{},
				"JIRA":   &configTestValidatingMockModel{},
			},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Nil(t, m.GetValidationOverlay(), "no overlay when valid")
				assert.True(t, m.ShowSaveConfirm(), "should show save confirmation")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
				if form, ok := tc.forms[section]; ok {
					return form
				}
				return &configTestMockModel{}
			}
			onSave := func(state *ConfigState) error { return nil }

			m := NewConfigModel(DefaultTheme(), WithFormFactory(factory), WithOnSave(onSave))

			// Visit all sections to populate cache
			for section := range tc.forms {
				m.Update(SidebarSelectionChangedMsg{Section: section})
			}

			// Trigger save flow
			m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
			tc.assert(t, m)
		})
	}
}

func TestConfigModel_QuitContext_SaveFailure(t *testing.T) {
	saveErr := errors.New("disk full")
	onSave := func(state *ConfigState) error { return saveErr }

	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory), WithOnSave(onSave))
	m.state.MarkDirty()

	// q while dirty shows quit confirmation
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	assert.True(t, m.ShowSaveConfirm())

	// "Yes" attempts save but fails - should stay with error, not quit
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	assert.Contains(t, m.GetSaveError(), "disk full")
	assert.True(t, m.IsDirty(), "dirty flag should remain set on save failure")
	// cmd should NOT be tea.Quit since save failed
	assert.Nil(t, cmd, "should not quit when save fails")
}

// configTestValidatingMockModel is a mock that implements Validator.
type configTestValidatingMockModel struct {
	validationErrors []string
}

func (m *configTestValidatingMockModel) Init() tea.Cmd                       { return nil }
func (m *configTestValidatingMockModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *configTestValidatingMockModel) View() string                        { return "mock" }
func (m *configTestValidatingMockModel) Focus() tea.Cmd                      { return nil }
func (m *configTestValidatingMockModel) Blur() tea.Cmd                       { return nil }
func (m *configTestValidatingMockModel) Validate() []string                  { return m.validationErrors }

// configTestFlushingMockModel is a mock that implements Flusher.
type configTestFlushingMockModel struct {
	onFlush func(state *ConfigState)
}

func (m *configTestFlushingMockModel) Init() tea.Cmd                       { return nil }
func (m *configTestFlushingMockModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *configTestFlushingMockModel) View() string                        { return "mock" }
func (m *configTestFlushingMockModel) Focus() tea.Cmd                      { return nil }
func (m *configTestFlushingMockModel) Blur() tea.Cmd                       { return nil }
func (m *configTestFlushingMockModel) FlushToState(state *ConfigState) {
	if m.onFlush != nil {
		m.onFlush(state)
	}
}

// configTestFocusTrackingMockModel tracks Focus() calls and returns a sentinel cmd.
type configTestFocusTrackingMockModel struct {
	focusCount int
}

func (m *configTestFocusTrackingMockModel) Init() tea.Cmd { return nil }
func (m *configTestFocusTrackingMockModel) Update(tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}
func (m *configTestFocusTrackingMockModel) View() string { return "mock" }
func (m *configTestFocusTrackingMockModel) Focus() tea.Cmd {
	m.focusCount++
	return func() tea.Msg { return nil }
}
func (m *configTestFocusTrackingMockModel) Blur() tea.Cmd { return nil }

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
