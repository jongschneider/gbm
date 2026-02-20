package config

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	"github.com/stretchr/testify/assert"
)

func TestNewBrowsingKeys(t *testing.T) {
	keys := NewBrowsingKeys()

	testCases := []struct {
		assert func(t *testing.T, keys BrowsingKeyMap)
		name   string
	}{
		{
			name: "tab navigates to next section tab",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.NextTab.Enabled())
				assert.Contains(t, keys.NextTab.Keys(), "tab")
				assert.NotEmpty(t, keys.NextTab.Help().Desc)
			},
		},
		{
			name: "shift-tab navigates to prev section tab",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.PrevTab.Enabled())
				assert.Contains(t, keys.PrevTab.Keys(), "shift+tab")
				assert.NotEmpty(t, keys.PrevTab.Help().Desc)
			},
		},
		{
			name: "up/k navigates up",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Up.Enabled())
				assert.Contains(t, keys.Up.Keys(), "up")
				assert.Contains(t, keys.Up.Keys(), "k")
				assert.NotEmpty(t, keys.Up.Help().Desc)
			},
		},
		{
			name: "down/j navigates down",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Down.Enabled())
				assert.Contains(t, keys.Down.Keys(), "down")
				assert.Contains(t, keys.Down.Keys(), "j")
				assert.NotEmpty(t, keys.Down.Help().Desc)
			},
		},
		{
			name: "curly braces jump between groups",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.PrevGroup.Enabled())
				assert.Contains(t, keys.PrevGroup.Keys(), "{")
				assert.True(t, keys.NextGroup.Enabled())
				assert.Contains(t, keys.NextGroup.Keys(), "}")
			},
		},
		{
			name: "g/G jumps to first/last field",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.First.Enabled())
				assert.Contains(t, keys.First.Keys(), "g")
				assert.True(t, keys.Last.Enabled())
				assert.Contains(t, keys.Last.Keys(), "G")
			},
		},
		{
			name: "e edits the focused field",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Edit.Enabled())
				assert.Contains(t, keys.Edit.Keys(), "e")
				assert.NotEmpty(t, keys.Edit.Help().Desc)
			},
		},
		{
			name: "enter triggers save and quit",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.SaveQuit.Enabled())
				assert.Contains(t, keys.SaveQuit.Keys(), "enter")
				assert.Contains(t, keys.SaveQuit.Help().Desc, "save")
			},
		},
		{
			name: "s saves config",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Save.Enabled())
				assert.Contains(t, keys.Save.Keys(), "s")
			},
		},
		{
			name: "r resets field and R resets all",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Reset.Enabled())
				assert.Contains(t, keys.Reset.Keys(), "r")
				assert.True(t, keys.ResetAll.Enabled())
				assert.Contains(t, keys.ResetAll.Keys(), "R")
			},
		},
		{
			name: "slash opens search",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Search.Enabled())
				assert.Contains(t, keys.Search.Keys(), "/")
			},
		},
		{
			name: "question mark opens help",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Help.Enabled())
				assert.Contains(t, keys.Help.Keys(), "?")
			},
		},
		{
			name: "a adds entry and d deletes entry",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Add.Enabled())
				assert.Contains(t, keys.Add.Keys(), "a")
				assert.True(t, keys.Delete.Enabled())
				assert.Contains(t, keys.Delete.Keys(), "d")
			},
		},
		{
			name: "q and ctrl-c quit",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				assert.True(t, keys.Quit.Enabled())
				assert.Contains(t, keys.Quit.Keys(), "q")
				assert.True(t, keys.ForceQuit.Enabled())
				assert.Contains(t, keys.ForceQuit.Keys(), "ctrl+c")
			},
		},
		{
			name: "all bindings have help text for footer rendering",
			assert: func(t *testing.T, keys BrowsingKeyMap) {
				t.Helper()
				bindings := []key.Binding{
					keys.NextTab, keys.PrevTab, keys.Up, keys.Down,
					keys.PrevGroup, keys.NextGroup, keys.First, keys.Last,
					keys.Edit, keys.SaveQuit, keys.Save, keys.Reset,
					keys.ResetAll, keys.Search, keys.Help, keys.Add,
					keys.Delete, keys.Quit, keys.ForceQuit,
				}
				for _, b := range bindings {
					assert.NotEmpty(t, b.Help().Key, "binding %v missing help key", b.Keys())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, keys)
		})
	}
}

func TestNewEditingKeys(t *testing.T) {
	keys := NewEditingKeys()

	testCases := []struct {
		assert func(t *testing.T, keys EditingKeyMap)
		name   string
	}{
		{
			name: "enter confirms edit",
			assert: func(t *testing.T, keys EditingKeyMap) {
				t.Helper()
				assert.True(t, keys.Confirm.Enabled())
				assert.Contains(t, keys.Confirm.Keys(), "enter")
				assert.NotEmpty(t, keys.Confirm.Help().Desc)
			},
		},
		{
			name: "esc cancels edit",
			assert: func(t *testing.T, keys EditingKeyMap) {
				t.Helper()
				assert.True(t, keys.Cancel.Enabled())
				assert.Contains(t, keys.Cancel.Keys(), "esc")
			},
		},
		{
			name: "ctrl-c cancels edit",
			assert: func(t *testing.T, keys EditingKeyMap) {
				t.Helper()
				assert.True(t, keys.ForceQuit.Enabled())
				assert.Contains(t, keys.ForceQuit.Keys(), "ctrl+c")
			},
		},
		{
			name: "tab is disabled during editing",
			assert: func(t *testing.T, keys EditingKeyMap) {
				t.Helper()
				assert.False(t, keys.Tab.Enabled(), "tab must be disabled during editing")
			},
		},
		{
			name: "shift-tab is disabled during editing",
			assert: func(t *testing.T, keys EditingKeyMap) {
				t.Helper()
				assert.False(t, keys.ShiftTab.Enabled(), "shift-tab must be disabled during editing")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, keys)
		})
	}
}

func TestNewSearchKeys(t *testing.T) {
	keys := NewSearchKeys()

	testCases := []struct {
		assert func(t *testing.T, keys SearchKeyMap)
		name   string
	}{
		{
			name: "esc closes search",
			assert: func(t *testing.T, keys SearchKeyMap) {
				t.Helper()
				assert.True(t, keys.Close.Enabled())
				assert.Contains(t, keys.Close.Keys(), "esc")
				assert.NotEmpty(t, keys.Close.Help().Desc)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, keys)
		})
	}
}

func TestNewListOverlayKeys(t *testing.T) {
	keys := NewListOverlayKeys()

	testCases := []struct {
		assert func(t *testing.T, keys ListOverlayKeyMap)
		name   string
	}{
		{
			name: "up/down navigate items",
			assert: func(t *testing.T, keys ListOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Up.Enabled())
				assert.Contains(t, keys.Up.Keys(), "up")
				assert.True(t, keys.Down.Enabled())
				assert.Contains(t, keys.Down.Keys(), "down")
			},
		},
		{
			name: "a adds and d deletes items",
			assert: func(t *testing.T, keys ListOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Add.Enabled())
				assert.Contains(t, keys.Add.Keys(), "a")
				assert.True(t, keys.Delete.Enabled())
				assert.Contains(t, keys.Delete.Keys(), "d")
			},
		},
		{
			name: "enter confirms and esc discards",
			assert: func(t *testing.T, keys ListOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Confirm.Enabled())
				assert.Contains(t, keys.Confirm.Keys(), "enter")
				assert.True(t, keys.Cancel.Enabled())
				assert.Contains(t, keys.Cancel.Keys(), "esc")
			},
		},
		{
			name: "all bindings have help text",
			assert: func(t *testing.T, keys ListOverlayKeyMap) {
				t.Helper()
				bindings := []key.Binding{
					keys.Up, keys.Down, keys.Add, keys.Delete,
					keys.Confirm, keys.Cancel,
				}
				for _, b := range bindings {
					assert.NotEmpty(t, b.Help().Key, "binding %v missing help key", b.Keys())
					assert.NotEmpty(t, b.Help().Desc, "binding %v missing help desc", b.Keys())
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, keys)
		})
	}
}

func TestNewEditorOverlayKeys(t *testing.T) {
	keys := NewEditorOverlayKeys()

	testCases := []struct {
		assert func(t *testing.T, keys EditorOverlayKeyMap)
		name   string
	}{
		{
			name: "up/down navigate fields",
			assert: func(t *testing.T, keys EditorOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Up.Enabled())
				assert.Contains(t, keys.Up.Keys(), "up")
				assert.True(t, keys.Down.Enabled())
				assert.Contains(t, keys.Down.Keys(), "down")
			},
		},
		{
			name: "e edits focused field",
			assert: func(t *testing.T, keys EditorOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Edit.Enabled())
				assert.Contains(t, keys.Edit.Keys(), "e")
			},
		},
		{
			name: "r renames entry",
			assert: func(t *testing.T, keys EditorOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Rename.Enabled())
				assert.Contains(t, keys.Rename.Keys(), "r")
			},
		},
		{
			name: "enter confirms and esc cancels",
			assert: func(t *testing.T, keys EditorOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Confirm.Enabled())
				assert.Contains(t, keys.Confirm.Keys(), "enter")
				assert.True(t, keys.Cancel.Enabled())
				assert.Contains(t, keys.Cancel.Keys(), "esc")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, keys)
		})
	}
}

func TestNewHelpOverlayKeys(t *testing.T) {
	keys := NewHelpOverlayKeys()

	testCases := []struct {
		assert func(t *testing.T, keys HelpOverlayKeyMap)
		name   string
	}{
		{
			name: "esc closes help",
			assert: func(t *testing.T, keys HelpOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Close.Enabled())
				assert.Contains(t, keys.Close.Keys(), "esc")
				assert.NotEmpty(t, keys.Close.Help().Desc)
			},
		},
		{
			name: "question mark also closes help",
			assert: func(t *testing.T, keys HelpOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.CloseHelp.Enabled())
				assert.Contains(t, keys.CloseHelp.Keys(), "?")
				assert.NotEmpty(t, keys.CloseHelp.Help().Desc)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, keys)
		})
	}
}

func TestNewErrorsOverlayKeys(t *testing.T) {
	keys := NewErrorsOverlayKeys()

	testCases := []struct {
		assert func(t *testing.T, keys ErrorsOverlayKeyMap)
		name   string
	}{
		{
			name: "up/down navigate errors",
			assert: func(t *testing.T, keys ErrorsOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Up.Enabled())
				assert.Contains(t, keys.Up.Keys(), "up")
				assert.True(t, keys.Down.Enabled())
				assert.Contains(t, keys.Down.Keys(), "down")
			},
		},
		{
			name: "enter jumps to error field",
			assert: func(t *testing.T, keys ErrorsOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Jump.Enabled())
				assert.Contains(t, keys.Jump.Keys(), "enter")
				assert.Contains(t, keys.Jump.Help().Desc, "jump")
			},
		},
		{
			name: "esc closes overlay",
			assert: func(t *testing.T, keys ErrorsOverlayKeyMap) {
				t.Helper()
				assert.True(t, keys.Cancel.Enabled())
				assert.Contains(t, keys.Cancel.Keys(), "esc")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, keys)
		})
	}
}

func TestNewConfirmationKeys(t *testing.T) {
	keys := NewConfirmationKeys()

	testCases := []struct {
		assert func(t *testing.T, keys ConfirmationKeyMap)
		name   string
	}{
		{
			name: "y confirms",
			assert: func(t *testing.T, keys ConfirmationKeyMap) {
				t.Helper()
				assert.True(t, keys.Confirm.Enabled())
				assert.Contains(t, keys.Confirm.Keys(), "y")
				assert.NotEmpty(t, keys.Confirm.Help().Desc)
			},
		},
		{
			name: "n denies",
			assert: func(t *testing.T, keys ConfirmationKeyMap) {
				t.Helper()
				assert.True(t, keys.Deny.Enabled())
				assert.Contains(t, keys.Deny.Keys(), "n")
				assert.NotEmpty(t, keys.Deny.Help().Desc)
			},
		},
		{
			name: "esc cancels",
			assert: func(t *testing.T, keys ConfirmationKeyMap) {
				t.Helper()
				assert.True(t, keys.Cancel.Enabled())
				assert.Contains(t, keys.Cancel.Keys(), "esc")
				assert.NotEmpty(t, keys.Cancel.Help().Desc)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, keys)
		})
	}
}

func TestEditHelpVerb(t *testing.T) {
	testCases := []struct {
		assert    func(t *testing.T, verb string)
		name      string
		fieldType string
	}{
		{
			name:      "string fields show edit",
			fieldType: "string",
			assert: func(t *testing.T, verb string) {
				t.Helper()
				assert.Equal(t, "edit", verb)
			},
		},
		{
			name:      "sensitive_string fields show edit",
			fieldType: "sensitive_string",
			assert: func(t *testing.T, verb string) {
				t.Helper()
				assert.Equal(t, "edit", verb)
			},
		},
		{
			name:      "int fields show edit",
			fieldType: "int",
			assert: func(t *testing.T, verb string) {
				t.Helper()
				assert.Equal(t, "edit", verb)
			},
		},
		{
			name:      "bool fields show toggle",
			fieldType: "bool",
			assert: func(t *testing.T, verb string) {
				t.Helper()
				assert.Equal(t, "toggle", verb)
			},
		},
		{
			name:      "string_list fields show open",
			fieldType: "string_list",
			assert: func(t *testing.T, verb string) {
				t.Helper()
				assert.Equal(t, "open", verb)
			},
		},
		{
			name:      "object_list fields show open",
			fieldType: "object_list",
			assert: func(t *testing.T, verb string) {
				t.Helper()
				assert.Equal(t, "open", verb)
			},
		},
		{
			name:      "unknown field type defaults to edit",
			fieldType: "unknown",
			assert: func(t *testing.T, verb string) {
				t.Helper()
				assert.Equal(t, "edit", verb)
			},
		},
		{
			name:      "empty field type defaults to edit",
			fieldType: "",
			assert: func(t *testing.T, verb string) {
				t.Helper()
				assert.Equal(t, "edit", verb)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			verb := EditHelpVerb(tc.fieldType)
			tc.assert(t, verb)
		})
	}
}
