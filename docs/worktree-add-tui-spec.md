# Worktree Add TUI Specification

This document captures the requirements and user experience for the `gbm worktree add` TUI command, to guide a reimplementation with cleaner component architecture.

## Working Approach

**IMPORTANT:** This project follows an incremental development approach:

1. **One feature at a time** - Work on a single feature from `prd.json`
2. **Commit after each feature** - Leave codebase in clean, working state with descriptive commit message
3. **Update progress.md** - Record what was done, any issues, and decisions made
4. **Update prd.json** - Change `"passes": false` → `"passes": true` ONLY when feature is complete and working
5. **Revert if broken** - Use `git revert` or `git checkout` to recover if changes break things

**Files:**
- `docs/prd.json` - Feature list with pass/fail status (edit only `passes` field)
- `docs/progress.md` - Human-readable progress log and notes
- `docs/worktree-add-tui-spec.md` - This specification (reference only)

**It is unacceptable to:**
- Remove or edit feature descriptions in prd.json
- Mark features as passing when they don't work
- Work on multiple features simultaneously
- Leave the codebase in a broken state

---

## Overview

When `gbm2 worktree add` is run without arguments, it launches an interactive TUI that guides the user through creating a worktree.

**Scope for Test Implementation:** Feature and Hotfix workflows only.

---

## Workflows

### Feature Workflow

**Steps:**
1. Select workflow type → "Feature"
2. Select/enter worktree name (JIRA filterable selector)
3. Enter/edit branch name (pre-filled with `feature/{ISSUE}-{slug}`)
4. **Conditional:** Check if branch exists
   - If exists → go to execution
   - If missing → select base branch
5. Confirm creation (shows branch name and base)
6. Execute (create worktree, copy files, etc.)

**Naming:**
- Worktree directory: `{ISSUE_KEY}` (e.g., `INGSVC-6468`)
- Branch: `feature/{ISSUE_KEY}-{slugified-summary}` (e.g., `feature/INGSVC-6468-email-ical-extra-text-time-parsing`)

---

### Hotfix Workflow

**Steps:**
1. Select workflow type → "Hotfix"
2. Select/enter worktree name (JIRA filterable selector)
3. Select base branch (typically production/release) - **asked before branch name**
4. Enter/edit branch name (pre-filled with `hotfix/{ISSUE}-{slug}`)
5. Confirm creation
6. Execute

**Naming:**
- Worktree directory: `HOTFIX_{ISSUE_KEY}` (e.g., `HOTFIX_INGSVC-6468`)
- Branch: `hotfix/{ISSUE_KEY}-{slugified-summary}` (e.g., `hotfix/INGSVC-6468-email-ical-extra-text-time-parsing`)

**Key Differences from Feature:**
- Base branch is asked **immediately after worktree name** (not conditionally)
- Worktree directory has `HOTFIX_` prefix
- Branch has `hotfix/` prefix instead of `feature/`

---

## Screen Specifications

### Screen 1: Workflow Type Selector

```
┃ What type of worktree do you want to create?
┃ > Feature
┃   Bug Fix
┃   Hotfix
┃   Mergeback

↑ up • ↓ down • / filter • enter submit

ctrl+c: cancel
```

**Component:** Single-select list
**Options:** Feature, Bug Fix, Hotfix, Mergeback
**For test implementation:** Only Feature and Hotfix

---

### Screen 2: Worktree Name (JIRA Selector)

```
Worktree name
Select JIRA ticket or enter custom name

> Type to filter or enter custom value...



▸ INGSVC-6468: EMAIL: ical extra text time parsing
  INGSVC-6457: New Integration - Facebook Business Pages
  INGSVC-6446: Microsoft Copilot - Update the media type to AI Interaction
  ...

  •••
↑/↓ navigate • enter select • esc cancel

ctrl+c: cancel
```

**Component:** Filterable select with custom input
**Data Source:** JIRA issues (assigned to user)
**Display Format:** `{KEY}: {Summary}`
**Allows:** Typing custom value OR selecting from list
**Filtering:** Real-time as user types

---

### Screen 3: Branch Name Input

```
┃ Branch name
┃ Edit if needed
┃ > feature/INGSVC-6468-email-ical-extra-text-time-parsing

enter submit

ctrl+c: cancel
```

**Component:** Text input with default value
**Default:** Generated from workflow type + JIRA issue
**Format:** `{workflow_prefix}/{ISSUE_KEY}-{slugified-summary}`
- Feature: `feature/...`
- Hotfix: `hotfix/...`

---

### Screen 4: Base Branch Selector

```
Base branch
Branch 'feature/INGSVC-6468-...' doesn't exist. Select base:

> Type to filter or enter custom value...



▸ master (worktree: master)
  main
  production-2025-03-1
  production-2025-05-1
  ...

  ••••
↑/↓ navigate • enter select • esc cancel

ctrl+c: cancel
```

**Component:** Filterable select
**Data Source:** Git branches (local + remote)
**Display Format:** `{branch_name}` with optional `(worktree: {name})` suffix
**When shown:**
- Feature: Only if branch doesn't exist
- Hotfix: Always (with different prompt: "Select branch (typically production or release)")

---

### Screen 5: Confirmation

```
┃ Create branch 'feature/INGSVC-6468-email-ical-extra-text-time-parsing'?
┃ From base: master
┃
┃                               Yes     No

←/→ toggle • enter submit • y Yes • n No

ctrl+c: cancel
```

**Component:** Yes/No confirmation
**Shows:** Branch name, base branch
**Shortcuts:** `y` for Yes, `n` for No, arrows to toggle
**Used by:** Both Feature and Hotfix workflows

---

### Screen 6: Execution

No visible screen - TUI exits and worktree is created.

**Actions performed:**
1. Create git worktree
2. Copy files (if configured)
3. Create JIRA markdown (if JIRA issue)

**Output:** Path to stdout, messages to stderr

---

## UI Components Needed

Based on the screens above, these reusable components are needed:

| Component | Description | Used In |
|-----------|-------------|---------|
| **Selector** | Single-choice from static list | Workflow type |
| **FilterableSelect** | Searchable list + custom input | JIRA issues, branches |
| **TextInput** | Single-line editable text | Branch name |
| **Confirm** | Yes/No with summary | Feature confirmation |

---

## Navigation Patterns

| Key | Action |
|-----|--------|
| `Enter` | Submit/confirm current step |
| `Escape` | Go back one step |
| `Ctrl+C` | Cancel entire workflow |
| `↑/↓` | Navigate list items |
| `y/n` | Quick select for confirmations |
| `/` | Filter mode (on selectors) |

---

## Data Flow

```
WorkflowState {
  WorkflowType   string   // "feature", "hotfix"
  WorktreeName   string   // "INGSVC-6468"
  BranchName     string   // "feature/INGSVC-6468-..."
  BaseBranch     string   // "master"
  JiraIssue      *Issue   // Full JIRA issue data
}
```

Each screen modifies part of this state. On completion, state is used to execute git commands.

---

## Implementation Notes

### Current Pain Points (from existing implementation)
1. State machine library adds complexity
2. Tight coupling between FSM states and UI
3. Hard to reuse UI components across workflows
4. Screen flicker was an issue (now fixed with single Bubble Tea program)

### Goals for Test Implementation
1. **Component-based:** Reusable UI primitives
2. **Declarative flows:** Define workflow as sequence of steps
3. **Separation:** UI components vs workflow logic vs git operations
4. **Testable:** Each component testable in isolation

### Proposed Architecture

```
cmd/service/
  testadd.go           # New "test add" command entry point

pkg/tui/               # New UI package
  components/
    selector.go        # Single-select list
    filterable.go      # Filterable select + custom input
    textinput.go       # Text input field
    confirm.go         # Yes/No confirmation
  wizard/
    wizard.go          # Multi-step flow container
    step.go            # Step abstraction

  # Each component implements tea.Model
  # Wizard coordinates step transitions
```

---

## Decisions Made

| Question | Decision |
|----------|----------|
| JIRA data source | **Mock data** with async behavior simulation (loading states, cache invalidation) |
| Execution | **Dry-run only** - demonstrate UI without creating worktrees |
| Approach | **Discovery first** - answer architecture questions before implementation |

---

## Open Architecture Questions

### 1. Window/Component Resizing

How should components handle terminal resize events?

- Does each component need to know its available width/height?
- Should there be a layout system that distributes space?
- How does the current implementation handle this?

### 2. Theming

How should styling be managed?

- Hardcoded lipgloss styles per component?
- Centralized theme object passed to components?
- CSS-like approach with style inheritance?

### 3. Async Data Loading

The JIRA selector needs to handle:
- Initial loading state (spinner while fetching)
- Cached data (show immediately, refresh in background)
- Error states (network failure, auth error)
- Incremental loading (pagination?)

What patterns work well in Bubble Tea for this?

### 4. Component Communication

How do components communicate?
- Parent-child: Props down, events up?
- Shared state object (like WorkflowState)?
- Message passing through Bubble Tea's Cmd system?

### 5. Focus Management

When a wizard has multiple steps:
- How does focus transfer between components?
- Can components be "inactive" but still visible?
- How does tab-order work (if at all)?

### 6. Validation

Where does input validation live?
- In the component (TextInput validates itself)?
- In the wizard (validates before proceeding)?
- In a separate validation layer?

---

## Research Completed

### huh Library Architecture (for wizard flows)

**Hierarchical Structure:**
```
Form
  └── Groups (screens/steps)
        └── Fields (inputs on each screen)
```

**Key Patterns:**

| Pattern | How huh Does It |
|---------|-----------------|
| **Multi-step flow** | `Selector[Group]` manages current step, `nextGroupMsg`/`prevGroupMsg` for transitions |
| **Field interface** | Each field is a full `tea.Model` with `Focus()`/`Blur()` methods |
| **Focus management** | `nextFieldMsg` → current field `Blur()`, next field `Focus()` |
| **Theming** | `Theme` struct with `Focused`/`Blurred` `FieldStyles`, passed via `WithTheme()` |
| **Validation** | Validate on `Blur()`, blocks progression if errors exist |
| **Async data** | `Eval[T]` type with function + bindings hash for cache invalidation |

**The Eval[T] Pattern for Async Data:**
```go
type Eval[T any] struct {
    val      T           // Cached value
    fn       func() T    // Function to compute value
    bindings any         // Dependencies (hashed for cache key)
    loading  bool        // Shows spinner if loading > 25ms
    cache    map[uint64]T
}

// Usage - async options loading:
field.OptionsFunc(func() []Option[T] {
    return fetchJiraIssues()  // Called async, result cached
}, &someBindings)
```

**Field Interface:**
```go
type Field interface {
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() string

    Blur() tea.Cmd   // Called when leaving field
    Focus() tea.Cmd  // Called when entering field
    Error() error    // Current validation error
    Skip() bool      // Should this field be skipped?

    WithTheme(*Theme) Field
    WithWidth(int) Field
    GetKey() string
    GetValue() any
}
```

**Navigation Flow:**
```go
// In Group.nextField():
blurCmd := g.selector.Selected().Blur()  // Current field loses focus
if g.selector.OnLast() {
    return []tea.Cmd{blurCmd, nextGroup}  // Signal form to advance
}
g.selector.Next()
focusCmd := g.selector.Selected().Focus()  // New field gains focus
return []tea.Cmd{blurCmd, focusCmd}
```

### gh-dash Architecture (for dashboards)

See `docs/gh-dash-tui-analysis.md` for full analysis. Key patterns:
- `ProgramContext` for shared state (dimensions, theme, services)
- `Section` interface for polymorphic list views
- Task-based async with spinner feedback
- Message routing via `SectionMsg` wrapper

### Comparison: Which Pattern for What?

| Concern | gh-dash (Dashboard) | huh (Wizard) | **Our Choice** |
|---------|---------------------|--------------|----------------|
| **Layout** | Multiple sections visible | One step at a time | **huh** - wizard is linear |
| **Navigation** | Tabs between sections | Linear next/prev | **huh** - steps are sequential |
| **Async** | Task system with spinner | `Eval[T]` with caching | **Hybrid** - Eval for data, Task for operations |
| **Context** | `ProgramContext` everywhere | `Theme` passed down | **gh-dash** - need services too |
| **Theming** | `Styles` in context | `Theme` struct | **huh** - simpler for forms |

### Recommended Hybrid Approach

1. **Use huh's Form/Group/Field hierarchy** for wizard structure
2. **Use huh's Eval[T] pattern** for async JIRA data loading
3. **Use gh-dash's ProgramContext** for sharing services (git, jira)
4. **Use gh-dash's Task pattern** for execution step (actual git operations)

### Remaining Research

1. ~~Look at how `huh` handles these concerns~~ ✅ Done
2. ~~Look at how `bubbles` components are designed~~ (covered by huh analysis)
3. ~~See if there are any good examples of multi-step wizards in Bubble Tea~~ ✅ huh is the example
4. ~~Understand the existing `filterable_select.go` implementation~~ ✅ Done

### Existing FilterableSelect Analysis

**Location:** `cmd/service/filterable_select.go`

**What Works Well:**
- Clean structure: title, description, text input, list
- Supports selection from list OR custom text entry
- ESC = go back, Ctrl+C = cancel, Enter = select
- Handles window resize
- Has `StepModel` interface: `IsComplete()`, `IsCancelled()`, `GetSelected()`

**What's Missing (for new architecture):**

| Gap | Current | Needed |
|-----|---------|--------|
| Theming | Hardcoded lipgloss styles | `WithTheme(*Theme)` |
| Async data | Items at construction | `Eval[T]` + `OptionsFunc()` |
| Focus lifecycle | None | `Focus()`/`Blur()` methods |
| Standalone program | `Run()` creates new program (flicker) | Embed in parent form |
| Validation | None | `Error()` + validate callback |

**Key Insight:** The existing `StepModel` interface is a good foundation:
```go
type StepModel interface {
    IsComplete() bool
    IsCancelled() bool
    GetSelected() string
}
```

This can evolve into a fuller `Field` interface like huh's.

---

## Proposed Component Architecture

### Field Interface (evolved from StepModel)

```go
// pkg/tui/field.go
type Field interface {
    // Bubble Tea Model
    Init() tea.Cmd
    Update(tea.Msg) (tea.Model, tea.Cmd)
    View() string

    // Lifecycle
    Focus() tea.Cmd
    Blur() tea.Cmd

    // State
    IsComplete() bool
    IsCancelled() bool
    Error() error
    Skip() bool

    // Configuration
    WithTheme(*Theme) Field
    WithWidth(int) Field

    // Value access
    GetKey() string
    GetValue() any
}
```

### Wizard (Form equivalent)

```go
// pkg/tui/wizard.go
type Wizard struct {
    steps    []Step
    current  int
    ctx      *Context  // Services, theme, dimensions
    complete bool
    cancelled bool
}

type Step struct {
    Name   string
    Field  Field
    Skip   func(*WorkflowState) bool  // Conditional skip
}

// Navigation via messages
type nextStepMsg struct{}
type prevStepMsg struct{}
```

### Context (hybrid of gh-dash and huh)

```go
// pkg/tui/context.go
type Context struct {
    // Services
    GitService  git.ServiceInterface
    JiraService jira.ServiceInterface

    // Dimensions (updated on resize)
    Width  int
    Height int

    // Theme
    Theme *Theme

    // Workflow state (data collected so far)
    State *WorkflowState
}

type WorkflowState struct {
    WorkflowType string      // "feature", "hotfix"
    WorktreeName string      // "INGSVC-6468"
    BranchName   string      // "feature/INGSVC-6468-..."
    BaseBranch   string      // "master"
    JiraIssue    *jira.Issue // Full issue data
}
```

### Async Data Loading (Eval pattern)

```go
// pkg/tui/eval.go
type Eval[T any] struct {
    value    T
    fetch    func() (T, error)
    loading  bool
    err      error
    cacheKey string
}

func (e *Eval[T]) Get() T              // Returns cached or triggers fetch
func (e *Eval[T]) IsLoading() bool     // For spinner display
func (e *Eval[T]) Invalidate()         // Clear cache, re-fetch
```

### File Structure

```
pkg/tui/
├── field.go           # Field interface
├── wizard.go          # Wizard (multi-step form)
├── context.go         # Context + WorkflowState
├── theme.go           # Theme struct
├── eval.go            # Async data loading
│
├── fields/
│   ├── selector.go       # Static list selection (workflow type)
│   ├── filterable.go     # Filterable select (JIRA, branches)
│   ├── textinput.go      # Text input (branch name)
│   └── confirm.go        # Yes/No confirmation
│
└── components/
    ├── spinner.go        # Loading spinner
    └── help.go           # Help/keybinding hints
```

### Example: Feature Workflow Definition

```go
func NewFeatureWizard(ctx *Context) *Wizard {
    return &Wizard{
        ctx: ctx,
        steps: []Step{
            {
                Name: "workflow_type",
                Field: fields.NewSelector("What type?", []string{"Feature", "Hotfix"}),
            },
            {
                Name: "worktree_name",
                Field: fields.NewFilterable("Worktree name", "Select JIRA or enter custom").
                    WithOptionsFunc(func() []Option {
                        return ctx.JiraService.FetchIssues()  // Async
                    }),
            },
            {
                Name: "branch_name",
                Field: fields.NewTextInput("Branch name").
                    WithDefaultFunc(func() string {
                        return generateBranchName(ctx.State)
                    }),
            },
            {
                Name: "base_branch",
                Field: fields.NewFilterable("Base branch", "").
                    WithOptionsFunc(func() []Option {
                        return ctx.GitService.ListBranches()
                    }),
                Skip: func(s *WorkflowState) bool {
                    return ctx.GitService.BranchExists(s.BranchName)
                },
            },
            {
                Name: "confirm",
                Field: fields.NewConfirm("Create branch?"),
            },
        },
    }
}
```
