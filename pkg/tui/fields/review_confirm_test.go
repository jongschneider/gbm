package fields

import (
	"fmt"
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestReviewConfirm() (*ReviewConfirm, *tui.WorkflowState) {
	state := &tui.WorkflowState{
		WorktreeName: "my-feature",
		BranchName:   "feature/my-feature",
		BaseBranch:   "main",
	}
	attrs := []ReviewAttribute{
		{Label: "Worktree Name", Key: tui.FieldKeyWorktreeName, Editable: true},
		{Label: "Branch Name", Key: tui.FieldKeyBranchName, Editable: true},
		{Label: "Base Branch", Key: tui.FieldKeyBaseBranch, Editable: true},
	}
	rc := NewReviewConfirm("confirm", "Review & Create", state, attrs, "/tmp/worktrees")
	rc.Focus()
	return rc, state
}

func TestReviewConfirm_Navigation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		assert func(t *testing.T, rc *ReviewConfirm)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "initial cursor on Create button",
			keys: nil,
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.Equal(t, len(rc.attrs), rc.cursor, "cursor should start on Create button")
			},
		},
		{
			name: "up moves to last attribute",
			keys: []tea.KeyMsg{{Type: tea.KeyUp}},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.Equal(t, len(rc.attrs)-1, rc.cursor)
			},
		},
		{
			name: "down from Create moves to Cancel",
			keys: []tea.KeyMsg{{Type: tea.KeyDown}},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.Equal(t, len(rc.attrs)+1, rc.cursor)
			},
		},
		{
			name: "navigate to first attribute with repeated up",
			keys: []tea.KeyMsg{
				{Type: tea.KeyUp}, // base branch
				{Type: tea.KeyUp}, // branch name
				{Type: tea.KeyUp}, // worktree name
			},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.Equal(t, 0, rc.cursor)
			},
		},
		{
			name: "up at top stays at top",
			keys: []tea.KeyMsg{
				{Type: tea.KeyUp},
				{Type: tea.KeyUp},
				{Type: tea.KeyUp},
				{Type: tea.KeyUp}, // already at 0
			},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.Equal(t, 0, rc.cursor)
			},
		},
		{
			name: "tab wraps around",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // Cancel
				{Type: tea.KeyTab},  // tab wraps to 0
			},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.Equal(t, 0, rc.cursor)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc, _ := newTestReviewConfirm()
			for _, key := range tc.keys {
				rc.Update(key)
			}
			tc.assert(t, rc)
		})
	}
}

func TestReviewConfirm_Confirm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		assert    func(t *testing.T, rc *ReviewConfirm)
		assertCmd func(t *testing.T, cmd tea.Cmd)
		name      string
		keys      []tea.KeyMsg
	}{
		{
			name: "enter on Create button confirms",
			keys: []tea.KeyMsg{{Type: tea.KeyEnter}},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.True(t, rc.IsComplete())
				assert.False(t, rc.IsCancelled())
				assert.True(t, rc.GetValue().(bool))
			},
			assertCmd: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, ok := msg.(tui.NextStepMsg)
				assert.True(t, ok, "expected NextStepMsg")
			},
		},
		{
			name: "y shortcut confirms",
			keys: []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'y'}}},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.True(t, rc.IsComplete())
				assert.False(t, rc.IsCancelled())
			},
			assertCmd: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, ok := msg.(tui.NextStepMsg)
				assert.True(t, ok)
			},
		},
		{
			name: "enter on Cancel button cancels",
			keys: []tea.KeyMsg{
				{Type: tea.KeyDown}, // Cancel
				{Type: tea.KeyEnter},
			},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.True(t, rc.IsComplete())
				assert.True(t, rc.IsCancelled())
			},
			assertCmd: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, ok := msg.(tui.CancelMsg)
				assert.True(t, ok, "expected CancelMsg")
			},
		},
		{
			name: "n shortcut cancels",
			keys: []tea.KeyMsg{{Type: tea.KeyRunes, Runes: []rune{'n'}}},
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.True(t, rc.IsCancelled())
			},
			assertCmd: func(t *testing.T, cmd tea.Cmd) {
				t.Helper()
				require.NotNil(t, cmd)
				msg := cmd()
				_, ok := msg.(tui.CancelMsg)
				assert.True(t, ok)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc, _ := newTestReviewConfirm()
			var cmd tea.Cmd
			for _, key := range tc.keys {
				_, cmd = rc.Update(key)
			}
			tc.assert(t, rc)
			tc.assertCmd(t, cmd)
		})
	}
}

func TestReviewConfirm_InlineEditing(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setup  func(rc *ReviewConfirm)
		assert func(t *testing.T, rc *ReviewConfirm, state *tui.WorkflowState)
		name   string
		keys   []tea.Msg
	}{
		{
			name: "enter on editable attribute starts editing",
			setup: func(rc *ReviewConfirm) {
				// Navigate to worktree name (index 0)
				rc.cursor = 0
			},
			keys: []tea.Msg{tea.KeyMsg{Type: tea.KeyEnter}},
			assert: func(t *testing.T, rc *ReviewConfirm, _ *tui.WorkflowState) {
				t.Helper()
				assert.True(t, rc.editing)
				assert.Equal(t, "my-feature", rc.editInput.Value())
			},
		},
		{
			name: "typing and enter confirms edit to state",
			setup: func(rc *ReviewConfirm) {
				rc.cursor = 0
			},
			keys: func() []tea.Msg {
				msgs := []tea.Msg{
					tea.KeyMsg{Type: tea.KeyEnter}, // start editing
				}
				// Clear existing text
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyCtrlU})
				// Type new value
				for _, ch := range "renamed-wt" {
					msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				}
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter}) // confirm edit
				return msgs
			}(),
			assert: func(t *testing.T, rc *ReviewConfirm, state *tui.WorkflowState) {
				t.Helper()
				assert.False(t, rc.editing)
				assert.Equal(t, "renamed-wt", state.WorktreeName)
				assert.False(t, rc.IsComplete(), "editing should not complete the field")
			},
		},
		{
			name: "escape cancels edit without changing state",
			setup: func(rc *ReviewConfirm) {
				rc.cursor = 1 // branch name
			},
			keys: func() []tea.Msg {
				msgs := []tea.Msg{
					tea.KeyMsg{Type: tea.KeyEnter}, // start editing
				}
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyCtrlU})
				for _, ch := range "changed" {
					msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				}
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEsc}) // cancel edit
				return msgs
			}(),
			assert: func(t *testing.T, rc *ReviewConfirm, state *tui.WorkflowState) {
				t.Helper()
				assert.False(t, rc.editing)
				assert.Equal(t, "feature/my-feature", state.BranchName, "branch name should be unchanged")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			rc, state := newTestReviewConfirm()
			if tc.setup != nil {
				tc.setup(rc)
			}
			for _, msg := range tc.keys {
				rc.Update(msg)
			}
			tc.assert(t, rc, state)
		})
	}
}

func TestReviewConfirm_View(t *testing.T) {
	t.Parallel()

	rc, _ := newTestReviewConfirm()
	view := rc.View()

	assert.Contains(t, view, "Review & Create")
	assert.Contains(t, view, "Worktree Name")
	assert.Contains(t, view, "my-feature")
	assert.Contains(t, view, "Branch Name")
	assert.Contains(t, view, "feature/my-feature")
	assert.Contains(t, view, "Base Branch")
	assert.Contains(t, view, "main")
	assert.Contains(t, view, "Path")
	assert.Contains(t, view, "/tmp/worktrees/my-feature")
	assert.Contains(t, view, "Create")
	assert.Contains(t, view, "Cancel")
}

func TestReviewConfirm_CustomFields(t *testing.T) {
	t.Parallel()

	state := &tui.WorkflowState{}
	state.SetField("source_branch", "feature/x")
	state.SetField("target_branch", "main")

	attrs := []ReviewAttribute{
		{Label: "Source", Key: "source_branch", Editable: true},
		{Label: "Target", Key: "target_branch", Editable: true},
	}
	rc := NewReviewConfirm("confirm", "Review Merge", state, attrs, "")
	rc.Focus()

	assert.Equal(t, "feature/x", rc.getAttrValue("source_branch"))
	assert.Equal(t, "main", rc.getAttrValue("target_branch"))

	// Edit a custom field
	rc.cursor = 0
	rc.Update(tea.KeyMsg{Type: tea.KeyEnter}) // start editing
	rc.Update(tea.KeyMsg{Type: tea.KeyCtrlU}) // clear
	for _, ch := range "develop" {
		rc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	rc.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm edit

	assert.Equal(t, "develop", state.GetField("source_branch"))
}

func TestReviewConfirm_NonEditableAttribute(t *testing.T) {
	t.Parallel()

	state := &tui.WorkflowState{WorktreeName: "test"}
	attrs := []ReviewAttribute{
		{Label: "Name", Key: tui.FieldKeyWorktreeName, Editable: false},
	}
	rc := NewReviewConfirm("confirm", "Review", state, attrs, "")
	rc.Focus()

	// Navigate to the non-editable attribute
	rc.cursor = 0
	_, cmd := rc.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.False(t, rc.editing, "should not enter edit mode for non-editable attribute")
	assert.Nil(t, cmd)
}

func TestReviewConfirm_Validation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		setup  func(rc *ReviewConfirm)
		assert func(t *testing.T, rc *ReviewConfirm, state *tui.WorkflowState)
		name   string
		keys   []tea.Msg
	}{
		{
			name: "validator rejects edit and shows error",
			setup: func(rc *ReviewConfirm) {
				rc.cursor = 0 // worktree name
			},
			keys: func() []tea.Msg {
				msgs := []tea.Msg{
					tea.KeyMsg{Type: tea.KeyEnter}, // start editing
					tea.KeyMsg{Type: tea.KeyCtrlU}, // clear
				}
				for _, ch := range "existing-wt" {
					msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				}
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter}) // try to confirm
				return msgs
			}(),
			assert: func(t *testing.T, rc *ReviewConfirm, state *tui.WorkflowState) {
				t.Helper()
				assert.True(t, rc.editing, "should still be editing after validation failure")
				assert.NotNil(t, rc.attrErrors[0], "should have a validation error")
				assert.Contains(t, rc.attrErrors[0].Error(), "already exists")
				assert.Equal(t, "my-feature", state.WorktreeName, "state should not be updated on validation failure")
			},
		},
		{
			name: "validator passes on valid edit",
			setup: func(rc *ReviewConfirm) {
				rc.cursor = 0
			},
			keys: func() []tea.Msg {
				msgs := []tea.Msg{
					tea.KeyMsg{Type: tea.KeyEnter}, // start editing
					tea.KeyMsg{Type: tea.KeyCtrlU}, // clear
				}
				for _, ch := range "new-name" {
					msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				}
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter}) // confirm
				return msgs
			}(),
			assert: func(t *testing.T, rc *ReviewConfirm, state *tui.WorkflowState) {
				t.Helper()
				assert.False(t, rc.editing, "should exit editing after validation passes")
				assert.Nil(t, rc.attrErrors[0], "should have no validation error")
				assert.Equal(t, "new-name", state.WorktreeName)
			},
		},
		{
			name: "error clears when user types",
			setup: func(rc *ReviewConfirm) {
				rc.cursor = 0
			},
			keys: func() []tea.Msg {
				msgs := []tea.Msg{
					tea.KeyMsg{Type: tea.KeyEnter}, // start editing
					tea.KeyMsg{Type: tea.KeyCtrlU}, // clear
				}
				for _, ch := range "existing-wt" {
					msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				}
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter}) // fail validation
				// Now type another character — error should clear
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
				return msgs
			}(),
			assert: func(t *testing.T, rc *ReviewConfirm, _ *tui.WorkflowState) {
				t.Helper()
				assert.True(t, rc.editing)
				assert.Nil(t, rc.attrErrors[0], "error should clear when typing")
			},
		},
		{
			name: "escape clears error",
			setup: func(rc *ReviewConfirm) {
				rc.cursor = 0
			},
			keys: func() []tea.Msg {
				msgs := []tea.Msg{
					tea.KeyMsg{Type: tea.KeyEnter}, // start editing
					tea.KeyMsg{Type: tea.KeyCtrlU}, // clear
				}
				for _, ch := range "existing-wt" {
					msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				}
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter}) // fail validation
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEsc})   // cancel edit
				return msgs
			}(),
			assert: func(t *testing.T, rc *ReviewConfirm, state *tui.WorkflowState) {
				t.Helper()
				assert.False(t, rc.editing)
				assert.Nil(t, rc.attrErrors[0])
				assert.Equal(t, "my-feature", state.WorktreeName, "state unchanged after cancelled edit")
			},
		},
		{
			name: "view shows error when present",
			setup: func(rc *ReviewConfirm) {
				rc.cursor = 0
			},
			keys: func() []tea.Msg {
				msgs := []tea.Msg{
					tea.KeyMsg{Type: tea.KeyEnter}, // start editing
					tea.KeyMsg{Type: tea.KeyCtrlU}, // clear
				}
				for _, ch := range "existing-wt" {
					msgs = append(msgs, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				}
				msgs = append(msgs, tea.KeyMsg{Type: tea.KeyEnter}) // fail validation
				return msgs
			}(),
			assert: func(t *testing.T, rc *ReviewConfirm, _ *tui.WorkflowState) {
				t.Helper()
				view := rc.View()
				assert.Contains(t, view, "already exists")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			state := &tui.WorkflowState{
				WorktreeName: "my-feature",
				BranchName:   "feature/my-feature",
				BaseBranch:   "main",
			}
			attrs := []ReviewAttribute{
				{
					Label:    "Worktree Name",
					Key:      tui.FieldKeyWorktreeName,
					Editable: true,
					Validator: func(name string) error {
						if name == "existing-wt" {
							return fmt.Errorf("worktree %q already exists", name)
						}
						return nil
					},
				},
				{Label: "Branch Name", Key: tui.FieldKeyBranchName, Editable: true},
			}
			rc := NewReviewConfirm("confirm", "Review", state, attrs, "/tmp/wt")
			rc.Focus()

			if tc.setup != nil {
				tc.setup(rc)
			}
			for _, msg := range tc.keys {
				rc.Update(msg)
			}
			tc.assert(t, rc, state)
		})
	}
}

func TestReviewConfirm_OnFocusValidation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		assert func(t *testing.T, rc *ReviewConfirm)
		name   string
		wtName string
	}{
		{
			name:   "errors shown on focus when value already invalid",
			wtName: "taken-wt",
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.NotNil(t, rc.attrErrors[0], "should have error on worktree name")
				assert.Contains(t, rc.attrErrors[0].Error(), "already exists")
				assert.Equal(t, 0, rc.cursor, "cursor should move to first errored attribute")
				assert.True(t, rc.hasValidationErrors())
			},
		},
		{
			name:   "no errors on focus when values are valid",
			wtName: "brand-new",
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.Nil(t, rc.attrErrors[0])
				assert.Equal(t, 1, rc.cursor, "cursor should stay on Create button")
				assert.False(t, rc.hasValidationErrors())
			},
		},
		{
			name:   "create button blocked when errors exist",
			wtName: "taken-wt",
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				// Try pressing y — should not complete
				rc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
				assert.False(t, rc.IsComplete(), "y shortcut should be blocked")

				// Navigate to Create and press Enter — should not complete
				rc.cursor = len(rc.attrs) // Create button
				rc.Update(tea.KeyMsg{Type: tea.KeyEnter})
				assert.False(t, rc.IsComplete(), "Create button should be blocked")
			},
		},
		{
			name:   "create unblocked after fixing error",
			wtName: "taken-wt",
			assert: func(t *testing.T, rc *ReviewConfirm) {
				t.Helper()
				assert.True(t, rc.hasValidationErrors())

				// Edit the worktree name to a valid value
				rc.cursor = 0
				rc.Update(tea.KeyMsg{Type: tea.KeyEnter}) // start editing
				rc.Update(tea.KeyMsg{Type: tea.KeyCtrlU}) // clear
				for _, ch := range "valid-name" {
					rc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
				}
				rc.Update(tea.KeyMsg{Type: tea.KeyEnter}) // confirm edit

				assert.False(t, rc.hasValidationErrors(), "errors should be cleared")

				// Now y should work
				rc.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
				assert.True(t, rc.IsComplete())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := &tui.WorkflowState{
				WorktreeName: tc.wtName,
				BranchName:   "feature/test",
			}
			attrs := []ReviewAttribute{
				{
					Label:    "Worktree Name",
					Key:      tui.FieldKeyWorktreeName,
					Editable: true,
					Validator: func(name string) error {
						if name == "taken-wt" {
							return fmt.Errorf("worktree %q already exists", name)
						}
						return nil
					},
				},
			}
			rc := NewReviewConfirm("confirm", "Review", state, attrs, "")
			rc.Focus() // triggers validateAll

			tc.assert(t, rc)
		})
	}
}

func TestReviewConfirm_PathComputation(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		assert func(t *testing.T, path string)
		name   string
		wtDir  string
		wtName string
	}{
		{
			name:   "computes path from dir and name",
			wtDir:  "/home/user/worktrees",
			wtName: "my-feature",
			assert: func(t *testing.T, path string) {
				t.Helper()
				assert.Equal(t, "/home/user/worktrees/my-feature", path)
			},
		},
		{
			name:   "empty dir returns empty path",
			wtDir:  "",
			wtName: "my-feature",
			assert: func(t *testing.T, path string) {
				t.Helper()
				assert.Empty(t, path)
			},
		},
		{
			name:   "empty name returns empty path",
			wtDir:  "/tmp/wt",
			wtName: "",
			assert: func(t *testing.T, path string) {
				t.Helper()
				assert.Empty(t, path)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			state := &tui.WorkflowState{WorktreeName: tc.wtName}
			attrs := []ReviewAttribute{
				{Label: "Name", Key: tui.FieldKeyWorktreeName, Editable: true},
			}
			rc := NewReviewConfirm("confirm", "Review", state, attrs, tc.wtDir)
			tc.assert(t, rc.computePath())
		})
	}
}
