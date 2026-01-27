package config

import (
	"gbm/pkg/tui"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

// fileCopyFormModel wraps FileCopyForm for teatest, quitting on BackBoundaryMsg.
type fileCopyFormModel struct {
	form *FileCopyForm
}

func newFileCopyFormModel(f *FileCopyForm) *fileCopyFormModel {
	return &fileCopyFormModel{form: f}
}

func (m *fileCopyFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *fileCopyFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(tui.BackBoundaryMsg); ok {
		return m, tea.Quit
	}

	model, cmd := m.form.Update(msg)
	if f, ok := model.(*FileCopyForm); ok {
		m.form = f
	}
	return m, cmd
}

func (m *fileCopyFormModel) View() string {
	return m.form.View()
}

func TestFileCopyForm_EmptyRulesRender(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: nil,
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	finalOutput := ""
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		finalOutput = string(bts)
		return len(finalOutput) > 0
	}, teatest.WithDuration(100*time.Millisecond))

	assert.Contains(t, finalOutput, "File Copy Rules")
	assert.Contains(t, finalOutput, "No rules configured")
}

func TestFileCopyForm_WithRulesRender(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: []FileCopyRule{
			{SourceWorktree: "main", Files: []string{".env", "config/"}},
			{SourceWorktree: "develop", Files: []string{".envrc"}},
		},
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	finalOutput := ""
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		finalOutput = string(bts)
		return len(finalOutput) > 0
	}, teatest.WithDuration(100*time.Millisecond))

	assert.Contains(t, finalOutput, "File Copy Rules")
	assert.Contains(t, finalOutput, "main")
	assert.NotContains(t, finalOutput, "No rules configured")
}

func TestFileCopyForm_AddModal(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: nil,
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalAdd, form.GetModalState(), "Add modal should be shown")
}

func TestFileCopyForm_EditModal(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: []FileCopyRule{
			{SourceWorktree: "main", Files: []string{".env"}},
		},
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalEdit, form.GetModalState(), "Edit modal should be shown")
}

func TestFileCopyForm_DeleteModal(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: []FileCopyRule{
			{SourceWorktree: "main", Files: []string{".env"}},
		},
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})

	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalDelete, form.GetModalState(), "Delete modal should be shown")
}

func TestFileCopyForm_DeleteConfirm(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: []FileCopyRule{
			{SourceWorktree: "main", Files: []string{".env"}},
		},
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	assert.Len(t, form.GetRules(), 1, "Should have 1 rule initially")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	time.Sleep(50 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, form.GetRules(), 0, "Rule should be deleted")
	assert.Equal(t, ModalNone, form.GetModalState(), "Modal should be closed")
}

func TestFileCopyForm_DeleteCancel(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: []FileCopyRule{
			{SourceWorktree: "main", Files: []string{".env"}},
		},
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	time.Sleep(50 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, form.GetRules(), 1, "Rule should not be deleted")
	assert.Equal(t, ModalNone, form.GetModalState(), "Modal should be closed")
}

func TestFileCopyForm_SaveFlow(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	saveCalled := false
	var savedRules []FileCopyRule

	config := FileCopyFormConfig{
		Rules: []FileCopyRule{
			{SourceWorktree: "main", Files: []string{".env"}},
		},
		Theme: tui.DefaultTheme(),
		OnSave: func(rules []FileCopyRule) error {
			saveCalled = true
			savedRules = rules
			return nil
		},
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	assert.True(t, saveCalled, "OnSave callback should have been called")
	assert.Len(t, savedRules, 1, "Should save 1 rule")
	assert.True(t, form.IsComplete(), "Form should be marked as submitted")
}

func TestFileCopyForm_DiscardFlow(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: nil,
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalDiscard, form.GetModalState(), "Discard modal should be shown")

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	assert.True(t, form.IsCancelled(), "Form should be marked as cancelled")
}

func TestFileCopyForm_KeepEditingFlow(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: nil,
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	time.Sleep(50 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	time.Sleep(50 * time.Millisecond)

	assert.False(t, form.IsCancelled(), "Form should not be cancelled")
	assert.Equal(t, ModalNone, form.GetModalState(), "Modal should be closed")
}

func TestFileCopyForm_EscapeKey(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: nil,
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})

	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	assert.True(t, form.IsCancelled(), "Form should be marked as cancelled")
}

func TestFileCopyForm_EscapeModalClosesModal(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: nil,
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalAdd, form.GetModalState(), "Add modal should be shown")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalNone, form.GetModalState(), "Modal should be closed")
	assert.False(t, form.IsCancelled(), "Form should not be cancelled")
}

func TestFormatFilesPreview(t *testing.T) {
	testCases := []struct {
		name   string
		files  []string
		expect func(t *testing.T, got string)
	}{
		{
			name:  "empty files",
			files: nil,
			expect: func(t *testing.T, got string) {
				assert.Equal(t, "(no files)", got)
			},
		},
		{
			name:  "single file",
			files: []string{".env"},
			expect: func(t *testing.T, got string) {
				assert.Equal(t, ".env", got)
			},
		},
		{
			name:  "two files",
			files: []string{".env", "config/"},
			expect: func(t *testing.T, got string) {
				assert.Equal(t, ".env, config/", got)
			},
		},
		{
			name:  "three or more files",
			files: []string{".env", "config/", ".envrc"},
			expect: func(t *testing.T, got string) {
				assert.Equal(t, ".env, config/, ...", got)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := formatFilesPreview(tc.files)
			tc.expect(t, got)
		})
	}
}

func TestParseFilesList(t *testing.T) {
	testCases := []struct {
		name   string
		input  string
		expect func(t *testing.T, got []string)
	}{
		{
			name:  "empty string",
			input: "",
			expect: func(t *testing.T, got []string) {
				assert.Nil(t, got)
			},
		},
		{
			name:  "single file",
			input: ".env",
			expect: func(t *testing.T, got []string) {
				assert.Equal(t, []string{".env"}, got)
			},
		},
		{
			name:  "multiple files with spaces",
			input: ".env, config/, .envrc",
			expect: func(t *testing.T, got []string) {
				assert.Equal(t, []string{".env", "config/", ".envrc"}, got)
			},
		},
		{
			name:  "files with extra whitespace",
			input: "  .env  ,  config/  ",
			expect: func(t *testing.T, got []string) {
				assert.Equal(t, []string{".env", "config/"}, got)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseFilesList(tc.input)
			tc.expect(t, got)
		})
	}
}

func TestFileCopyForm_OpenFilePicker(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: nil,
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalAdd, form.GetModalState(), "Add modal should be shown")

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	time.Sleep(50 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalFilePicker, form.GetModalState(), "FilePicker modal should be shown")
}

func TestFileCopyForm_FilePickerEscapeReturnsToEditModal(t *testing.T) {
	t.Parallel()
	lipgloss.SetColorProfile(termenv.Ascii)

	config := FileCopyFormConfig{
		Rules: nil,
		Theme: tui.DefaultTheme(),
	}

	form := NewFileCopyForm(config)
	model := newFileCopyFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	time.Sleep(50 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	time.Sleep(50 * time.Millisecond)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalFilePicker, form.GetModalState(), "FilePicker modal should be shown")

	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	time.Sleep(50 * time.Millisecond)

	assert.Equal(t, ModalAdd, form.GetModalState(), "Should return to Add modal")
}

func TestContainsPath(t *testing.T) {
	testCases := []struct {
		name   string
		paths  []string
		path   string
		expect func(t *testing.T, got bool)
	}{
		{
			name:  "empty paths",
			paths: []string{},
			path:  "/test",
			expect: func(t *testing.T, got bool) {
				assert.False(t, got)
			},
		},
		{
			name:  "path exists",
			paths: []string{"/path1", "/path2"},
			path:  "/path2",
			expect: func(t *testing.T, got bool) {
				assert.True(t, got)
			},
		},
		{
			name:  "path does not exist",
			paths: []string{"/path1", "/path2"},
			path:  "/path3",
			expect: func(t *testing.T, got bool) {
				assert.False(t, got)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := containsPath(tc.paths, tc.path)
			tc.expect(t, got)
		})
	}
}
