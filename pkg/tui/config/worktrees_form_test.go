package config

import (
	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestWorktreesForm_NewWorktreesForm(t *testing.T) {
	testCases := []struct {
		expect    func(t *testing.T, form *WorktreesForm)
		expectErr func(t *testing.T, err error)
		name      string
		config    WorktreesFormConfig
	}{
		{
			name: "creates form with empty worktrees",
			config: WorktreesFormConfig{
				Worktrees: []WorktreeEntry{},
			},
			expect: func(t *testing.T, form *WorktreesForm) {
				assert.NotNil(t, form)
				assert.Empty(t, form.worktrees)
				assert.Equal(t, WorktreeModalNone, form.modalState)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "creates form with worktrees",
			config: WorktreesFormConfig{
				Worktrees: []WorktreeEntry{
					{Name: "feature", Branch: "feature/test", MergeInto: "main"},
					{Name: "hotfix", Branch: "hotfix/fix", Description: "urgent fix"},
				},
			},
			expect: func(t *testing.T, form *WorktreesForm) {
				assert.Len(t, form.worktrees, 2)
				assert.Equal(t, "feature", form.worktrees[0].Name)
				assert.Equal(t, "hotfix", form.worktrees[1].Name)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "uses default theme when not provided",
			config: WorktreesFormConfig{
				Theme: nil,
			},
			expect: func(t *testing.T, form *WorktreesForm) {
				assert.NotNil(t, form.theme)
			},
			expectErr: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			form := NewWorktreesForm(tc.config)
			tc.expect(t, form)
			tc.expectErr(t, nil)
		})
	}
}

func TestWorktreesForm_OpenAddModal(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	f := model.(*WorktreesForm)

	assert.Equal(t, WorktreeModalAdd, f.modalState)
	assert.Equal(t, -1, f.editingIdx)
	assert.NotNil(t, f.nameField)
	assert.NotNil(t, f.branchField)
}

func TestWorktreesForm_OpenEditModal(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "test", Branch: "test-branch", MergeInto: "main"},
		},
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	f := model.(*WorktreesForm)

	assert.Equal(t, WorktreeModalEdit, f.modalState)
	assert.Equal(t, 0, f.editingIdx)
}

func TestWorktreesForm_OpenEditModal_EmptyTable(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	f := model.(*WorktreesForm)

	assert.Equal(t, WorktreeModalNone, f.modalState)
}

func TestWorktreesForm_OpenDeleteModal(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "test", Branch: "test-branch"},
		},
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	f := model.(*WorktreesForm)

	assert.Equal(t, WorktreeModalDelete, f.modalState)
	assert.NotNil(t, f.confirmField)
}

func TestWorktreesForm_ConfirmDelete(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "test", Branch: "test-branch"},
		},
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	f := model.(*WorktreesForm)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	f = model.(*WorktreesForm)

	assert.Empty(t, f.worktrees)
	assert.Equal(t, WorktreeModalNone, f.modalState)
}

func TestWorktreesForm_CancelDelete(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "test", Branch: "test-branch"},
		},
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	f := model.(*WorktreesForm)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	f = model.(*WorktreesForm)

	assert.Len(t, f.worktrees, 1)
	assert.Equal(t, WorktreeModalNone, f.modalState)
}

func TestWorktreesForm_AddWorktree(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	f := model.(*WorktreesForm)

	f.nameField = fields.NewTextInput("name", "Name", "").WithDefault("newworktree").WithTheme(f.theme)
	f.branchField = fields.NewTextInput("branch", "Branch", "").WithDefault("feature/new").WithTheme(f.theme)
	f.mergeIntoField = fields.NewTextInput("merge_into", "Merge Into", "").WithDefault("main").WithTheme(f.theme)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	f = model.(*WorktreesForm)

	assert.Len(t, f.worktrees, 1)
	assert.Equal(t, "newworktree", f.worktrees[0].Name)
	assert.Equal(t, "feature/new", f.worktrees[0].Branch)
	assert.Equal(t, WorktreeModalNone, f.modalState)
}

func TestWorktreesForm_AddWorktree_ValidationErrors(t *testing.T) {
	testCases := []struct {
		name        string
		nameVal     string
		branchVal   string
		expectedErr string
	}{
		{
			name:        "empty name",
			nameVal:     "",
			branchVal:   "branch",
			expectedErr: "Name is required",
		},
		{
			name:        "invalid name characters",
			nameVal:     "invalid name!",
			branchVal:   "branch",
			expectedErr: "Invalid name: use only alphanumeric, -, _",
		},
		{
			name:        "empty branch",
			nameVal:     "valid",
			branchVal:   "",
			expectedErr: "Branch is required",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			form := NewWorktreesForm(WorktreesFormConfig{})

			model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
			f := model.(*WorktreesForm)

			f.nameField = fields.NewTextInput("name", "Name", "").WithDefault(tc.nameVal).WithTheme(f.theme)
			f.branchField = fields.NewTextInput("branch", "Branch", "").WithDefault(tc.branchVal).WithTheme(f.theme)

			model, _ = f.Update(tea.KeyMsg{Type: tea.KeyEnter})
			f = model.(*WorktreesForm)

			assert.Equal(t, tc.expectedErr, f.validationError)
			assert.Equal(t, WorktreeModalAdd, f.modalState)
		})
	}
}

func TestWorktreesForm_AddWorktree_DuplicateName(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "existing", Branch: "branch"},
		},
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	f := model.(*WorktreesForm)

	f.nameField = fields.NewTextInput("name", "Name", "").WithDefault("existing").WithTheme(f.theme)
	f.branchField = fields.NewTextInput("branch", "Branch", "").WithDefault("new-branch").WithTheme(f.theme)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	f = model.(*WorktreesForm)

	assert.Contains(t, f.validationError, "already exists")
	assert.Len(t, f.worktrees, 1)
}

func TestWorktreesForm_EditWorktree(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "original", Branch: "old-branch"},
		},
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	f := model.(*WorktreesForm)

	f.nameField = fields.NewTextInput("name", "Name", "").WithDefault("updated").WithTheme(f.theme)
	f.branchField = fields.NewTextInput("branch", "Branch", "").WithDefault("new-branch").WithTheme(f.theme)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyEnter})
	f = model.(*WorktreesForm)

	assert.Equal(t, "updated", f.worktrees[0].Name)
	assert.Equal(t, "new-branch", f.worktrees[0].Branch)
}

func TestWorktreesForm_EscapeClosesModal(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	f := model.(*WorktreesForm)
	assert.Equal(t, WorktreeModalAdd, f.modalState)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyEsc})
	f = model.(*WorktreesForm)
	assert.Equal(t, WorktreeModalNone, f.modalState)
}

func TestWorktreesForm_TabCyclesFocus(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	f := model.(*WorktreesForm)

	assert.Equal(t, 0, f.modalFocusIdx)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyTab})
	f = model.(*WorktreesForm)
	assert.Equal(t, 1, f.modalFocusIdx)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyTab})
	f = model.(*WorktreesForm)
	assert.Equal(t, 2, f.modalFocusIdx)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyTab})
	f = model.(*WorktreesForm)
	assert.Equal(t, 3, f.modalFocusIdx)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyTab})
	f = model.(*WorktreesForm)
	assert.Equal(t, 0, f.modalFocusIdx)
}

func TestWorktreesForm_ShiftTabCyclesFocusBackward(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	f := model.(*WorktreesForm)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	f = model.(*WorktreesForm)
	assert.Equal(t, 3, f.modalFocusIdx)
}

func TestWorktreesForm_DiscardModal(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	f := model.(*WorktreesForm)

	assert.Equal(t, WorktreeModalDiscard, f.modalState)
}

func TestWorktreesForm_DiscardConfirmed(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	f := model.(*WorktreesForm)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	f = model.(*WorktreesForm)

	assert.True(t, f.cancelled)
}

func TestWorktreesForm_DiscardCancelled(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	f := model.(*WorktreesForm)

	model, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	f = model.(*WorktreesForm)

	assert.False(t, f.cancelled)
	assert.Equal(t, WorktreeModalNone, f.modalState)
}

func TestWorktreesForm_Save(t *testing.T) {
	var savedWorktrees []WorktreeEntry
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "test", Branch: "test-branch"},
		},
		OnSave: func(wts []WorktreeEntry) error {
			savedWorktrees = wts
			return nil
		},
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	f := model.(*WorktreesForm)

	assert.True(t, f.submitted)
	assert.Len(t, savedWorktrees, 1)
}

func TestWorktreesForm_View(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "test", Branch: "test-branch", MergeInto: "main"},
		},
		Theme: tui.DefaultTheme(),
	})

	view := form.View()

	assert.Contains(t, view, "Worktrees")
	assert.Contains(t, view, "a=add")
	assert.Contains(t, view, "e=edit")
	assert.Contains(t, view, "d=delete")
}

func TestWorktreesForm_ViewEmptyTable(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Theme: tui.DefaultTheme(),
	})

	view := form.View()

	assert.Contains(t, view, "No worktrees configured")
}

func TestWorktreesForm_ViewAddModal(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Theme: tui.DefaultTheme(),
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	f := model.(*WorktreesForm)

	view := f.View()

	assert.Contains(t, view, "Add Worktree")
	assert.Contains(t, view, "Name")
	assert.Contains(t, view, "Branch")
}

func TestWorktreesForm_ViewEditModal(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "test", Branch: "test-branch"},
		},
		Theme: tui.DefaultTheme(),
	})

	model, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	f := model.(*WorktreesForm)

	view := f.View()

	assert.Contains(t, view, "Edit Worktree")
}

func TestWorktreesForm_GetWorktrees(t *testing.T) {
	form := NewWorktreesForm(WorktreesFormConfig{
		Worktrees: []WorktreeEntry{
			{Name: "test", Branch: "test-branch"},
		},
	})

	wts := form.GetWorktrees()
	assert.Len(t, wts, 1)
	assert.Equal(t, "test", wts[0].Name)
}
