# Testing & Validation Strategy for TUI Phase 2

## Overview

This document outlines the comprehensive testing and validation approach for Phase 2 improvements (Navigator, Teatest, Custom Fields) that addresses your core pain points:

1. **Async patterns**: Spinners/progress for JIRA/Git without blocking event loop
2. **Testability**: Fast feedback loop for TUI validation without manual clicking
3. **Responsive design**: Terminal resizes + visual inspection
4. **Reusability**: Navigator enables proper composition across workflows

---

## Architecture: Testing Stack

### Layer 1: Unit Testing (go test)

**Tools**: `testify/assert`, mocked services, `teatest` (minimal)

**Scope**:
- Individual field behavior (Filterable, TextInput, etc.)
- Message routing (NextStepMsg, PrevStepMsg, etc.)
- Async command execution
- Custom field storage

**What's NOT tested here**: Full terminal rendering, visual appearance

---

### Layer 2: Integration Testing (go test + teatest)

**Tools**: `teatest` from Bubble Tea, mocked services

**Scope**:
- Full wizard flows end-to-end
- Message sequencing (Init → Focus → Blur → NextStepMsg)
- Field initialization order
- Window resize handling
- State accumulation across steps

**What's NOT tested here**: Real terminal appearance, latency

---

### Layer 3: Visual Validation (VHS recordings)

**Tools**: `vhs` (terminal recorder), committed .tape files

**Scope**:
- Actual TUI appearance and interactions
- Loading spinners and progress states
- Error states and user feedback
- Multi-step workflows
- Merge suggestions, branch names, etc.

**Workflow**:
```
1. Create .tape script (user interaction script)
2. Run: vhs < script.tape  → generates script.gif
3. Commit .tape + .gif to repo
4. On CI: regenerate gifs, diff against committed version
5. If different: flag for review (visual regression)
```

**Integration with Ralph**: Before closing a story, Ralph:
- Runs `go test ./...` (unit + integration)
- Runs VHS recording for affected workflow
- Commits artifacts
- Reports: "X tests pass, Y VHS recordings generated"

---

## Why This Stack Works For Your Pain Points

### Pain Point 1: "Struggling with async activities (JIRA/Git) + spinners"

**Solution**:
1. **Teatest integration tests** verify message flow:
   - Init field → returns FetchCmd
   - FetchCmd completes → FetchMsg arrives
   - Update() handles FetchMsg → updates state
   - View() shows spinner while loading

2. **VHS recordings** show the user what they see:
   - Spinner animation playing while data loads
   - Data appears when ready
   - No blocking, responsive to ESC

---

### Pain Point 2: "Hard to create feedback loop + validate TUI"

**Solution**:
1. **Fast teatest loop**: `go test ./... -v` (< 1s)
   - No real terminal needed
   - Deterministic inputs (mock delays)
   - Binary assertions (pass/fail)

2. **Visual teatest assertions**: Capture rendered View()
   ```go
   tm := teatest.NewTestModel(model)
   tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
   output := tm.GetViewAsString()
   assert.Contains(t, output, "Loading...")
   ```

3. **VHS for final validation**: See actual appearance
   - Run once per story
   - Committed to repo as baseline
   - Quick visual scan

---

### Pain Point 3: "Terminal resizes + responsive feedback"

**Solution**:
1. **Teatest can send WindowSizeMsg**:
   ```go
   tm := teatest.NewTestModel(wizard)
   tm.Send(tea.WindowSizeMsg{Width: 120, Height: 30})
   tm.Send(tea.WindowSizeMsg{Width: 60, Height: 15})  // Resize
   // Assert layout adapts
   ```

2. **VHS can test with real resizes**:
   ```
   Pause 500ms
   Type "resize 120 40"  # VHS built-in resize
   Pause 1000ms
   ```

---

### Pain Point 4: "Don't understand Bubble Tea patterns"

**Solution**:
1. **Reference implementations**: Each field type gets teatest examples
2. **Pattern templates**: Copy-paste async command pattern
3. **Architecture comments**: Inline documentation (from spec §9)

---

## Testing Patterns (By Component)

### Pattern 1: Async Field with Loading Spinner

**Location**: `pkg/tui/async/messages.go` (new)

```go
// FetchMsg[T any] signals completion of async fetch
type FetchMsg[T any] struct {
    Value T
    Err   error
}

// FetchCmd returns a command that fetches asynchronously
func FetchCmd[T any](fetch func() (T, error)) tea.Cmd {
    return func() tea.Msg {
        value, err := fetch()
        return FetchMsg[T]{Value: value, Err: err}
    }
}
```

**Unit test pattern** (`fields/filterable_test.go`):

```go
func TestFilterableLoadingState(t *testing.T) {
    // Setup
    mockFetch := func() ([]Option, error) {
        time.Sleep(100 * time.Millisecond)
        return []Option{{Label: "opt1", Value: "v1"}}, nil
    }
    f := NewFilterable("test", "title", "desc", []Option{}).
        WithOptionsFunc(mockFetch)
    
    // Init should return FetchCmd
    cmd := f.Init()
    assert.NotNil(t, cmd)
    
    // Send FetchMsg when fetch completes
    msg := cmd()
    fetchMsg, ok := msg.(async.FetchMsg[[]Option])
    assert.True(t, ok)
    assert.NoError(t, fetchMsg.Err)
    assert.Len(t, fetchMsg.Value, 1)
    
    // Update with FetchMsg should populate options
    f2, _ := f.Update(fetchMsg)
    filterable := f2.(*Filterable)
    assert.Len(t, filterable.options, 1)
}
```

**Integration test pattern** (`wizard_test.go`):

```go
func TestWizardAsyncFieldFlow(t *testing.T) {
    // Create wizard with async filterable
    mockJira := testing.NewMockJiraService()
    ctx := tui.NewContext().WithJiraService(mockJira)
    
    filterable := NewFilterable("jira", "Select Issue", "", []Option{}).
        WithOptionsFunc(func() ([]Option, error) {
            issues, err := mockJira.FetchIssues()
            opts := make([]Option, len(issues))
            for i, issue := range issues {
                opts[i] = Option{Label: issue.Key + ": " + issue.Summary, Value: issue.Key}
            }
            return opts, err
        })
    
    w := tui.NewWizard([]tui.Step{
        {Name: "jira_select", Field: filterable},
        {Name: "confirm", Field: NewConfirm("confirm", "Proceed?")},
    }, ctx)
    
    tm := teatest.NewTestModel(w)
    
    // Init starts the fetch
    tm.Send(tm.Init())
    
    // Simulate delay + fetch completion
    time.Sleep(200 * time.Millisecond)
    
    // Send enter to select option
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    
    // Wizard should advance
    final := tm.FinalModel.(*tui.Wizard)
    assert.True(t, final.currentField().GetKey() == "confirm")
}
```

---

### Pattern 2: Navigator (Multi-Screen Composition)

**Location**: `pkg/tui/navigator.go` (new)

```go
type Navigator struct {
    stack   []tea.Model
    current tea.Model
}

func (n *Navigator) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Delegate to current model
    updated, cmd := n.current.Update(msg)
    n.current = updated
    
    // Check for navigation messages
    if navMsg, ok := msg.(NavigateMsg); ok {
        n.Push(navMsg.Model)
        return n, navMsg.Model.Init()
    }
    
    return n, cmd
}
```

**Test pattern**:

```go
func TestNavigatorTransitions(t *testing.T) {
    typeSelector := NewTypeSelector()
    nav := tui.NewNavigator(typeSelector)
    
    tm := teatest.NewTestModel(nav)
    
    // Select "feature" workflow
    tm.Send(tea.KeyMsg{String: "f"})
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
    
    // Navigator should push workflow wizard
    final := tm.FinalModel.(*Navigator)
    assert.Len(t, final.stack, 2)
}
```

---

### Pattern 3: VHS Recording Script

**Location**: `spec/vhs/testadd_feature_workflow.tape`

```tape
# Testadd feature workflow demo
# Shows: type selection → jira fetch + spinner → branch name generation → confirmation

Output "testadd_feature_workflow.gif"
Set Theme "Catppuccin Mocha"
Set Height 30
Set Width 120
Set Margin 10
Set MarginFill "#1e1e2e"

# Run testadd with 500ms simulated latency
Type "go run ./cmd/service testadd --delay 500"
Sleep 500ms
Enter

# Wait for initial render
Sleep 500ms

# Select Feature workflow
Type "f"
Enter

# Wait for JIRA fetch to start (should see spinner)
Sleep 1000ms

# JIRA issues should be loaded now, select first one
Type "PROJ"
Sleep 300ms
Enter

# Branch name should be auto-generated, just confirm
Sleep 300ms
Enter

# Select main as base branch
Type "main"
Sleep 300ms
Enter

# Confirmation step
Sleep 300ms
Enter

# Wait for completion message
Sleep 500ms
```

**To generate**:
```bash
cd spec/vhs
vhs < testadd_feature_workflow.tape
# Creates testadd_feature_workflow.gif
git add testadd_feature_workflow.tape testadd_feature_workflow.gif
```

---

## Testing Checklist (Per Phase 2 Story)

### For each story, verify:

- [ ] **Unit tests**:
  - `go test ./... -v` passes
  - > 80% coverage for changed files
  - Edge cases tested (empty lists, errors, resize, etc.)

- [ ] **Integration tests** (teatest):
  - Full workflow from Init to completion
  - Message ordering correct (Init → Focus → selection → NextStepMsg)
  - Async operations handled (spinner shown, data loaded, updated view)
  - Error states shown to user

- [ ] **VHS recording**:
  - Created `.tape` script for new workflow/feature
  - Generated `.gif` shows expected behavior
  - Committed to repo as baseline

- [ ] **Manual smoke test** (optional, ~5min):
  - Run `go run ./cmd/service testadd`
  - Interact manually to verify
  - Can be skipped if teatest + VHS comprehensive

---

## Running Tests in Ralph Loop

### Test command Ralph should use:

```bash
# Unit + integration tests (fast)
go test ./... -v -race

# Coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html

# VHS recording (if spec includes .tape file)
if [ -f "spec/vhs/${STORY_WORKFLOW}.tape" ]; then
    cd spec/vhs
    vhs < "${STORY_WORKFLOW}.tape"
    cd -
fi
```

### On test failure:

1. **go test fails**: Show error, stop
2. **VHS differs from baseline**: Generate new baseline, flag for review
3. **Coverage drops**: Show which lines, require explanation

---

## File Structure

```
pkg/tui/
├── async/
│   ├── messages.go          (NEW: FetchMsg, FetchCmd)
│   ├── eval.go              (existing, may refactor)
│   └── eval_test.go
├── navigator.go             (NEW: root model)
├── navigator_test.go        (NEW)
├── wizard.go
├── wizard_test.go           (UPDATE: add async patterns)
├── fields/
│   ├── filterable.go        (UPDATE: use FetchCmd)
│   ├── filterable_test.go   (UPDATE: add async tests)
│   └── ...
└── context.go               (UPDATE: add CustomFields)

spec/
├── TESTING_VALIDATION_STRATEGY.md  (THIS FILE)
├── vhs/
│   ├── testadd_feature_workflow.tape
│   ├── testadd_feature_workflow.gif
│   ├── testadd_bug_workflow.tape
│   ├── testadd_bug_workflow.gif
│   ├── testadd_merge_workflow.tape
│   ├── testadd_merge_workflow.gif
│   └── ... (one per major workflow)
└── ...
```

---

## Success Criteria

- [ ] All unit tests pass (`go test ./...`)
- [ ] All integration tests pass (with teatest)
- [ ] VHS recordings commit cleanly (no noisy diffs)
- [ ] New async patterns documented with examples
- [ ] Navigator enables clean multi-screen composition
- [ ] Custom field storage works without Wizard modifications
- [ ] Can run full test suite in < 10s
- [ ] New dev can understand patterns from tests + inline comments

---

## References

- [BUBBLETEA.md](./BUBBLETEA.md) - Best practices
- [TUI_IMPROVEMENTS.md](./TUI_IMPROVEMENTS.md) - Implementation spec
- [Teatest documentation](https://github.com/charmbracelet/bubbletea/tree/main/testutil)
- [VHS documentation](https://github.com/charmbracelet/vhs)
