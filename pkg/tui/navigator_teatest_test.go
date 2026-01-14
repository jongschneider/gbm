package tui

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Set ASCII color profile for consistent test output across environments
	lipgloss.SetColorProfile(termenv.Ascii)
}

// =============================================================================
// TT-006: Navigator stack push/pop tests
// =============================================================================

// navigatorTestModel is a simple model for teatest that displays distinct content.
type navigatorTestModel struct {
	name    string
	initCmd tea.Cmd
}

func newNavigatorTestModel(name string) *navigatorTestModel {
	return &navigatorTestModel{name: name}
}

func (m *navigatorTestModel) Init() tea.Cmd {
	return m.initCmd
}

func (m *navigatorTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *navigatorTestModel) View() string {
	return "SCREEN_" + m.name + "\n"
}

// navigatorWrapper wraps a Navigator for teatest, handling quit commands.
type navigatorWrapper struct {
	nav *Navigator
}

func newNavigatorWrapper(nav *Navigator) *navigatorWrapper {
	return &navigatorWrapper{nav: nav}
}

func (w *navigatorWrapper) Init() tea.Cmd {
	return w.nav.Init()
}

func (w *navigatorWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Ctrl+C to quit
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyCtrlC {
		return w, tea.Quit
	}
	_, cmd := w.nav.Update(msg)
	return w, cmd
}

func (w *navigatorWrapper) View() string {
	return w.nav.View()
}

// TestNavigator_PushAddsModelToStack verifies that Push() adds a model to the stack
// and it becomes the current model in a real Bubble Tea program context.
func TestNavigator_PushAddsModelToStack(t *testing.T) {
	model1 := newNavigatorTestModel("ONE")
	nav := NewNavigator(model1)

	tm := teatest.NewTestModel(t, newNavigatorWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render showing model 1
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("SCREEN_ONE"))
	}, teatest.WithDuration(time.Second))

	// Verify initial depth
	assert.Equal(t, 1, nav.Depth(), "Navigator should have depth 1 initially")

	// Push a second model
	model2 := newNavigatorTestModel("TWO")
	nav.Push(model2)

	// Verify depth increased
	assert.Equal(t, 2, nav.Depth(), "Navigator should have depth 2 after push")

	// Give time for any pending updates
	time.Sleep(50 * time.Millisecond)

	// Verify the new model is now current
	assert.Contains(t, nav.View(), "SCREEN_TWO",
		"View() should show pushed model after Push()")
	assert.NotContains(t, nav.View(), "SCREEN_ONE",
		"View() should NOT show original model after Push()")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestNavigator_PopRemovesAndReturnsTopModel verifies that Pop() removes and returns
// the top model, revealing the previous model underneath.
func TestNavigator_PopRemovesAndReturnsTopModel(t *testing.T) {
	model1 := newNavigatorTestModel("FIRST")
	model2 := newNavigatorTestModel("SECOND")

	nav := NewNavigator(model1)
	nav.Push(model2)

	tm := teatest.NewTestModel(t, newNavigatorWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for render showing model 2 (top of stack)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("SCREEN_SECOND"))
	}, teatest.WithDuration(time.Second))

	// Verify depth before pop
	assert.Equal(t, 2, nav.Depth(), "Navigator should have depth 2 before pop")

	// Pop the top model
	popped := nav.Pop()

	// Verify popped model is correct
	assert.NotNil(t, popped, "Pop() should return the popped model")
	poppedModel := popped.(*navigatorTestModel)
	assert.Equal(t, "SECOND", poppedModel.name, "Popped model should be 'SECOND'")

	// Verify depth decreased
	assert.Equal(t, 1, nav.Depth(), "Navigator should have depth 1 after pop")

	// Verify the first model is now current
	assert.Contains(t, nav.View(), "SCREEN_FIRST",
		"View() should show first model after Pop()")
	assert.NotContains(t, nav.View(), "SCREEN_SECOND",
		"View() should NOT show popped model after Pop()")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestNavigator_DepthReflectsStackSize verifies that Depth() accurately reflects
// the current stack size through multiple push/pop operations.
func TestNavigator_DepthReflectsStackSize(t *testing.T) {
	model1 := newNavigatorTestModel("M1")
	nav := NewNavigator(model1)

	tm := teatest.NewTestModel(t, newNavigatorWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("SCREEN_M1"))
	}, teatest.WithDuration(time.Second))

	// Verify initial depth
	assert.Equal(t, 1, nav.Depth(), "Initial depth should be 1")

	// Push multiple models and verify depth
	nav.Push(newNavigatorTestModel("M2"))
	assert.Equal(t, 2, nav.Depth(), "Depth should be 2 after first push")

	nav.Push(newNavigatorTestModel("M3"))
	assert.Equal(t, 3, nav.Depth(), "Depth should be 3 after second push")

	nav.Push(newNavigatorTestModel("M4"))
	assert.Equal(t, 4, nav.Depth(), "Depth should be 4 after third push")

	// Pop and verify depth decreases
	nav.Pop()
	assert.Equal(t, 3, nav.Depth(), "Depth should be 3 after first pop")

	nav.Pop()
	assert.Equal(t, 2, nav.Depth(), "Depth should be 2 after second pop")

	nav.Pop()
	assert.Equal(t, 1, nav.Depth(), "Depth should be 1 after third pop")

	// Pop the last one
	nav.Pop()
	assert.Equal(t, 0, nav.Depth(), "Depth should be 0 after popping all")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestNavigator_CurrentReturnsTopModel verifies that Current() returns the current
// top model without removing it from the stack.
func TestNavigator_CurrentReturnsTopModel(t *testing.T) {
	model1 := newNavigatorTestModel("BASE")
	model2 := newNavigatorTestModel("TOP")

	nav := NewNavigator(model1)
	nav.Push(model2)

	tm := teatest.NewTestModel(t, newNavigatorWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for render showing top model
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("SCREEN_TOP"))
	}, teatest.WithDuration(time.Second))

	// Get current without removing
	current := nav.Current()
	assert.NotNil(t, current, "Current() should return a model")

	currentModel := current.(*navigatorTestModel)
	assert.Equal(t, "TOP", currentModel.name, "Current() should return the top model")

	// Verify depth unchanged (model not removed)
	assert.Equal(t, 2, nav.Depth(), "Depth should remain 2 after Current()")

	// Current() multiple times should return the same model
	current2 := nav.Current()
	assert.Equal(t, current, current2, "Current() should consistently return the same model")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestNavigator_EmptyStackReturnsNilFromCurrent verifies that Current() returns nil
// when the stack is empty.
func TestNavigator_EmptyStackReturnsNilFromCurrent(t *testing.T) {
	// Create navigator with nil initial (empty stack)
	nav := NewNavigator(nil)

	// Verify empty state
	assert.True(t, nav.IsEmpty(), "Navigator should be empty")
	assert.Equal(t, 0, nav.Depth(), "Depth should be 0 for empty navigator")

	// Current should return nil
	current := nav.Current()
	assert.Nil(t, current, "Current() should return nil for empty stack")

	// Pop should also return nil
	popped := nav.Pop()
	assert.Nil(t, popped, "Pop() should return nil for empty stack")

	// View should return empty string
	assert.Equal(t, "", nav.View(), "View() should return empty string for empty stack")
}

// TestNavigator_PushPopSequence verifies correct behavior through a sequence
// of push and pop operations in a real program context.
func TestNavigator_PushPopSequence(t *testing.T) {
	model1 := newNavigatorTestModel("ALPHA")
	nav := NewNavigator(model1)

	tm := teatest.NewTestModel(t, newNavigatorWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("SCREEN_ALPHA"))
	}, teatest.WithDuration(time.Second))

	// Push BETA
	nav.Push(newNavigatorTestModel("BETA"))
	assert.Contains(t, nav.View(), "SCREEN_BETA")
	assert.Equal(t, 2, nav.Depth())

	// Push GAMMA
	nav.Push(newNavigatorTestModel("GAMMA"))
	assert.Contains(t, nav.View(), "SCREEN_GAMMA")
	assert.Equal(t, 3, nav.Depth())

	// Pop GAMMA - should reveal BETA
	popped := nav.Pop()
	assert.Equal(t, "GAMMA", popped.(*navigatorTestModel).name)
	assert.Contains(t, nav.View(), "SCREEN_BETA")
	assert.Equal(t, 2, nav.Depth())

	// Pop BETA - should reveal ALPHA
	popped = nav.Pop()
	assert.Equal(t, "BETA", popped.(*navigatorTestModel).name)
	assert.Contains(t, nav.View(), "SCREEN_ALPHA")
	assert.Equal(t, 1, nav.Depth())

	// Pop ALPHA - should empty the stack
	popped = nav.Pop()
	assert.Equal(t, "ALPHA", popped.(*navigatorTestModel).name)
	assert.True(t, nav.IsEmpty())
	assert.Equal(t, 0, nav.Depth())
	assert.Equal(t, "", nav.View())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestNavigator_NilPushIsIgnored verifies that pushing nil does not affect the stack.
func TestNavigator_NilPushIsIgnored(t *testing.T) {
	model1 := newNavigatorTestModel("ONLY")
	nav := NewNavigator(model1)

	tm := teatest.NewTestModel(t, newNavigatorWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("SCREEN_ONLY"))
	}, teatest.WithDuration(time.Second))

	// Verify initial state
	assert.Equal(t, 1, nav.Depth())

	// Push nil - should be ignored
	nav.Push(nil)

	// Verify depth unchanged
	assert.Equal(t, 1, nav.Depth(), "Push(nil) should not increase depth")
	assert.Contains(t, nav.View(), "SCREEN_ONLY",
		"View() should still show original model after Push(nil)")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
