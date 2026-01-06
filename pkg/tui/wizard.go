package tui

import tea "github.com/charmbracelet/bubbletea"

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
