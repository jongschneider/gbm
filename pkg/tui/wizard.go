package tui

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Step represents a single step in a wizard workflow.
type Step struct {
	// Name identifies the step for debugging and logging.
	Name string

	// Field is the form component for this step.
	Field Field

	// Skip is an optional function that determines if this step should be skipped.
	// If nil, the step is never skipped. If the function returns true, the step is skipped.
	Skip func(*WorkflowState) bool
}

// Wizard orchestrates a multi-step form flow.
type Wizard struct {
	steps   []Step
	current int
	ctx     *Context

	cancelled bool
	complete  bool
}

// NewWizard creates a new Wizard with the given steps and context.
func NewWizard(steps []Step, ctx *Context) *Wizard {
	if ctx == nil {
		ctx = NewContext()
	}
	return &Wizard{
		steps:   steps,
		current: 0,
		ctx:     ctx,
	}
}

// Init initializes the wizard and focuses the first non-skipped step.
func (w *Wizard) Init() tea.Cmd {
	if len(w.steps) == 0 {
		w.complete = true
		return tea.Quit
	}

	// Find first non-skipped step
	w.current = w.findNextStep(-1)
	if w.current >= len(w.steps) {
		// All steps skipped
		w.complete = true
		return tea.Quit
	}

	// Initialize current field and focus it
	initCmd := w.currentField().Init()
	focusCmd := w.currentField().Focus()
	return tea.Batch(initCmd, focusCmd)
}

// Update handles messages and delegates to the current step's field.
func (w *Wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle global key events
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyCtrlC:
			w.cancelled = true
			return w, tea.Quit

		case tea.KeyEsc:
			wizard, cmd := w.handleBack()
			return wizard, cmd
		}
	}

	// Handle window resize
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		w.ctx.Width = sizeMsg.Width
		w.ctx.Height = sizeMsg.Height
		// Update current field dimensions
		if w.current < len(w.steps) {
			field := w.currentField().WithWidth(sizeMsg.Width).WithHeight(sizeMsg.Height)
			w.steps[w.current].Field = field
		}
		return w, nil
	}

	// Handle step navigation messages
	switch msg.(type) {
	case NextStepMsg:
		wizard, cmd := w.handleNext()
		return wizard, cmd
	case PrevStepMsg:
		wizard, cmd := w.handleBack()
		return wizard, cmd
	case CancelMsg:
		w.cancelled = true
		return w, tea.Quit
	}

	// Delegate to current field
	if w.current < len(w.steps) {
		field, cmd := w.currentField().Update(msg)
		w.steps[w.current].Field = field
		return w, cmd
	}

	return w, nil
}

// View renders only the current step.
func (w *Wizard) View() string {
	if w.current >= len(w.steps) {
		return ""
	}
	return w.currentField().View()
}

// IsComplete returns true if the wizard finished all steps.
func (w *Wizard) IsComplete() bool {
	return w.complete
}

// IsCancelled returns true if the wizard was cancelled by the user.
func (w *Wizard) IsCancelled() bool {
	return w.cancelled
}

// State returns the workflow state with all collected values.
func (w *Wizard) State() *WorkflowState {
	return w.ctx.State
}

// Context returns the wizard's context.
func (w *Wizard) Context() *Context {
	return w.ctx
}

// currentField returns the Field of the current step.
func (w *Wizard) currentField() Field {
	return w.steps[w.current].Field
}

// handleNext advances to the next non-skipped step.
func (w *Wizard) handleNext() (*Wizard, tea.Cmd) {
	// Blur current field
	blurCmd := w.currentField().Blur()

	// Store the current field's value in workflow state
	w.storeFieldValue()

	// Find next non-skipped step
	nextIdx := w.findNextStep(w.current)
	if nextIdx >= len(w.steps) {
		// All remaining steps skipped or done
		w.complete = true
		return w, tea.Batch(blurCmd, tea.Quit)
	}

	w.current = nextIdx

	// Apply default values for the next field based on workflow context
	w.applyFieldDefaults()

	// Initialize and focus the new current field
	initCmd := w.currentField().Init()
	focusCmd := w.currentField().Focus()
	return w, tea.Batch(blurCmd, initCmd, focusCmd)
}

// handleBack goes back to the previous non-skipped step.
func (w *Wizard) handleBack() (*Wizard, tea.Cmd) {
	if w.current == 0 {
		// Already at first step, do nothing or cancel
		return w, nil
	}

	// Blur current field
	blurCmd := w.currentField().Blur()

	// Find previous non-skipped step
	prevIdx := w.findPrevStep(w.current)
	if prevIdx < 0 {
		// No previous step available
		return w, blurCmd
	}

	w.current = prevIdx

	// Focus the previous field (don't re-init, keep existing value)
	focusCmd := w.currentField().Focus()
	return w, tea.Batch(blurCmd, focusCmd)
}

// findNextStep finds the next step index that is not skipped, starting after fromIdx.
func (w *Wizard) findNextStep(fromIdx int) int {
	for i := fromIdx + 1; i < len(w.steps); i++ {
		step := w.steps[i]
		if step.Skip == nil || !step.Skip(w.ctx.State) {
			return i
		}
	}
	return len(w.steps) // No more steps
}

// findPrevStep finds the previous step index that is not skipped, starting before fromIdx.
func (w *Wizard) findPrevStep(fromIdx int) int {
	for i := fromIdx - 1; i >= 0; i-- {
		step := w.steps[i]
		if step.Skip == nil || !step.Skip(w.ctx.State) {
			return i
		}
	}
	return -1 // No previous step
}

// storeFieldValue stores the current field's value in the workflow state.
// This uses the field's key to determine where to store the value.
func (w *Wizard) storeFieldValue() {
	field := w.currentField()
	key := field.GetKey()
	value := field.GetValue()

	switch key {
	case "workflow_type":
		if v, ok := value.(string); ok {
			w.ctx.State.WorkflowType = v
		}
	case "worktree_name":
		if v, ok := value.(string); ok {
			w.ctx.State.WorktreeName = v
		}
	case "branch_name":
		if v, ok := value.(string); ok {
			w.ctx.State.BranchName = v
		}
	case "base_branch":
		if v, ok := value.(string); ok {
			w.ctx.State.BaseBranch = v
		}
	case "jira_issue":
		if v, ok := value.(*JiraIssue); ok {
			w.ctx.State.JiraIssue = v
		}
	}
}

// applyFieldDefaults applies default values to the current field based on workflow context.
// This is called when transitioning to a new step.
// Currently, this applies auto-generated branch names when:
// - The current field is "branch_name" (TextInput)
// - The worktree_name has been set (either from JIRA issue or custom input)
func (w *Wizard) applyFieldDefaults() {
	field := w.currentField()
	key := field.GetKey()

	// Only apply defaults to branch_name TextInput fields
	if key != "branch_name" {
		return
	}

	// Get the worktree name from state (set in previous step)
	worktreeName := w.ctx.State.WorktreeName
	if worktreeName == "" {
		return
	}

	// Generate a default branch name based on the workflow type and worktree name
	defaultBranchName := w.calculateDefaultBranchName(worktreeName)
	if defaultBranchName == "" {
		return
	}

	// Try to apply the default value using reflection
	// This avoids circular import issues by not directly importing fields package
	w.applyTextInputDefault(field, defaultBranchName)
}

// applyTextInputDefault attempts to apply a default value to a TextInput field using reflection.
// This avoids circular imports by using type-based method lookup.
func (w *Wizard) applyTextInputDefault(field Field, defaultValue string) {
	// Use reflection to call WithDefault method if it exists
	type withDefaulter interface {
		WithDefault(string) Field
	}

	if wd, ok := field.(withDefaulter); ok {
		updatedField := wd.WithDefault(defaultValue)
		w.steps[w.current].Field = updatedField
	}
}

// calculateDefaultBranchName calculates a default branch name based on the worktree name.
// If the worktree name is a JIRA issue key, it looks up the full issue to get the summary
// and generates a slug-based branch name. Otherwise, it generates one from the custom name.
// It respects the workflow type to determine the prefix (feature/, bug/, or hotfix/).
func (w *Wizard) calculateDefaultBranchName(worktreeName string) string {
	// Determine workflow prefix based on state workflow type or step names
	prefix := "feature/"
	switch w.ctx.State.WorkflowType {
	case "hotfix":
		prefix = "hotfix/"
	case "bug":
		prefix = "bug/"
	}
	// Also check if this is a hotfix workflow by looking for HOTFIX_ prefix in current state
	// (This happens when we're in ProcessHotfixWorkflow context)
	if w.isProbablyHotfix() {
		prefix = "hotfix/"
	}

	// Try to find the full JIRA issue to get the summary for a better branch name
	if w.ctx.JiraService != nil {
		issues, err := w.ctx.JiraService.FetchIssues()
		if err == nil {
			// Look for a matching JIRA issue
			for _, issue := range issues {
				if issue.Key == worktreeName {
					// Found the issue - generate branch name with proper prefix
					return generateBranchName(worktreeName, issue.Summary, prefix)
				}
			}
		}
	}

	// Fallback: generate from custom name (worktree name)
	return generateBranchNameFromCustom(worktreeName, prefix)
}

// isProbablyHotfix checks if the current wizard is likely a hotfix workflow.
// This is a heuristic based on step names.
func (w *Wizard) isProbablyHotfix() bool {
	// Check if we have a base_branch step that comes BEFORE branch_name step
	baseBranchIdx := -1
	branchNameIdx := -1

	for i, step := range w.steps {
		if step.Name == "base_branch" {
			baseBranchIdx = i
		}
		if step.Name == "branch_name" {
			branchNameIdx = i
		}
	}

	// In hotfix workflows, base_branch (step 2) comes before branch_name (step 3)
	// In feature workflows, branch_name (step 2) comes before base_branch (step 3)
	return baseBranchIdx >= 0 && branchNameIdx >= 0 && baseBranchIdx < branchNameIdx
}

// Helper functions needed by the wizard
func generateBranchName(issueKey, summary, prefix string) string {
	slug := slugify(summary)
	return prefix + issueKey + "_" + slug
}

func generateBranchNameFromCustom(customName, prefix string) string {
	slug := slugify(customName)
	return prefix + slug
}

func slugify(s string) string {
	// Convert to lowercase
	s = strings.ToLower(s)

	// Replace spaces with underscores (not hyphens for custom names)
	s = strings.ReplaceAll(s, " ", "_")

	// Remove special characters, keep only alphanumeric, hyphens, and underscores
	s = regexp.MustCompile(`[^\w-]`).ReplaceAllString(s, "")

	// Remove multiple consecutive underscores/hyphens
	s = regexp.MustCompile(`[-_]+`).ReplaceAllString(s, "_")

	// Trim underscores and hyphens from start and end
	s = strings.Trim(s, "_-")

	return s
}
