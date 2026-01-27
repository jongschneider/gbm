// Package fields provides concrete Field implementations for wizard forms.
package fields

import (
	"gbm/pkg/tui"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
)

// FilePicker is a field that allows the user to select files from the filesystem.
type FilePicker struct {
	err           error
	theme         *tui.Theme
	title         string
	description   string
	key           string
	filepicker    filepicker.Model
	selectedFiles []string
	width         int
	height        int
	focused       bool
	cancelled     bool
	complete      bool
	currentDir    string
	allowedTypes  []string
	dirAllowed    bool
	multiSelect   bool
	onSelect      func(path string)
}

// NewFilePicker creates a new FilePicker field with the given title and description.
func NewFilePicker(key, title, description string) *FilePicker {
	fp := filepicker.New()
	fp.DirAllowed = true
	fp.FileAllowed = true
	fp.ShowHidden = false
	fp.ShowPermissions = false
	fp.ShowSize = false

	currentDir, err := os.Getwd()
	if err != nil {
		currentDir, _ = os.UserHomeDir()
	}
	fp.CurrentDirectory = currentDir

	return &FilePicker{
		key:           key,
		title:         title,
		description:   description,
		filepicker:    fp,
		theme:         tui.DefaultTheme(),
		currentDir:    currentDir,
		selectedFiles: []string{},
		dirAllowed:    true,
	}
}

// WithCurrentDir sets the initial directory for the filepicker.
func (f *FilePicker) WithCurrentDir(dir string) *FilePicker {
	f.currentDir = dir
	f.filepicker.CurrentDirectory = dir
	return f
}

// WithAllowedTypes sets the allowed file extensions (e.g., ".go", ".yaml").
func (f *FilePicker) WithAllowedTypes(types []string) *FilePicker {
	f.allowedTypes = types
	f.filepicker.AllowedTypes = types
	return f
}

// WithDirAllowed sets whether directories can be selected.
func (f *FilePicker) WithDirAllowed(allowed bool) *FilePicker {
	f.dirAllowed = allowed
	f.filepicker.DirAllowed = allowed
	return f
}

// WithMultiSelect enables multi-file selection mode.
func (f *FilePicker) WithMultiSelect(enabled bool) *FilePicker {
	f.multiSelect = enabled
	return f
}

// WithOnSelect sets a callback function to be called when a file is selected.
func (f *FilePicker) WithOnSelect(fn func(path string)) *FilePicker {
	f.onSelect = fn
	return f
}

// Init implements Field.Init.
func (f *FilePicker) Init() tea.Cmd {
	return f.filepicker.Init()
}

// Update implements Field.Update.
func (f *FilePicker) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
	if !f.focused {
		return f, nil
	}

	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyEsc:
			f.cancelled = true
			return f, func() tea.Msg { return tui.CancelMsg{} }

		case tea.KeyEnter:
			if f.multiSelect && len(f.selectedFiles) > 0 {
				f.complete = true
				return f, func() tea.Msg { return tui.NextStepMsg{} }
			}
		}

		if keyMsg.String() == "space" && f.multiSelect {
			if selected, path := f.filepicker.DidSelectFile(msg); selected {
				if !f.isAlreadySelected(path) {
					f.selectedFiles = append(f.selectedFiles, path)
					if f.onSelect != nil {
						f.onSelect(path)
					}
				}
				return f, nil
			}
		}
	}

	f.filepicker, cmd = f.filepicker.Update(msg)

	if selected, path := f.filepicker.DidSelectFile(msg); selected {
		if f.multiSelect {
			if !f.isAlreadySelected(path) {
				f.selectedFiles = append(f.selectedFiles, path)
				if f.onSelect != nil {
					f.onSelect(path)
				}
			}
		} else {
			f.selectedFiles = []string{path}
			if f.onSelect != nil {
				f.onSelect(path)
			}
			f.complete = true
			return f, func() tea.Msg { return tui.NextStepMsg{} }
		}
	}

	return f, cmd
}

// isAlreadySelected checks if a path is already in the selected files list.
func (f *FilePicker) isAlreadySelected(path string) bool {
	for _, p := range f.selectedFiles {
		if p == path {
			return true
		}
	}
	return false
}

// View implements Field.View.
func (f *FilePicker) View() string {
	var b strings.Builder

	styles := f.getStyles()

	b.WriteString(styles.Title.Render(f.title))
	b.WriteString("\n")

	if f.description != "" {
		b.WriteString(styles.Description.Render(f.description))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(f.filepicker.View())

	if len(f.selectedFiles) > 0 {
		b.WriteString("\n\n")
		b.WriteString(styles.Description.Render("Selected files:"))
		for _, path := range f.selectedFiles {
			b.WriteString("\n  • " + path)
		}
	}

	b.WriteString("\n\n")

	helpText := "↑↓ navigate • → open dir • ← parent dir • Enter select"
	if f.multiSelect {
		helpText = "↑↓ navigate • → open • ← back • Space add file • Enter confirm"
	}
	b.WriteString(styles.Description.Render(helpText + " • Esc cancel"))

	return b.String()
}

// Focus implements Field.Focus.
func (f *FilePicker) Focus() tea.Cmd {
	f.focused = true
	return f.filepicker.Init()
}

// Blur implements Field.Blur.
func (f *FilePicker) Blur() tea.Cmd {
	f.focused = false
	return nil
}

// IsComplete implements Field.IsComplete.
func (f *FilePicker) IsComplete() bool {
	return f.complete
}

// IsCancelled implements Field.IsCancelled.
func (f *FilePicker) IsCancelled() bool {
	return f.cancelled
}

// Error implements Field.Error.
func (f *FilePicker) Error() error {
	return f.err
}

// Skip implements Field.Skip.
func (f *FilePicker) Skip() bool {
	return false
}

// WithTheme implements Field.WithTheme.
func (f *FilePicker) WithTheme(theme *tui.Theme) tui.Field {
	f.theme = theme
	return f
}

// WithWidth implements Field.WithWidth.
func (f *FilePicker) WithWidth(width int) tui.Field {
	f.width = width
	return f
}

// WithHeight implements Field.WithHeight.
func (f *FilePicker) WithHeight(height int) tui.Field {
	f.height = height
	f.filepicker.SetHeight(height - 6)
	return f
}

// GetKey implements Field.GetKey.
func (f *FilePicker) GetKey() string {
	return f.key
}

// GetValue implements Field.GetValue.
func (f *FilePicker) GetValue() any {
	return f.selectedFiles
}

// GetSelectedFiles returns the list of selected file paths.
func (f *FilePicker) GetSelectedFiles() []string {
	return f.selectedFiles
}

// ClearSelection clears all selected files.
func (f *FilePicker) ClearSelection() {
	f.selectedFiles = []string{}
}

// RemoveSelection removes a file from the selection.
func (f *FilePicker) RemoveSelection(path string) {
	newFiles := make([]string, 0, len(f.selectedFiles))
	for _, p := range f.selectedFiles {
		if p != path {
			newFiles = append(newFiles, p)
		}
	}
	f.selectedFiles = newFiles
}

// getStyles returns the appropriate styles based on focus state.
func (f *FilePicker) getStyles() tui.FieldStyles {
	if f.theme == nil {
		f.theme = tui.DefaultTheme()
	}
	if f.focused {
		return f.theme.Focused
	}
	return f.theme.Blurred
}

// Ensure FilePicker implements Field interface at compile time.
var _ tui.Field = (*FilePicker)(nil)
