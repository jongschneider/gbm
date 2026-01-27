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
			name: "creates with sidebar as initial model",
			expect: func(t *testing.T, m *ConfigModel) {
				assert.NotNil(t, m.sidebar)
				assert.NotNil(t, m.nav)
				assert.NotNil(t, m.theme)
			},
		},
		{
			name: "navigator has sidebar on stack",
			expect: func(t *testing.T, m *ConfigModel) {
				assert.Equal(t, 1, m.nav.Depth())
				assert.Equal(t, m.sidebar, m.nav.Current())
			},
		},
		{
			name: "creates with empty state by default",
			expect: func(t *testing.T, m *ConfigModel) {
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
		return nil
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))

	// Simulate sidebar selection
	m.Update(SidebarSelectionMsg{Section: "Basics"})

	assert.True(t, factoryCalled)
}

func TestConfigModel_SidebarSelection_PushesForm(t *testing.T) {
	// Create a simple mock form
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))
	assert.Equal(t, 1, m.nav.Depth())

	// Simulate sidebar selection
	m.Update(SidebarSelectionMsg{Section: "Basics"})

	// Form should be pushed onto navigator
	assert.Equal(t, 2, m.nav.Depth())
	assert.Equal(t, mockForm, m.nav.Current())
}

func TestConfigModel_BackBoundary_PopsForm(t *testing.T) {
	mockForm := &configTestMockModel{}
	factory := func(section string, state *ConfigState, theme *Theme, onUpdate func()) tea.Model {
		return mockForm
	}

	m := NewConfigModel(DefaultTheme(), WithFormFactory(factory))

	// Push form
	m.Update(SidebarSelectionMsg{Section: "Basics"})
	assert.Equal(t, 2, m.nav.Depth())

	// Back should pop
	m.Update(BackBoundaryMsg{})
	assert.Equal(t, 1, m.nav.Depth())
	assert.Equal(t, m.sidebar, m.nav.Current())
}

func TestConfigModel_DirtyState(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	assert.False(t, m.IsDirty())

	// Manually mark as dirty
	m.state.dirty = true
	assert.True(t, m.IsDirty())
}

func TestConfigModel_Reset(t *testing.T) {
	resetCalled := false
	onReset := func() (*ConfigState, error) {
		resetCalled = true
		return &ConfigState{DefaultBranch: "main"}, nil
	}

	m := NewConfigModel(DefaultTheme(), WithOnReset(onReset))
	m.state.DefaultBranch = "develop"
	m.state.dirty = true

	// Simulate 'r' key at sidebar level
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	assert.True(t, resetCalled)
	assert.Equal(t, "main", m.state.DefaultBranch)
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
	assert.Equal(t, m.nav, m.GetNavigator())
	assert.Equal(t, m.theme, m.GetTheme())
	assert.NotNil(t, m.GetState())
}

func TestConfigModel_WindowSize(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	m.Update(sizeMsg)

	assert.Equal(t, 100, m.width)
	assert.Equal(t, 50, m.height)
	assert.Equal(t, 100, m.sidebar.width)
	assert.Equal(t, 50, m.sidebar.height)
}

func TestConfigModel_View(t *testing.T) {
	m := NewConfigModel(DefaultTheme())
	view := m.View()

	// Should render sidebar initially
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Basics")
}

// configTestMockModel is a simple mock tea.Model for testing.
type configTestMockModel struct{}

func (m *configTestMockModel) Init() tea.Cmd                       { return nil }
func (m *configTestMockModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *configTestMockModel) View() string                        { return "mock" }
