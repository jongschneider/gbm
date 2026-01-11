# TUI Package Architecture

This document describes the architecture of the `pkg/tui` package, a Bubble Tea-based terminal user interface system for interactive wizards and workflows.

## Overview

The TUI package provides a production-ready framework for building interactive terminal interfaces using the Bubble Tea framework. It follows best practices for event loop performance, message routing, and state management.

**Key Design Principles:**
- Non-blocking event loop (async/FetchCmd pattern)
- Composable field components with consistent interface
- Stack-based navigation for multi-screen workflows
- Extensible state management with custom fields

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                     Application Layer                        │
│                  (cmd/service/worktree_*)                    │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                      Navigator                               │
│          (Root Model - Stack-Based Navigation)              │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                      Wizard                                  │
│         (Field Orchestrator - Step-by-Step Forms)           │
├──────────────────────────┬──────────────────────────────────┤
│      FieldImpl 1          │      FieldImpl 2                  │
│  (Textinput/Selector)    │  (Filterable/Custom)             │
│  + Validation            │  + Async Loading                 │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                    Async Messages                            │
│         (FetchMsg/FetchCmd - Non-Blocking I/O)             │
└──────────────────────────┬──────────────────────────────────┘
                           │
┌──────────────────────────▼──────────────────────────────────┐
│                   External Services                          │
│            (Git, JIRA, Repo Config)                         │
└─────────────────────────────────────────────────────────────┘
```

## Component Details

### 1. Navigator (pkg/tui/navigator.go)

**Purpose**: Manage navigation between different screens/models in a stack-based manner.

**Key Methods**:
- `Init()` - delegates to current model
- `Update(msg)` - delegates to current, handles NavigateMsg
- `View()` - renders current model's view
- `Push(model)` - push new model to top of stack
- `Pop()` - pop current model from stack
- `Depth()` - get current stack depth

**Design Pattern**:
```
Stack: [Screen1, Screen2, Screen3] ← top
       View() returns Screen3's View()
```

**Benefits**:
- Simple back navigation (Pop)
- Decoupled screen models
- No complex wrapper models

**See**: [navigator.go](./navigator.go), [navigator_test.go](./navigator_test.go)

### 2. Wizard (pkg/tui/wizard.go)

**Purpose**: Orchestrate a sequence of input fields to collect workflow data.

**Key Methods**:
- `Init()` - initialize first field
- `Update(msg)` - route messages to current field, handle navigation
- `View()` - render current field with navigation hints
- `Fields()` - list all configured steps

**Field Routing**:
```go
// Wizard receives tea.Msg
// Routes to current field.Update(msg)
// Field returns (Field, tea.Cmd)
// Wizard advances or revisits based on field completion
```

**Key Features**:
- Sequential step-through
- Field validation before advancing
- Back/forward navigation
- Stores field values in WorkflowState

**See**: [wizard.go](./wizard.go), [wizard_test.go](./wizard_test.go)

### 3. Fields (pkg/tui/fields/)

**Purpose**: Reusable input components for different data types.

**Field Interface** (pkg/tui/field.go):
```go
type Field interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (Field, tea.Cmd)
    View() string
    Focus() tea.Cmd
    Blur() tea.Cmd
    GetKey() string
    GetValue() any
    IsComplete() bool
    // ... more methods
}
```

**Built-in Fields**:

#### Textinput (fields/textinput.go)
- Simple string input with validation
- Character limit support
- Placeholder text
- Password masking (future)

#### Selector (fields/selector.go)
- Fixed list of options
- Cursor-based selection
- Up/Down navigation
- Enter to confirm

#### Filterable (fields/filterable.go) - **Async Pattern**
- Dynamic filtering of large lists
- **Async option loading** with FetchCmd
- Spinner animation while loading
- Error handling for failed loads
- User can cancel during load

**Async Filterable Flow**:
```
1. Init() returns FetchCmd (starts async load)
2. FetchCmd executes in Bubble Tea runtime
3. Returns FetchMsg[[]Option] with data/error
4. Update() handles FetchMsg, populates options
5. View() shows spinner during load, data after
```

**See**: [fields/filterable.go](./fields/filterable.go), [fields/filterable_async_test.go](./fields/filterable_async_test.go)

### 4. Async Messages (pkg/tui/async/messages.go)

**Purpose**: Enable non-blocking asynchronous operations without blocking the event loop.

**Key Types**:
```go
type FetchMsg[T any] struct {
    Value T
    Err   error
}

func FetchCmd[T any](fetch func() (T, error)) tea.Cmd
```

**Pattern**:
```go
// In Field.Init() or Update():
return async.FetchCmd(func() ([]Option, error) {
    // This runs in Bubble Tea's async executor
    // Doesn't block the event loop
    return fetchFromJIRA()
})

// In Update(), handle the result:
case fetchMsg := msg.(async.FetchMsg[[]Option]):
    if fetchMsg.Err != nil {
        f.loadErr = fetchMsg.Err
    } else {
        f.options = fetchMsg.Value
    }
```

**Best Practices**:
- Return FetchCmd from Init() for initial loads
- Handle FetchMsg in Update() to apply results
- Show spinner/loading state while fetching
- Show error messages on failure
- Allow user to cancel (Ctrl+C) while loading

**See**: [async/messages.go](./async/messages.go), [async/messages_test.go](./async/messages_test.go)

### 5. WorkflowState (pkg/tui/context.go)

**Purpose**: Hold all data collected across wizard steps.

**Standard Fields**:
- `WorkflowType` - "feature" / "bug" / "hotfix" / "merge"
- `WorktreeName` - JIRA issue or custom name
- `BranchName` - auto-generated or custom
- `BaseBranch` - source branch for new work
- `JiraIssue` - populated JIRA issue details

**Custom Fields** (Story 8):
```go
// Store arbitrary data without modifying Wizard
state.SetField("merge_strategy", "squash")
state.SetField("review_count", 2)

// Retrieve with type assertion
strategy, _ := state.GetField("merge_strategy").(string)
```

**Benefits**:
- Wizard doesn't need to know about all possible fields
- New workflows can add fields without code changes
- Type-safe with runtime type assertion

**See**: [context.go](./context.go), [context_custom_fields_test.go](./context_custom_fields_test.go)

### 6. Theme (pkg/tui/theme.go)

**Purpose**: Centralized styling for all UI components.

**Features**:
- Focused/Blurred styles for each element
- Title, description, input, error colors
- Consistent look across fields
- Easy to customize

## Message Flow

### Single Field Message Flow
```
Bubble Tea Event Loop
         │
         ▼
    Wizard.Update(msg)
         │
         ├─→ Current Field.Update(msg)
         │         │
         │         ├─→ Handle keyboard input
         │         ├─→ Handle async FetchMsg
         │         ├─→ Update internal state
         │         └─→ Return (Field, Cmd)
         │
         ├─→ Check field completion
         ├─→ Store value in WorkflowState
         └─→ Advance to next step (or repeat)
```

### Async Operation Flow
```
Field.Init()
     │
     └─→ return FetchCmd(fetchFunc)
              │
              ▼
         Bubble Tea Runtime
              │
              ├─→ Execute fetchFunc() in background
              │         │
              │         ├─→ I/O operation (network, file, etc)
              │         └─→ Return (data, error)
              │
              └─→ Send FetchMsg[T] back to Wizard
                       │
                       ▼
                   Wizard.Update(FetchMsg)
                       │
                       ▼
                   Field.Update(FetchMsg)
                       │
                       ├─→ Populate options from FetchMsg.Value
                       ├─→ Set loadErr from FetchMsg.Err
                       ├─→ Set isLoading = false
                       └─→ Update view to show data/error
```

## Testing Strategy

### Unit Tests (Story 4 - Teatest Helpers)
- Test fields in isolation with mock data
- Verify message handling without async
- Check view rendering and validation

**Example**:
```go
func TestFilterable_Update_HandlesFetchMsg(t *testing.T) {
    f := NewFilterable("key", "Select", "", []Option{})
    f.isLoading = true
    
    fetchMsg := async.FetchMsg[[]Option]{
        Value: []Option{...},
        Err: nil,
    }
    
    field, _ := f.Update(fetchMsg)
    f = field.(*Filterable)
    
    assert.False(t, f.isLoading)
    assert.Equal(t, 2, len(f.options))
}
```

**See**: [testutil/teatest_helpers.go](../../testutil/teatest_helpers.go)

### Integration Tests (Story 5 - Async Integration)
- Test complete flows: Wizard → Field → Async → View
- Verify field sequencing
- Check error recovery

**Example**:
```go
func TestFilterable_AsyncLoadsOnFocus(t *testing.T) {
    f := NewFilterable("key", "Select", "", []Option{})
    f.WithOptionsFuncAsync(func() ([]Option, error) {
        return []Option{{Label: "A", Value: "a"}}, nil
    })
    
    cmd := f.Init()
    msg := cmd()  // Execute the FetchCmd
    
    fetchMsg := msg.(async.FetchMsg[[]Option])
    field, _ := f.Update(fetchMsg)
    
    assert.Equal(t, 1, len(field.(*Filterable).options))
}
```

**See**: [fields/filterable_async_test.go](./fields/filterable_async_test.go)

## Extending the TUI

### Adding a New Field Type

1. Implement the `Field` interface:
```go
type MyField struct {
    key          string
    value        string
    focused      bool
    // ... other state
}

func (f *MyField) Init() tea.Cmd { ... }
func (f *MyField) Update(msg tea.Msg) (Field, tea.Cmd) { ... }
func (f *MyField) View() string { ... }
// ... implement other Field methods
```

2. Add validation/completion logic:
```go
func (f *MyField) IsComplete() bool {
    return len(strings.TrimSpace(f.value)) > 0
}
```

3. Handle async operations if needed:
```go
func (f *MyField) Init() tea.Cmd {
    if f.needsAsyncLoad {
        return async.FetchCmd(f.loadData)
    }
    return nil
}

func (f *MyField) Update(msg tea.Msg) (Field, tea.Cmd) {
    case fetchMsg := msg.(async.FetchMsg[MyData]):
        f.data = fetchMsg.Value
        f.loadErr = fetchMsg.Err
        return f, nil
}
```

### Adding a New Workflow

1. Create workflow steps using built-in fields:
```go
func NewFeatureWorkflow(ctx *tui.Context) []tui.Step {
    return []tui.Step{
        SelectWorkflowType(),
        SelectJiraIssue(ctx),
        SelectBaseBranch(ctx),
        ConfirmAction(ctx),
    }
}
```

2. Use Navigator for multi-screen flows:
```go
type FeatureWorkflow struct {
    nav *tui.Navigator
    ctx *tui.Context
}

func (fw *FeatureWorkflow) Init() tea.Cmd {
    wizard := tui.NewWizard(NewFeatureWorkflow(fw.ctx), fw.ctx)
    fw.nav.Push(wizard)
    return fw.nav.Init()
}
```

3. Store custom fields as needed:
```go
fw.ctx.State.SetField("merge_strategy", "squash")
```

## Performance Considerations

### Event Loop Responsiveness
- **DO**: Use FetchCmd for I/O operations
- **DON'T**: Call blocking functions in Update()
- **DO**: Show spinner/progress during async operations

**Before** (blocking):
```go
func (f *Filterable) Update(msg tea.Msg) (Field, tea.Cmd) {
    // ❌ BLOCKS EVENT LOOP - freezes UI
    options, _ := f.optionsFunc.Get()
    f.options = options
    return f, nil
}
```

**After** (non-blocking):
```go
func (f *Filterable) Init() tea.Cmd {
    // ✅ DOESN'T BLOCK - returns immediately
    return async.FetchCmd(f.optionsFunc)
}

func (f *Filterable) Update(msg tea.Msg) (Field, tea.Cmd) {
    case fetchMsg := msg.(async.FetchMsg[[]Option]):
        f.options = fetchMsg.Value
        return f, nil
}
```

### Message Ordering
- Use `tea.Sequence()` when operations must happen in order
- Prefer `tea.Batch()` for independent operations
- Document dependencies between fields

## Common Patterns

### Conditional Fields
```go
func shouldShowField(ctx *tui.Context) bool {
    return ctx.State.WorkflowType == "feature"
}
```

### Field Defaults
```go
func setFieldDefault(ctx *tui.Context, key string) string {
    switch key {
    case "branchName":
        return generateBranchName(ctx.State.WorktreeName)
    default:
        return ""
    }
}
```

### Error Recovery
```go
func (f *Filterable) Update(msg tea.Msg) (Field, tea.Cmd) {
    case fetchMsg := msg.(async.FetchMsg[[]Option]):
        if fetchMsg.Err != nil {
            // Show error but allow user to continue/retry
            f.loadErr = fetchMsg.Err
            f.options = []Option{} // fallback
        }
        return f, nil
}
```

## Debugging Tips

### Enable Message Logging
```go
func (w *Wizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Log incoming messages for debugging
    log.Printf("Wizard received: %T: %+v", msg, msg)
    // ... rest of implementation
}
```

### View Current State
```go
func DebugView(ctx *tui.Context) string {
    return fmt.Sprintf("State: %+v\nCustomFields: %+v",
        ctx.State, ctx.State.CustomFields)
}
```

### Test Async with Delays
```go
func TestWithSlowFetch(t *testing.T) {
    f := NewFilterable("key", "Select", "", []Option{})
    f.WithOptionsFuncAsync(func() ([]Option, error) {
        time.Sleep(100 * time.Millisecond) // Simulate slowness
        return []Option{...}, nil
    })
    // Verify spinner is shown during wait
    assert.Contains(t, f.View(), "Loading")
}
```

## References

- **Bubble Tea**: https://github.com/charmbracelet/bubbletea
- **Best Practices**: [spec/BUBBLETEA.md](../../spec/BUBBLETEA.md)
- **Implementation Guide**: [spec/TUI_IMPROVEMENTS.md](../../spec/TUI_IMPROVEMENTS.md)
- **Testing Strategy**: [spec/TESTING_VALIDATION_STRATEGY.md](../../spec/TESTING_VALIDATION_STRATEGY.md)

## Future Enhancements

- [ ] Responsive layout helpers (handle terminal resize)
- [ ] Composite fields (nested field groups)
- [ ] Custom validators framework
- [ ] Undo/redo support for field values
- [ ] Keyboard shortcuts customization
- [ ] Accessibility improvements (screen reader support)
