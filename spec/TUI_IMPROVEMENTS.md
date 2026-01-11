# TUI Package Improvements Specification

## Executive Summary

The TUI package demonstrates solid foundational architecture with proper abstractions (Field interface, Context pattern, Wizard orchestration). However, it has several areas where implementation deviates from Bubble Tea best practices, particularly around event loop performance, error handling, testing, and message/command architecture.

This spec outlines improvements to align with the [BUBBLETEA.md best practices](./BUBBLETEA.md) and ensure production-readiness.

---

## Issues and Improvements

### 1. **Event Loop Performance**

#### Current Issues

- **Blocking async operations**: `Eval[T].Get()` in [async/eval.go](../pkg/tui/async/eval.go) blocks the event loop when fetching data synchronously on first call
- **Missing command-based async pattern**: Field implementations don't use Bubble Tea's command pattern for async operations
- **Synchronous JIRA/Git calls in Update()**: [wizard.go:323-338](../pkg/tui/wizard.go#L323-L338) calls `FetchIssues()` directly in `Update()`, blocking the loop

**Best Practice Reference**: [BUBBLETEA.md §1](./BUBBLETEA.md#1-keep-the-event-loop-fast)

#### Improvements

**A. Convert Async Operations to Commands**

Create a new message type for async completions and return commands instead of blocking:

```go
// pkg/tui/async/messages.go
package async

// FetchMsg represents completion of an async fetch operation.
type FetchMsg[T any] struct {
    Value T
    Err   error
}

// FetchCmd returns a command that fetches a value asynchronously.
func FetchCmd[T any](fetch func() (T, error)) tea.Cmd {
    return func() tea.Msg {
        value, err := fetch()
        return FetchMsg[T]{Value: value, Err: err}
    }
}
```

**B. Update Fields to Use Async Commands**

Modify field implementations to:
- Return a command on Init() to start async operations
- Handle FetchMsg to update state when data arrives
- Show loading state while fetching

Example for Filterable field with JIRA issues:

```go
// pkg/tui/fields/filterable.go
type Filterable struct {
    // ... existing fields ...
    isLoading bool
    loadErr   error
}

func (f *Filterable) Init() tea.Cmd {
    if f.fetchFunc != nil {
        return async.FetchCmd(f.fetchFunc)
    }
    return nil
}

func (f *Filterable) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
    switch msg := msg.(type) {
    case async.FetchMsg[[]string]:
        if msg.Err != nil {
            f.loadErr = msg.Err
        } else {
            f.options = msg.Value
        }
        f.isLoading = false
        return f, nil
    }
    // ... rest of update logic
}

func (f *Filterable) View() string {
    if f.isLoading {
        return "Loading..." // Or spinner animation
    }
    if f.loadErr != nil {
        return fmt.Sprintf("Error: %v", f.loadErr)
    }
    // ... normal rendering
}
```

**C. Update Wizard to Use Command-Based Branch Name Generation**

Instead of calling `calculateDefaultBranchName()` in `applyFieldDefaults()`:

```go
// pkg/tui/wizard.go
type generatedBranchNameMsg struct {
    branchName string
    err        error
}

func (w *Wizard) applyFieldDefaults() {
    field := w.currentField()
    if field.GetKey() != FieldKeyBranchName {
        return
    }
    
    // Return a command to calculate the default asynchronously
    return func() tea.Msg {
        defaultName := w.calculateDefaultBranchName(w.ctx.State.WorktreeName)
        return generatedBranchNameMsg{branchName: defaultName}
    }
}
```

**Impact**: Eliminates blocking calls in Update() loop. Fields become responsive during network operations.

---

### 2. **Message Ordering and Command Sequencing**

#### Current Issues

- **Unordered field initialization**: [wizard.go:175-177](../pkg/tui/wizard.go#L175-L177) uses `tea.Batch()` for Init/Focus commands, causing potential ordering issues
- **No sequential command pattern**: Multi-step transitions don't guarantee order
- **Async field population races**: Filterable selectors may render before options are populated

**Best Practice Reference**: [BUBBLETEA.md §4](./BUBBLETEA.md#4-messages-are-not-ordered-when-sent-concurrently)

#### Improvements

**A. Use `tea.Sequence()` for Ordered Operations**

Replace `tea.Batch()` with `tea.Sequence()` where order matters:

```go
// pkg/tui/wizard.go - handleNext()
func (w *Wizard) handleNext() (*Wizard, tea.Cmd) {
    blurCmd := w.currentField().Blur()
    w.storeFieldValue()
    
    nextIdx := w.findNextStep(w.current)
    if nextIdx >= len(w.steps) {
        w.complete = true
        // Blur first, then quit (in order)
        return w, tea.Sequence(blurCmd, tea.Quit)
    }
    
    w.current = nextIdx
    w.applyFieldDefaults()
    
    // SEQUENTIAL: Init must complete before Focus
    initCmd := w.currentField().Init()
    focusCmd := w.currentField().Focus()
    return w, tea.Sequence(
        blurCmd,
        initCmd,
        focusCmd,
    )
}
```

**B. Document Message Ordering Assumptions**

Add documentation to Field interface:

```go
// pkg/tui/field.go
// Focus() is called AFTER Init() to ensure fields are ready.
// Focus() tea.Cmd
```

**Impact**: Prevents race conditions in field initialization. Ensures data is available before rendering.

---

### 3. **Hierarchical Model Tree (Missing)**

#### Current Issues

- **No root model**: [worktree_testadd.go](../cmd/service/worktree_testadd.go) creates a wrapper model for wizard transitions, but lacks proper composition
- **Manual message routing**: testaddWrapperModel manually handles stage transitions
- **No global message distribution**: Window resize and other global events aren't broadcast
- **Ad-hoc navigation state management**: Stage switching is imperative rather than declarative

**Best Practice Reference**: [BUBBLETEA.md §5](./BUBBLETEA.md#5-build-a-hierarchical-model-tree)

#### Improvements

**A. Create a Root Model Architecture**

```go
// pkg/tui/navigator.go - NEW FILE
package tui

import tea "github.com/charmbracelet/bubbletea"

// Navigator manages transitions between wizard models in a multi-step flow.
// It provides:
// - Model composition and routing
// - Global message distribution (window resize, etc.)
// - Navigation state management with back/forward
// - Model caching for efficient transitions
type Navigator struct {
    // Stack for navigation history
    stack []tea.Model
    
    // Current model (top of stack)
    current tea.Model
    
    // Callbacks for custom transitions
    onNavigate func(model tea.Model)
}

// NewNavigator creates a new Navigator with an initial model.
func NewNavigator(initial tea.Model) *Navigator {
    return &Navigator{
        stack:   []tea.Model{initial},
        current: initial,
    }
}

// Push adds a new model to the navigation stack.
func (n *Navigator) Push(model tea.Model) {
    n.stack = append(n.stack, model)
    n.current = model
    if n.onNavigate != nil {
        n.onNavigate(model)
    }
}

// Pop returns to the previous model.
func (n *Navigator) Pop() tea.Model {
    if len(n.stack) <= 1 {
        return nil // At root, can't go back
    }
    n.stack = n.stack[:len(n.stack)-1]
    n.current = n.stack[len(n.stack)-1]
    if n.onNavigate != nil {
        n.onNavigate(n.current)
    }
    return n.current
}

// Peek returns the current model without modifying stack.
func (n *Navigator) Peek() tea.Model {
    return n.current
}

// Init initializes the current model.
func (n *Navigator) Init() tea.Cmd {
    if model, ok := n.current.(tea.Model); ok {
        return model.Init()
    }
    return nil
}

// Update delegates to current model and handles navigation messages.
func (n *Navigator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Broadcast window resize to all models for responsive layouts
    if _, ok := msg.(tea.WindowSizeMsg); ok {
        // Update current model's dimensions
    }
    
    // Delegate to current model
    updated, cmd := n.current.Update(msg)
    n.current = updated
    
    return n, cmd
}

// View delegates to current model.
func (n *Navigator) View() string {
    if model, ok := n.current.(tea.Model); ok {
        return model.View()
    }
    return ""
}
```

**B. Refactor testaddWrapperModel to Use Navigator**

```go
// cmd/service/worktree_testadd.go
type testaddWrapperModel struct {
    navigator *tui.Navigator
    ctx       *tui.Context
    stepsMap  map[string][]tui.Step
}

func (m *testaddWrapperModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle global keys at root level
    if km, ok := msg.(tea.KeyMsg); ok && km.Type == tea.KeyCtrlC {
        return m, tea.Quit
    }
    
    // Handle back boundary (user pressed ESC at first step)
    if _, ok := msg.(tui.BackBoundaryMsg); ok {
        prevModel := m.navigator.Pop()
        if prevModel == nil {
            // At root, quit
            return m, tea.Quit
        }
        return m, prevModel.Init()
    }
    
    // Delegate to current wizard
    currentWizard := m.navigator.Peek().(*tui.Wizard)
    updated, cmd := currentWizard.Update(msg)
    
    if wizard, ok := updated.(*tui.Wizard); ok {
        m.navigator.current = wizard
        
        // Check if wizard completed
        if wizard.IsComplete() {
            selectedType := wizard.State().WorkflowType
            if steps, ok := m.stepsMap[selectedType]; ok {
                // Create next wizard and push to stack
                nextWizard := tui.NewWizard(steps, m.ctx)
                nextWizard.State().WorkflowType = selectedType
                m.navigator.Push(nextWizard)
                return m, nextWizard.Init()
            }
        }
    }
    
    return m, cmd
}
```

**Impact**: Better scalability for multi-screen flows. Cleaner message routing. Easier testing of individual screens.

---

### 4. **Layout and Dimensions Management**

#### Current Issues

- **No layout helpers**: [field.go](../pkg/tui/field.go) defines `WithWidth()` and `WithHeight()` but they're barely used
- **Hardcoded sizes in views**: No consistent use of `lipgloss.Height()` / `lipgloss.Width()` for calculations
- **No responsive height calculation**: Fields don't adapt when terminal is resized

**Best Practice Reference**: [BUBBLETEA.md §6](./BUBBLETEA.md#6-use-lipgloss-for-layout-arithmetic)

#### Improvements

**A. Create Layout Helper Package**

```go
// pkg/tui/layout/layout.go - NEW FILE
package layout

import "github.com/charmbracelet/lipgloss"

// Constraints defines available space for a component.
type Constraints struct {
    MaxWidth  int
    MaxHeight int
}

// CalculateContentHeight calculates available height after reserving space for header/footer.
func CalculateContentHeight(totalHeight, headerHeight, footerHeight int) int {
    content := totalHeight - headerHeight - footerHeight
    if content < 0 {
        return 0
    }
    return content
}

// ApplyPadding applies margin to a rendered string and returns new dimensions.
func ApplyPadding(rendered string, hPad, vPad int) (string, int, int) {
    width := lipgloss.Width(rendered) + 2*hPad
    height := lipgloss.Height(rendered) + 2*vPad
    return lipgloss.NewStyle().
        Padding(vPad, hPad).
        Render(rendered), width, height
}
```

**B. Update Field Views to Use Constraints**

```go
// pkg/tui/field.go - update interface
type Field interface {
    // ... existing methods ...
    
    // ViewWithConstraints renders with explicit space constraints.
    ViewWithConstraints(constraints layout.Constraints) string
}
```

**C. Update Wizard to Manage Layout**

```go
// pkg/tui/wizard.go
func (w *Wizard) View() string {
    if w.current >= len(w.steps) {
        return ""
    }
    
    field := w.currentField()
    
    // Use layout constraints
    constraints := layout.Constraints{
        MaxWidth:  w.ctx.Width,
        MaxHeight: w.ctx.Height - 3, // Reserve space for footer
    }
    
    // Render with constraints
    view := field.ViewWithConstraints(constraints)
    
    // Add footer
    footer := w.renderFooter()
    return lipgloss.JoinVertical(
        lipgloss.Top,
        view,
        footer,
    )
}

func (w *Wizard) renderFooter() string {
    helpText := "[↑↓ Navigate] [Enter Confirm] [Esc Back] [Ctrl+C Quit]"
    return lipgloss.NewStyle().
        Foreground(lipgloss.Color("240")).
        Render(helpText)
}
```

**Impact**: Responsive layouts. Easier to maintain. Automatic adaptation to terminal size changes.

---

### 5. **Testing and Demos**

#### Current Issues

- **No teatest usage**: No integration tests for wizard flows
- **Limited test coverage**: Only [wizard_navigation_test.go](../pkg/tui/wizard_navigation_test.go) and basic table tests
- **No demo/recording scripts**: No VHS scripts for documentation
- **Manual testing required**: testadd is the only way to verify UI behavior

**Best Practice Reference**: [BUBBLETEA.md §8-9](./BUBBLETEA.md#8-test-with-teatest) and [BUBBLETEA.md §9](./BUBBLETEA.md#9-record-demos-with-vhs)

#### Improvements

**A. Add Comprehensive Teatest Suite**

```go
// pkg/tui/wizard_test.go - NEW TESTS
package tui

import (
    "testing"
    "time"
    
    "github.com/charmbracelet/x/exp/teatest"
    tea "github.com/charmbracelet/bubbletea"
    "github.com/stretchr/testify/assert"
)

func TestWizardCompleteFlow(t *testing.T) {
    // Create wizard with simple test steps
    steps := []Step{
        {
            Name:  "step1",
            Field: NewMockField("value1"),
        },
        {
            Name:  "step2",
            Field: NewMockField("value2"),
        },
    }
    
    wizard := NewWizard(steps, NewContext())
    tm := teatest.NewTestModel(t, wizard)
    
    // Wait for initial render
    teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
        return strings.Contains(string(b), "step1")
    }, teatest.WithDuration(time.Second))
    
    // Simulate navigation
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    
    // Verify transition to step2
    teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
        return strings.Contains(string(b), "step2")
    }, teatest.WithDuration(time.Second))
    
    // Complete workflow
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
    
    // Assert final state
    if w, ok := tm.FinalModel().(*Wizard); ok {
        assert.True(t, w.IsComplete())
    }
}

func TestWizardBackNavigation(t *testing.T) {
    steps := []Step{
        {Name: "step1", Field: NewMockField("v1")},
        {Name: "step2", Field: NewMockField("v2")},
    }
    
    wizard := NewWizard(steps, NewContext())
    tm := teatest.NewTestModel(t, wizard)
    
    // Navigate to step2
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
        return strings.Contains(string(b), "step2")
    }, teatest.WithDuration(time.Second))
    
    // Go back to step1
    tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
    teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
        return strings.Contains(string(b), "step1")
    }, teatest.WithDuration(time.Second))
}

func TestWizardBackBoundary(t *testing.T) {
    steps := []Step{
        {Name: "step1", Field: NewMockField("v1")},
    }
    
    wizard := NewWizard(steps, NewContext())
    tm := teatest.NewTestModel(t, wizard)
    
    // Try to go back at first step
    tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
    
    // Should receive BackBoundaryMsg (check in final state)
    tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
```

**B. Create VHS Demo Script**

```bash
# demos/wizard-feature-flow.tape
Output demo-feature-flow.gif

Set Shell "bash"
Set FontSize 14
Set Width 1200
Set Height 800
Set Framerate 24

Hide
Type `cd /path/to/project` Enter
Sleep 0.5s
Type `go run cmd/gbm/main.go worktree testadd` Enter
Sleep 1.5s
Show

# Select "feature" workflow type
Type "j" # Down
Sleep 0.3s
Type Enter
Sleep 1s

# Select JIRA issue
Type "PROJ" # Filter
Sleep 0.5s
Type Enter
Sleep 1s

# Auto-populated branch name
Sleep 0.5s
Type Enter
Sleep 1s

# Select base branch
Type "j" # Down
Sleep 0.3s
Type Enter
Sleep 1s

# Confirm
Type Enter
Sleep 0.5s

Screenshot demo-complete.png
```

**Impact**: Confidence in wizard behavior. Documentation with screenshots. Easy regression testing.

---

### 6. **Error Handling**

#### Current Issues

- **Nil error checks**: [fields/selector.go:152-154](../pkg/tui/fields/selector.go#L152-L154) always returns `nil`
- **Silent failures**: Async operations don't communicate errors to users
- **No error recovery**: Fields can't recover from failed operations
- **Missing validation messages**: TextInput has no validation feedback

**Best Practice Reference**: AGENTS.md Error Handling guidelines

#### Improvements

**A. Create Error Field Interface**

```go
// pkg/tui/field.go - enhance interface
type Field interface {
    // ... existing methods ...
    
    // SetError sets an error message to display to the user.
    SetError(error) Field
    
    // ClearError clears any error state.
    ClearError() Field
}
```

**B. Update Field Implementations**

```go
// pkg/tui/fields/textinput.go
type TextInput struct {
    // ... existing fields ...
    validationErr error
}

func (t *TextInput) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
    // ... handle input ...
    
    // Validate on each keystroke
    if err := t.validate(); err != nil {
        t.validationErr = err
        return t, nil
    }
    
    return t, nil
}

func (t *TextInput) View() string {
    view := t.renderInput()
    
    // Render validation error if present
    if t.validationErr != nil {
        errorView := lipgloss.NewStyle().
            Foreground(lipgloss.Color("196")).
            Render(t.validationErr.Error())
        return lipgloss.JoinVertical(lipgloss.Top, view, errorView)
    }
    
    return view
}

func (t *TextInput) SetError(err error) tui.Field {
    t.validationErr = err
    return t
}

func (t *TextInput) ClearError() tui.Field {
    t.validationErr = nil
    return t
}
```

**C. Handle Async Errors in Fields**

```go
// pkg/tui/fields/filterable.go
func (f *Filterable) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
    switch msg := msg.(type) {
    case async.FetchMsg[[]string]:
        f.isLoading = false
        if msg.Err != nil {
            // Show error but allow retry
            return f.SetError(msg.Err), nil
        }
        f.options = msg.Value
        f.ClearError()
        return f, nil
    }
    return f, nil
}
```

**Impact**: Better UX. Users understand why actions fail. Enables retry logic.

---

### 7. **Message Dumping for Debugging**

#### Current Issues

- **No message logging**: Can't debug complex wizard flows
- **Missing DEBUG mode**: No facility to enable detailed logging

**Best Practice Reference**: [BUBBLETEA.md §2](./BUBBLETEA.md#2-debug-with-message-dumps)

#### Improvements

**A. Add Message Dumping to Wizard**

```go
// pkg/tui/wizard.go
type Wizard struct {
    // ... existing fields ...
    dump io.Writer // For DEBUG mode
}

func NewWizardWithDebug(steps []Step, ctx *Context, debugWriter io.Writer) *Wizard {
    return &Wizard{
        steps:   steps,
        current: 0,
        ctx:     ctx,
        dump:    debugWriter,
    }
}

func (w *Wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if w.dump != nil {
        // Log message details
        fmt.Fprintf(w.dump, "[%s] %T: %+v\n", 
            time.Now().Format("15:04:05"), 
            msg, 
            msg)
    }
    
    // ... rest of Update logic ...
}
```

**B. Activate in testadd**

```go
// cmd/service/worktree_testadd.go
func runWorktreeTestaddCommand(cmd *cobra.Command, delayMs int, withConfig bool) error {
    // ... existing code ...
    
    var debugLog io.Writer
    if _, ok := os.LookupEnv("DEBUG"); ok {
        f, err := os.OpenFile("tui-debug.log", os.O_CREATE|os.O_WRONLY, 0o644)
        if err != nil {
            return fmt.Errorf("failed to open debug log: %w", err)
        }
        defer f.Close()
        debugLog = f
    }
    
    typeWizard := tui.NewWizardWithDebug([]tui.Step{...}, ctx, debugLog)
    wrapper := newTestaddWrapperModel(ctx, stepsMap, typeWizard)
    // ...
}
```

**Usage**: `DEBUG=1 go run . & tail -f tui-debug.log`

**Impact**: Dramatically faster debugging of complex wizard flows.

---

### 8. **Custom Field State Management**

#### Current Issues

- **Hard-coded field keys**: [wizard.go:234-255](../pkg/tui/wizard.go#L234-L255) has a switch statement for each field key
- **No generic field tracking**: New fields require modifying Wizard.storeFieldValue()
- **Merge workflow custom fields unsupported**: [workflows.go:265-301](../pkg/tui/workflows/workflows.go#L265-L301) can't track source/target branch

#### Improvements

**A. Create Generic Field Storage**

```go
// pkg/tui/context.go - enhance WorkflowState
type WorkflowState struct {
    // Standard fields
    WorkflowType string
    WorktreeName string
    BranchName   string
    BaseBranch   string
    JiraIssue    *JiraIssue
    
    // Generic field storage for custom fields
    CustomFields map[string]interface{}
}

func (ws *WorkflowState) SetField(key string, value interface{}) {
    if ws.CustomFields == nil {
        ws.CustomFields = make(map[string]interface{})
    }
    ws.CustomFields[key] = value
}

func (ws *WorkflowState) GetField(key string) interface{} {
    if ws.CustomFields == nil {
        return nil
    }
    return ws.CustomFields[key]
}
```

**B. Update Wizard to Use Generic Storage**

```go
// pkg/tui/wizard.go
func (w *Wizard) storeFieldValue() {
    field := w.currentField()
    key := field.GetKey()
    value := field.GetValue()
    
    // Try to store in standard fields first
    switch key {
    case FieldKeyWorkflowType:
        if v, ok := value.(string); ok {
            w.ctx.State.WorkflowType = v
        }
    // ... other standard fields ...
    default:
        // Store in custom fields map
        w.ctx.State.SetField(key, value)
    }
}
```

**C. Update Merge Workflow**

```go
// pkg/tui/workflows/workflows.go
func ProcessMergeWorkflow(wizard *tui.Wizard, ctx *tui.Context) error {
    state := wizard.State()
    
    // Retrieve from custom fields
    sourceBranch, _ := state.GetField("source_branch").(string)
    targetBranch, _ := state.GetField("target_branch").(string)
    
    // ... generate worktree and branch names ...
}
```

**Impact**: Extensible field system. No Wizard modifications needed for new fields.

---

### 9. **Documentation and Comments**

#### Current Issues

- **Missing best practices docs**: No reference to [BUBBLETEA.md](./BUBBLETEA.md) in code
- **No architecture diagram**: Complex Wizard/Field interaction not visualized
- **Undocumented patterns**: Skip logic, stage transitions, command routing need explanation

#### Improvements

**A. Add Architecture Documentation**

Create [spec/TUI_ARCHITECTURE.md](./TUI_ARCHITECTURE.md) with:
- Component diagram (Field → Wizard → Navigator)
- Message flow diagram
- State transition examples
- Best practices checklist

**B. Add Inline Architecture Comments**

```go
// pkg/tui/wizard.go

// Wizard orchestrates a multi-step form flow following the Elm Architecture pattern.
// See BUBBLETEA.md §5 for hierarchical model design.
//
// Architecture:
//   Input Messages (KeyMsg, etc.)
//          ↓
//   Update() - delegates to current Field
//          ↓
//   Field processes and returns NextStepMsg/PrevStepMsg
//          ↓
//   Wizard transitions to next/prev non-skipped step
//          ↓
//   View() renders current Field's UI
//
// All async operations (JIRA fetches, etc.) return Commands (see async/messages.go)
// rather than blocking the event loop (BUBBLETEA.md §1).
type Wizard struct {
    // ...
}
```

**Impact**: Easier onboarding. Better maintainability.

---

## Implementation Priority

### Phase 1 (Critical - Event Loop)
1. Convert async operations to commands (§1)
2. Fix message ordering with tea.Sequence (§2)
3. Add error handling (§6)

### Phase 2 (Important - Testing & Architecture)
4. Create Navigator for model composition (§3)
5. Add teatest suite (§5)
6. Generic field storage (§8)

### Phase 3 (Enhancement - Polish)
7. Layout helpers and responsive design (§4)
8. Debug message dumping (§7)
9. Documentation (§9)

---

## Success Criteria

- [ ] Event loop never blocks (Update/View < 16ms)
- [ ] 80%+ test coverage with teatest
- [ ] No synchronous service calls in event loop
- [ ] Responsive UI resizes correctly
- [ ] Custom fields work without Wizard modifications
- [ ] Debug mode captures all messages for analysis
- [ ] All async operations show loading state

---

## References

- [Bubble Tea Best Practices](./BUBBLETEA.md)
- [AGENTS.md Development Rules](../AGENTS.md)
- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [PUG Implementation](https://github.com/leg100/pug) - Reference for complex TUI patterns
