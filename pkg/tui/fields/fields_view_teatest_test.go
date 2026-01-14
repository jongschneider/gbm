package fields

import (
	"bytes"
	"fmt"
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

// =============================================================================
// TT-013: TextInput typing and submission tests
// =============================================================================

// TestTextInput_TypingUpdatesValue verifies that typing characters updates
// the internal input value correctly.
func TestTextInput_TypingUpdatesValue(t *testing.T) {
	ti := NewTextInput("name", "Enter name", "Your full name")
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Verify initial state
	assert.Equal(t, "", ti.textInput.Value(), "initial text input should be empty")

	// Type some characters
	tm.Type("John Doe")
	time.Sleep(100 * time.Millisecond)

	// Verify typed text is in the input
	assert.Equal(t, "John Doe", ti.textInput.Value(), "typed text should be in input")
	assert.False(t, ti.IsComplete(), "should not be complete until Enter pressed")

	// Quit without submitting
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTextInput_EnterSubmitsValue verifies that pressing Enter submits
// the current input value and marks the field as complete.
func TestTextInput_EnterSubmitsValue(t *testing.T) {
	ti := NewTextInput("name", "Enter name", "")
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Type a value
	tm.Type("Alice Smith")
	time.Sleep(100 * time.Millisecond)

	// Verify not complete before Enter
	assert.False(t, ti.IsComplete(), "should not be complete before Enter")
	assert.Equal(t, "", ti.GetValue(), "GetValue should be empty before submission")

	// Press Enter to submit
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Model should quit on completion (fieldModel checks IsComplete)
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify submission
	assert.True(t, ti.IsComplete(), "IsComplete() should be true after Enter")
	assert.Equal(t, "Alice Smith", ti.GetValue(), "GetValue() should return submitted value")
}

// TestTextInput_SubmittedValueIsTrimmed verifies that whitespace is trimmed
// from the submitted value.
func TestTextInput_SubmittedValueIsTrimmed(t *testing.T) {
	t.Run("leading and trailing whitespace trimmed", func(t *testing.T) {
		ti := NewTextInput("name", "Enter name", "")
		model := newFieldModel(ti, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter name"))
		}, teatest.WithDuration(time.Second))

		// Type value with leading and trailing spaces
		tm.Type("  Bob Jones  ")
		time.Sleep(100 * time.Millisecond)

		// Press Enter to submit
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify trimmed value
		assert.Equal(t, "Bob Jones", ti.GetValue(), "submitted value should be trimmed")
		assert.True(t, ti.IsComplete(), "should be complete")
	})

	t.Run("only whitespace becomes empty string", func(t *testing.T) {
		ti := NewTextInput("name", "Enter name", "")
		model := newFieldModel(ti, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter name"))
		}, teatest.WithDuration(time.Second))

		// Type only spaces
		tm.Type("   ")
		time.Sleep(100 * time.Millisecond)

		// Press Enter to submit
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify trimmed to empty
		assert.Equal(t, "", ti.GetValue(), "whitespace-only input should become empty string")
		assert.True(t, ti.IsComplete(), "should be complete")
	})
}

// TestTextInput_NextStepMsgSentOnEnter verifies that NextStepMsg is sent
// when Enter is pressed, which is detected by fieldModel returning tea.Quit.
func TestTextInput_NextStepMsgSentOnEnter(t *testing.T) {
	ti := NewTextInput("test", "Test Input", "")
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Test Input"))
	}, teatest.WithDuration(time.Second))

	// Verify field is not complete
	assert.False(t, ti.IsComplete(), "should not be complete before Enter")

	// Type something
	tm.Type("test value")
	time.Sleep(100 * time.Millisecond)

	// Press Enter - this triggers NextStepMsg which causes fieldModel to quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// The model should finish because fieldModel.Update checks IsComplete()
	// and returns tea.Quit when the field completes
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify completion - the fact that WaitFinished succeeded confirms
	// the message flow worked correctly (NextStepMsg -> IsComplete -> tea.Quit)
	assert.True(t, ti.IsComplete(), "IsComplete() should be true, indicating NextStepMsg flow")
	assert.Equal(t, "test value", ti.GetValue(), "submitted value should be stored")
}

// TestTextInput_IsCompleteAfterSubmission verifies that IsComplete() returns
// true after successful submission and false before.
func TestTextInput_IsCompleteAfterSubmission(t *testing.T) {
	ti := NewTextInput("field", "Enter text", "Some description")
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter text"))
	}, teatest.WithDuration(time.Second))

	// Initial state
	assert.False(t, ti.IsComplete(), "IsComplete() should be false initially")
	assert.False(t, ti.IsCancelled(), "IsCancelled() should be false")

	// Type and submit
	tm.Type("my value")
	time.Sleep(100 * time.Millisecond)

	// Still not complete until Enter
	assert.False(t, ti.IsComplete(), "IsComplete() should still be false before Enter")

	// Submit
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Now complete
	assert.True(t, ti.IsComplete(), "IsComplete() should be true after submission")
	assert.False(t, ti.IsCancelled(), "IsCancelled() should still be false")
}

// TestTextInput_EmptySubmission verifies behavior when submitting empty input.
func TestTextInput_EmptySubmission(t *testing.T) {
	ti := NewTextInput("optional", "Optional field", "Leave blank if not needed")
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Optional field"))
	}, teatest.WithDuration(time.Second))

	// Don't type anything, just press Enter
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify empty submission works
	assert.True(t, ti.IsComplete(), "empty submission should complete")
	assert.Equal(t, "", ti.GetValue(), "GetValue() should return empty string")
}

// =============================================================================
// TT-014: TextInput validation tests
// =============================================================================

// TestTextInput_ValidatorCalledOnEnter verifies that the validator function
// is called when Enter is pressed.
func TestTextInput_ValidatorCalledOnEnter(t *testing.T) {
	validatorCalled := false
	validator := func(value string) error {
		validatorCalled = true
		return nil
	}

	ti := NewTextInput("name", "Enter name", "").WithValidator(validator)
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Verify validator not called yet
	assert.False(t, validatorCalled, "validator should not be called before Enter")

	// Type a value
	tm.Type("test value")
	time.Sleep(100 * time.Millisecond)

	// Validator still not called
	assert.False(t, validatorCalled, "validator should not be called on typing")

	// Press Enter - this should trigger validator
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Validator should now be called
	assert.True(t, validatorCalled, "validator should be called on Enter")
	assert.True(t, ti.IsComplete(), "should complete with valid input")
}

// TestTextInput_ValidationErrorPreventsSubmission verifies that validation errors
// prevent the field from completing and the value from being submitted.
func TestTextInput_ValidationErrorPreventsSubmission(t *testing.T) {
	validator := func(value string) error {
		if len(value) < 5 {
			return fmt.Errorf("value must be at least 5 characters")
		}
		return nil
	}

	ti := NewTextInput("name", "Enter name", "").WithValidator(validator)
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Type a short value (less than 5 chars)
	tm.Type("abc")
	time.Sleep(100 * time.Millisecond)

	// Press Enter - this should fail validation
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(100 * time.Millisecond)

	// Verify field is NOT complete
	assert.False(t, ti.IsComplete(), "should not complete with invalid input")
	assert.Equal(t, "", ti.GetValue(), "value should not be set on validation failure")
	assert.NotNil(t, ti.err, "error should be set")
	assert.Contains(t, ti.err.Error(), "at least 5 characters")

	// Quit the test
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTextInput_ErrorMessageDisplayedInView verifies that validation error
// messages are displayed in the View() output.
func TestTextInput_ErrorMessageDisplayedInView(t *testing.T) {
	validator := func(value string) error {
		if value == "" {
			return fmt.Errorf("name is required")
		}
		return nil
	}

	ti := NewTextInput("name", "Enter name", "").WithValidator(validator)
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Press Enter with empty input - should fail validation
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(100 * time.Millisecond)

	// Verify error is set
	assert.NotNil(t, ti.err, "error should be set after validation failure")

	// Check that View() contains the error message
	view := ti.View()
	assert.Contains(t, view, "name is required", "View() should display error message")
	assert.True(t, strings.HasSuffix(view, "\n"), "View() should still end with newline")

	// Quit the test
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTextInput_ErrorClearsOnTyping verifies that validation errors are cleared
// when the user starts typing again.
func TestTextInput_ErrorClearsOnTyping(t *testing.T) {
	validator := func(value string) error {
		if value == "" {
			return fmt.Errorf("value required")
		}
		return nil
	}

	ti := NewTextInput("name", "Enter name", "").WithValidator(validator)
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Press Enter with empty input - should fail validation
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(100 * time.Millisecond)

	// Verify error is set
	assert.NotNil(t, ti.err, "error should be set after validation failure")
	assert.Contains(t, ti.View(), "value required", "View() should show error")

	// Start typing - error should clear
	tm.Type("a")
	time.Sleep(100 * time.Millisecond)

	// Error should be cleared
	assert.Nil(t, ti.err, "error should be cleared after typing")
	assert.NotContains(t, ti.View(), "value required", "View() should not show error after typing")

	// Quit the test
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTextInput_ValidInputAfterErrorSucceeds verifies that after a validation
// error, correcting the input and pressing Enter again succeeds.
func TestTextInput_ValidInputAfterErrorSucceeds(t *testing.T) {
	validator := func(value string) error {
		if len(value) < 3 {
			return fmt.Errorf("minimum 3 characters required")
		}
		return nil
	}

	ti := NewTextInput("code", "Enter code", "").WithValidator(validator)
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter code"))
	}, teatest.WithDuration(time.Second))

	// Type short value and submit
	tm.Type("ab")
	time.Sleep(100 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(100 * time.Millisecond)

	// Should fail validation
	assert.False(t, ti.IsComplete(), "should not complete with short input")
	assert.NotNil(t, ti.err, "error should be set")

	// Add more characters to make valid
	tm.Type("cdef")
	time.Sleep(100 * time.Millisecond)

	// Error should be cleared
	assert.Nil(t, ti.err, "error should be cleared after typing")

	// Now submit again - should succeed
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Should now be complete with the full value
	assert.True(t, ti.IsComplete(), "should complete with valid input")
	assert.Equal(t, "abcdef", ti.GetValue(), "value should be the corrected input")
	assert.Nil(t, ti.err, "no error should remain")
}

// TestTextInput_ValidationWithTrimmedValue verifies that validation is run
// against the trimmed value, matching what gets stored.
func TestTextInput_ValidationWithTrimmedValue(t *testing.T) {
	var validatedValue string
	validator := func(value string) error {
		validatedValue = value
		if value == "" {
			return fmt.Errorf("empty not allowed")
		}
		return nil
	}

	ti := NewTextInput("name", "Enter name", "").WithValidator(validator)
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Type value with whitespace
	tm.Type("  hello  ")
	time.Sleep(100 * time.Millisecond)

	// Submit
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Validator should receive trimmed value
	assert.Equal(t, "hello", validatedValue, "validator should receive trimmed value")
	assert.Equal(t, "hello", ti.GetValue(), "stored value should be trimmed")
	assert.True(t, ti.IsComplete(), "should complete")
}

// TestTextInput_NoValidatorAllowsAnyInput verifies that without a validator,
// any input (including empty) is accepted.
func TestTextInput_NoValidatorAllowsAnyInput(t *testing.T) {
	ti := NewTextInput("optional", "Optional field", "")
	// No validator set
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Optional field"))
	}, teatest.WithDuration(time.Second))

	// Submit empty value
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Should succeed without any validator
	assert.True(t, ti.IsComplete(), "should complete without validator")
	assert.Equal(t, "", ti.GetValue(), "empty value should be accepted")
	assert.Nil(t, ti.err, "no error should be set")
}

// =============================================================================
// TT-015: TextInput default value tests
// =============================================================================

// TestTextInput_WithDefaultSetsValueOnFocus verifies that WithDefault() sets
// the initial value when the field is focused.
func TestTextInput_WithDefaultSetsValueOnFocus(t *testing.T) {
	ti := NewTextInput("name", "Enter name", "Your full name")
	ti.WithDefault("John Doe")
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Verify the default value was set on focus (focus happens in Init)
	assert.Equal(t, "John Doe", ti.textInput.Value(), "default value should be set after focus")

	// View should display the default value
	view := ti.View()
	assert.Contains(t, view, "John Doe", "view should contain default value")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTextInput_CursorAtEndOfDefault verifies that the cursor is positioned
// at the end of the default value after focus.
func TestTextInput_CursorAtEndOfDefault(t *testing.T) {
	ti := NewTextInput("name", "Enter name", "")
	ti.WithDefault("Hello World")
	model := newFieldModel(ti, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Enter name"))
	}, teatest.WithDuration(time.Second))

	// Verify cursor position is at end of default value
	assert.Equal(t, len("Hello World"), ti.textInput.Position(), "cursor should be at end of default value")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestTextInput_DefaultValueCanBeEdited verifies that the default value can
// be modified by typing.
func TestTextInput_DefaultValueCanBeEdited(t *testing.T) {
	t.Run("append to default value", func(t *testing.T) {
		ti := NewTextInput("name", "Enter name", "")
		ti.WithDefault("Hello")
		model := newFieldModel(ti, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter name"))
		}, teatest.WithDuration(time.Second))

		// Verify default value is set
		assert.Equal(t, "Hello", ti.textInput.Value())

		// Type additional characters (cursor at end, so this appends)
		tm.Type(" World")
		time.Sleep(100 * time.Millisecond)

		// Verify the combined value
		assert.Equal(t, "Hello World", ti.textInput.Value(), "typing should append to default value")

		// Submit and verify
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, ti.IsComplete())
		assert.Equal(t, "Hello World", ti.GetValue())
	})

	t.Run("backspace removes from default value", func(t *testing.T) {
		ti := NewTextInput("name", "Enter name", "")
		ti.WithDefault("Hello")
		model := newFieldModel(ti, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter name"))
		}, teatest.WithDuration(time.Second))

		// Press backspace a few times
		tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})
		tm.Send(tea.KeyMsg{Type: tea.KeyBackspace})
		time.Sleep(100 * time.Millisecond)

		// Verify characters were removed
		assert.Equal(t, "Hel", ti.textInput.Value(), "backspace should remove characters from default")

		// Submit and verify
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, ti.IsComplete())
		assert.Equal(t, "Hel", ti.GetValue())
	})

	t.Run("replace entire default value", func(t *testing.T) {
		ti := NewTextInput("name", "Enter name", "")
		ti.WithDefault("Old Value")
		model := newFieldModel(ti, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter name"))
		}, teatest.WithDuration(time.Second))

		// Clear with Ctrl+U (delete to start of line)
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlU})
		time.Sleep(50 * time.Millisecond)

		// Type new value
		tm.Type("New Value")
		time.Sleep(100 * time.Millisecond)

		// Submit and verify
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, ti.IsComplete())
		assert.Equal(t, "New Value", ti.GetValue())
	})
}

// TestTextInput_EmptyDefaultValueHandled verifies that an empty default value
// is handled gracefully.
func TestTextInput_EmptyDefaultValueHandled(t *testing.T) {
	t.Run("empty string default", func(t *testing.T) {
		ti := NewTextInput("name", "Enter name", "")
		ti.WithDefault("")
		model := newFieldModel(ti, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter name"))
		}, teatest.WithDuration(time.Second))

		// Verify text input is empty
		assert.Equal(t, "", ti.textInput.Value(), "empty default should result in empty input")

		// Should still be able to type and submit
		tm.Type("New Text")
		time.Sleep(100 * time.Millisecond)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, ti.IsComplete())
		assert.Equal(t, "New Text", ti.GetValue())
	})

	t.Run("no WithDefault called", func(t *testing.T) {
		ti := NewTextInput("name", "Enter name", "")
		// No WithDefault call
		model := newFieldModel(ti, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter name"))
		}, teatest.WithDuration(time.Second))

		// Verify text input starts empty
		assert.Equal(t, "", ti.textInput.Value(), "input should start empty without WithDefault")

		// Type and submit
		tm.Type("Typed Value")
		time.Sleep(100 * time.Millisecond)

		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, ti.IsComplete())
		assert.Equal(t, "Typed Value", ti.GetValue())
	})
}

// =============================================================================
// TT-017: Confirm Enter submission tests
// =============================================================================

// TestConfirm_EnterWithYesSelected verifies that pressing Enter with Yes
// selected completes the field with value true and sends NextStepMsg.
func TestConfirm_EnterWithYesSelected(t *testing.T) {
	c := NewConfirm("proceed", "Do you want to proceed?")
	model := newFieldModel(c, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Do you want to proceed?"))
	}, teatest.WithDuration(time.Second))

	// Verify initial state: Yes is selected by default
	assert.True(t, c.selected, "Yes should be selected by default")
	assert.False(t, c.IsComplete(), "should not be complete before Enter")
	assert.False(t, c.IsCancelled(), "should not be cancelled before Enter")

	// Press Enter to confirm Yes selection
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Model should quit on completion (fieldModel checks IsComplete)
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify confirmation with Yes
	assert.True(t, c.IsComplete(), "IsComplete() should be true after Enter")
	assert.False(t, c.IsCancelled(), "IsCancelled() should be false for Yes")
	assert.Equal(t, true, c.GetValue(), "GetValue() should return true for Yes")
}

// TestConfirm_EnterWithNoSelected verifies that pressing Enter with No
// selected completes the field with value false and sends CancelMsg.
func TestConfirm_EnterWithNoSelected(t *testing.T) {
	c := NewConfirm("proceed", "Do you want to proceed?")
	model := newFieldModel(c, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Do you want to proceed?"))
	}, teatest.WithDuration(time.Second))

	// Navigate to No using right arrow
	tm.Send(tea.KeyMsg{Type: tea.KeyRight})
	time.Sleep(50 * time.Millisecond)

	// Verify No is now selected
	assert.False(t, c.selected, "No should be selected after right arrow")

	// Press Enter to confirm No selection
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Model should quit on completion (fieldModel checks IsCancelled)
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify confirmation with No
	assert.True(t, c.IsComplete(), "IsComplete() should be true after Enter")
	assert.True(t, c.IsCancelled(), "IsCancelled() should be true for No")
	assert.Equal(t, false, c.GetValue(), "GetValue() should return false for No")
}

// TestConfirm_GetValueReturnsCorrectBoolean verifies that GetValue() returns
// the correct boolean based on which button was selected when Enter was pressed.
func TestConfirm_GetValueReturnsCorrectBoolean(t *testing.T) {
	t.Run("returns true when Yes confirmed", func(t *testing.T) {
		c := NewConfirm("test", "Confirm?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Confirm?"))
		}, teatest.WithDuration(time.Second))

		// GetValue should return false (zero value) before completion
		assert.Equal(t, false, c.GetValue(), "GetValue() before completion should be false")

		// Confirm with Yes selected (default)
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.Equal(t, true, c.GetValue(), "GetValue() should be true after Yes confirmed")
	})

	t.Run("returns false when No confirmed", func(t *testing.T) {
		c := NewConfirm("test", "Confirm?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Confirm?"))
		}, teatest.WithDuration(time.Second))

		// Navigate to No with tab
		tm.Send(tea.KeyMsg{Type: tea.KeyTab})
		time.Sleep(50 * time.Millisecond)

		// Confirm with No selected
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.Equal(t, false, c.GetValue(), "GetValue() should be false after No confirmed")
	})
}

// TestConfirm_IsCancelledReflectsNoSelection verifies that IsCancelled()
// returns true only when No is selected and Enter is pressed.
func TestConfirm_IsCancelledReflectsNoSelection(t *testing.T) {
	t.Run("false when Yes confirmed", func(t *testing.T) {
		c := NewConfirm("test", "Continue?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Continue?"))
		}, teatest.WithDuration(time.Second))

		// IsCancelled should be false initially
		assert.False(t, c.IsCancelled(), "IsCancelled() should be false initially")

		// Confirm Yes
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.False(t, c.IsCancelled(), "IsCancelled() should be false when Yes confirmed")
		assert.True(t, c.IsComplete(), "IsComplete() should be true")
	})

	t.Run("true when No confirmed", func(t *testing.T) {
		c := NewConfirm("test", "Continue?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Continue?"))
		}, teatest.WithDuration(time.Second))

		// Navigate to No using vim-style 'l' key
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("l")})
		time.Sleep(50 * time.Millisecond)

		assert.False(t, c.selected, "No should be selected after 'l' press")

		// Confirm No
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, c.IsCancelled(), "IsCancelled() should be true when No confirmed")
		assert.True(t, c.IsComplete(), "IsComplete() should also be true")
	})

	t.Run("true when navigated back to No and confirmed", func(t *testing.T) {
		c := NewConfirm("test", "Proceed?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Proceed?"))
		}, teatest.WithDuration(time.Second))

		// Navigate: Yes -> No -> Yes -> No
		tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // to No
		time.Sleep(50 * time.Millisecond)
		assert.False(t, c.selected)

		tm.Send(tea.KeyMsg{Type: tea.KeyTab}) // back to Yes
		time.Sleep(50 * time.Millisecond)
		assert.True(t, c.selected)

		tm.Send(tea.KeyMsg{Type: tea.KeyRight}) // to No again
		time.Sleep(50 * time.Millisecond)
		assert.False(t, c.selected)

		// Confirm No
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, c.IsCancelled(), "IsCancelled() should be true")
		assert.Equal(t, false, c.GetValue(), "GetValue() should be false")
	})
}

// TestConfirm_EnterWithSummary verifies that the Confirm field with summary
// text works correctly when Enter is pressed.
func TestConfirm_EnterWithSummary(t *testing.T) {
	c := NewConfirm("confirm", "Create branch?").WithSummary("This will create a new feature branch")
	model := newFieldModel(c, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for render including summary
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Create branch?")) &&
			bytes.Contains(bts, []byte("This will create a new feature branch"))
	}, teatest.WithDuration(time.Second))

	// Verify field renders correctly with summary
	view := c.View()
	assert.Contains(t, view, "Create branch?")
	assert.Contains(t, view, "This will create a new feature branch")

	// Confirm with Yes
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify completion
	assert.True(t, c.IsComplete(), "should be complete")
	assert.Equal(t, true, c.GetValue(), "should confirm with true")
}

// =============================================================================
// TT-009: Filterable custom value entry tests
// =============================================================================

// TestFilterable_CustomValue_NoMatchesMessage verifies that typing non-matching
// text shows the 'No matches' message in the view.
func TestFilterable_CustomValue_NoMatchesMessage(t *testing.T) {
	options := []Option{
		{Label: "Apple", Value: "apple"},
		{Label: "Banana", Value: "banana"},
		{Label: "Cherry", Value: "cherry"},
	}

	t.Run("shows no matches message for non-matching input", func(t *testing.T) {
		f := NewFilterable("fruit", "Select fruit", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select fruit"))
		}, teatest.WithDuration(time.Second))

		// Verify initial state shows all 3 options
		assert.Equal(t, 3, len(f.filtered), "initial state should show all 3 options")

		// Type text that doesn't match any option
		tm.Type("xyz")
		time.Sleep(100 * time.Millisecond)

		// Verify no matches state
		assert.Equal(t, 0, len(f.filtered), "filter should have no matches for 'xyz'")

		// Verify the view shows the no matches message with the custom value hint
		view := f.View()
		assert.Contains(t, view, "No matches")
		assert.Contains(t, view, "xyz", "should show the typed value in the message")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})

	t.Run("shows plain no matches for empty filtered input", func(t *testing.T) {
		f := NewFilterable("fruit", "Select fruit", "", options)
		f.Focus()
		f.textInput.Focus()

		// Set value then clear it, simulating backspace scenario
		f.textInput.SetValue("xyz")
		f.filterOptions()

		// Verify we have no matches with non-empty input
		assert.Equal(t, 0, len(f.filtered))
		view := f.View()
		assert.Contains(t, view, "No matches")
		assert.Contains(t, view, `"xyz"`)

		// Now clear the input
		f.textInput.SetValue("")
		f.filterOptions()

		// With empty input, should show all options again
		assert.Equal(t, 3, len(f.filtered), "empty input should show all options")
	})
}

// TestFilterable_CustomValue_EnterSubmitsTypedText verifies that pressing Enter
// when there are no matches uses the typed text as a custom value.
func TestFilterable_CustomValue_EnterSubmitsTypedText(t *testing.T) {
	options := []Option{
		{Label: "Apple", Value: "apple"},
		{Label: "Banana", Value: "banana"},
	}

	t.Run("enter submits custom value when no matches", func(t *testing.T) {
		f := NewFilterable("fruit", "Select fruit", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select fruit"))
		}, teatest.WithDuration(time.Second))

		// Verify field is not complete
		assert.False(t, f.IsComplete(), "should not be complete before Enter")

		// Type a custom value that doesn't match any option
		tm.Type("Orange")
		time.Sleep(100 * time.Millisecond)

		// Verify no matches
		assert.Equal(t, 0, len(f.filtered), "should have no matches for 'Orange'")

		// Press Enter to submit custom value
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify custom value was submitted
		assert.True(t, f.IsComplete(), "should be complete after Enter")
		assert.Equal(t, "Orange", f.GetValue(), "GetValue() should return the custom typed value")
	})

	t.Run("custom value works with partial match that becomes no match", func(t *testing.T) {
		f := NewFilterable("fruit", "Select fruit", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select fruit"))
		}, teatest.WithDuration(time.Second))

		// Type 'a' - should match Apple
		tm.Type("a")
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, 2, len(f.filtered), "should match Apple and Banana")

		// Type more to make it not match anything
		tm.Type("xyz")
		time.Sleep(100 * time.Millisecond)
		assert.Equal(t, 0, len(f.filtered), "should have no matches for 'axyz'")

		// Press Enter to submit custom value
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify custom value was submitted
		assert.True(t, f.IsComplete(), "should be complete")
		assert.Equal(t, "axyz", f.GetValue(), "GetValue() should return 'axyz'")
	})
}

// TestFilterable_CustomValue_IsTrimmed verifies that custom values are trimmed
// of leading and trailing whitespace before being stored.
func TestFilterable_CustomValue_IsTrimmed(t *testing.T) {
	options := []Option{
		{Label: "Option A", Value: "a"},
	}

	t.Run("leading and trailing spaces are trimmed", func(t *testing.T) {
		f := NewFilterable("custom", "Enter value", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter value"))
		}, teatest.WithDuration(time.Second))

		// Type value with leading and trailing spaces
		tm.Type("  custom value  ")
		time.Sleep(100 * time.Millisecond)

		// Verify no matches (so custom value will be used)
		assert.Equal(t, 0, len(f.filtered), "should have no matches")

		// Press Enter to submit
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify value is trimmed
		assert.True(t, f.IsComplete(), "should be complete")
		assert.Equal(t, "custom value", f.GetValue(), "GetValue() should return trimmed value")
	})

	t.Run("internal spaces are preserved", func(t *testing.T) {
		f := NewFilterable("custom", "Enter value", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Enter value"))
		}, teatest.WithDuration(time.Second))

		// Type value with internal spaces
		tm.Type("hello   world")
		time.Sleep(100 * time.Millisecond)

		// Press Enter to submit
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify internal spaces preserved
		assert.Equal(t, "hello   world", f.GetValue(), "internal spaces should be preserved")
	})
}

// =============================================================================
// TT-010: Filterable arrow key navigation tests
// =============================================================================

// TestFilterable_ArrowNavigation_UpMovesUp verifies that pressing Up arrow
// moves the cursor up in the filtered list.
func TestFilterable_ArrowNavigation_UpMovesUp(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	f := NewFilterable("test", "Select option", "", options)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select option"))
	}, teatest.WithDuration(time.Second))

	// Initial cursor should be at 0
	assert.Equal(t, 0, f.cursor, "initial cursor should be at 0")

	// Move down first so we can test moving up
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, f.cursor, "cursor should be at 1 after down")

	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, f.cursor, "cursor should be at 2 after second down")

	// Now test Up arrow
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, f.cursor, "cursor should move to 1 after up arrow")

	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, f.cursor, "cursor should move to 0 after second up arrow")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFilterable_ArrowNavigation_DownMovesDown verifies that pressing Down arrow
// moves the cursor down in the filtered list.
func TestFilterable_ArrowNavigation_DownMovesDown(t *testing.T) {
	options := []Option{
		{Label: "Alpha", Value: "a"},
		{Label: "Beta", Value: "b"},
		{Label: "Gamma", Value: "g"},
		{Label: "Delta", Value: "d"},
	}

	f := NewFilterable("test", "Select option", "", options)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select option"))
	}, teatest.WithDuration(time.Second))

	// Initial cursor should be at 0
	assert.Equal(t, 0, f.cursor, "initial cursor should be at 0")

	// Press Down arrow - cursor should move to 1
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, f.cursor, "cursor should be at 1 after first down")

	// Press Down arrow again - cursor should move to 2
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 2, f.cursor, "cursor should be at 2 after second down")

	// Press Down arrow again - cursor should move to 3
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 3, f.cursor, "cursor should be at 3 after third down")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFilterable_ArrowNavigation_WrapTopToBottom verifies that pressing Up arrow
// at the first item wraps to the last item.
func TestFilterable_ArrowNavigation_WrapTopToBottom(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	f := NewFilterable("test", "Select option", "", options)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select option"))
	}, teatest.WithDuration(time.Second))

	// Initial cursor should be at 0 (first item)
	assert.Equal(t, 0, f.cursor, "initial cursor should be at 0")

	// Press Up arrow at first item - should wrap to last item (index 2)
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 2, f.cursor, "cursor should wrap to last item (index 2) after up at first")
	assert.Equal(t, "Third", f.filtered[f.cursor].Label, "cursor should be on Third after wrap")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFilterable_ArrowNavigation_WrapBottomToTop verifies that pressing Down arrow
// at the last item wraps to the first item.
func TestFilterable_ArrowNavigation_WrapBottomToTop(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	f := NewFilterable("test", "Select option", "", options)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select option"))
	}, teatest.WithDuration(time.Second))

	// Move to last item
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 2, f.cursor, "cursor should be at last item (index 2)")
	assert.Equal(t, "Third", f.filtered[f.cursor].Label, "cursor should be on Third")

	// Press Down arrow at last item - should wrap to first item (index 0)
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, 0, f.cursor, "cursor should wrap to first item (index 0) after down at last")
	assert.Equal(t, "First", f.filtered[f.cursor].Label, "cursor should be on First after wrap")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFilterable_ArrowNavigation_ViewportScrollsDown verifies that the viewport
// scrolls when cursor moves below visible area.
func TestFilterable_ArrowNavigation_ViewportScrollsDown(t *testing.T) {
	// Create many options to force scrolling
	options := make([]Option, 20)
	for i := range options {
		options[i] = Option{
			Label: fmt.Sprintf("Option %02d", i+1),
			Value: fmt.Sprintf("%d", i+1),
		}
	}

	f := NewFilterable("test", "Select option", "", options)
	// Set a small height to force viewport scrolling
	f.WithHeight(10) // With header lines, this allows only ~5 visible items
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 10))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render - look for the first option
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Option 01"))
	}, teatest.WithDuration(time.Second))

	// Initial viewport offset should be 0
	assert.Equal(t, 0, f.viewportOffset, "initial viewport offset should be 0")
	assert.Equal(t, 0, f.cursor, "initial cursor should be 0")

	// Move down multiple times to force scrolling
	for i := 0; i < 8; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(30 * time.Millisecond)
	}

	// Cursor should be at position 8
	assert.Equal(t, 8, f.cursor, "cursor should be at position 8 after 8 down presses")

	// Viewport should have scrolled to keep cursor visible
	// The exact offset depends on visibleItemCount(), but it should be > 0
	assert.Greater(t, f.viewportOffset, 0, "viewport should have scrolled down")
	assert.LessOrEqual(t, f.viewportOffset, f.cursor, "cursor should be >= viewport offset")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFilterable_ArrowNavigation_ViewportScrollsUp verifies that the viewport
// scrolls when cursor moves above visible area.
func TestFilterable_ArrowNavigation_ViewportScrollsUp(t *testing.T) {
	// Create many options to force scrolling
	options := make([]Option, 20)
	for i := range options {
		options[i] = Option{
			Label: fmt.Sprintf("Option %02d", i+1),
			Value: fmt.Sprintf("%d", i+1),
		}
	}

	f := NewFilterable("test", "Select option", "", options)
	// Set a small height to force viewport scrolling
	f.WithHeight(10)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 10))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render - look for the first option
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Option 01"))
	}, teatest.WithDuration(time.Second))

	// Move down multiple times to scroll viewport down
	for i := 0; i < 10; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(30 * time.Millisecond)
	}

	// Record the viewport offset after scrolling down
	viewportAfterDown := f.viewportOffset
	assert.Greater(t, viewportAfterDown, 0, "viewport should have scrolled down")

	// Now move up multiple times
	for i := 0; i < 8; i++ {
		tm.Send(tea.KeyMsg{Type: tea.KeyUp})
		time.Sleep(30 * time.Millisecond)
	}

	// Cursor should be at position 2 (10 - 8)
	assert.Equal(t, 2, f.cursor, "cursor should be at position 2 after moving up")

	// Viewport should have scrolled up to keep cursor visible
	assert.Less(t, f.viewportOffset, viewportAfterDown, "viewport should have scrolled up")
	assert.LessOrEqual(t, f.viewportOffset, f.cursor, "viewport should show cursor")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFilterable_ArrowNavigation_CtrlUpDown verifies that Ctrl+K and Ctrl+J
// also work as alternative up/down navigation keys.
func TestFilterable_ArrowNavigation_CtrlUpDown(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	t.Run("ctrl+j moves down", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		assert.Equal(t, 0, f.cursor, "initial cursor should be 0")

		// Ctrl+J should move down
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlJ})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, f.cursor, "cursor should move to 1 after Ctrl+J")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})

	t.Run("ctrl+k moves up", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Move down first
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 2, f.cursor, "cursor should be at 2")

		// Ctrl+K should move up
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlK})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, f.cursor, "cursor should move to 1 after Ctrl+K")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
}

// TestFilterable_ArrowNavigation_EmptyList verifies that navigation doesn't
// crash or misbehave with an empty options list.
func TestFilterable_ArrowNavigation_EmptyList(t *testing.T) {
	f := NewFilterable("test", "Select option", "", []Option{})
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select option"))
	}, teatest.WithDuration(time.Second))

	// With empty list, cursor should be -1 or operations should be no-ops
	initialCursor := f.cursor

	// These should not crash or change state dramatically
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	time.Sleep(50 * time.Millisecond)

	// Verify no crash occurred and we can still quit
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Cursor should remain unchanged with empty list
	assert.Equal(t, initialCursor, f.cursor, "cursor should not change with empty list")
}

// TestFilterable_ArrowNavigation_SingleOption verifies navigation behavior
// with just a single option.
func TestFilterable_ArrowNavigation_SingleOption(t *testing.T) {
	options := []Option{
		{Label: "Only Option", Value: "only"},
	}

	f := NewFilterable("test", "Select option", "", options)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select option"))
	}, teatest.WithDuration(time.Second))

	assert.Equal(t, 0, f.cursor, "initial cursor should be 0")

	// Down should wrap back to 0
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, f.cursor, "cursor should wrap to 0 with single option")

	// Up should also wrap back to 0
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, f.cursor, "cursor should wrap to 0 with single option")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFilterable_ArrowNavigation_AfterFiltering verifies that navigation works
// correctly after filtering reduces the options list.
func TestFilterable_ArrowNavigation_AfterFiltering(t *testing.T) {
	options := []Option{
		{Label: "Apple", Value: "apple"},
		{Label: "Apricot", Value: "apricot"},
		{Label: "Banana", Value: "banana"},
		{Label: "Cherry", Value: "cherry"},
	}

	f := NewFilterable("fruit", "Select fruit", "", options)
	model := newFieldModel(f, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select fruit"))
	}, teatest.WithDuration(time.Second))

	// Initially 4 options
	assert.Equal(t, 4, len(f.filtered), "should have 4 options initially")

	// Type 'ap' to filter to Apple and Apricot
	tm.Type("ap")
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 2, len(f.filtered), "should have 2 options after filtering")
	assert.Equal(t, 0, f.cursor, "cursor should reset to 0 after filter")

	// Navigate down
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, f.cursor, "cursor should be at 1")
	assert.Equal(t, "Apricot", f.filtered[f.cursor].Label, "should be on Apricot")

	// Navigate down again - should wrap to 0
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 0, f.cursor, "cursor should wrap to 0")
	assert.Equal(t, "Apple", f.filtered[f.cursor].Label, "should be on Apple")

	// Navigate up - should wrap to 1 (last in filtered list)
	tm.Send(tea.KeyMsg{Type: tea.KeyUp})
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, 1, f.cursor, "cursor should wrap to 1")
	assert.Equal(t, "Apricot", f.filtered[f.cursor].Label, "should be on Apricot")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestFilterable_CustomValue_EmptyInputHandling verifies that empty input
// is handled gracefully when pressing Enter.
func TestFilterable_CustomValue_EmptyInputHandling(t *testing.T) {
	options := []Option{
		{Label: "Option A", Value: "a"},
		{Label: "Option B", Value: "b"},
	}

	t.Run("empty input selects first option", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Verify options are shown (empty input shows all)
		assert.Equal(t, 2, len(f.filtered), "should show all options with empty input")

		// Press Enter without typing anything - should select first option
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify first option was selected (not empty string)
		assert.True(t, f.IsComplete(), "should be complete")
		assert.Equal(t, "a", f.GetValue(), "should select first option value when input is empty")
	})

	t.Run("whitespace-only input with matches selects highlighted option", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", options)
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Type only spaces - this still shows all options since trim is applied to filter
		tm.Type("   ")
		time.Sleep(100 * time.Millisecond)

		// Move to second option
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)

		// Press Enter - should select highlighted option
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify second option was selected
		assert.True(t, f.IsComplete(), "should be complete")
		assert.Equal(t, "b", f.GetValue(), "should select second option")
	})

	t.Run("empty options list with empty input returns empty string", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", []Option{})
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Press Enter with no options and no input
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify empty string result
		assert.True(t, f.IsComplete(), "should be complete")
		assert.Equal(t, "", f.GetValue(), "should return empty string with no options and no input")
	})

	t.Run("empty options list with custom input uses custom value", func(t *testing.T) {
		f := NewFilterable("test", "Select option", "", []Option{})
		model := newFieldModel(f, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Type custom value
		tm.Type("custom")
		time.Sleep(100 * time.Millisecond)

		// Press Enter
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify custom value used
		assert.True(t, f.IsComplete(), "should be complete")
		assert.Equal(t, "custom", f.GetValue(), "should return custom value")
	})
}

// =============================================================================
// TT-011: Selector Enter selection tests
// =============================================================================

// TestSelector_EnterSelectsHighlightedOption verifies that pressing Enter
// selects the currently highlighted option in the Selector.
func TestSelector_EnterSelectsHighlightedOption(t *testing.T) {
	options := []Option{
		{Label: "Option Alpha", Value: "alpha"},
		{Label: "Option Beta", Value: "beta"},
		{Label: "Option Gamma", Value: "gamma"},
	}

	t.Run("selects first option by default", func(t *testing.T) {
		s := NewSelector("test", "Select option", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Verify initial cursor position is 0
		assert.Equal(t, 0, s.cursor, "initial cursor should be at index 0")
		assert.False(t, s.IsComplete(), "should not be complete before Enter")

		// Press Enter to select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify selected value is the first option
		assert.Equal(t, "alpha", s.selected, "should select Option Alpha's value")
		assert.Equal(t, "alpha", s.GetValue(), "GetValue() should return selected value")
		assert.True(t, s.IsComplete(), "IsComplete() should be true after Enter")
	})

	t.Run("selects second option after down navigation", func(t *testing.T) {
		s := NewSelector("test", "Select option", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select option"))
		}, teatest.WithDuration(time.Second))

		// Navigate to second option
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, s.cursor, "cursor should be at index 1 after down press")

		// Press Enter to select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify selected value is the second option
		assert.Equal(t, "beta", s.selected, "should select Option Beta's value")
		assert.Equal(t, "beta", s.GetValue(), "GetValue() should return selected value")
		assert.True(t, s.IsComplete(), "IsComplete() should be true after Enter")
	})

	t.Run("selects third option after two down presses", func(t *testing.T) {
		s := NewSelector("test", "Select option", options)
		model := newFieldModel(s, 10)

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

		assert.Equal(t, 2, s.cursor, "cursor should be at index 2 after two down presses")

		// Press Enter to select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify selected value is the third option
		assert.Equal(t, "gamma", s.selected, "should select Option Gamma's value")
		assert.Equal(t, "gamma", s.GetValue(), "GetValue() should return selected value")
		assert.True(t, s.IsComplete(), "IsComplete() should be true after Enter")
	})
}

// TestSelector_SelectedValueMatchesOptionValue verifies that the selected value
// matches the option's Value field, not the Label.
func TestSelector_SelectedValueMatchesOptionValue(t *testing.T) {
	options := []Option{
		{Label: "Display Name One", Value: "value-1"},
		{Label: "Display Name Two", Value: "value-2"},
		{Label: "Display Name Three", Value: "value-3"},
	}

	s := NewSelector("items", "Select item", options)
	model := newFieldModel(s, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select item"))
	}, teatest.WithDuration(time.Second))

	// Navigate to second option and select
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify value is the Value field, not Label
	assert.Equal(t, "value-2", s.GetValue(), "GetValue() should return Value field, not Label")
	assert.NotEqual(t, "Display Name Two", s.GetValue(), "should not return the Label")
}

// TestSelector_NextStepMsgSentOnEnter verifies that NextStepMsg is sent
// when Enter is pressed, causing the fieldModel to quit (via IsComplete check).
func TestSelector_NextStepMsgSentOnEnter(t *testing.T) {
	options := []Option{
		{Label: "Only Option", Value: "only"},
	}

	s := NewSelector("test", "Select", options)
	model := newFieldModel(s, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select"))
	}, teatest.WithDuration(time.Second))

	// Verify field is not complete before Enter
	assert.False(t, s.IsComplete(), "should not be complete before Enter")

	// Press Enter - this should trigger NextStepMsg which causes fieldModel to quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// The model should finish because fieldModel.Update checks IsComplete()
	// and returns tea.Quit when the field completes
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify completion state
	assert.True(t, s.IsComplete(), "IsComplete() should be true, indicating NextStepMsg flow")
	assert.Equal(t, "only", s.GetValue(), "selected value should be stored")
}

// TestSelector_IsCompleteAfterSelection verifies that IsComplete() returns
// true after selection and false before.
func TestSelector_IsCompleteAfterSelection(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
	}

	s := NewSelector("field", "Choose one", options)
	model := newFieldModel(s, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Choose one"))
	}, teatest.WithDuration(time.Second))

	// Initial state
	assert.False(t, s.IsComplete(), "IsComplete() should be false initially")
	assert.False(t, s.IsCancelled(), "IsCancelled() should be false")
	assert.Equal(t, "", s.GetValue(), "GetValue() should be empty before selection")

	// Navigate and check - still not complete
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	time.Sleep(50 * time.Millisecond)
	assert.False(t, s.IsComplete(), "IsComplete() should still be false before Enter")

	// Submit
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Now complete
	assert.True(t, s.IsComplete(), "IsComplete() should be true after selection")
	assert.False(t, s.IsCancelled(), "IsCancelled() should still be false")
	assert.Equal(t, "2", s.GetValue(), "GetValue() should return selected value")
}

// TestSelector_EmptyOptionsDoesNotCrash verifies that pressing Enter
// with an empty options list doesn't crash and doesn't mark as complete.
func TestSelector_EmptyOptionsDoesNotCrash(t *testing.T) {
	s := NewSelector("test", "Select", []Option{})
	model := newFieldModel(s, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Select"))
	}, teatest.WithDuration(time.Second))

	// Press Enter - should not crash and should not complete
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	time.Sleep(50 * time.Millisecond)

	// Verify not complete (no options to select)
	assert.False(t, s.IsComplete(), "should not be complete with no options")
	assert.Equal(t, "", s.GetValue(), "GetValue() should be empty")

	// Quit cleanly
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

// TestSelector_NavigationAfterWrapping verifies Enter works correctly
// after cursor has wrapped around the options list.
func TestSelector_NavigationAfterWrapping(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	t.Run("select after wrapping from bottom to top", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// Move to last item
		tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // 1
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // 2
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 2, s.cursor, "should be at index 2")

		// Wrap to first
		tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // wraps to 0
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 0, s.cursor, "should wrap to index 0")

		// Select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.Equal(t, "1", s.GetValue(), "should select first option after wrap")
	})

	t.Run("select after wrapping from top to bottom", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// At first item, wrap to last
		tm.Send(tea.KeyMsg{Type: tea.KeyUp}) // wraps to 2
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 2, s.cursor, "should wrap to index 2")

		// Select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.Equal(t, "3", s.GetValue(), "should select third option after wrap")
	})
}

// =============================================================================
// TT-012: Selector vim key navigation tests
// =============================================================================

// TestSelector_JKeyMovesCursorDown verifies that pressing 'j' moves the cursor
// down in vim-style navigation.
func TestSelector_JKeyMovesCursorDown(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	t.Run("j moves cursor down one position", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// Verify initial cursor is at 0
		assert.Equal(t, 0, s.cursor, "initial cursor should be at index 0")

		// Press 'j' to move down
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, s.cursor, "cursor should move to index 1 after j press")

		// Press 'j' again
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 2, s.cursor, "cursor should move to index 2 after second j press")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})

	t.Run("j wraps from bottom to top", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// Move to last item
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // 1
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // 2
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 2, s.cursor, "should be at last index")

		// Press j to wrap
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 0, s.cursor, "j should wrap from bottom to top")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
}

// TestSelector_KKeyMovesCursorUp verifies that pressing 'k' moves the cursor
// up in vim-style navigation.
func TestSelector_KKeyMovesCursorUp(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	t.Run("k moves cursor up one position", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// First move down to have room to go up
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // 1
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // 2
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 2, s.cursor, "should be at index 2")

		// Press 'k' to move up
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, s.cursor, "cursor should move to index 1 after k press")

		// Press 'k' again
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 0, s.cursor, "cursor should move to index 0 after second k press")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})

	t.Run("k wraps from top to bottom", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// At first item, press k to wrap
		assert.Equal(t, 0, s.cursor, "should start at index 0")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 2, s.cursor, "k should wrap from top to bottom")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
}

// TestSelector_CtrlJMovesCursorDown verifies that Ctrl+J moves the cursor down.
func TestSelector_CtrlJMovesCursorDown(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	t.Run("ctrl+j moves cursor down", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		assert.Equal(t, 0, s.cursor, "initial cursor should be at index 0")

		// Press Ctrl+J to move down
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlJ})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, s.cursor, "cursor should move to index 1 after ctrl+j")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})

	t.Run("ctrl+j wraps from bottom to top", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// Move to last position
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlJ}) // 1
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlJ}) // 2
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 2, s.cursor, "should be at last index")

		// Wrap
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlJ})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 0, s.cursor, "ctrl+j should wrap from bottom to top")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
}

// TestSelector_CtrlKMovesCursorUp verifies that Ctrl+K moves the cursor up.
func TestSelector_CtrlKMovesCursorUp(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	t.Run("ctrl+k moves cursor up", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// Move down first
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
		time.Sleep(50 * time.Millisecond)
		assert.Equal(t, 2, s.cursor, "should be at index 2")

		// Press Ctrl+K to move up
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlK})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, s.cursor, "cursor should move to index 1 after ctrl+k")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})

	t.Run("ctrl+k wraps from top to bottom", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		assert.Equal(t, 0, s.cursor, "should start at index 0")

		// Press Ctrl+K to wrap
		tm.Send(tea.KeyMsg{Type: tea.KeyCtrlK})
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 2, s.cursor, "ctrl+k should wrap from top to bottom")

		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
	})
}

// TestSelector_VimKeysThenSelect verifies that vim keys can be used to navigate
// and then Enter selects the correct option.
func TestSelector_VimKeysThenSelect(t *testing.T) {
	options := []Option{
		{Label: "First", Value: "1"},
		{Label: "Second", Value: "2"},
		{Label: "Third", Value: "3"},
	}

	t.Run("j navigation then enter selects correct option", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// Navigate with j to second option
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
		time.Sleep(50 * time.Millisecond)

		// Select with Enter
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, s.IsComplete(), "should be complete after enter")
		assert.Equal(t, "2", s.GetValue(), "should select second option")
	})

	t.Run("k navigation then enter selects correct option", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// Navigate with k to wrap to last option
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
		time.Sleep(50 * time.Millisecond)

		// Select with Enter
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, s.IsComplete(), "should be complete after enter")
		assert.Equal(t, "3", s.GetValue(), "should select third option after k wrap")
	})

	t.Run("mixed vim and arrow navigation then select", func(t *testing.T) {
		s := NewSelector("test", "Select", options)
		model := newFieldModel(s, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Select"))
		}, teatest.WithDuration(time.Second))

		// Mix j and arrow keys
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")}) // 1
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyDown}) // 2
		time.Sleep(50 * time.Millisecond)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")}) // 1
		time.Sleep(50 * time.Millisecond)

		assert.Equal(t, 1, s.cursor, "cursor should be at index 1")

		// Select
		tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.Equal(t, "2", s.GetValue(), "should select second option")
	})
}

// =============================================================================
// TT-016: Confirm y/n shortcut keys tests
// =============================================================================

// TestConfirm_YKeyImmediatelyConfirmsTrue verifies that pressing 'y' immediately
// confirms with true value without needing to press Enter.
func TestConfirm_YKeyImmediatelyConfirmsTrue(t *testing.T) {
	t.Run("lowercase y confirms with true", func(t *testing.T) {
		c := NewConfirm("proceed", "Do you want to proceed?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		// Wait for initial render
		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Do you want to proceed?"))
		}, teatest.WithDuration(time.Second))

		// Verify initial state
		assert.False(t, c.IsComplete(), "should not be complete before y press")

		// Press 'y' to immediately confirm
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

		// Model should quit on completion
		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		// Verify confirmation
		assert.True(t, c.IsComplete(), "IsComplete() should be true after y press")
		assert.False(t, c.IsCancelled(), "IsCancelled() should be false")
		assert.Equal(t, true, c.GetValue(), "GetValue() should return true")
	})

	t.Run("uppercase Y confirms with true", func(t *testing.T) {
		c := NewConfirm("proceed", "Continue?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Continue?"))
		}, teatest.WithDuration(time.Second))

		// Press 'Y' (uppercase)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("Y")})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, c.IsComplete(), "IsComplete() should be true after Y press")
		assert.False(t, c.IsCancelled(), "IsCancelled() should be false")
		assert.Equal(t, true, c.GetValue(), "GetValue() should return true")
	})
}

// TestConfirm_NKeyImmediatelyCancelsWithFalse verifies that pressing 'n'
// immediately cancels with false value and sends CancelMsg.
func TestConfirm_NKeyImmediatelyCancelsWithFalse(t *testing.T) {
	t.Run("lowercase n cancels with false", func(t *testing.T) {
		c := NewConfirm("proceed", "Do you want to proceed?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Do you want to proceed?"))
		}, teatest.WithDuration(time.Second))

		assert.False(t, c.IsComplete(), "should not be complete before n press")

		// Press 'n' to immediately cancel
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, c.IsComplete(), "IsComplete() should be true after n press")
		assert.True(t, c.IsCancelled(), "IsCancelled() should be true")
		assert.Equal(t, false, c.GetValue(), "GetValue() should return false")
	})

	t.Run("uppercase N cancels with false", func(t *testing.T) {
		c := NewConfirm("proceed", "Continue?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Continue?"))
		}, teatest.WithDuration(time.Second))

		// Press 'N' (uppercase)
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, c.IsComplete(), "IsComplete() should be true after N press")
		assert.True(t, c.IsCancelled(), "IsCancelled() should be true")
		assert.Equal(t, false, c.GetValue(), "GetValue() should return false")
	})
}

// TestConfirm_YNShortcutsIgnoreCurrentSelection verifies that y/n shortcuts
// work regardless of which button is currently selected.
func TestConfirm_YNShortcutsIgnoreCurrentSelection(t *testing.T) {
	t.Run("y confirms true even when No is selected", func(t *testing.T) {
		c := NewConfirm("test", "Proceed?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Proceed?"))
		}, teatest.WithDuration(time.Second))

		// Navigate to No
		tm.Send(tea.KeyMsg{Type: tea.KeyRight})
		time.Sleep(50 * time.Millisecond)
		assert.False(t, c.selected, "No should be selected")

		// Press 'y' - should still confirm with true
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, c.IsComplete(), "should be complete")
		assert.False(t, c.IsCancelled(), "should not be cancelled")
		assert.Equal(t, true, c.GetValue(), "y should confirm true regardless of selection")
	})

	t.Run("n cancels false even when Yes is selected", func(t *testing.T) {
		c := NewConfirm("test", "Proceed?")
		model := newFieldModel(c, 10)

		tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
		t.Cleanup(func() { _ = tm.Quit() })

		teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
			return bytes.Contains(bts, []byte("Proceed?"))
		}, teatest.WithDuration(time.Second))

		// Yes is selected by default
		assert.True(t, c.selected, "Yes should be selected by default")

		// Press 'n' - should still cancel with false
		tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

		tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

		assert.True(t, c.IsComplete(), "should be complete")
		assert.True(t, c.IsCancelled(), "should be cancelled")
		assert.Equal(t, false, c.GetValue(), "n should cancel false regardless of selection")
	})
}

// TestConfirm_CancelMsgSentOnNKey verifies that CancelMsg is sent when pressing
// 'n' or 'N' keys.
func TestConfirm_CancelMsgSentOnNKey(t *testing.T) {
	// This test verifies CancelMsg behavior by checking IsCancelled() state
	// which is only set when CancelMsg would be sent (lines 77-81 in confirm.go)
	c := NewConfirm("test", "Delete file?")
	model := newFieldModel(c, 10)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() { _ = tm.Quit() })

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Delete file?"))
	}, teatest.WithDuration(time.Second))

	// IsCancelled should be false initially
	assert.False(t, c.IsCancelled(), "IsCancelled() should be false initially")

	// Press 'n' - this triggers CancelMsg
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify CancelMsg was sent (indicated by IsCancelled being true)
	assert.True(t, c.IsCancelled(), "IsCancelled() should be true after n press (CancelMsg sent)")
	assert.True(t, c.IsComplete(), "IsComplete() should also be true")
}
