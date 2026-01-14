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
	time.Sleep(10 * time.Millisecond)

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

// =============================================================================
// TT-007: Navigator message-based navigation tests
// =============================================================================

// navigatorMsgTestModel is a model that can trigger NavigateMsg via a command.
// It responds to specific key presses to trigger navigation.
type navigatorMsgTestModel struct {
	name        string
	initCalled  bool
	navigateTo  tea.Model
	initCommand tea.Cmd
}

func newNavigatorMsgTestModel(name string) *navigatorMsgTestModel {
	return &navigatorMsgTestModel{name: name}
}

func (m *navigatorMsgTestModel) WithNavigateTarget(target tea.Model) *navigatorMsgTestModel {
	m.navigateTo = target
	return m
}

func (m *navigatorMsgTestModel) WithInitCommand(cmd tea.Cmd) *navigatorMsgTestModel {
	m.initCommand = cmd
	return m
}

func (m *navigatorMsgTestModel) Init() tea.Cmd {
	m.initCalled = true
	return m.initCommand
}

func (m *navigatorMsgTestModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		// 'n' key triggers navigation to the target model
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 && keyMsg.Runes[0] == 'n' {
			if m.navigateTo != nil {
				return m, func() tea.Msg {
					return NewNavigateMsg(m.navigateTo)
				}
			}
		}
	}
	return m, nil
}

func (m *navigatorMsgTestModel) View() string {
	return "MSG_SCREEN_" + m.name + "\n"
}

// navigatorMsgWrapper wraps Navigator for testing NavigateMsg handling.
type navigatorMsgWrapper struct {
	nav *Navigator
}

func newNavigatorMsgWrapper(nav *Navigator) *navigatorMsgWrapper {
	return &navigatorMsgWrapper{nav: nav}
}

func (w *navigatorMsgWrapper) Init() tea.Cmd {
	return w.nav.Init()
}

func (w *navigatorMsgWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Ctrl+C to quit
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyCtrlC {
		return w, tea.Quit
	}
	_, cmd := w.nav.Update(msg)
	return w, cmd
}

func (w *navigatorMsgWrapper) View() string {
	return w.nav.View()
}

// TestNavigator_NavigateMsgPushesTargetOntoStack verifies that NavigateMsg pushes
// the target model onto the navigation stack.
func TestNavigator_NavigateMsgPushesTargetOntoStack(t *testing.T) {
	targetModel := newNavigatorMsgTestModel("TARGET")
	initialModel := newNavigatorMsgTestModel("INITIAL").WithNavigateTarget(targetModel)
	nav := NewNavigator(initialModel)

	tm := teatest.NewTestModel(t, newNavigatorMsgWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("MSG_SCREEN_INITIAL"))
	}, teatest.WithDuration(time.Second))

	// Verify initial state
	assert.Equal(t, 1, nav.Depth(), "Initial depth should be 1")
	assert.Contains(t, nav.View(), "MSG_SCREEN_INITIAL")

	// Send 'n' to trigger navigation
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Wait for navigation to complete - look for target screen
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("MSG_SCREEN_TARGET"))
	}, teatest.WithDuration(time.Second))

	// Verify target is now on stack
	assert.Equal(t, 2, nav.Depth(), "Depth should be 2 after navigation")
	assert.Contains(t, nav.View(), "MSG_SCREEN_TARGET",
		"View() should delegate to newly pushed target model")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// initTrackerMsg is sent by initTrackerModel to verify Init() was called.
type initTrackerMsg struct {
	modelName string
}

// initTrackerModel tracks whether Init() was called via a message.
type initTrackerModel struct {
	name       string
	initCalled bool
}

func newInitTrackerModel(name string) *initTrackerModel {
	return &initTrackerModel{name: name}
}

func (m *initTrackerModel) Init() tea.Cmd {
	m.initCalled = true
	return func() tea.Msg {
		return initTrackerMsg{modelName: m.name}
	}
}

func (m *initTrackerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *initTrackerModel) View() string {
	return "TRACKER_" + m.name + "\n"
}

// TestNavigator_TargetInitIsCalledAfterPush verifies that the target model's Init()
// is called after it is pushed onto the stack via NavigateMsg.
func TestNavigator_TargetInitIsCalledAfterPush(t *testing.T) {
	targetModel := newInitTrackerModel("TARGET")
	initialModel := newNavigatorMsgTestModel("INITIAL").WithNavigateTarget(targetModel)
	nav := NewNavigator(initialModel)

	tm := teatest.NewTestModel(t, newNavigatorMsgWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("MSG_SCREEN_INITIAL"))
	}, teatest.WithDuration(time.Second))

	// Verify target Init() not called yet
	assert.False(t, targetModel.initCalled, "Target Init() should not be called before navigation")

	// Send 'n' to trigger navigation
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Wait for the target model to appear
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("TRACKER_TARGET"))
	}, teatest.WithDuration(time.Second))

	// Verify target Init() was called
	assert.True(t, targetModel.initCalled, "Target Init() should be called after navigation")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestNavigator_ViewDelegatesToNewlyPushedModel verifies that View() delegates
// to the newly pushed model after NavigateMsg.
func TestNavigator_ViewDelegatesToNewlyPushedModel(t *testing.T) {
	targetModel := newNavigatorMsgTestModel("SECOND_SCREEN")
	initialModel := newNavigatorMsgTestModel("FIRST_SCREEN").WithNavigateTarget(targetModel)
	nav := NewNavigator(initialModel)

	tm := teatest.NewTestModel(t, newNavigatorMsgWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("MSG_SCREEN_FIRST_SCREEN"))
	}, teatest.WithDuration(time.Second))

	// Verify initial view
	assert.Contains(t, nav.View(), "MSG_SCREEN_FIRST_SCREEN")
	assert.NotContains(t, nav.View(), "MSG_SCREEN_SECOND_SCREEN")

	// Send 'n' to trigger navigation
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Wait for navigation to complete
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("MSG_SCREEN_SECOND_SCREEN"))
	}, teatest.WithDuration(time.Second))

	// Verify view now shows target model
	assert.Contains(t, nav.View(), "MSG_SCREEN_SECOND_SCREEN",
		"View() should show newly pushed model")
	assert.NotContains(t, nav.View(), "MSG_SCREEN_FIRST_SCREEN",
		"View() should NOT show the previous model")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// updateDelegatingModel is a model that tracks whether Update() receives messages.
type updateDelegatingModel struct {
	name            string
	lastMsg         tea.Msg
	updateCallCount int
	navigateTo      tea.Model
}

func newUpdateDelegatingModel(name string) *updateDelegatingModel {
	return &updateDelegatingModel{name: name}
}

func (m *updateDelegatingModel) WithNavigateTarget(target tea.Model) *updateDelegatingModel {
	m.navigateTo = target
	return m
}

func (m *updateDelegatingModel) Init() tea.Cmd {
	return nil
}

func (m *updateDelegatingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.updateCallCount++
	m.lastMsg = msg

	// 'n' key triggers navigation
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		if keyMsg.Type == tea.KeyRunes && len(keyMsg.Runes) > 0 && keyMsg.Runes[0] == 'n' {
			if m.navigateTo != nil {
				return m, func() tea.Msg {
					return NewNavigateMsg(m.navigateTo)
				}
			}
		}
	}
	return m, nil
}

func (m *updateDelegatingModel) View() string {
	return "UPDATE_MODEL_" + m.name + "\n"
}

// TestNavigator_UpdateDelegatesToCurrentModel verifies that Update() delegates
// messages to the current (top) model on the stack.
func TestNavigator_UpdateDelegatesToCurrentModel(t *testing.T) {
	targetModel := newUpdateDelegatingModel("TARGET")
	initialModel := newUpdateDelegatingModel("INITIAL").WithNavigateTarget(targetModel)
	nav := NewNavigator(initialModel)

	tm := teatest.NewTestModel(t, newNavigatorMsgWrapper(nav), teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("UPDATE_MODEL_INITIAL"))
	}, teatest.WithDuration(time.Second))

	// Reset call count after init
	initialModel.updateCallCount = 0
	targetModel.updateCallCount = 0

	// Send a message to initial model
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// Give time for update to process
	time.Sleep(20 * time.Millisecond)

	// Verify initial model received the message
	assert.Greater(t, initialModel.updateCallCount, 0, "Initial model should receive Update() calls")
	assert.Equal(t, 0, targetModel.updateCallCount, "Target model should not receive Update() calls yet")

	// Navigate to target
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	// Wait for navigation
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("UPDATE_MODEL_TARGET"))
	}, teatest.WithDuration(time.Second))

	// Reset counters
	initialModel.updateCallCount = 0
	targetModel.updateCallCount = 0

	// Send a message - should now go to target
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Give time for update to process
	time.Sleep(20 * time.Millisecond)

	// Verify target model receives the message now
	assert.Greater(t, targetModel.updateCallCount, 0, "Target model should receive Update() calls after navigation")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
