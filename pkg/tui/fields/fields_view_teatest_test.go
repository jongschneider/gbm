package fields

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"gbm/pkg/tui"

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

// fieldModel wraps a Field to implement tea.Model for teatest.
// It tracks render count and quits after maxRenders or on Enter key.
type fieldModel struct {
	field      tui.Field
	renders    int
	maxRenders int
}

func newFieldModel(field tui.Field, maxRenders int) *fieldModel {
	return &fieldModel{
		field:      field,
		maxRenders: maxRenders,
	}
}

func (m *fieldModel) Init() tea.Cmd {
	// Focus the field and get its init command
	focusCmd := m.field.Focus()
	initCmd := m.field.Init()
	return tea.Batch(focusCmd, initCmd)
}

func (m *fieldModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	}

	field, cmd := m.field.Update(msg)
	m.field = field

	// Check if field is complete (e.g., Enter was pressed)
	if m.field.IsComplete() || m.field.IsCancelled() {
		return m, tea.Quit
	}

	return m, cmd
}

func (m *fieldModel) View() string {
	m.renders++
	return m.field.View()
}

// TestFilterable_View_EndsWithNewline verifies Filterable's View() always ends with newline.
// This tests the fix for terminal rendering issues where missing trailing newlines
// caused display artifacts in Bubble Tea applications.
func TestFilterable_View_EndsWithNewline(t *testing.T) {
	options := []Option{
		{Label: "Option A", Value: "a"},
		{Label: "Option B", Value: "b"},
		{Label: "Option C", Value: "c"},
	}

	t.Run("with options focused", func(t *testing.T) {
		f := NewFilterable("test", "Select an option", "Description text", options)
		f.Focus()

		view := f.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"View() with options must end with newline. Got: %q", view)
		assert.Contains(t, view, "Select an option")
		assert.Contains(t, view, "Option A")
	})

	t.Run("with options blurred", func(t *testing.T) {
		f := NewFilterable("test", "Select an option", "", options)
		f.Blur()

		view := f.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Blurred View() must end with newline. Got: %q", view)
	})

	t.Run("with no matches", func(t *testing.T) {
		f := NewFilterable("test", "Select an option", "", options)
		f.Focus()
		f.textInput.Focus()
		f.textInput.SetValue("xyz") // No matches
		f.filterOptions()

		view := f.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"View() with no matches must end with newline. Got: %q", view)
		assert.Contains(t, view, "No matches")
	})

	t.Run("empty options list", func(t *testing.T) {
		f := NewFilterable("test", "Select an option", "", []Option{})
		f.Focus()

		view := f.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"View() with empty options must end with newline. Got: %q", view)
	})
}

// TestFilterable_View_LoadingState verifies loading state rendering ends with newline.
func TestFilterable_View_LoadingState(t *testing.T) {
	// Create filterable with async loading
	f := NewFilterable("test", "Loading Test", "Wait for options", nil)
	f.isLoading = true
	f.Focus()

	view := f.View()
	assert.True(t, strings.HasSuffix(view, "\n"),
		"View() in loading state must end with newline. Got: %q", view)
	assert.Contains(t, view, "Loading options...")
}

// TestFilterable_View_ErrorState verifies error state rendering ends with newline.
func TestFilterable_View_ErrorState(t *testing.T) {
	f := NewFilterable("test", "Error Test", "", nil)
	f.loadErr = assert.AnError
	f.Focus()

	view := f.View()
	assert.True(t, strings.HasSuffix(view, "\n"),
		"View() in error state must end with newline. Got: %q", view)
	assert.Contains(t, view, "Error loading options")
}

// TestSelector_View_EndsWithNewline verifies Selector's View() always ends with newline.
func TestSelector_View_EndsWithNewline(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
	}

	t.Run("focused", func(t *testing.T) {
		s := NewSelector("test", "Choose one", options)
		s.Focus()

		view := s.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Selector View() must end with newline. Got: %q", view)
		assert.Contains(t, view, "Choose one")
		assert.Contains(t, view, "First")
	})

	t.Run("blurred", func(t *testing.T) {
		s := NewSelector("test", "Choose one", options)
		s.Blur()

		view := s.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Blurred Selector View() must end with newline. Got: %q", view)
	})

	t.Run("single option", func(t *testing.T) {
		s := NewSelector("test", "Choose", []Option{{Label: "Only", Value: "only"}})
		s.Focus()

		view := s.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Single option View() must end with newline. Got: %q", view)
	})
}

// TestTextInput_View_EndsWithNewline verifies TextInput's View() always ends with newline.
func TestTextInput_View_EndsWithNewline(t *testing.T) {
	t.Run("focused without error", func(t *testing.T) {
		ti := NewTextInput("test", "Enter name", "Your full name")
		ti.Focus()

		view := ti.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"TextInput View() must end with newline. Got: %q", view)
		assert.Contains(t, view, "Enter name")
		assert.Contains(t, view, "Your full name")
	})

	t.Run("with validation error", func(t *testing.T) {
		ti := NewTextInput("test", "Enter name", "")
		ti.Focus()
		ti.err = assert.AnError // Simulate validation error

		view := ti.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"TextInput View() with error must end with newline. Got: %q", view)
	})

	t.Run("without description", func(t *testing.T) {
		ti := NewTextInput("test", "Enter name", "")
		ti.Focus()

		view := ti.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"TextInput View() without description must end with newline. Got: %q", view)
	})

	t.Run("blurred", func(t *testing.T) {
		ti := NewTextInput("test", "Enter name", "Description")
		ti.Blur()

		view := ti.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Blurred TextInput View() must end with newline. Got: %q", view)
	})
}

// TestConfirm_View_EndsWithNewline verifies Confirm's View() always ends with newline.
func TestConfirm_View_EndsWithNewline(t *testing.T) {
	t.Run("focused yes selected", func(t *testing.T) {
		c := NewConfirm("test", "Proceed?")
		c.Focus()

		view := c.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Confirm View() must end with newline. Got: %q", view)
		assert.Contains(t, view, "Proceed?")
		assert.Contains(t, view, "Yes")
		assert.Contains(t, view, "No")
	})

	t.Run("focused no selected", func(t *testing.T) {
		c := NewConfirm("test", "Proceed?")
		c.Focus()
		c.selected = false

		view := c.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Confirm View() with No selected must end with newline. Got: %q", view)
	})

	t.Run("with summary", func(t *testing.T) {
		c := NewConfirm("test", "Proceed?").WithSummary("This will make changes")
		c.Focus()

		view := c.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Confirm View() with summary must end with newline. Got: %q", view)
		assert.Contains(t, view, "This will make changes")
	})

	t.Run("blurred", func(t *testing.T) {
		c := NewConfirm("test", "Proceed?")
		c.Blur()

		view := c.View()
		assert.True(t, strings.HasSuffix(view, "\n"),
			"Blurred Confirm View() must end with newline. Got: %q", view)
	})
}

// TestFilterable_Interactive_Typing tests interactive typing behavior using teatest.
// This verifies the TUI responds correctly to user input in a real program context.
func TestFilterable_Interactive_Typing(t *testing.T) {
	options := []Option{
		{Label: "Apple", Value: "apple"},
		{Label: "Banana", Value: "banana"},
		{Label: "Cherry", Value: "cherry"},
		{Label: "Apricot", Value: "apricot"},
	}

	t.Run("typing updates filter state", func(t *testing.T) {
		f := NewFilterable("fruit", "Select fruit", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select fruit"))
		}, teatest.WithDuration(time.Second))

		// Verify initial state shows all options
		assert.Equal(t, 4, len(f.filtered), "initial state should show all 4 options")

		// Type 'ap' to filter
		tm.Type("ap")

		// Allow time for messages to process
		time.Sleep(100 * time.Millisecond)

		// Verify filter was applied - only Apple and Apricot match 'ap'
		assert.Equal(t, 2, len(f.filtered), "after typing 'ap', should have 2 matches")
		assert.Equal(t, "ap", f.textInput.Value(), "text input should contain 'ap'")

		// Type more to narrow down
		tm.Type("r")
		time.Sleep(100 * time.Millisecond)

		// Only Apricot matches 'apr'
		assert.Equal(t, 1, len(f.filtered), "after typing 'apr', should have 1 match")
		assert.Equal(t, "Apricot", f.filtered[0].Label, "the match should be Apricot")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
}

// TestSelector_Interactive_Navigation tests arrow key navigation using teatest.
func TestSelector_Interactive_Navigation(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	s := NewSelector("test", "Select", options)
	model := newFieldModel(s, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select"))
	}, teatest.WithDuration(time.Second))

	// Press down arrow to move to second option
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	// Give it time to process
	time.Sleep(50 * time.Millisecond)

	// Press down again
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})

	time.Sleep(50 * time.Millisecond)

	// Quit and verify final state
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify cursor moved (selector should be at Third after two down presses)
	assert.Equal(t, 2, s.cursor, "cursor should be at index 2 after two down presses")
}

// TestConfirm_Interactive_Toggle tests tab toggling using teatest.
func TestConfirm_Interactive_Toggle(t *testing.T) {
	c := NewConfirm("test", "Continue?")
	model := newFieldModel(c, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Initial state should be Yes selected
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Continue?"))
	}, teatest.WithDuration(time.Second))

	assert.True(t, c.selected, "initial selection should be Yes")

	// Press tab to toggle
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	time.Sleep(50 * time.Millisecond)

	assert.False(t, c.selected, "after tab, selection should be No")

	// Press tab again
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	time.Sleep(50 * time.Millisecond)

	assert.True(t, c.selected, "after second tab, selection should be Yes again")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// =============================================================================
// TT-008: Filterable Enter selection tests
// =============================================================================

// TestFilterable_EnterSelectsHighlightedOption verifies that pressing Enter
// selects the currently highlighted option in the filtered list.
func TestFilterable_EnterSelectsHighlightedOption(t *testing.T) {
	options := []Option{
		{Label: "Option Alpha", Value: "alpha"},
		{Label: "Option Beta", Value: "beta"},
		{Label: "Option Gamma", Value: "gamma"},
	}

	t.Run("selects first option by default", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Verify initial cursor position is 0
		assert.Equal(t, 0, f.cursor, "initial cursor should be at index 0")
		assert.False(t, f.IsComplete(), "should not be complete before Enter")

		// Press Enter to select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify selected value is the first option
		assert.Equal(t, "alpha", f.selected, "should select Option Alpha's value")
		assert.Equal(t, "alpha", f.GetValue(), "GetValue() should return selected value")
		assert.True(t, f.IsComplete(), "IsComplete() should be true after Enter")
	})

	t.Run("selects second option after down navigation", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Navigate to second option
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, f.cursor, "cursor should be at index 1 after down press")

		// Press Enter to select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify selected value is the second option
		assert.Equal(t, "beta", f.selected, "should select Option Beta's value")
		assert.Equal(t, "beta", f.GetValue(), "GetValue() should return selected value")
		assert.True(t, f.IsComplete(), "IsComplete() should be true after Enter")
	})

	t.Run("selects third option after two down presses", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Navigate to third option
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 2, f.cursor, "cursor should be at index 2 after two down presses")

		// Press Enter to select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify selected value is the third option
		assert.Equal(t, "gamma", f.selected, "should select Option Gamma's value")
		assert.Equal(t, "gamma", f.GetValue(), "GetValue() should return selected value")
		assert.True(t, f.IsComplete(), "IsComplete() should be true after Enter")
	})
}

// TestFilterable_EnterWithFilteredResults verifies Enter works correctly
// when the options list has been filtered by typing.
func TestFilterable_EnterWithFilteredResults(t *testing.T) {
	options := []Option{
		{Label: "Apple", Value: "apple"},
		{Label: "Banana", Value: "banana"},
		{Label: "Apricot", Value: "apricot"},
		{Label: "Cherry", Value: "cherry"},
	}

	t.Run("selects from filtered list", func(t *testing.T) {
		f := NewFilterable("fruit", "Select fruit", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select fruit"))
		}, teatest.WithDuration(time.Second))

		// Type 'ap' to filter to Apple and Apricot
		tm.Type("ap")
		time.Sleep(100 * time.Millisecond)

		// Verify filter was applied
		assert.Equal(t, 2, len(f.filtered), "filter should show 2 options (Apple, Apricot)")
		assert.Equal(t, 0, f.cursor, "cursor should be at 0")

		// Navigate to second filtered option (Apricot)
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, f.cursor, "cursor should be at 1")
		assert.Equal(t, "Apricot", f.filtered[f.cursor].Label, "cursor should be on Apricot")

		// Press Enter to select Apricot
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify the filtered selection
		assert.Equal(t, "apricot", f.selected, "should select Apricot's value")
		assert.Equal(t, "apricot", f.GetValue(), "GetValue() should return apricot")
		assert.True(t, f.IsComplete(), "IsComplete() should be true")
	})
}

// TestFilterable_NextStepMsgSentOnEnter verifies that NextStepMsg is sent
// when Enter is pressed, causing the fieldModel to quit (via IsComplete check).
func TestFilterable_NextStepMsgSentOnEnter(t *testing.T) {
	options := []Option{
		{Label: "Only Option", Value: "only"},
	}

	f := NewFilterable("test", "Select", "", options)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select"))
	}, teatest.WithDuration(time.Second))

	// Verify field is not complete before Enter
	assert.False(t, f.IsComplete(), "should not be complete before Enter")

	// Press Enter - this should trigger NextStepMsg which causes fieldModel to quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// The model should finish because fieldModel.Update checks IsComplete()
	// and returns tea.Quit when the field completes
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify completion state
	assert.True(t, f.IsComplete(), "IsComplete() should be true, indicating NextStepMsg flow")
	assert.Equal(t, "only", f.GetValue(), "selected value should be stored")
}

// TestFilterable_ValueStoredAfterSelection verifies GetValue() returns
// the selected option's value after Enter is pressed.
func TestFilterable_ValueStoredAfterSelection(t *testing.T) {
	options := []Option{
		{Label: "Item One", Value: "value-1"},
		{Label: "Item Two", Value: "value-2"},
		{Label: "Item Three", Value: "value-3"},
	}

	f := NewFilterable("items", "Select item", "Pick one", options)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select item"))
	}, teatest.WithDuration(time.Second))

	// GetValue should be empty string before selection
	assert.Equal(t, "", f.GetValue(), "GetValue() should be empty before selection")

	// Navigate to Item Two and select
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify value is stored correctly
	assert.Equal(t, "value-2", f.GetValue(), "GetValue() should return the selected option's Value field")
	assert.Equal(t, f.selected, f.GetValue().(string), "GetValue() should match internal selected field")
}
