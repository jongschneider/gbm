# Screen Flicker Analysis and Solution

## Problem

When using `gbm2 wt add`, each time you confirm a screen, the TUI disappears back to the terminal briefly before showing the next screen. This creates a jarring, unprofessional UX.

## Root Cause

The issue is architectural - it stems from how the FSM state handlers interact with Bubble Tea's program model:

1. **FSM Loop**: `Run()` in `worktree_fsm.go` has a main loop that advances through states
2. **State Handlers**: Each state (e.g., `runFeatureWorktreeName()`) creates a `WizardModel` and calls `wizard.Run()`
3. **Wizard Program Creation**: `wizard.Run()` calls `tea.NewProgram(m, tea.WithAltScreen()).Run()`
4. **Program Lifecycle**: Each `.Run()` call:
   - Enters alt screen (hides normal terminal)
   - Displays the form
   - User confirms
   - **Exits alt screen** when `.Run()` returns
   - Returns control to FSM loop

The sequence for two states looks like:

```
1. FSM calls runFeatureWorktreeName()
2. Creates wizard, calls wizard.Run()
3. NEW program created with WithAltScreen()
4. Alt screen ENTERED, form shown
5. User confirms
6. Program.Run() EXITS, Alt screen EXITED
7. Returns to FSM loop
8. FSM calls runFeatureBranchName()
9. Creates wizard, calls wizard.Run()
10. NEW program created with WithAltScreen()
11. Alt screen RE-ENTERED, next form shown
```

The problem is steps 6 and 10-11: exiting and re-entering alt screen causes visible flicker.

## Why Previous Fixes Don't Work

Several approaches were attempted:

### Approach 1: Suppress alt screen on nested wizards
- Don't call `tea.WithAltScreen()` for subsequent wizard calls
- **Doesn't work**: Once alt screen exits (step 6), we're already out of alt screen mode. Subsequent programs can't re-enter it if they don't request it.

### Approach 2: Reuse program instance
- Keep a program reference and try to reuse it across states
- **Doesn't work**: Bubble Tea programs are single-use. Once `.Run()` returns, the program is done. You cannot call `.Run()` again on the same program instance.

### Approach 3: Have FSM state handlers return models instead of running them
- Have state handlers return `WizardModel` instead of calling `wizard.Run()`
- Wrap FSM in a single Bubble Tea program that manages state transitions
- **Architectural issue**: This requires massive refactoring of all ~30 state handler functions

## Correct Solution

The **only** way to truly fix this is to have a **single Bubble Tea program** that stays alive through the entire workflow:

```
1. FSM creates a SINGLE tea.Program with WithAltScreen()
2. Program ENTERS alt screen ONCE at the start
3. FSM loop runs INSIDE the program's Update() method (not as a separate loop)
4. When state changes, the program updates its view but NEVER exits
5. Program stays in alt screen throughout the entire workflow
6. Only when workflow is COMPLETELY done does program.Run() return
```

### Implementation Requirements

1. **Refactor `worktree_fsm.go`**:
   - Change state handlers from `func(...) (string, error)` that run wizards
   - To `func(...) (*WizardModel, error)` that return UI models
   - Or have them return a `tea.Model` interface

2. **Create FSM-aware Bubble Tea model**:
   - New `WorktreeFSMModel struct` that implements `tea.Model`
   - Holds reference to `*WorktreeAddFSM` and current UI model
   - `Init()`: Advances FSM to first state, gets wizard model
   - `Update()`: Delegates to current wizard's Update, checks for wizard completion
   - When wizard completes, advances FSM to next state
   - When FSM reaches terminal state, returns `tea.Quit`

3. **Modify `wizard.Run()` pattern**:
   - Wizards shouldn't call `tea.NewProgram().Run()`
   - Instead, return themselves as models to be rendered
   - Or accept a reference to a parent program for updates

### Example Pseudo-code

```go
// Current (broken):
func (w *WorktreeAddFSM) runFeatureWorktreeName() (string, error) {
    wizard := NewWizard(...)
    wizard.Run()  // Creates NEW program, exits alt screen
    return EventComplete, nil
}

// Fixed:
func (w *WorktreeAddFSM) getUIForFeatureWorktreeName() tea.Model {
    return NewWizard(...)  // Returns model without running
}

// New FSM-aware Bubble Tea model:
type FSMModel struct {
    fsm *WorktreeAddFSM
    currentUI tea.Model
}

func (m FSMModel) Init() tea.Cmd {
    m.currentUI = m.fsm.getUIForState(m.fsm.CurrentState())
    return m.currentUI.Init()
}

func (m FSMModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Update current UI
    updatedUI, cmd := m.currentUI.Update(msg)
    m.currentUI = updatedUI
    
    // Check if UI (wizard) is complete
    if wizard, ok := updatedUI.(WizardModel); ok && wizard.completed {
        // Advance FSM
        event, _ := m.fsm.runCurrentState()  // Gets event without running UI
        m.fsm.fsm.Event(event)
        
        // Get next UI
        m.currentUI = m.fsm.getUIForState(m.fsm.CurrentState())
        return m, m.currentUI.Init()
    }
    
    return m, cmd
}

func (m FSMModel) View() string {
    return m.currentUI.View()
}

// Usage:
func RunWorktreeAddTUI(ctx context.Context, fsm *WorktreeAddFSM) error {
    model := FSMModel{fsm: fsm}
    program := tea.NewProgram(&model, tea.WithAltScreen())
    _, err := program.Run()
    return err
}
```

## Effort Estimation

- **Refactoring effort**: 3-4 hours
  - Modify all ~30 state handler functions
  - Test each workflow path
  - Debug edge cases with FSM transitions

- **Risk**: Medium
  - Complex state machine refactoring
  - Many interdependent changes
  - Could introduce new bugs if not careful

## Recommended Next Steps

1. Create new file `cmd/service/fsm_tui_model.go` with FSM-aware Bubble Tea model
2. Create new file `cmd/service/worktree_fsm_ui.go` with UI-returning state handlers
3. Gradually migrate state handlers one at a time
4. Test each workflow thoroughly
5. Remove old `wizard.Run()` based state handlers once all are migrated
