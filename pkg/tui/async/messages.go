// Package async provides utilities for asynchronous operations in TUI with caching.
package async

import tea "github.com/charmbracelet/bubbletea"

// FetchMsg represents completion of an async fetch operation.
// It carries the result (Value) and any error (Err) that occurred during the operation.
// This message is used to communicate back to the UI model that an async operation has completed.
type FetchMsg[T any] struct {
	Value T
	Err   error
}

// FetchCmd returns a command that fetches a value asynchronously without blocking the event loop.
// The provided fetch function is executed in a separate goroutine, and its result is returned
// as a FetchMsg[T] to the model's Update method.
//
// Usage:
//
//	func (f *MyField) Init() tea.Cmd {
//	    return async.FetchCmd(func() ([]string, error) {
//	        return f.fetchOptions()
//	    })
//	}
//
//	func (f *MyField) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
//	    switch msg := msg.(type) {
//	    case async.FetchMsg[[]string]:
//	        if msg.Err != nil {
//	            f.loadErr = msg.Err
//	        } else {
//	            f.options = msg.Value
//	        }
//	        f.isLoading = false
//	        return f, nil
//	    }
//	    return f, nil
//	}
func FetchCmd[T any](fetch func() (T, error)) tea.Cmd {
	return func() tea.Msg {
		value, err := fetch()
		return FetchMsg[T]{Value: value, Err: err}
	}
}
