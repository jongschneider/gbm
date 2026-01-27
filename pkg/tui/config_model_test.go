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
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel(DefaultTheme())
			tc.expect(t, m)
		})
	}
}

func TestConfigModel_Accessors(t *testing.T) {
	m := NewConfigModel(DefaultTheme())

	assert.Equal(t, m.sidebar, m.GetSidebar())
	assert.Equal(t, m.nav, m.GetNavigator())
	assert.Equal(t, m.theme, m.GetTheme())
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
