package config

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFieldRow(ft FieldType) *FieldRow {
	meta := FieldMeta{
		Key:   "test.field",
		Label: "Test Field",
		Type:  ft,
	}
	fr := NewFieldRow(meta, nil)
	fr.SetWidth(72)
	fr.SetLabelWidth(15)
	return fr
}

func TestFieldRow_BrowsingRendersLabelAndValue(t *testing.T) {
	testCases := []struct {
		value  any
		assert func(t *testing.T, view string)
		name   string
		ft     FieldType
	}{
		{
			name:  "string value",
			ft:    String,
			value: "main",
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Test Field")
				assert.Contains(t, view, "main")
			},
		},
		{
			name:  "int value",
			ft:    Int,
			value: 42,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Test Field")
				assert.Contains(t, view, "42")
			},
		},
		{
			name:  "int64 value",
			ft:    Int,
			value: int64(99),
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "99")
			},
		},
		{
			name:  "bool true",
			ft:    Bool,
			value: true,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "yes")
			},
		},
		{
			name:  "bool false",
			ft:    Bool,
			value: false,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "no")
			},
		},
		{
			name:  "string list",
			ft:    StringList,
			value: []string{"a", "b", "c"},
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "a, b, c")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newTestFieldRow(tc.ft)
			fr.SetValue(tc.value)
			view := fr.View()
			tc.assert(t, view)
		})
	}
}

func TestFieldRow_FocusIndicator(t *testing.T) {
	testCases := []struct {
		assert  func(t *testing.T, view string)
		name    string
		focused bool
	}{
		{
			name:    "focused shows cursor",
			focused: true,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, ">")
			},
		},
		{
			name:    "unfocused no cursor",
			focused: false,
			assert: func(t *testing.T, view string) {
				t.Helper()
				// The view should start with spaces, not ">"
				stripped := stripAnsi(view)
				assert.Equal(t, byte(' '), stripped[0], "unfocused row should not start with >")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newTestFieldRow(String)
			fr.SetValue("value")
			fr.SetFocused(tc.focused)
			view := fr.View()
			tc.assert(t, view)
		})
	}
}

func TestFieldRow_DirtyMarker(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, view string)
		name   string
		dirty  bool
	}{
		{
			name:  "dirty shows asterisk",
			dirty: true,
			assert: func(t *testing.T, view string) {
				t.Helper()
				stripped := stripAnsi(view)
				assert.Contains(t, stripped, "*")
			},
		},
		{
			name:  "clean no asterisk",
			dirty: false,
			assert: func(t *testing.T, view string) {
				t.Helper()
				stripped := stripAnsi(view)
				// Should not contain * in prefix area (first 4 chars)
				prefix := stripped[:4]
				assert.NotContains(t, prefix, "*")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newTestFieldRow(String)
			fr.SetValue("value")
			fr.SetDirty(tc.dirty)
			view := fr.View()
			tc.assert(t, view)
		})
	}
}

func TestFieldRow_BoolStyling(t *testing.T) {
	t.Run("true renders yes", func(t *testing.T) {
		fr := newTestFieldRow(Bool)
		fr.SetValue(true)
		view := fr.View()
		assert.Contains(t, view, "yes")
	})

	t.Run("false renders no", func(t *testing.T) {
		fr := newTestFieldRow(Bool)
		fr.SetValue(false)
		view := fr.View()
		assert.Contains(t, view, "no")
	})
}

func TestFieldRow_SensitiveMasking(t *testing.T) {
	t.Run("unfocused masks value", func(t *testing.T) {
		fr := newTestFieldRow(SensitiveString)
		fr.SetValue("secret123")
		fr.SetFocused(false)
		view := fr.View()
		assert.Contains(t, view, "********")
		assert.NotContains(t, view, "secret123")
	})

	t.Run("focused reveals value", func(t *testing.T) {
		fr := newTestFieldRow(SensitiveString)
		fr.SetValue("secret123")
		fr.SetFocused(true)
		view := fr.View()
		assert.Contains(t, view, "secret123")
	})
}

func TestFieldRow_EmptyValueRendersPlaceholder(t *testing.T) {
	testCases := []struct {
		value any
		name  string
		ft    FieldType
	}{
		{name: "nil value", ft: String, value: nil},
		{name: "empty string", ft: String, value: ""},
		{name: "empty string list", ft: StringList, value: []string{}},
		{name: "nil string list", ft: StringList, value: nil},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newTestFieldRow(tc.ft)
			fr.SetValue(tc.value)
			view := fr.View()
			assert.Contains(t, view, "--")
		})
	}
}

func TestFieldRow_EnterEditingString(t *testing.T) {
	fr := newTestFieldRow(String)
	fr.SetValue("original")

	cmd := fr.EnterEditing()

	assert.Equal(t, FieldEditing, fr.State())
	assert.NotNil(t, cmd, "should return blink command")
}

func TestFieldRow_EnterEditingBoolIsNoOp(t *testing.T) {
	fr := newTestFieldRow(Bool)
	fr.SetValue(true)

	cmd := fr.EnterEditing()

	assert.Equal(t, FieldBrowsing, fr.State(), "bool should not enter editing")
	assert.Nil(t, cmd)
}

func TestFieldRow_ToggleBool(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, result any, err error)
		name   string
		start  bool
	}{
		{
			name:  "true toggles to false",
			start: true,
			assert: func(t *testing.T, result any, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, false, result)
			},
		},
		{
			name:  "false toggles to true",
			start: false,
			assert: func(t *testing.T, result any, err error) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, true, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newTestFieldRow(Bool)
			fr.SetValue(tc.start)
			result, err := fr.ToggleBool()
			tc.assert(t, result, err)
		})
	}
}

func TestFieldRow_ToggleBoolOnNonBoolErrors(t *testing.T) {
	fr := newTestFieldRow(String)
	fr.SetValue("hello")
	_, err := fr.ToggleBool()
	assert.Error(t, err)
}

func TestFieldRow_CommitEditingSuccess(t *testing.T) {
	testCases := []struct {
		value  any
		assert func(t *testing.T, result any, err error, fr *FieldRow)
		name   string
		input  string
		ft     FieldType
	}{
		{
			name:  "string commit",
			ft:    String,
			value: "old",
			input: "new",
			assert: func(t *testing.T, result any, err error, fr *FieldRow) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, "new", result)
				assert.Equal(t, FieldBrowsing, fr.State())
			},
		},
		{
			name:  "int commit",
			ft:    Int,
			value: 10,
			input: "42",
			assert: func(t *testing.T, result any, err error, fr *FieldRow) {
				t.Helper()
				require.NoError(t, err)
				assert.Equal(t, 42, result)
				assert.Equal(t, FieldBrowsing, fr.State())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newTestFieldRow(tc.ft)
			fr.SetValue(tc.value)
			fr.EnterEditing()
			fr.input.SetValue(tc.input)
			result, err := fr.CommitEditing()
			tc.assert(t, result, err, fr)
		})
	}
}

func TestFieldRow_CommitEditingValidationError(t *testing.T) {
	meta := FieldMeta{
		Key:      "test.field",
		Label:    "Test Field",
		Type:     Int,
		Validate: ValidatePositiveInt,
	}
	fr := NewFieldRow(meta, nil)
	fr.SetWidth(72)
	fr.SetLabelWidth(15)
	fr.SetValue(10)
	fr.EnterEditing()
	fr.input.SetValue("-5")

	result, err := fr.CommitEditing()

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, FieldEditing, fr.State(), "should stay in editing on error")
}

func TestFieldRow_CommitEditingCoercionError(t *testing.T) {
	fr := newTestFieldRow(Int)
	fr.SetValue(10)
	fr.EnterEditing()
	fr.input.SetValue("not-a-number")

	result, err := fr.CommitEditing()

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, FieldEditing, fr.State(), "should stay in editing on coercion error")
}

func TestFieldRow_CancelEditing(t *testing.T) {
	fr := newTestFieldRow(String)
	fr.SetValue("original")
	fr.EnterEditing()
	require.Equal(t, FieldEditing, fr.State())

	fr.CancelEditing()

	assert.Equal(t, FieldBrowsing, fr.State())
}

func TestFieldRow_EditingViewShowsInput(t *testing.T) {
	fr := newTestFieldRow(String)
	fr.SetValue("hello")
	fr.EnterEditing()

	view := fr.View()

	assert.Contains(t, view, ">")
	assert.Contains(t, view, "Test Field")
}

func TestFieldRow_EditingViewShowsDescription(t *testing.T) {
	meta := FieldMeta{
		Key:         "test.field",
		Label:       "Test Field",
		Type:        String,
		Description: "Enter a value here",
	}
	fr := NewFieldRow(meta, nil)
	fr.SetWidth(72)
	fr.SetLabelWidth(15)
	fr.SetValue("hello")
	fr.EnterEditing()

	view := fr.View()

	assert.Contains(t, view, "Enter a value here")
}

func TestFieldRow_EditingViewShowsValidationError(t *testing.T) {
	meta := FieldMeta{
		Key:      "test.field",
		Label:    "Test Field",
		Type:     Int,
		Validate: ValidatePositiveInt,
	}
	fr := NewFieldRow(meta, nil)
	fr.SetWidth(72)
	fr.SetLabelWidth(15)
	fr.SetValue(10)
	fr.EnterEditing()
	fr.input.SetValue("not-a-number")

	// Trigger the error.
	_, err := fr.CommitEditing()
	require.Error(t, err)

	view := fr.View()
	assert.Contains(t, view, "cannot convert")
}

func TestFieldRow_Truncation(t *testing.T) {
	fr := newTestFieldRow(String)
	longValue := strings.Repeat("x", 200)
	fr.SetValue(longValue)
	fr.SetWidth(40)

	view := fr.View()
	stripped := stripAnsi(view)

	assert.Contains(t, stripped, "...")
	assert.Less(t, len(stripped), 200, "view should be truncated")
}

func TestFieldRow_UpdateInputForwardsMessages(t *testing.T) {
	fr := newTestFieldRow(String)
	fr.SetValue("hello")
	fr.EnterEditing()

	// Send a key message.
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	cmd := fr.UpdateInput(msg)

	// Should not panic, may return a command.
	_ = cmd
	// The input should have the character appended.
	assert.Contains(t, fr.input.Value(), "x")
}

func TestFieldRow_UpdateInputNoOpWhenBrowsing(t *testing.T) {
	fr := newTestFieldRow(String)
	fr.SetValue("hello")

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}
	cmd := fr.UpdateInput(msg)

	assert.Nil(t, cmd)
}

func TestFieldRow_ObjectListCannotEdit(t *testing.T) {
	fr := newTestFieldRow(ObjectList)
	fr.SetValue("something")

	cmd := fr.EnterEditing()

	assert.Nil(t, cmd)
	assert.Equal(t, FieldBrowsing, fr.State())
}

func TestFieldRow_FormatValueForInput(t *testing.T) {
	testCases := []struct {
		value  any
		assert func(t *testing.T, result string)
		name   string
		ft     FieldType
	}{
		{
			name:  "string",
			ft:    String,
			value: "hello",
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "hello", result)
			},
		},
		{
			name:  "int",
			ft:    Int,
			value: 42,
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "42", result)
			},
		},
		{
			name:  "int64",
			ft:    Int,
			value: int64(99),
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "99", result)
			},
		},
		{
			name:  "string list",
			ft:    StringList,
			value: []string{"a", "b"},
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "a, b", result)
			},
		},
		{
			name:  "nil value",
			ft:    String,
			value: nil,
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Empty(t, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newTestFieldRow(tc.ft)
			fr.SetValue(tc.value)
			result := fr.formatValueForInput()
			tc.assert(t, result)
		})
	}
}

func TestFieldRow_StripAnsi(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, result string)
		name   string
		input  string
	}{
		{
			name:  "plain text unchanged",
			input: "hello world",
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "hello world", result)
			},
		},
		{
			name:  "removes ANSI codes",
			input: "\033[31mred\033[0m text",
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Equal(t, "red text", result)
			},
		},
		{
			name:  "empty string",
			input: "",
			assert: func(t *testing.T, result string) {
				t.Helper()
				assert.Empty(t, result)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := stripAnsi(tc.input)
			tc.assert(t, result)
		})
	}
}

func TestFieldRow_Accessors(t *testing.T) {
	fr := newTestFieldRow(String)
	fr.SetValue("hello")
	fr.SetDirty(true)
	fr.SetFocused(true)

	assert.Equal(t, "hello", fr.Value())
	assert.True(t, fr.IsDirty())
	assert.True(t, fr.IsFocused())
	assert.Equal(t, "test.field", fr.Meta().Key)
	assert.Equal(t, FieldBrowsing, fr.State())
}

func TestFieldRow_SensitiveEditingReveals(t *testing.T) {
	fr := newTestFieldRow(SensitiveString)
	fr.SetValue("secret")
	fr.EnterEditing()

	// During editing, the input should show normal echo mode (not masked).
	assert.Equal(t, textinput.EchoNormal, fr.input.EchoMode)
}

func TestFieldRow_SuggestionsOnEnterEditing(t *testing.T) {
	testCases := []struct {
		assert      func(t *testing.T, fr *FieldRow)
		name        string
		suggestions []string
	}{
		{
			name:        "sets suggestions when meta has them",
			suggestions: []string{"main", "master", "develop"},
			assert: func(t *testing.T, fr *FieldRow) {
				t.Helper()
				assert.True(t, fr.input.ShowSuggestions)
			},
		},
		{
			name:        "no suggestions when meta is empty",
			suggestions: nil,
			assert: func(t *testing.T, fr *FieldRow) {
				t.Helper()
				assert.False(t, fr.input.ShowSuggestions)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			meta := FieldMeta{
				Key:         "test.field",
				Label:       "Test Field",
				Type:        String,
				Suggestions: tc.suggestions,
			}
			fr := NewFieldRow(meta, nil)
			fr.SetWidth(72)
			fr.SetLabelWidth(15)
			fr.SetValue("m")
			fr.EnterEditing()
			tc.assert(t, fr)
		})
	}
}

func TestFieldRow_SetSuggestionsOverridesMeta(t *testing.T) {
	meta := FieldMeta{
		Key:         "test.field",
		Label:       "Test Field",
		Type:        String,
		Suggestions: []string{"original"},
	}
	fr := NewFieldRow(meta, nil)
	fr.SetWidth(72)
	fr.SetLabelWidth(15)
	fr.SetValue("")

	fr.SetSuggestions([]string{"override1", "override2"})
	fr.EnterEditing()

	assert.True(t, fr.input.ShowSuggestions)
	assert.Equal(t, []string{"override1", "override2"}, fr.meta.Suggestions)
}
