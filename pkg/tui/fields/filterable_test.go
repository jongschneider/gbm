package fields

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFilterable_AsyncOptions(t *testing.T) {
	testCases := []struct {
		name        string
		optionsFunc func() ([]Option, error)
		expect      func(t *testing.T, f *Filterable)
		expectErr   func(t *testing.T, err error)
	}{
		{
			name: "load async options successfully",
			optionsFunc: func() ([]Option, error) {
				return []Option{
					{Label: "Option A", Value: "a"},
					{Label: "Option B", Value: "b"},
				}, nil
			},
			expect: func(t *testing.T, f *Filterable) {
				assert.Equal(t, 2, len(f.options))
				assert.Equal(t, "Option A", f.options[0].Label)
				assert.Equal(t, "Option B", f.options[1].Label)
			},
			expectErr: nil,
		},
		{
			name: "async options loading error",
			optionsFunc: func() ([]Option, error) {
				return nil, errors.New("API error")
			},
			expect: func(t *testing.T, f *Filterable) {
				// Options should be empty due to error
				assert.Equal(t, 0, len(f.options))
			},
			expectErr: func(t *testing.T, err error) {
				assert.NotNil(t, err)
				assert.Equal(t, "API error", err.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFilterable("test", "Select", "", []Option{})
			f.WithOptionsFunc(tc.optionsFunc)

			// Simulate loading
			opts, err := f.optionsFunc.Get()
			f.options = opts
			if err == nil && len(opts) > 0 {
				f.filterOptions()
			}

			if tc.expectErr != nil {
				tc.expectErr(t, err)
			}
			if tc.expect != nil {
				tc.expect(t, f)
			}
		})
	}
}

func TestFilterable_WithOptionsFunc(t *testing.T) {
	testCases := []struct {
		name         string
		optionsFunc  func() ([]Option, error)
		expectLoaded bool
	}{
		{
			name: "options func is stored and can be queried",
			optionsFunc: func() ([]Option, error) {
				return []Option{{Label: "Test", Value: "test"}}, nil
			},
			expectLoaded: false, // Not loaded until Get() is called
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFilterable("key", "title", "desc", []Option{})
			result := f.WithOptionsFunc(tc.optionsFunc)

			// Should return self for chaining
			assert.Equal(t, f, result)

			// Should have optionsFunc set
			assert.NotNil(t, f.optionsFunc)

			// Should not be loaded yet
			assert.Equal(t, tc.expectLoaded, f.optionsFunc.IsLoaded())
		})
	}
}

func TestFilterable_AsyncOptionsIntegration(t *testing.T) {
	testCases := []struct {
		name        string
		optionsFunc func() ([]Option, error)
		expect      func(t *testing.T, f *Filterable)
	}{
		{
			name: "async options with filtering",
			optionsFunc: func() ([]Option, error) {
				time.Sleep(10 * time.Millisecond) // Simulate network latency
				return []Option{
					{Label: "JIRA-123", Value: "jira-123"},
					{Label: "JIRA-456", Value: "jira-456"},
					{Label: "Other-789", Value: "other-789"},
				}, nil
			},
			expect: func(t *testing.T, f *Filterable) {
				// Load options
				opts, err := f.optionsFunc.Get()
				assert.NoError(t, err)
				assert.Equal(t, 3, len(opts))

				// Apply to filterable and simulate typing "JIRA"
				f.options = opts
				f.textInput.SetValue("JIRA")
				f.filterOptions()

				// Should filter to 2 items
				assert.Equal(t, 2, len(f.filtered))
				assert.Equal(t, "JIRA-123", f.filtered[0].Label)
				assert.Equal(t, "JIRA-456", f.filtered[1].Label)
			},
		},
		{
			name: "async options are cached",
			optionsFunc: func() ([]Option, error) {
				return []Option{
					{Label: "Cached", Value: "cached"},
				}, nil
			},
			expect: func(t *testing.T, f *Filterable) {
				// First call
				opts1, err1 := f.optionsFunc.Get()
				assert.NoError(t, err1)
				assert.Equal(t, 1, len(opts1))

				// Second call should return same cached value
				opts2, err2 := f.optionsFunc.Get()
				assert.NoError(t, err2)
				assert.Equal(t, 1, len(opts2))

				// Should be same value
				assert.Equal(t, opts1[0].Value, opts2[0].Value)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFilterable("key", "title", "desc", []Option{})
			f.WithOptionsFunc(tc.optionsFunc)

			tc.expect(t, f)
		})
	}
}

func TestFilterable_SpinnerInitialization(t *testing.T) {
	f := NewFilterable("key", "title", "desc", []Option{})
	assert.NotNil(t, f.spinner)
	assert.NotEmpty(t, f.spinner.View())
}

func TestFilterable_ViewWithAsyncError(t *testing.T) {
	f := NewFilterable("key", "title", "desc", []Option{})
	f.focused = true
	f.asyncErr = fmt.Errorf("network error")

	view := f.View()
	assert.Contains(t, view, "Error loading options")
	assert.Contains(t, view, "network error")
}

func TestFilterable_SpinnerDisplayDuringAsyncLoad(t *testing.T) {
	// Create a Filterable with async options that have a delay
	f := NewFilterable("key", "title", "desc", []Option{})
	f.WithOptionsFunc(func() ([]Option, error) {
		time.Sleep(100 * time.Millisecond) // Simulate network delay
		return []Option{
			{Label: "Option A", Value: "a"},
		}, nil
	})

	// Before loading, optionsFunc exists but hasn't been called yet
	assert.NotNil(t, f.optionsFunc)
	assert.False(t, f.optionsFunc.IsLoaded())

	// View should show loading spinner
	f.focused = true
	view := f.View()
	assert.Contains(t, view, "Loading options")
	assert.NotEmpty(t, f.spinner.View())
}
