// Package config provides the TUI components for the gbm configuration editor.
package config

import "github.com/charmbracelet/bubbles/key"

// BrowsingKeyMap defines keybindings active during field browsing.
type BrowsingKeyMap struct {
	NextTab   key.Binding
	PrevTab   key.Binding
	Up        key.Binding
	Down      key.Binding
	PrevGroup key.Binding
	NextGroup key.Binding
	First     key.Binding
	Last      key.Binding
	Edit      key.Binding
	SaveQuit  key.Binding
	Save      key.Binding
	Reset     key.Binding
	ResetAll  key.Binding
	Search    key.Binding
	Help      key.Binding
	Add       key.Binding
	Delete    key.Binding
	Quit      key.Binding
	ForceQuit key.Binding
}

// EditingKeyMap defines keybindings active during inline field editing.
// Tab and shift-tab are explicitly disabled to prevent section switching.
type EditingKeyMap struct {
	Confirm   key.Binding
	Cancel    key.Binding
	ForceQuit key.Binding
	Tab       key.Binding // disabled -- no-op during editing
	ShiftTab  key.Binding // disabled -- no-op during editing
}

// SearchKeyMap defines keybindings active during field search/filter.
type SearchKeyMap struct {
	Close key.Binding
}

// ListOverlayKeyMap defines keybindings for the string list editing overlay.
type ListOverlayKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Add     key.Binding
	Delete  key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

// EditorOverlayKeyMap defines keybindings for the entry editor overlay
// (rule editor, worktree editor).
type EditorOverlayKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Edit    key.Binding
	Rename  key.Binding
	Confirm key.Binding
	Cancel  key.Binding
}

// HelpOverlayKeyMap defines keybindings for the full help reference overlay.
type HelpOverlayKeyMap struct {
	Close     key.Binding
	CloseHelp key.Binding
}

// ErrorsOverlayKeyMap defines keybindings for the validation errors overlay.
type ErrorsOverlayKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Jump   key.Binding
	Cancel key.Binding
}

// ConfirmationKeyMap defines keybindings for confirmation dialogs
// (dirty guard, reset all, discard changes).
type ConfirmationKeyMap struct {
	Confirm key.Binding
	Deny    key.Binding
	Cancel  key.Binding
}

// NewBrowsingKeys returns the default keybindings for browsing state.
func NewBrowsingKeys() BrowsingKeyMap {
	return BrowsingKeyMap{
		NextTab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-tab", "prev tab"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("\u2191/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("\u2193/j", "down"),
		),
		PrevGroup: key.NewBinding(
			key.WithKeys("{"),
			key.WithHelp("{", "prev group"),
		),
		NextGroup: key.NewBinding(
			key.WithKeys("}"),
			key.WithHelp("}", "next group"),
		),
		First: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "first field"),
		),
		Last: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "last field"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		SaveQuit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "save & quit"),
		),
		Save: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "save"),
		),
		Reset: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reset field"),
		),
		ResetAll: key.NewBinding(
			key.WithKeys("R"),
			key.WithHelp("R", "reset all"),
		),
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add entry"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete entry"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q"),
			key.WithHelp("q", "quit"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl-c", "quit"),
		),
	}
}

// NewEditingKeys returns the default keybindings for inline editing state.
// Tab and shift-tab are disabled to prevent accidental section switches.
func NewEditingKeys() EditingKeyMap {
	return EditingKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		ForceQuit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl-c", "cancel"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", ""),
			key.WithDisabled(),
		),
		ShiftTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("S-tab", ""),
			key.WithDisabled(),
		),
	}
}

// NewSearchKeys returns the default keybindings for field search state.
func NewSearchKeys() SearchKeyMap {
	return SearchKeyMap{
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close search"),
		),
	}
}

// NewListOverlayKeys returns the default keybindings for the list editing overlay.
func NewListOverlayKeys() ListOverlayKeyMap {
	return ListOverlayKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("\u2191", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("\u2193", "down"),
		),
		Add: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "add item"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete item"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "discard"),
		),
	}
}

// NewEditorOverlayKeys returns the default keybindings for the entry editor overlay.
func NewEditorOverlayKeys() EditorOverlayKeyMap {
	return EditorOverlayKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("\u2191", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("\u2193", "down"),
		),
		Edit: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "edit"),
		),
		Rename: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "rename"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// NewHelpOverlayKeys returns the default keybindings for the help overlay.
func NewHelpOverlayKeys() HelpOverlayKeyMap {
	return HelpOverlayKeyMap{
		Close: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close help"),
		),
		CloseHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "close help"),
		),
	}
}

// NewErrorsOverlayKeys returns the default keybindings for the errors overlay.
func NewErrorsOverlayKeys() ErrorsOverlayKeyMap {
	return ErrorsOverlayKeyMap{
		Up: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("\u2191", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("\u2193", "down"),
		),
		Jump: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "jump to error"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "close"),
		),
	}
}

// NewConfirmationKeys returns the default keybindings for confirmation dialogs.
func NewConfirmationKeys() ConfirmationKeyMap {
	return ConfirmationKeyMap{
		Confirm: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "confirm"),
		),
		Deny: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "deny"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// EditHelpVerb returns the appropriate help description for the edit key
// based on the focused field type. This supports field-type-aware footer
// rendering: "edit" for strings, "toggle" for bools, "open" for lists.
func EditHelpVerb(fieldType string) string {
	switch fieldType {
	case "bool":
		return "toggle"
	case "string_list", "object_list":
		return "open"
	default:
		return "edit"
	}
}
