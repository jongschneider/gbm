// Package tui provides terminal user interface components for wizard-based workflows.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// NavigateMsg is sent to transition between different screens/models in the navigator.
// The target should be pushed onto the navigation stack.
type NavigateMsg struct {
	target tea.Model
}

// NewNavigateMsg creates a new navigation message for transitioning to a target model.
func NewNavigateMsg(target tea.Model) NavigateMsg {
	return NavigateMsg{target: target}
}

// Navigator is a root model that manages navigation between multiple screens.
// It maintains a stack of models and delegates Init/Update/View to the current top model.
// This enables multi-screen workflows without requiring complex wrapper models.
//
// Usage:
//
//	initialModel := MyFirstScreen()
//	nav := NewNavigator(initialModel)
//	if _, err := tea.NewProgram(nav).Run(); err != nil {
//	    log.Fatal(err)
//	}
type Navigator struct {
	stack  []tea.Model
	width  int
	height int
}

// NewNavigator creates a new Navigator with the given initial model.
// The initial model becomes the first item on the stack.
// Returns an error if the initial model is nil.
func NewNavigator(initial tea.Model) *Navigator {
	if initial == nil {
		return &Navigator{stack: []tea.Model{}}
	}
	return &Navigator{
		stack: []tea.Model{initial},
	}
}

// Init implements tea.Model.Init.
// It delegates to the current model on top of the stack, or returns nil if the stack is empty.
func (n *Navigator) Init() tea.Cmd {
	if len(n.stack) == 0 {
		return nil
	}
	return n.current().Init()
}

// Update implements tea.Model.Update.
// It delegates to the current model on top of the stack.
// If the model returns a NavigateMsg, it pushes the target onto the stack.
// Returns nil if the stack is empty.
func (n *Navigator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if len(n.stack) == 0 {
		return n, nil
	}

	current := n.current()
	newModel, cmd := current.Update(msg)

	// Replace the current model with the updated one
	n.stack[len(n.stack)-1] = newModel

	// Check if the command returns a NavigateMsg
	if cmd != nil {
		return n, func() tea.Msg {
			result := cmd()
			if navMsg, ok := result.(NavigateMsg); ok {
				n.push(navMsg.target)
				// Return a command that delegates to the newly pushed model's Init
				return navMsg.target.Init()
			}
			return result
		}
	}

	return n, cmd
}

// View implements tea.Model.View.
// It delegates to the current model on top of the stack.
// Returns an empty string if the stack is empty.
func (n *Navigator) View() string {
	if len(n.stack) == 0 {
		return ""
	}
	return n.current().View()
}

// Push adds a new model to the top of the stack.
// The new model becomes the current model and will receive Update and View calls.
func (n *Navigator) Push(model tea.Model) {
	n.push(model)
}

// Pop removes the current model from the stack.
// Returns the removed model, or nil if the stack is empty.
// If there's only one model on the stack, popping it will leave the stack empty
// and View() will return an empty string.
func (n *Navigator) Pop() tea.Model {
	if len(n.stack) == 0 {
		return nil
	}
	model := n.stack[len(n.stack)-1]
	n.stack = n.stack[:len(n.stack)-1]
	return model
}

// SetSize sets the terminal width and height.
// This is called by the Bubble Tea runtime and should be propagated to models if needed.
func (n *Navigator) SetSize(width, height int) {
	n.width = width
	n.height = height
	// Optionally propagate to current model if it supports window size
}

// Depth returns the current depth of the navigation stack.
// Useful for debugging or determining if we're at the root level (depth == 1).
func (n *Navigator) Depth() int {
	return len(n.stack)
}

// Current returns the current model on top of the stack.
// Returns nil if the stack is empty.
func (n *Navigator) Current() tea.Model {
	if len(n.stack) == 0 {
		return nil
	}
	return n.stack[len(n.stack)-1]
}

// current returns the current model on top of the stack.
// Assumes the stack is not empty. (private helper)
func (n *Navigator) current() tea.Model {
	return n.Current()
}

// push adds a model to the stack (private method).
func (n *Navigator) push(model tea.Model) {
	if model == nil {
		return
	}
	n.stack = append(n.stack, model)
}

// HandleWindowSizeMsg processes a window size message and optionally propagates it.
// This can be called in Update() if the model needs to respond to terminal resizing.
func (n *Navigator) HandleWindowSizeMsg(width, height int) {
	n.SetSize(width, height)
}

// IsEmpty returns true if the navigation stack is empty.
func (n *Navigator) IsEmpty() bool {
	return len(n.stack) == 0
}

// String returns a string representation of the navigator state for debugging.
func (n *Navigator) String() string {
	return fmt.Sprintf("Navigator(depth=%d, width=%d, height=%d)", len(n.stack), n.width, n.height)
}
