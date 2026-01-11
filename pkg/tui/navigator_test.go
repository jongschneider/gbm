package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// mockModel is a simple model for testing navigator
type mockModel struct {
	name     string
	updated  bool
	viewed   bool
	initCmd  tea.Cmd
}

func (m *mockModel) Init() tea.Cmd {
	return m.initCmd
}

func (m *mockModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.updated = true
	return m, nil
}

func (m *mockModel) View() string {
	m.viewed = true
	return "Model: " + m.name
}

func TestNewNavigator(t *testing.T) {
	testCases := []struct {
		name        string
		initial     tea.Model
		expect      func(t *testing.T, nav *Navigator)
		description string
	}{
		{
			name:    "creates navigator with initial model",
			initial: &mockModel{name: "initial"},
			expect: func(t *testing.T, nav *Navigator) {
				assert.NotNil(t, nav, "navigator should be created")
				assert.Equal(t, 1, nav.Depth(), "should have one model on stack")
				assert.False(t, nav.IsEmpty(), "should not be empty")
			},
			description: "NewNavigator should add initial model to stack",
		},
		{
			name:    "creates navigator with nil initial",
			initial: nil,
			expect: func(t *testing.T, nav *Navigator) {
				assert.NotNil(t, nav, "navigator should be created")
				assert.Equal(t, 0, nav.Depth(), "should have empty stack")
				assert.True(t, nav.IsEmpty(), "should be empty")
			},
			description: "NewNavigator should handle nil initial model",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nav := NewNavigator(tc.initial)
			tc.expect(t, nav)
		})
	}
}

func TestNavigator_Init(t *testing.T) {
	testCases := []struct {
		name        string
		initial     tea.Model
		expect      func(t *testing.T, cmd tea.Cmd)
		description string
	}{
		{
			name: "delegates Init to current model",
			initial: &mockModel{
				name:    "model1",
				initCmd: func() tea.Msg { return nil },
			},
			expect: func(t *testing.T, cmd tea.Cmd) {
				assert.NotNil(t, cmd, "should return command from model")
			},
			description: "Init should delegate to current model",
		},
		{
			name:    "returns nil for empty stack",
			initial: nil,
			expect: func(t *testing.T, cmd tea.Cmd) {
				assert.Nil(t, cmd, "should return nil for empty stack")
			},
			description: "Init should return nil when stack is empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nav := NewNavigator(tc.initial)
			cmd := nav.Init()
			tc.expect(t, cmd)
		})
	}
}

func TestNavigator_Update(t *testing.T) {
	t.Run("delegates Update to current model", func(t *testing.T) {
		initial := &mockModel{name: "model1"}
		nav := NewNavigator(initial)

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		_, cmd := nav.Update(msg)

		assert.True(t, initial.updated, "current model should be updated")
		assert.Nil(t, cmd, "should return nil command when no navigation")
	})

	t.Run("Push adds model to navigation", func(t *testing.T) {
		initial := &mockModel{name: "model1"}
		nav := NewNavigator(initial)

		next := &mockModel{name: "model2"}
		nav.Push(next)

		assert.Equal(t, 2, nav.Depth(), "should have two models on stack after navigation")
		assert.Contains(t, nav.View(), "model2", "should display the pushed model")
	})

	t.Run("returns nil for empty stack", func(t *testing.T) {
		nav := NewNavigator(nil)
		_, cmd := nav.Update(tea.KeyMsg{})

		assert.Nil(t, cmd, "should return nil for empty stack")
	})
}

func TestNavigator_View(t *testing.T) {
	testCases := []struct {
		name        string
		initial     tea.Model
		expect      func(t *testing.T, view string)
		description string
	}{
		{
			name:    "delegates View to current model",
			initial: &mockModel{name: "test"},
			expect: func(t *testing.T, view string) {
				assert.Contains(t, view, "Model: test", "should render current model's view")
			},
			description: "View should delegate to current model",
		},
		{
			name:    "returns empty for empty stack",
			initial: nil,
			expect: func(t *testing.T, view string) {
				assert.Equal(t, "", view, "should return empty string for empty stack")
			},
			description: "View should return empty string when stack is empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nav := NewNavigator(tc.initial)
			view := nav.View()
			tc.expect(t, view)
		})
	}
}

func TestNavigator_Push(t *testing.T) {
	t.Run("adds model to stack", func(t *testing.T) {
		initial := &mockModel{name: "model1"}
		nav := NewNavigator(initial)
		assert.Equal(t, 1, nav.Depth())

		second := &mockModel{name: "model2"}
		nav.Push(second)

		assert.Equal(t, 2, nav.Depth(), "should have two models after push")
		assert.Equal(t, "Model: model2", nav.View(), "should display second model after push")
	})

	t.Run("ignores nil push", func(t *testing.T) {
		initial := &mockModel{name: "model1"}
		nav := NewNavigator(initial)

		nav.Push(nil)

		assert.Equal(t, 1, nav.Depth(), "should still have one model after nil push")
	})
}

func TestNavigator_Pop(t *testing.T) {
	testCases := []struct {
		name        string
		setup       func() *Navigator
		expect      func(t *testing.T, popped tea.Model, nav *Navigator)
		description string
	}{
		{
			name: "pops model from stack",
			setup: func() *Navigator {
				nav := NewNavigator(&mockModel{name: "model1"})
				nav.Push(&mockModel{name: "model2"})
				return nav
			},
			expect: func(t *testing.T, popped tea.Model, nav *Navigator) {
				assert.NotNil(t, popped, "should return popped model")
				mockModel := popped.(*mockModel)
				assert.Equal(t, "model2", mockModel.name, "should pop the last added model")
				assert.Equal(t, 1, nav.Depth(), "should have one model after pop")
			},
			description: "Pop should remove and return top model",
		},
		{
			name: "returns nil for empty stack",
			setup: func() *Navigator {
				return NewNavigator(nil)
			},
			expect: func(t *testing.T, popped tea.Model, nav *Navigator) {
				assert.Nil(t, popped, "should return nil for empty stack")
				assert.Equal(t, 0, nav.Depth(), "depth should remain 0")
			},
			description: "Pop should return nil when stack is empty",
		},
		{
			name: "can empty entire stack",
			setup: func() *Navigator {
				nav := NewNavigator(&mockModel{name: "model1"})
				nav.Push(&mockModel{name: "model2"})
				return nav
			},
			expect: func(t *testing.T, popped tea.Model, nav *Navigator) {
				nav.Pop() // Pop second
				nav.Pop() // Pop first
				assert.Equal(t, 0, nav.Depth(), "should be empty after popping all")
				assert.True(t, nav.IsEmpty(), "should report as empty")
			},
			description: "Pop should allow emptying the entire stack",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nav := tc.setup()
			popped := nav.Pop()
			tc.expect(t, popped, nav)
		})
	}
}

func TestNavigator_Depth(t *testing.T) {
	nav := NewNavigator(&mockModel{name: "model1"})
	assert.Equal(t, 1, nav.Depth())

	nav.Push(&mockModel{name: "model2"})
	assert.Equal(t, 2, nav.Depth())

	nav.Push(&mockModel{name: "model3"})
	assert.Equal(t, 3, nav.Depth())

	nav.Pop()
	assert.Equal(t, 2, nav.Depth())
}

func TestNavigator_IsEmpty(t *testing.T) {
	testCases := []struct {
		name     string
		isEmpty  bool
		setup    func() *Navigator
		description string
	}{
		{
			name:    "empty navigator",
			isEmpty: true,
			setup: func() *Navigator {
				return NewNavigator(nil)
			},
			description: "IsEmpty should return true for empty stack",
		},
		{
			name:    "non-empty navigator",
			isEmpty: false,
			setup: func() *Navigator {
				return NewNavigator(&mockModel{name: "model1"})
			},
			description: "IsEmpty should return false when stack has models",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nav := tc.setup()
			assert.Equal(t, tc.isEmpty, nav.IsEmpty(), tc.description)
		})
	}
}

func TestNavigator_SetSize(t *testing.T) {
	nav := NewNavigator(&mockModel{name: "model1"})

	nav.SetSize(120, 30)

	assert.Equal(t, 120, nav.width, "should set width")
	assert.Equal(t, 30, nav.height, "should set height")
}

func TestNavigator_HandleWindowSizeMsg(t *testing.T) {
	nav := NewNavigator(&mockModel{name: "model1"})

	nav.HandleWindowSizeMsg(100, 25)

	assert.Equal(t, 100, nav.width, "should handle window size message")
	assert.Equal(t, 25, nav.height, "should handle window size message")
}

func TestNavigator_String(t *testing.T) {
	nav := NewNavigator(&mockModel{name: "model1"})
	nav.Push(&mockModel{name: "model2"})

	str := nav.String()

	assert.Contains(t, str, "Navigator", "should contain Navigator in string")
	assert.Contains(t, str, "depth=2", "should show correct depth")
}

func TestNavigator_MultipleNavigations(t *testing.T) {
	t.Run("navigate through multiple screens", func(t *testing.T) {
		screen1 := &mockModel{name: "screen1"}
		nav := NewNavigator(screen1)

		screen2 := &mockModel{name: "screen2"}
		nav.Push(screen2)
		assert.Equal(t, "Model: screen2", nav.View())

		screen3 := &mockModel{name: "screen3"}
		nav.Push(screen3)
		assert.Equal(t, "Model: screen3", nav.View())

		nav.Pop()
		assert.Equal(t, "Model: screen2", nav.View())

		nav.Pop()
		assert.Equal(t, "Model: screen1", nav.View())

		nav.Pop()
		assert.True(t, nav.IsEmpty())
	})
}

func TestNavigator_NodesRenderCorrectly(t *testing.T) {
	t.Run("each model's view is rendered when active", func(t *testing.T) {
		model1 := &mockModel{name: "first"}
		model2 := &mockModel{name: "second"}
		model3 := &mockModel{name: "third"}

		nav := NewNavigator(model1)
		assert.Contains(t, nav.View(), "first")

		nav.Push(model2)
		assert.Contains(t, nav.View(), "second")
		assert.NotContains(t, nav.View(), "first")

		nav.Push(model3)
		assert.Contains(t, nav.View(), "third")
		assert.NotContains(t, nav.View(), "second")
		assert.NotContains(t, nav.View(), "first")
	})
}
