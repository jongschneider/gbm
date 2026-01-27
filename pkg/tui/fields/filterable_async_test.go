package fields

import (
	"errors"
	"gbm/pkg/tui/async"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterable_Init_WithAsyncOptions(t *testing.T) {
	testCases := []struct {
		name         string
		optionsFetch func() ([]Option, error)
		expect       func(t *testing.T, cmd tea.Cmd)
		description  string
	}{
		{
			name: "Init returns FetchCmd when optionsFetch is set",
			optionsFetch: func() ([]Option, error) {
				return []Option{{Label: "A", Value: "a"}}, nil
			},
			expect: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assert.NotNil(t, cmd, "Init should return a command when async options are configured")
				// Execute the command to verify it returns a FetchMsg
				msg := cmd()
				_, ok := msg.(async.FetchMsg[[]Option])
				assert.True(t, ok, "command should return FetchMsg[[]Option]")
			},
			description: "Init should return FetchCmd for async loading",
		},
		{
			name:         "Init returns textinput.Blink when no async options",
			optionsFetch: nil,
			expect: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				assert.NotNil(t, cmd, "Init should return a command (textinput.Blink)")
			},
			description: "Init should return textinput.Blink for static options",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFilterable("key", "title", "desc", []Option{})
			if tc.optionsFetch != nil {
				f.WithOptionsFuncAsync(tc.optionsFetch)
			}

			cmd := f.Init()
			tc.expect(t, cmd)
		})
	}
}

func TestFilterable_Update_HandlesFetchMsg(t *testing.T) {
	testCases := []struct {
		name        string
		fetchMsg    async.FetchMsg[[]Option]
		expect      func(t *testing.T, f *Filterable)
		expectErr   func(t *testing.T, err error)
		description string
	}{
		{
			name: "FetchMsg with successful data updates options",
			fetchMsg: async.FetchMsg[[]Option]{
				Value: []Option{
					{Label: "Option A", Value: "a"},
					{Label: "Option B", Value: "b"},
				},
				Err: nil,
			},
			expect: func(t *testing.T, f *Filterable) {
				t.Helper()
				assert.False(t, f.isLoading, "should not be loading after FetchMsg")
				assert.Len(t, f.options, 2, "options should be updated")
				assert.Equal(t, "a", f.options[0].Value)
				assert.Equal(t, "b", f.options[1].Value)
				assert.NoError(t, f.loadErr)
			},
			description: "successful fetch should populate options",
		},
		{
			name: "FetchMsg with error sets loadErr",
			fetchMsg: async.FetchMsg[[]Option]{
				Value: nil,
				Err:   errors.New("network error"),
			},
			expect: func(t *testing.T, f *Filterable) {
				t.Helper()
				assert.False(t, f.isLoading, "should not be loading after FetchMsg with error")
				assert.Empty(t, f.options, "options should remain empty on error")
				require.Error(t, f.loadErr)
				assert.Equal(t, "network error", f.loadErr.Error())
			},
			description: "failed fetch should set loadErr",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFilterable("key", "title", "desc", []Option{})
			f.isLoading = true
			f.focused = true

			field, cmd := f.Update(tc.fetchMsg)
			f = field.(*Filterable)

			assert.Nil(t, cmd, "FetchMsg handler should not return a command")
			tc.expect(t, f)
		})
	}
}

func TestFilterable_View_ShowsSpinnerWhileLoading(t *testing.T) {
	f := NewFilterable("key", "Select Option", "Choose one", []Option{})
	f.WithOptionsFuncAsync(func() ([]Option, error) {
		return []Option{{Label: "A", Value: "a"}}, nil
	})

	f.focused = true
	f.isLoading = true

	view := f.View()
	assert.Contains(t, view, "Loading options", "should show loading message")
	assert.Contains(t, view, "Select Option", "should show title")
}

func TestFilterable_View_ShowsErrorOnAsyncFailure(t *testing.T) {
	f := NewFilterable("key", "Select", "", []Option{})
	f.WithOptionsFuncAsync(func() ([]Option, error) {
		return nil, errors.New("API failed")
	})
	f.focused = true
	f.loadErr = errors.New("API failed")

	view := f.View()
	assert.Contains(t, view, "Error loading options", "should show error message")
	assert.Contains(t, view, "API failed", "should show error details")
}

func TestFilterable_Update_BlocksInputWhileLoading(t *testing.T) {
	testCases := []struct {
		name        string
		description string
		keyMsg      tea.KeyMsg
		shouldBlock bool
	}{
		{
			name:        "Enter blocked while loading",
			keyMsg:      tea.KeyMsg{Type: tea.KeyEnter},
			shouldBlock: true,
			description: "Enter should not allow submission while loading",
		},
		{
			name:        "j (filter) blocked while loading",
			keyMsg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")},
			shouldBlock: true,
			description: "j should not filter while loading",
		},
		{
			name:        "down (nav) blocked while loading",
			keyMsg:      tea.KeyMsg{Type: tea.KeyDown},
			shouldBlock: true,
			description: "down should not navigate while loading",
		},
		{
			name:        "typing blocked while loading",
			keyMsg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")},
			shouldBlock: true,
			description: "typing should be ignored while loading",
		},
		{
			name:        "Ctrl+C allowed while loading",
			keyMsg:      tea.KeyMsg{Type: tea.KeyCtrlC},
			shouldBlock: false,
			description: "Ctrl+C should be allowed to cancel",
		},
		{
			name:        "q allowed while loading",
			keyMsg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")},
			shouldBlock: false,
			description: "q should be allowed to quit",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewFilterable("key", "Select", "", []Option{})
			f.WithOptionsFuncAsync(func() ([]Option, error) {
				return []Option{{Label: "A", Value: "a"}}, nil
			})
			f.focused = true
			f.isLoading = true

			field, cmd := f.Update(tc.keyMsg)
			f = field.(*Filterable)

			if tc.shouldBlock {
				// Should return early without processing
				assert.False(t, f.IsComplete(), tc.description)
				// For most keys, cmd should be nil
				if tc.keyMsg.Type == tea.KeyEnter || tc.keyMsg.Type == tea.KeyRunes && len(tc.keyMsg.Runes) > 0 {
					assert.Nil(t, cmd, tc.description)
				}
			}
			// Allowed keys (Ctrl+C, q) don't set complete either, they're handled by wizard
		})
	}
}

func TestFilterable_AllowsInputAfterLoading(t *testing.T) {
	f := NewFilterable("key", "Select", "", []Option{})
	f.WithOptionsFuncAsync(func() ([]Option, error) {
		return []Option{
			{Label: "Option A", Value: "a"},
			{Label: "Option B", Value: "b"},
		}, nil
	})
	f.focused = true
	f.isLoading = false

	// Simulate successful async load
	fetchMsg := async.FetchMsg[[]Option]{
		Value: []Option{
			{Label: "Option A", Value: "a"},
			{Label: "Option B", Value: "b"},
		},
		Err: nil,
	}
	field, _ := f.Update(fetchMsg)
	f = field.(*Filterable)

	// Now Enter should be allowed
	keyMsg := tea.KeyMsg{Type: tea.KeyEnter}
	field, cmd := f.Update(keyMsg)
	f = field.(*Filterable)

	assert.True(t, f.IsComplete(), "should allow submission after loading")
	assert.NotNil(t, cmd, "should return command for completion")
	assert.Equal(t, "a", f.GetValue(), "should have selected first option")
}

func TestFilterable_SpinnerTicks(t *testing.T) {
	f := NewFilterable("key", "Select", "", []Option{})
	f.WithOptionsFuncAsync(func() ([]Option, error) {
		return []Option{{Label: "A", Value: "a"}}, nil
	})
	f.focused = true
	f.isLoading = true

	// Send spinner tick
	msg := f.spinner.Tick()
	field, cmd := f.Update(msg)
	f = field.(*Filterable)

	// Should return a tick command for animation and still be loading
	assert.NotNil(t, cmd, "should return spinner.Tick command while loading")
	assert.True(t, f.isLoading, "should still be loading after spinner tick")
}

func TestFilterable_WithOptionsFuncAsync_Chaining(t *testing.T) {
	f := NewFilterable("key", "title", "desc", []Option{})
	result := f.WithOptionsFuncAsync(func() ([]Option, error) {
		return []Option{{Label: "Test", Value: "test"}}, nil
	})

	// Should return self for chaining
	assert.Equal(t, f, result, "should return self for method chaining")
	assert.NotNil(t, f.optionsFetch, "optionsFetch should be set")
}

func TestFilterable_AsyncLoadsOnFocus(t *testing.T) {
	f := NewFilterable("key", "Select", "", []Option{})
	f.WithOptionsFuncAsync(func() ([]Option, error) {
		return []Option{{Label: "A", Value: "a"}}, nil
	})

	// Before focus, should not be loading
	assert.False(t, f.isLoading)

	// After Init (which is called by wizard)
	cmd := f.Init()
	assert.NotNil(t, cmd)

	// Execute the command to simulate async operation
	msg := cmd()
	fetchMsg, ok := msg.(async.FetchMsg[[]Option])
	assert.True(t, ok)

	// Update with the fetched options
	field, _ := f.Update(fetchMsg)
	f = field.(*Filterable)

	assert.False(t, f.isLoading, "should not be loading after successful fetch")
	assert.Len(t, f.options, 1, "should have loaded options")
}

func TestFilterable_FilteringWorksAfterAsyncLoad(t *testing.T) {
	f := NewFilterable("key", "Select", "", []Option{})
	f.WithOptionsFuncAsync(func() ([]Option, error) {
		return []Option{
			{Label: "JIRA-123", Value: "jira-123"},
			{Label: "JIRA-456", Value: "jira-456"},
			{Label: "Other-789", Value: "other-789"},
		}, nil
	})
	f.focused = true

	// Simulate async load completing
	fetchMsg := async.FetchMsg[[]Option]{
		Value: []Option{
			{Label: "JIRA-123", Value: "jira-123"},
			{Label: "JIRA-456", Value: "jira-456"},
			{Label: "Other-789", Value: "other-789"},
		},
		Err: nil,
	}
	field, _ := f.Update(fetchMsg)
	f = field.(*Filterable)

	// Now simulate typing to filter
	f.textInput.SetValue("JIRA")
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("JIRA")}
	field, _ = f.Update(keyMsg)
	f = field.(*Filterable)

	// Should filter options
	assert.Len(t, f.filtered, 2, "should filter to 2 JIRA items")
	assert.Equal(t, "jira-123", f.filtered[0].Value)
	assert.Equal(t, "jira-456", f.filtered[1].Value)
}

func TestFilterable_CanCustomizeErrorDisplay(t *testing.T) {
	testErr := errors.New("custom error message")
	f := NewFilterable("key", "Select", "", []Option{})
	f.WithOptionsFuncAsync(func() ([]Option, error) {
		return nil, testErr
	})
	f.focused = true
	f.loadErr = testErr

	view := f.View()
	assert.Contains(t, view, "Error loading options", "should show error message")
	assert.Contains(t, view, "custom error message", "should show custom error")
}

func TestFilterable_NavigationAfterAsyncLoad(t *testing.T) {
	f := NewFilterable("key", "Select", "", []Option{})
	f.WithOptionsFuncAsync(func() ([]Option, error) {
		return []Option{
			{Label: "A", Value: "a"},
			{Label: "B", Value: "b"},
			{Label: "C", Value: "c"},
		}, nil
	})
	f.focused = true

	// Load options
	fetchMsg := async.FetchMsg[[]Option]{
		Value: []Option{
			{Label: "A", Value: "a"},
			{Label: "B", Value: "b"},
			{Label: "C", Value: "c"},
		},
		Err: nil,
	}
	field, _ := f.Update(fetchMsg)
	f = field.(*Filterable)

	// Navigate down
	keyMsg := tea.KeyMsg{Type: tea.KeyDown}
	field, _ = f.Update(keyMsg)
	f = field.(*Filterable)
	assert.Equal(t, 1, f.cursor, "should move cursor down")

	// Navigate down again
	field, _ = f.Update(keyMsg)
	f = field.(*Filterable)
	assert.Equal(t, 2, f.cursor, "should move cursor down again")

	// Wrap around
	field, _ = f.Update(keyMsg)
	f = field.(*Filterable)
	assert.Equal(t, 0, f.cursor, "should wrap cursor to top")
}
