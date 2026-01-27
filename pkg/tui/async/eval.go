// Package async provides utilities for asynchronous operations in TUI with caching.
package async

import (
	"sync"
)

// Eval represents an async evaluation with caching.
// It lazy-loads a value using a fetch function and caches the result.
type Eval[T any] struct {
	value   T
	err     error
	fetch   func() (T, error)
	mu      sync.RWMutex
	loaded  bool
	loading bool
}

// New creates a new Eval that will fetch a value using the provided function.
func New[T any](fetch func() (T, error)) *Eval[T] {
	return &Eval[T]{
		fetch: fetch,
	}
}

// Get retrieves the cached value, fetching it if not already loaded.
// If the value is already loaded, it returns the cached value immediately.
// If currently loading, it returns an error indicating the load is in progress.
// On first load or after Invalidate(), this function blocks until the fetch completes.
func (e *Eval[T]) Get() (T, error) {
	e.mu.Lock()

	// If already loaded, return cached value
	if e.loaded {
		defer e.mu.Unlock()
		return e.value, e.err
	}

	// If currently loading, return error
	if e.loading {
		defer e.mu.Unlock()
		var zero T
		return zero, ErrLoading
	}

	// Mark as loading
	e.loading = true
	e.mu.Unlock()

	// Fetch the value (outside lock to avoid blocking)
	value, err := e.fetch()

	e.mu.Lock()
	defer e.mu.Unlock()

	// Store result
	e.value = value
	e.err = err
	e.loaded = true
	e.loading = false

	return value, err
}

// IsLoaded returns true if the value has been successfully fetched and cached.
func (e *Eval[T]) IsLoaded() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.loaded && e.err == nil
}

// IsLoading returns true if a fetch operation is currently in progress.
func (e *Eval[T]) IsLoading() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.loading
}

// Invalidate clears the cached value, forcing a fresh fetch on the next Get() call.
func (e *Eval[T]) Invalidate() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.loaded = false
	e.loading = false
	var zero T
	e.value = zero
	e.err = nil
}
