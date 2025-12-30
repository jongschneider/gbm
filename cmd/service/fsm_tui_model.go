package service

import (
	"context"
	"fmt"

	"gbm/internal/git"

	tea "github.com/charmbracelet/bubbletea"
)

// FSMModel is a Bubble Tea model that wraps the FSM and coordinates state transitions
// with a single persistent Bubble Tea program instance to avoid screen flicker.
// This is the key to fixing the screen flicker issue - we have ONE program that stays
// alive throughout the entire workflow, rather than creating a new program for each state.
type FSMModel struct {
	fsm              *WorktreeAddFSM
	currentUI        tea.Model
	ctx              context.Context
	shouldQuit       bool
	lastError        error
}

// NewFSMModel creates a new FSM-aware Bubble Tea model
func NewFSMModel(ctx context.Context, fsm *WorktreeAddFSM) *FSMModel {
	return &FSMModel{
		fsm: fsm,
		ctx: ctx,
	}
}

// Init initializes the FSM model and gets the first UI
func (m *FSMModel) Init() tea.Cmd {
	return m.initializeCurrentState()
}

// initializeCurrentState loads the UI model for the current FSM state
func (m *FSMModel) initializeCurrentState() tea.Cmd {
	state := m.fsm.fsm.Current()

	// Get the UI model for this state
	ui := m.getUIModelForState(state)
	if ui == nil {
		m.lastError = fmt.Errorf("unknown state: %s", state)
		return tea.Quit
	}

	m.currentUI = ui
	return ui.Init()
}

// getUIModelForState returns the Bubble Tea model for the given state
// Each state handler returns a WizardModel configured for that state
func (m *FSMModel) getUIModelForState(state string) tea.Model {
	// Non-interactive states (like branch checking and execution)
	switch state {
	case StateFeatureCheckBranch, StateFeatureExecuteCreate,
		StateHotfixExecuteCreate, StateMergebackExecuteCreate, StateMergebackExecuteMerge:
		return m.buildAsyncExecutionUI(state)
	}

	// Interactive wizard states - dispatch to FSM to build the UI
	wizard := m.fsm.buildUIForState(state)
	return wizard
}

// buildAsyncExecutionUI creates a UI for async operations
func (m *FSMModel) buildAsyncExecutionUI(state string) tea.Model {
	return &AsyncExecutionModel{
		fsm:   m.fsm,
		state: state,
		done:  false,
	}
}

// Update handles messages and coordinates FSM state transitions
func (m *FSMModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.shouldQuit {
		return m, tea.Quit
	}

	// Handle terminal states
	if m.handleTerminalState() {
		return m, tea.Quit
	}

	// Update current UI
	updatedUI, cmd := m.currentUI.Update(msg)
	m.currentUI = updatedUI

	// Check if the current UI (usually a WizardModel) is complete
	if wizard, ok := updatedUI.(WizardModel); ok {
		// Check for completion
		if wizard.cancelled {
			m.fsm.fsm.Event(m.ctx, EventCancel)
			return m, m.handleStateTransition()
		}
		if wizard.completed {
			// Extract data from wizard into FSM state
			// This happens because huh forms bind to pointers in the FSM state
			m.captureWizardData(&wizard)

			// Get the next event based on the current state
			event, err := m.getEventFromWizardCompletion(&wizard)
			if err != nil {
				m.lastError = err
				return m, tea.Quit
			}

			// Handle special case: confirm no -> cancel
			if event == EventConfirmNo {
				m.fsm.fsm.Event(m.ctx, EventConfirmNo)
				return m, m.handleStateTransition()
			}

			// Trigger FSM transition
			if err := m.fsm.fsm.Event(m.ctx, event); err != nil {
				m.lastError = err
				return m, tea.Quit
			}

			// Move to next state
			return m, m.handleStateTransition()
		}
	}

	// For async states, check if they're done
	if asyncUI, ok := updatedUI.(*AsyncExecutionModel); ok {
		if asyncUI.done {
			event := asyncUI.nextEvent
			if err := m.fsm.fsm.Event(m.ctx, event); err != nil {
				m.lastError = err
				return m, tea.Quit
			}
			return m, m.handleStateTransition()
		}
	}

	return m, cmd
}

// handleTerminalState checks if we're in a terminal state
// Returns true if a terminal state was reached and UI was set
func (m *FSMModel) handleTerminalState() bool {
	state := m.fsm.fsm.Current()
	switch state {
	case StateSuccess:
		title := "Success"
		message := "Worktree created successfully!"

		// If we have worktree info, include it
		if m.fsm.state.CreatedWorktree.Name != "" {
			message = formatWorktreeSuccessMessage(&m.fsm.state.CreatedWorktree)
		}

		m.currentUI = &TerminalUI{title: title, message: message}
		return true
	case StateCancelled:
		m.currentUI = &TerminalUI{title: "Cancelled", message: "Workflow cancelled."}
		return true
	case StateError:
		msg := "An error occurred"
		if m.fsm.state.LastError != nil {
			msg = m.fsm.state.LastError.Error()
		}
		m.currentUI = &TerminalUI{title: "Error", message: msg}
		return true
	}
	return false
}

// handleStateTransition updates the UI for the new current state
func (m *FSMModel) handleStateTransition() tea.Cmd {
	newState := m.fsm.fsm.Current()

	// On success, exit immediately (don't show TUI screen)
	if newState == StateSuccess {
		return tea.Quit
	}

	// For other terminal states, show a message
	if newState == StateCancelled || newState == StateError {
		if m.handleTerminalState() {
			return nil
		}
	}

	// Get the new UI
	ui := m.getUIModelForState(newState)
	if ui == nil {
		m.lastError = fmt.Errorf("unknown state: %s", newState)
		return tea.Quit
	}

	m.currentUI = ui
	return ui.Init()
}

// captureWizardData extracts and stores data from the completed wizard
func (m *FSMModel) captureWizardData(wizard *WizardModel) {
	state := m.fsm.fsm.Current()

	switch state {
	case StateSelectType:
		// Get the selected type from userData
		if v, ok := wizard.userData["selectedType"]; ok {
			if selectedType, ok := v.(*string); ok {
				m.fsm.selectedType = *selectedType
			}
		}

	case StateFeatureWorktreeName, StateHotfixWorktreeName:
		step, _ := wizard.GetStep(0)
		if model, ok := step.customModel.(FilterableSelectModel); ok {
			selected := model.GetSelected()
			m.fsm.state.WorktreeName = selected
			if state == StateHotfixWorktreeName {
				m.fsm.state.WorktreeName = HotfixPrefix + selected
			}
		}

	case StateFeatureBaseBranch, StateHotfixBaseBranch:
		step, _ := wizard.GetStep(0)
		if model, ok := step.customModel.(FilterableSelectModel); ok {
			m.fsm.state.BaseBranch = model.GetSelected()
		}

	case StateMergebackSourceBranch, StateMergebackTargetBranch,
		StateMergebackWorktreeName, StateMergebackBranchName:
		// These have userData with pointers to their values
		// The form's Value binding already updated these pointers
		// Just read from the userData to confirm
		if v, ok := wizard.userData[getStateDataKey(state)]; ok {
			if ptr, ok := v.(*string); ok {
				switch state {
				case StateMergebackSourceBranch:
					m.fsm.state.SourceBranch = *ptr
				case StateMergebackTargetBranch:
					m.fsm.state.TargetBranch = *ptr
				case StateMergebackWorktreeName:
					m.fsm.state.WorktreeName = *ptr
				case StateMergebackBranchName:
					m.fsm.state.BranchName = *ptr
				}
			}
		}
	}
}

// getStateDataKey returns the userData key for a given state
func getStateDataKey(state string) string {
	switch state {
	case StateMergebackSourceBranch:
		return "sourceBranch"
	case StateMergebackTargetBranch:
		return "targetBranch"
	case StateMergebackWorktreeName:
		return "worktreeName"
	case StateMergebackBranchName:
		return "branchName"
	default:
		return ""
	}
}

// getEventFromWizardCompletion determines the next FSM event based on the current state
func (m *FSMModel) getEventFromWizardCompletion(wizard *WizardModel) (string, error) {
	state := m.fsm.fsm.Current()

	// Handle confirmation states specially
	if state == StateFeatureConfirmCreate || state == StateMergebackConfirmMerge {
		// Check the confirm variable from userData
		if v, ok := wizard.userData["confirm"]; ok {
			if confirm, ok := v.(*bool); ok && !*confirm {
				return EventConfirmNo, nil
			}
		}
		return EventConfirmYes, nil
	}

	// States that simply complete and move forward
	switch state {
	case StateSelectType:
		// Need to get the selected type - it's bound in buildSelectTypeUI
		// Unfortunately we can't access it directly from the form
		// We'll rely on the old behavior where runSelectType set m.fsm.selectedType
		// For now, just return the appropriate event
		// This is a limitation of the current approach
		switch m.fsm.selectedType {
		case "feature":
			return EventSelectFeature, nil
		case "bug":
			return EventSelectBug, nil
		case "hotfix":
			return EventSelectHotfix, nil
		case "mergeback":
			return EventSelectMergeback, nil
		default:
			return "", fmt.Errorf("no workflow type selected")
		}
	case StateFeatureWorktreeName, StateFeatureBranchName,
		StateFeatureBaseBranch,
		StateHotfixWorktreeName, StateHotfixBaseBranch, StateHotfixBranchName,
		StateMergebackSourceBranch, StateMergebackTargetBranch,
		StateMergebackWorktreeName, StateMergebackBranchName:
		return EventComplete, nil
	}

	return "", fmt.Errorf("unable to determine event for state: %s", state)
}

// View renders the current UI
func (m *FSMModel) View() string {
	if m.currentUI != nil {
		return m.currentUI.View()
	}
	return ""
}

// GetLastError returns any error that occurred
func (m *FSMModel) GetLastError() error {
	return m.lastError
}

// TerminalUI displays a simple terminal message and exits on key press
type TerminalUI struct {
	title   string
	message string
}

func (t *TerminalUI) Init() tea.Cmd {
	return nil
}

func (t *TerminalUI) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		// Any key press exits and restores the terminal
		return t, tea.Quit
	}
	return t, nil
}

func (t *TerminalUI) View() string {
	// Simple formatted output without title
	return fmt.Sprintf("%s\n\nPress any key to exit...", t.message)
}

// AsyncExecutionModel handles non-interactive states that do background work
type AsyncExecutionModel struct {
	fsm       *WorktreeAddFSM
	state     string
	done      bool
	nextEvent string
	err       error
}

func (a *AsyncExecutionModel) Init() tea.Cmd {
	return func() tea.Msg {
		event, err := a.fsm.runCurrentState(a.state)
		return AsyncExecutionResult{event: event, err: err}
	}
}

func (a *AsyncExecutionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AsyncExecutionResult:
		a.done = true
		a.nextEvent = msg.event
		a.err = msg.err
		if msg.err != nil {
			a.nextEvent = EventError
		}
	}
	return a, nil
}

func (a *AsyncExecutionModel) View() string {
	return "Processing...\n"
}

// AsyncExecutionResult is a message sent when async execution completes
type AsyncExecutionResult struct {
	event string
	err   error
}

// formatWorktreeSuccessMessage returns a formatted success message for worktree creation
func formatWorktreeSuccessMessage(wt *git.Worktree) string {
	baseInfo := ""
	if wt.BaseBranch != "" {
		baseInfo = fmt.Sprintf("\n  Base:   %s", wt.BaseBranch)
	}
	return fmt.Sprintf("✓ Worktree created successfully!\n\n  Name:   %s\n  Path:   %s\n  Branch: %s%s\n  Commit: %s",
		wt.Name, wt.Path, wt.Branch, baseInfo, wt.Commit)
}

// RunWorktreeAddTUI runs the worktree add workflow with a single Bubble Tea program
func RunWorktreeAddTUI(ctx context.Context, fsm *WorktreeAddFSM) error {
	model := NewFSMModel(ctx, fsm)
	program := tea.NewProgram(model, tea.WithAltScreen())
	finalModel, err := program.Run()

	if err != nil {
		return err
	}

	if m, ok := finalModel.(*FSMModel); ok {
		// Print success message if workflow completed successfully
		if fsm.fsm.Current() == StateSuccess && fsm.state.CreatedWorktree.Name != "" {
			fmt.Printf("\n%s\n\n", formatWorktreeSuccessMessage(&fsm.state.CreatedWorktree))
		}
		return m.GetLastError()
	}

	return nil
}
