# Implementation Plan: `gbm wt testls`

**Created:** 2026-01-08
**Status:** Draft - Awaiting Approval
**Estimated Effort:** ~21 hours

---

## Overview

Build a reusable worktree list TUI in `pkg/tui/` that mirrors the functionality of the existing `worktree_table.go` but uses the component architecture established for `testadd`.

### Goals

1. Create reusable components (table, prompt, footer) in `pkg/tui/components/`
2. Implement a `Section` interface for list-based interactive views
3. Build `WorktreeSection` using these components
4. Create `gbm wt testls` command with mock services for testing

### Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Table foundation | `bubbles/table` | Battle-tested, handles scrolling/selection |
| Async pattern | Per-row loading | Matches current behavior, better UX than global spinner |
| Status symbols | Keep existing | `✓ ? ↑ ↓ ↕` are familiar to users |
| Help location | Bottom footer | Consistent with current TUI |
| Component reuse | Separate from Fields | Fields are wizard-oriented; Sections are list-action oriented |

---

## Architecture

### File Structure

```
pkg/tui/
├── wizard.go                       # Existing (unchanged)
├── field.go                        # Existing (unchanged)
├── section.go                      # 🆕 Section interface
├── context.go                      # Extend GitService interface
├── constants.go                    # Existing (unchanged)
├── theme.go                        # Existing (unchanged)
├── async/
│   └── eval.go                     # Existing (unchanged)
├── fields/                         # Existing (unchanged)
│   ├── selector.go
│   ├── filterable.go
│   ├── textinput.go
│   └── confirm.go
├── components/                     # 🆕 Shared UI primitives
│   ├── table/
│   │   └── table.go                # Wrapper around bubbles/table
│   ├── prompt/
│   │   └── prompt.go               # Inline y/n confirmation
│   └── footer/
│       └── footer.go               # Help text + messages
└── sections/                       # 🆕 Section implementations
    └── worktrees/
        ├── section.go              # Main WorktreeSection model
        ├── row.go                  # Row data + rendering
        └── status.go               # Per-row async status loading

cmd/service/
├── worktree_testadd.go             # Existing
└── worktree_testls.go              # 🆕 Test command

internal/testing/
└── mock_git.go                     # Extend with worktree/status mocks
```

### Relationship to Existing Code

```
┌─────────────────────────────────────────────────────────────┐
│                        pkg/tui/                              │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────┐         ┌─────────────┐                    │
│  │   Wizard    │         │   Section   │  ← 🆕 New          │
│  │  (testadd)  │         │  (testls)   │                    │
│  └──────┬──────┘         └──────┬──────┘                    │
│         │                       │                            │
│         ▼                       ▼                            │
│  ┌─────────────┐         ┌─────────────┐                    │
│  │   Fields    │         │ Components  │  ← 🆕 New          │
│  │ (Selector,  │         │  (Table,    │                    │
│  │ Filterable) │         │   Prompt,   │                    │
│  └─────────────┘         │   Footer)   │                    │
│                          └─────────────┘                    │
│         │                       │                            │
│         └───────────┬───────────┘                           │
│                     ▼                                        │
│              ┌─────────────┐                                 │
│              │   Context   │                                 │
│              │   Theme     │                                 │
│              │ async.Eval  │                                 │
│              └─────────────┘                                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## Component Specifications

### 1. Section Interface

**File:** `pkg/tui/section.go`

```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// Section defines the interface for list-based interactive views.
// Unlike Field (wizard-oriented), Section handles browsing + inline actions.
type Section interface {
    // Bubble Tea Model
    Init() tea.Cmd
    Update(tea.Msg) (Section, tea.Cmd)
    View() string

    // Lifecycle
    Focus() tea.Cmd
    Blur() tea.Cmd

    // Configuration
    WithTheme(*Theme) Section
    WithWidth(int) Section
    WithHeight(int) Section
}

// SectionCompleteMsg signals the section completed with a result.
// For worktree list, Result contains the selected worktree path.
type SectionCompleteMsg struct {
    Result any
}

// SectionCancelMsg signals the user cancelled/quit without selection.
type SectionCancelMsg struct{}
```

### 2. GitService Interface Extension

**File:** `pkg/tui/context.go` (extend existing)

```go
// GitService defines the interface for git operations needed by the TUI.
type GitService interface {
    // Existing methods (for wizard)
    BranchExists(branch string) (bool, error)
    ListBranches() ([]string, error)

    // New methods (for worktree list)
    ListWorktrees() ([]Worktree, error)
    GetBranchStatus(worktreePath string) (*BranchStatus, error)
    PullWorktree(worktreePath string) error
    PushWorktree(worktreePath string) error
    RemoveWorktree(name string) error
    GetCurrentWorktree() (*Worktree, error)
}

// Worktree represents a git worktree for the TUI.
type Worktree struct {
    Name   string
    Branch string
    Path   string
    IsBare bool
}

// BranchStatus represents the sync status of a branch with its remote.
type BranchStatus struct {
    Ahead    int
    Behind   int
    UpToDate bool
    NoRemote bool
}
```

### 3. Table Component (Responsive)

**File:** `pkg/tui/components/table/table.go`

Wrapper around `bubbles/table` with theme integration, row metadata support, and **responsive column widths**.

```go
package table

import (
    "gbm/pkg/tui"

    "github.com/charmbracelet/bubbles/table"
    tea "github.com/charmbracelet/bubbletea"
)

// Column defines a table column configuration with responsive behavior.
type Column struct {
    Title    string
    Width    int   // Base width (used when not growing)
    MinWidth int   // Minimum width (for grow columns)
    MaxWidth int   // Maximum width (0 = unlimited)
    Grow     bool  // If true, column expands to fill available space
    Priority int   // Hide priority: lower = hidden first when space is tight (0 = never hide)
}

// Row represents a table row with optional metadata.
type Row struct {
    Cells []string
    Data  any   // Underlying data (e.g., *WorktreeRow)
}

// Model wraps bubbles/table with theme support and responsive columns.
type Model struct {
    table       table.Model
    columns     []Column      // Original column config
    rows        []Row
    focused     bool
    theme       *tui.Theme
    width       int           // Available width
    height      int           // Available height
}

// New creates a new table with the given columns.
func New(columns []Column, theme *tui.Theme) Model

// SetRows replaces all rows in the table.
func (m *Model) SetRows(rows []Row)

// SetWidth updates the table width and recalculates column widths.
func (m *Model) SetWidth(width int)

// SetHeight updates the table height.
func (m *Model) SetHeight(height int)

// SelectedRow returns the currently selected row, or nil.
func (m Model) SelectedRow() *Row

// Cursor returns the current cursor position.
func (m Model) Cursor() int

// Standard Bubble Tea methods
func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd)
func (m Model) View() string

// Focus/Blur for lifecycle management
func (m *Model) Focus()
func (m *Model) Blur()

// computeColumnWidths calculates actual column widths based on available space.
// Algorithm:
// 1. Start with all columns at their base Width
// 2. If total > available: hide columns by Priority (lowest first) until it fits
// 3. If total < available: distribute extra space to Grow columns proportionally
// 4. Respect MinWidth and MaxWidth constraints
func (m *Model) computeColumnWidths() []int
```

**Responsive Column Configuration Example:**

```go
columns := []table.Column{
    {Title: "Name", Width: 25, MinWidth: 15, Grow: true, Priority: 0},   // Never hide, grows
    {Title: "Branch", Width: 40, MinWidth: 20, Grow: true, Priority: 0}, // Never hide, grows
    {Title: "Kind", Width: 10, Priority: 2},                              // Hide second
    {Title: "Git Status", Width: 15, Priority: 1},                        // Hide first
}
```

**Behavior at different terminal widths:**

| Terminal Width | Name | Branch | Kind | Git Status |
|----------------|------|--------|------|------------|
| 120+ chars | 30 | 50 | 10 | 15 | All visible, grow columns expanded |
| 100 chars | 25 | 40 | 10 | 15 | Base widths |
| 80 chars | 25 | 40 | 10 | — | Git Status hidden (Priority 1) |
| 60 chars | 25 | 40 | — | — | Kind also hidden (Priority 2) |
| 40 chars | 20 | 20 | — | — | MinWidth enforced for grow columns |

### 4. Prompt Component

**File:** `pkg/tui/components/prompt/prompt.go`

Inline confirmation dialog for destructive actions.

```go
package prompt

import (
    "gbm/pkg/tui"

    tea "github.com/charmbracelet/bubbletea"
)

// ConfirmMsg is sent when the user confirms the prompt.
type ConfirmMsg struct{}

// CancelMsg is sent when the user cancels the prompt.
type CancelMsg struct{}

// Model represents an inline confirmation prompt.
type Model struct {
    visible  bool
    question string  // e.g., "Delete worktree 'feature-x'?"
    selected bool    // true = Yes highlighted, false = No highlighted
    theme    *tui.Theme
}

// New creates a new prompt (hidden by default).
func New(theme *tui.Theme) Model

// Show displays the prompt with the given question.
func (m *Model) Show(question string)

// Hide hides the prompt.
func (m *Model) Hide()

// IsVisible returns whether the prompt is currently shown.
func (m Model) IsVisible() bool

// Update handles key input.
// Keys: y/Y → ConfirmMsg, n/N/esc → CancelMsg,
//       left/right/tab → toggle, enter → confirm selection
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd)

// View renders the prompt.
// Example: "Delete worktree 'feature-x'? [Yes] No"
func (m Model) View() string
```

### 5. Footer Component

**File:** `pkg/tui/components/footer/footer.go`

Help text and transient messages.

```go
package footer

import (
    "gbm/pkg/tui"

    tea "github.com/charmbracelet/bubbletea"
)

// MessageStyle defines the visual style for messages.
type MessageStyle int

const (
    MessageInfo MessageStyle = iota
    MessageSuccess
    MessageError
)

// Model represents the footer with help and messages.
type Model struct {
    helpText string
    message  string
    msgStyle MessageStyle
    theme    *tui.Theme
    width    int
}

// New creates a new footer with the given help text.
func New(helpText string, theme *tui.Theme) Model

// SetMessage displays a transient message.
func (m *Model) SetMessage(msg string, style MessageStyle)

// ClearMessage removes the current message.
func (m *Model) ClearMessage()

// SetWidth updates the footer width.
func (m *Model) SetWidth(width int)

// View renders the footer.
// Layout: [message if present] + [help text]
func (m Model) View() string
```

### 6. WorktreeSection

#### Row Data (`pkg/tui/sections/worktrees/row.go`)

```go
package worktrees

import (
    "fmt"

    "gbm/pkg/tui"
    "gbm/pkg/tui/components/table"
)

// WorktreeRow holds data for a single worktree in the list.
type WorktreeRow struct {
    Name      string
    Branch    string
    Path      string
    Kind      string  // "tracked" or "ad hoc"
    IsCurrent bool

    // Status (per-row async)
    Status  *tui.BranchStatus
    Loading bool
    Error   error
}

// ToTableRow converts WorktreeRow to a table.Row for display.
func (w WorktreeRow) ToTableRow() table.Row {
    name := w.Name
    if w.IsCurrent {
        name = "* " + name
    }

    gitStatus := formatStatus(w.Status, w.Loading, w.Error)

    return table.Row{
        Cells: []string{name, w.Branch, w.Kind, gitStatus},
        Data:  &w,
    }
}

// formatStatus returns the status symbol for display.
func formatStatus(status *tui.BranchStatus, loading bool, err error) string {
    if loading {
        return "⋯"
    }
    if err != nil {
        return "✗"
    }
    if status == nil {
        return "—"
    }
    if status.NoRemote {
        return "?"
    }
    if status.UpToDate {
        return "✓"
    }
    if status.Ahead > 0 && status.Behind > 0 {
        return fmt.Sprintf("↕ %d↑%d↓", status.Ahead, status.Behind)
    }
    if status.Ahead > 0 {
        return fmt.Sprintf("↑ %d", status.Ahead)
    }
    if status.Behind > 0 {
        return fmt.Sprintf("↓ %d", status.Behind)
    }
    return "—"
}
```

#### Status Loader (`pkg/tui/sections/worktrees/status.go`)

```go
package worktrees

import (
    "gbm/pkg/tui"
    "gbm/pkg/tui/async"

    tea "github.com/charmbracelet/bubbletea"
)

// StatusLoadedMsg is sent when a worktree's status has been fetched.
type StatusLoadedMsg struct {
    Name   string
    Status *tui.BranchStatus
    Error  error
}

// StatusLoader manages per-row async status loading.
type StatusLoader struct {
    evaluators map[string]*async.Eval[*tui.BranchStatus]
    git        tui.GitService
}

// NewStatusLoader creates a new status loader.
func NewStatusLoader(git tui.GitService) *StatusLoader

// LoadStatus starts async loading for a single worktree.
// Returns a command that will send StatusLoadedMsg when complete.
func (l *StatusLoader) LoadStatus(name, path string) tea.Cmd

// LoadAll starts async loading for all worktrees concurrently.
func (l *StatusLoader) LoadAll(worktrees []WorktreeRow) tea.Cmd

// Refresh invalidates and reloads status for a single worktree.
func (l *StatusLoader) Refresh(name, path string) tea.Cmd

// IsLoading returns whether a worktree's status is currently loading.
func (l *StatusLoader) IsLoading(name string) bool
```

#### Section Model (`pkg/tui/sections/worktrees/section.go`)

```go
package worktrees

import (
    "gbm/pkg/tui"
    "gbm/pkg/tui/components/footer"
    "gbm/pkg/tui/components/prompt"
    "gbm/pkg/tui/components/table"

    tea "github.com/charmbracelet/bubbletea"
)

// Model implements tui.Section for worktree browsing.
type Model struct {
    // Components
    table  table.Model
    prompt prompt.Model
    footer footer.Model

    // Data
    worktrees    []WorktreeRow
    trackedBranches map[string]bool

    // Async
    statusLoader *StatusLoader

    // State
    focused       bool
    selectedPath  string  // Set on selection, used for output

    // Context
    ctx    *tui.Context
    theme  *tui.Theme
    width  int
    height int
}

// New creates a new WorktreeSection.
func New(ctx *tui.Context) *Model

// SelectedPath returns the path of the selected worktree (after exit).
func (m Model) SelectedPath() string

// SelectedName returns the name of the selected worktree (after exit).
func (m Model) SelectedName() string

// --- tui.Section interface ---

func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tui.Section, tea.Cmd)
func (m Model) View() string
func (m *Model) Focus() tea.Cmd
func (m *Model) Blur() tea.Cmd
func (m Model) WithTheme(theme *tui.Theme) tui.Section
func (m Model) WithWidth(width int) tui.Section
func (m Model) WithHeight(height int) tui.Section
```

**Update Logic:**

```go
func (m Model) Update(msg tea.Msg) (tui.Section, tea.Cmd) {
    // Handle window resize - propagate to components
    if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
        m.width = sizeMsg.Width
        m.height = sizeMsg.Height

        // Propagate to table (triggers column recalculation)
        m.table.SetWidth(sizeMsg.Width)
        m.table.SetHeight(sizeMsg.Height - footerHeight)

        // Propagate to footer
        m.footer.SetWidth(sizeMsg.Width)

        return m, nil
    }

    // Handle prompt if visible
    if m.prompt.IsVisible() {
        return m.handlePromptUpdate(msg)
    }

    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKeyMsg(msg)

    case StatusLoadedMsg:
        return m.handleStatusLoaded(msg)

    case PullCompleteMsg:
        return m.handlePullComplete(msg)

    case PushCompleteMsg:
        return m.handlePushComplete(msg)

    case DeleteCompleteMsg:
        return m.handleDeleteComplete(msg)
    }

    // Delegate to table
    var cmd tea.Cmd
    m.table, cmd = m.table.Update(msg)
    return m, cmd
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tui.Section, tea.Cmd) {
    switch msg.String() {
    case "q", "esc", "ctrl+c":
        return m, func() tea.Msg { return tui.SectionCancelMsg{} }

    case "enter", " ":
        // Select worktree
        if row := m.table.SelectedRow(); row != nil {
            wt := row.Data.(*WorktreeRow)
            m.selectedPath = wt.Path
            return m, func() tea.Msg {
                return tui.SectionCompleteMsg{Result: wt.Path}
            }
        }

    case "l":
        // Pull
        return m.startPull()

    case "p":
        // Push (if not tracked)
        return m.startPush()

    case "d":
        // Delete (show prompt)
        return m.showDeletePrompt()
    }

    // Navigation handled by table
    var cmd tea.Cmd
    m.table, cmd = m.table.Update(msg)
    return m, cmd
}
```

---

## Test Command

**File:** `cmd/service/worktree_testls.go`

```go
package service

import (
    "fmt"
    "os"
    "time"

    "gbm/internal/testing"
    "gbm/pkg/tui"
    "gbm/pkg/tui/sections/worktrees"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/spf13/cobra"
)

func newWorktreeTestlsCommand(svc *Service) *cobra.Command {
    var delayMs int

    cmd := &cobra.Command{
        Use:   "testls",
        Short: "Test the worktree list TUI with mock data",
        Long: `Launch the interactive worktree list TUI with mock Git services.

This command is useful for testing and developing the list UI without
affecting real repositories.

Features tested:
- Table display with columns (Name, Branch, Kind, Git Status)
- Navigation (up/down arrows, j/k)
- Selection (enter/space to switch)
- Pull action (l key)
- Push action (p key, disabled for tracked branches)
- Delete with confirmation (d key)
- Per-row async status loading

No actual worktrees are modified (dry-run mode).`,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runWorktreeTestlsCommand(delayMs)
        },
    }

    cmd.Flags().IntVar(&delayMs, "delay", 0,
        "simulate network latency in milliseconds (0-5000)")

    return cmd
}

func runWorktreeTestlsCommand(delayMs int) error {
    // Validate delay
    if delayMs < 0 || delayMs > 5000 {
        return fmt.Errorf("delay must be between 0 and 5000 milliseconds")
    }

    // Create mock git service
    mockGit := testing.NewMockGitService().
        WithWorktrees([]tui.Worktree{
            {Name: "main", Branch: "main", Path: "/repo/worktrees/main"},
            {Name: "feature-auth", Branch: "feature/auth", Path: "/repo/worktrees/feature-auth"},
            {Name: "feature-dashboard", Branch: "feature/dashboard", Path: "/repo/worktrees/feature-dashboard"},
            {Name: "bugfix-login", Branch: "bugfix/login", Path: "/repo/worktrees/bugfix-login"},
            {Name: "hotfix-security", Branch: "hotfix/security", Path: "/repo/worktrees/hotfix-security"},
            {Name: "release-v2", Branch: "release/v2.0", Path: "/repo/worktrees/release-v2"},
        }).
        WithBranchStatuses(map[string]*tui.BranchStatus{
            "main":              {UpToDate: true},
            "feature-auth":     {Ahead: 2},
            "feature-dashboard": {Behind: 3},
            "bugfix-login":     {Ahead: 1, Behind: 2},
            "hotfix-security":  {NoRemote: true},
            "release-v2":       {UpToDate: true},
        }).
        WithTrackedBranches([]string{"main", "release/v2.0"}).
        WithCurrentWorktree("feature-auth")

    if delayMs > 0 {
        mockGit = mockGit.WithDelay(time.Duration(delayMs) * time.Millisecond)
    }

    // Create context
    ctx := tui.NewContext().
        WithDimensions(100, 30).
        WithTheme(tui.DefaultTheme()).
        WithGitService(mockGit)

    // Create section
    section := worktrees.New(ctx)

    // Open TTY for TUI
    input, err := os.Open("/dev/tty")
    if err != nil {
        return fmt.Errorf("failed to open /dev/tty: %w", err)
    }
    defer input.Close()

    // Run TUI
    p := tea.NewProgram(section,
        tea.WithInput(input),
        tea.WithAltScreen())

    finalModel, err := p.Run()
    if err != nil {
        return fmt.Errorf("testls error: %w", err)
    }

    // Handle result
    if s, ok := finalModel.(*worktrees.Model); ok {
        if path := s.SelectedPath(); path != "" {
            fmt.Println(path)  // stdout - machine readable
            fmt.Fprintf(os.Stderr, "✓ Selected worktree: %s\n", s.SelectedName())
        }
    }

    return nil
}
```

---

## Mock GitService Extensions

**File:** `internal/testing/mock_git.go` (extend existing)

```go
// Add to MockGitService struct
type MockGitService struct {
    // Existing fields...

    // New fields for worktree list
    worktrees       []tui.Worktree
    branchStatuses  map[string]*tui.BranchStatus
    trackedBranches map[string]bool
    currentWorktree string
}

// WithWorktrees sets the mock worktrees.
func (m *MockGitService) WithWorktrees(wts []tui.Worktree) *MockGitService {
    m.worktrees = wts
    return m
}

// WithBranchStatuses sets the mock branch statuses (keyed by branch name).
func (m *MockGitService) WithBranchStatuses(statuses map[string]*tui.BranchStatus) *MockGitService {
    m.branchStatuses = statuses
    return m
}

// WithTrackedBranches sets which branches are "tracked" (push disabled).
func (m *MockGitService) WithTrackedBranches(branches []string) *MockGitService {
    m.trackedBranches = make(map[string]bool)
    for _, b := range branches {
        m.trackedBranches[b] = true
    }
    return m
}

// WithCurrentWorktree sets which worktree is "current".
func (m *MockGitService) WithCurrentWorktree(name string) *MockGitService {
    m.currentWorktree = name
    return m
}

// ListWorktrees returns mock worktrees.
func (m *MockGitService) ListWorktrees() ([]tui.Worktree, error) {
    m.delay()
    return m.worktrees, nil
}

// GetBranchStatus returns mock status for a worktree.
func (m *MockGitService) GetBranchStatus(worktreePath string) (*tui.BranchStatus, error) {
    m.delay()
    // Find worktree by path, look up status by branch
    for _, wt := range m.worktrees {
        if wt.Path == worktreePath {
            if status, ok := m.branchStatuses[wt.Branch]; ok {
                return status, nil
            }
        }
    }
    return &tui.BranchStatus{UpToDate: true}, nil
}

// PullWorktree simulates a pull operation.
func (m *MockGitService) PullWorktree(worktreePath string) error {
    m.delay()
    return nil
}

// PushWorktree simulates a push operation.
func (m *MockGitService) PushWorktree(worktreePath string) error {
    m.delay()
    return nil
}

// RemoveWorktree simulates worktree removal.
func (m *MockGitService) RemoveWorktree(name string) error {
    m.delay()
    // Remove from mock list
    for i, wt := range m.worktrees {
        if wt.Name == name {
            m.worktrees = append(m.worktrees[:i], m.worktrees[i+1:]...)
            break
        }
    }
    return nil
}

// GetCurrentWorktree returns the mock current worktree.
func (m *MockGitService) GetCurrentWorktree() (*tui.Worktree, error) {
    for _, wt := range m.worktrees {
        if wt.Name == m.currentWorktree {
            return &wt, nil
        }
    }
    return nil, nil
}

// IsTrackedBranch returns whether a branch is tracked.
func (m *MockGitService) IsTrackedBranch(branch string) bool {
    return m.trackedBranches[branch]
}
```

---

## Status Symbols Reference

| State | Symbol | Description |
|-------|--------|-------------|
| Loading | `⋯` | Status fetch in progress |
| Up to date | `✓` | Branch matches remote |
| No remote | `?` | No upstream tracking branch |
| Ahead only | `↑ N` | N commits ahead of remote |
| Behind only | `↓ N` | N commits behind remote |
| Diverged | `↕ N↑M↓` | N ahead, M behind |
| Error | `✗` | Failed to fetch status |
| N/A | `—` | Bare repo or status unavailable |

---

## Keybindings

| Key | Action | Notes |
|-----|--------|-------|
| `↑` / `k` | Move up | Navigate list |
| `↓` / `j` | Move down | Navigate list |
| `enter` / `space` | Select | Output path and exit |
| `l` | Pull | Pulls selected worktree |
| `p` | Push | Pushes selected worktree (disabled for tracked) |
| `d` | Delete | Shows confirmation prompt |
| `q` / `esc` | Quit | Exit without selection |
| `ctrl+c` | Force quit | Exit immediately |

**In confirmation prompt:**

| Key | Action |
|-----|--------|
| `y` / `Y` | Confirm (immediate) |
| `n` / `N` / `esc` | Cancel (immediate) |
| `←` / `→` / `tab` | Toggle Yes/No |
| `enter` | Confirm current selection |

---

## Implementation Order

| # | Task | Files | Est. | Dependencies |
|---|------|-------|------|--------------|
| 1 | Section interface | `pkg/tui/section.go` | 1h | None |
| 2 | Extend GitService interface | `pkg/tui/context.go` | 1h | None |
| 3 | Table component (responsive) | `pkg/tui/components/table/table.go` | 4h | Theme |
| 4 | Prompt component | `pkg/tui/components/prompt/prompt.go` | 2h | Theme |
| 5 | Footer component | `pkg/tui/components/footer/footer.go` | 2h | Theme |
| 6 | Row data + rendering | `pkg/tui/sections/worktrees/row.go` | 2h | Table |
| 7 | Status loader | `pkg/tui/sections/worktrees/status.go` | 3h | async.Eval |
| 8 | WorktreeSection model | `pkg/tui/sections/worktrees/section.go` | 4h | All above |
| 9 | Mock GitService extensions | `internal/testing/mock_git.go` | 2h | GitService |
| 10 | testls command | `cmd/service/worktree_testls.go` | 2h | Section |

**Total: ~23 hours**

Note: Table component increased from 2h to 4h to account for responsive column width calculation algorithm.

---

## Testing & Debugging Tools

### Interactive Tmux Session Skill

We have `skills/interactive-tmux-session.md` available for visual testing of the TUI. This allows:

1. Creating a tmux session in a temp directory
2. User attaches to watch in real-time
3. Claude executes commands via `tmux send-keys`
4. Output captured via `tmux capture-pane`

**Usage for TUI testing:**

```bash
# Setup
TMPDIR=$(mktemp -d)
cd "$TMPDIR"
# Copy built binary or set up test environment
tmux new-session -d -s "testls_$(date +%s)" -c "$TMPDIR"
SESSION=$(tmux list-sessions | grep "testls_" | cut -d: -f1 | tail -1)
echo "Attach: tmux attach-session -t $SESSION"

# After user attaches, run TUI
tmux send-keys -t "$SESSION" "gbm wt testls --delay 500" Enter
sleep 3
tmux capture-pane -t "$SESSION" -p  # See what's rendered
```

### Skills to Create During Implementation

As we implement, capture reusable patterns as skills:

| Potential Skill | Purpose |
|-----------------|---------|
| `tui-visual-test.md` | Standard workflow for visually testing Bubble Tea TUIs |
| `tui-screenshot.md` | Capture TUI state to file for comparison/debugging |
| `mock-service-debug.md` | Debugging mock service behavior with logging |
| `async-timing-debug.md` | Debugging race conditions in async TUI updates |

**Skill creation triggers:**
- When we solve a debugging problem in a reusable way
- When we find ourselves repeating the same testing pattern
- When we discover useful tmux/terminal tricks for TUI development

---

## Success Criteria

1. `gbm wt testls` launches and displays mock worktrees in a table
2. Navigation works (↑/↓/j/k)
3. Selection outputs path to stdout and exits
4. Status loading shows `⋯` then resolves to actual status
5. Pull action (`l`) shows loading state, then refreshes status
6. Push action (`p`) works for non-tracked, shows error for tracked
7. Delete action (`d`) shows confirmation, removes on confirm
8. `--delay` flag simulates network latency
9. **Responsive design**: Table columns adjust on terminal resize
   - Grow columns expand/contract with available space
   - Low-priority columns hide when terminal is narrow
   - MinWidth constraints are respected
10. All existing `testadd` functionality continues to work

---

## Future Considerations

Items explicitly deferred for later:

1. **Search/filter** - Type to filter worktrees by name
2. **Configurable keybindings** - YAML-based keybinding configuration
3. **Multiple sections** - Tabs for worktrees/branches/tags
4. **Detail sidebar** - Rich info about selected worktree
5. **Mouse support** - Click to select

These can be added incrementally once the foundation is solid.
