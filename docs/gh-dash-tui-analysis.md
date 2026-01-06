# gh-dash TUI Architecture Analysis

**Analysis Date:** 2026-01-01
**Last Updated:** 2026-01-03
**Project:** [gh-dash](https://github.com/dlvhdr/gh-dash)
**Purpose:** Identify patterns and best practices for enhancing gbm's TUI functionality

---

## Current State of gbm (as of 2026-01-03)

**✅ What's Already Been Done**

Since the initial analysis, gbm has implemented several foundational improvements:

- **✅ Git Service Organization** - Clean separation with 5 focused files (service.go, worktree.go, branch.go, init.go, clone.go)
- **✅ Typed Errors** - GitError with classification and rich context (similar to gh-dash error handling)
- **✅ Configuration System** - Template expansion with {gitroot} variables (P2.1 complete)
- **✅ Comprehensive Documentation** - CLAUDE.md with examples, inline godoc comments (P4.1 complete)
- **✅ Testing Infrastructure** - E2E tests, testify patterns, testutil package (P1.2 complete)
- **✅ Flag Override Pattern** - Clear precedence for flags > config > defaults (P1.3 complete)
- **✅ Universal Output Pattern** - stdout/stderr separation, /dev/tty for TUI (P1.1 complete)
- **✅ Automatic File Copying** - gitignore pattern matching, source resolution (P2.2 complete)

**❌ What's Still Missing (TUI-Specific)**

The TUI implementation (`cmd/service/worktree_*.go`) has NOT been refactored and would still benefit from:

- **Component-based architecture** - Currently monolithic TUI files
- **Context propagation** - No ProgramContext pattern for shared state
- **Task-based async operations** - Git operations block UI, no spinner feedback
- **Section abstraction** - Single view, hard to extend with new section types
- **Configurable keybindings** - Keybindings are hardcoded
- **Multiple sections/views** - Cannot have multiple configured worktree views

**Focus Areas:** The remaining recommendations focus on TUI architecture improvements that gh-dash demonstrates exceptionally well.

---

## Executive Summary

**gh-dash** is a sophisticated terminal dashboard for GitHub PRs and Issues built with Bubble Tea. The codebase demonstrates exceptional TUI architecture with:

- **87 Go files** organized into clear layers (cmd, data, TUI, config)
- **19 distinct component types** with interface-based composition
- **Task-based async operations** with excellent UX feedback
- **Highly configurable** UI (keybindings, layouts, themes) via YAML
- **Production-ready** patterns for complex state management and data fetching

**Key Insight:** While gbm has excellent git service organization, configuration, and testing, the **TUI architecture** can still benefit significantly from gh-dash's component-based patterns, context propagation, and task management.

---

## Table of Contents

0. [Current State of gbm](#current-state-of-gbm-as-of-2026-01-03)
1. [Project Structure](#1-project-structure)
2. [TUI Organization](#2-tui-organization)
3. [Key Architectural Patterns](#3-key-architectural-patterns)
4. [Libraries & Technologies](#4-libraries--technologies)
5. [Component Hierarchy](#5-component-hierarchy)
6. [Data Flow & State Management](#6-data-flow--state-management)
7. [Best Practices Observed](#7-best-practices-observed)
8. [Recommendations for gbm](#8-recommendations-for-gbm)
9. [Specific gh-dash Features Worth Adopting](#9-specific-gh-dash-features-worth-adopting)
10. [Priority Recommendations](#10-priority-recommendations)

---

## 1. Project Structure

```
deps/gh-dash/
├── cmd/                           # CLI entry points
│   ├── root.go                    # Cobra root command, TUI initialization
│   └── sponsors.go                # Secondary commands
│
├── internal/
│   ├── config/                    # Configuration management
│   │   ├── config.go              # Main config struct and loading
│   │   ├── parser.go              # YAML parsing with koanf
│   │   └── defaults.go            # Default configuration
│   │
│   ├── data/                      # GitHub API data layer
│   │   ├── data.go                # GraphQL client setup
│   │   ├── pull_requests.go       # PR queries and mutations
│   │   ├── issues.go              # Issue queries
│   │   └── enrichers/             # Incremental data loading
│   │
│   ├── git/                       # Git operations
│   │   └── git.go                 # Local git repo detection
│   │
│   ├── tui/                       # Terminal UI (Bubble Tea)
│   │   ├── ui.go                  # Root model (1095 lines)
│   │   │
│   │   ├── components/            # 45 Go files - UI components
│   │   │   ├── section/           # Base section abstraction
│   │   │   ├── prssection/        # PR list section
│   │   │   ├── issuessection/     # Issue list section
│   │   │   ├── reposection/       # Repository branches section
│   │   │   ├── prview/            # PR detail view
│   │   │   ├── issueview/         # Issue detail view
│   │   │   ├── table/             # Generic table component
│   │   │   ├── listviewport/      # Scrollable list
│   │   │   ├── sidebar/           # Preview sidebar
│   │   │   ├── footer/            # Status and help footer
│   │   │   ├── tabs/              # Tab navigation
│   │   │   ├── search/            # Search input
│   │   │   ├── prompt/            # Confirmation dialogs
│   │   │   ├── inputbox/          # Multi-line text input
│   │   │   └── tasks/             # Async GitHub operations
│   │   │
│   │   ├── context/               # Shared state and styles
│   │   │   ├── context.go         # ProgramContext struct
│   │   │   └── styles.go          # Computed lipgloss styles
│   │   │
│   │   ├── keys/                  # Keybinding management
│   │   │   └── keys.go            # KeyMap with rebinding
│   │   │
│   │   ├── markdown/              # Markdown rendering
│   │   │   └── markdown.go        # Glamour integration
│   │   │
│   │   └── theme/                 # Theme system
│   │       └── theme.go           # Color schemes
│   │
│   └── utils/                     # Shared utilities
│       ├── utils.go               # String helpers, formatting
│       └── notifications.go       # Desktop notifications
│
├── docs/                          # Documentation website (Astro)
└── test/                          # Integration tests
```

### Key Observations

**✅ Clear Separation of Concerns:**
- **cmd/** - CLI entry and initialization
- **data/** - All GitHub API interactions isolated
- **tui/** - Pure UI logic, no API calls
- **config/** - Configuration management separate from business logic

**✅ Component-Based Architecture:**
- 45 component files, each focused on a single UI element
- Components compose via embedding and interfaces
- Reusable primitives (table, viewport, search)

**✅ Layered Design:**
- Data layer doesn't know about TUI
- TUI components receive data via messages
- Configuration drives behavior without code changes

---

## 2. TUI Organization

### 2.1 Main Model Structure

**File:** `internal/tui/ui.go` (1095 lines)

```go
type Model struct {
    // Keybindings
    keys *keys.KeyMap

    // Layout components
    sidebar       sidebar.Model
    footer        footer.Model
    tabs          tabs.Model

    // Detail views
    prView        prview.Model        // PR detail sidebar
    issueSidebar  issueview.Model     // Issue detail sidebar
    branchSidebar branchsidebar.Model // Branch detail sidebar

    // Sections (main content areas)
    repo          section.Section     // Repository branches
    prs           []section.Section   // PR sections (can have multiple)
    issues        []section.Section   // Issue sections (can have multiple)
    currSectionId int                 // Active section index

    // Async operations
    taskSpinner   spinner.Model
    tasks         map[string]context.Task

    // Global context
    ctx *context.ProgramContext
}
```

**Key Design Decisions:**

1. **Multiple section support** - `prs` and `issues` are slices, allowing multiple configured views
2. **Polymorphic sections** - All sections implement `section.Section` interface
3. **Separate detail views** - Detail sidebars are distinct from list sections
4. **Task tracking** - Map-based task state with spinner integration
5. **Context propagation** - Single source of truth for dimensions, theme, config

### 2.2 Component Categories

Components are organized into **7 functional categories:**

| Category | Components | Purpose |
|----------|-----------|---------|
| **Layout** | `tabs`, `sidebar`, `footer`, `carousel` | Application chrome and navigation |
| **Sections** | `section` (base), `prssection`, `issuessection`, `reposection` | Main content areas with lists |
| **Detail Views** | `prview`, `issueview`, `branchsidebar` | Focused views for selected items |
| **Rows** | `prrow`, `issuerow`, `branch` | Individual list item rendering |
| **Primitives** | `table`, `listviewport`, `search`, `prompt`, `inputbox` | Reusable building blocks |
| **Tasks** | `tasks` (pr, issue, comment, notifications) | Async GitHub operations |
| **Shared** | `context`, `keys`, `markdown`, `theme` | Cross-cutting concerns |

### 2.3 File Size Distribution

**Component Complexity** (lines of code):

- **Large components** (500+ lines):
  - `ui.go` (1095) - Main model orchestration
  - `prview/prview.go` (755) - PR detail view with tabs
  - `section/section.go` (674) - Base section implementation
  - `table/table.go` (598) - Generic table component

- **Medium components** (200-500 lines):
  - `prssection/prssection.go` (441)
  - `issueview/issueview.go` (431)
  - `listviewport/listviewport.go` (388)
  - `issuessection/issuessection.go` (284)

- **Small components** (< 200 lines):
  - Most row components (50-150 lines)
  - Primitives like `search`, `prompt`, `inputbox`
  - Utilities and helpers

**Insight:** Complexity is concentrated in orchestration (`ui.go`) and base abstractions (`section`, `table`). Individual components are kept small and focused.

---

## 3. Key Architectural Patterns

### 3.1 Interface-Based Section Abstraction

**Problem Solved:** Different content types (PRs, Issues, Branches) need common table/navigation/search behavior.

**Solution:** `section.Section` interface with shared base implementation.

```go
// internal/tui/components/section/section.go
type Section interface {
    Identifier             // GetId(), GetType()
    Component              // Update(), View()
    Table                  // NumRows(), CurrRow(), NextRow(), etc.
    Search                 // SetIsSearching(), GetFilters(), etc.
    PromptConfirmation     // ShowPrompt(), ClosePrompt()

    GetConfig() config.SectionConfig
    UpdateProgramContext(ctx *context.ProgramContext)
    MakeSectionCmd(cmd tea.Cmd) tea.Cmd
}

type BaseModel struct {
    Id                    int
    Type                  string
    Config                config.SectionConfig
    Ctx                   *context.ProgramContext
    SearchBar             search.Model
    Table                 table.Model
    PromptConfirmationBox prompt.Model
    // ... pagination, loading state, etc.
}
```

**Usage:**

```go
type PrsSectionModel struct {
    section.BaseModel  // Embedded base
    Prs []prrow.Data   // Specific data
}

// Specific implementation
func (m *PrsSectionModel) Update(msg tea.Msg) tea.Cmd {
    // Handle PR-specific messages
    switch msg := msg.(type) {
    case UpdatePRMsg:
        // Update specific PR in list
    }

    // Delegate to base for common behavior
    return m.BaseModel.Update(msg)
}
```

**Benefits:**

✅ **Polymorphism** - Main model treats all sections uniformly
✅ **Code reuse** - Table, search, pagination implemented once
✅ **Consistency** - All sections have same keyboard navigation
✅ **Extensibility** - New section types just implement interface

**Application to gbm:**

This pattern would allow gbm to have:
- Worktree list section
- Branch list section
- Recent activity section
- All with common navigation, search, and table behavior

### 3.2 Context Propagation Pattern

**Problem Solved:** Components need shared state (dimensions, theme, config) without tight coupling.

**Solution:** `ProgramContext` struct passed through all components.

```go
// internal/tui/context/context.go
type ProgramContext struct {
    // Dimensions (updated on window resize)
    ScreenWidth       int
    ScreenHeight      int
    MainContentWidth  int
    MainContentHeight int

    // Configuration
    Config *config.Config
    View   config.ViewType  // PRs, Issues, or Repo

    // State
    RepoPath string
    User     string
    Error    error

    // Styling (computed from theme + dimensions)
    Theme  theme.Theme
    Styles Styles  // Cached lipgloss styles

    // Task management
    StartTask func(task Task) tea.Cmd
}

type Styles struct {
    Section      SectionStyles
    Table        TableStyles
    Sidebar      SidebarStyles
    Footer       FooterStyles
    ListViewPort ListViewPortStyles
}
```

**Propagation Flow:**

1. **Root model** creates context with initial state
2. **Window resize** updates dimensions in context
3. **Context recomputed** with new styles
4. **UpdateProgramContext()** called on all components
5. **Components re-render** with updated dimensions/styles

```go
// Example: Handling window resize
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.ctx.ScreenWidth = msg.Width
        m.ctx.ScreenHeight = msg.Height
        m.ctx.MainContentWidth = msg.Width - sidebarWidth
        m.ctx.MainContentHeight = msg.Height - footerHeight

        // Recompute styles
        m.ctx.Styles = computeStyles(m.ctx)

        // Propagate to all components
        m.footer.UpdateProgramContext(m.ctx)
        m.sidebar.UpdateProgramContext(m.ctx)
        for i := range m.prs {
            m.prs[i].UpdateProgramContext(m.ctx)
        }
    }
}
```

**Benefits:**

✅ **Decoupling** - Components don't hold references to each other
✅ **Consistency** - Single source of truth for dimensions/theme
✅ **Performance** - Styles computed once, reused everywhere
✅ **Testability** - Components can be tested with mock context

**Application to gbm:**

gbm could use this to:
- Share git service, JIRA service across components
- Propagate terminal dimensions for responsive layout
- Centralize theme/styling configuration
- Provide consistent error display mechanism

### 3.3 Task-Based Async Operations

**Problem Solved:** GitHub API calls and git operations are slow; need good UX feedback.

**Solution:** Task tracking system with spinner integration and message-based completion.

```go
// internal/tui/context/context.go
type Task struct {
    Id           string
    StartText    string
    FinishedText string
    State        State      // TaskStart, TaskFinished, TaskError
    Error        error
    StartTime    time.Time
    FinishedTime *time.Time
}

type TaskStartMsg struct{ Task Task }
type TaskFinishedMsg struct{ Task Task }
```

**Typical Flow:**

```go
// 1. Define task in tasks/pr.go
func ClosePR(ctx *context.ProgramContext, sectionId int, pr data.RowData) tea.Cmd {
    taskId := buildTaskId("pr_close", pr.Number)

    return fireTask(ctx, GitHubTask{
        Id:           taskId,
        Section:      sectionId,
        StartText:    fmt.Sprintf("Closing PR #%d", pr.Number),
        FinishedText: fmt.Sprintf("PR #%d has been closed", pr.Number),

        // Command to execute (uses gh CLI)
        Args: []string{"pr", "close", fmt.Sprint(pr.Number), "-R", repoName},

        // Message to send on completion
        Msg: func(c *exec.Cmd, err error) tea.Msg {
            if err != nil {
                return TaskErrorMsg{/* ... */}
            }
            return UpdatePRMsg{
                PrNumber: pr.Number,
                IsClosed: &closed,
            }
        },
    })
}

// 2. Start task from component
case key.Matches(msg, m.keys.Close):
    return m, tasks.ClosePR(m.ctx, m.Id, m.getCurrRow())

// 3. Root model handles task lifecycle
case TaskStartMsg:
    m.tasks[msg.Id] = msg.Task
    m.taskSpinner, cmd = m.taskSpinner.Update(msg)  // Show spinner

case TaskFinishedMsg:
    delete(m.tasks, msg.Id)  // Clean up
    // Spinner auto-hides, shows success message

case UpdatePRMsg:
    // Update data in section
    m.prs[sectionId].UpdateRow(msg.PrNumber, msg)
```

**Task Implementation** (simplified):

```go
func fireTask(ctx *ProgramContext, task GitHubTask) tea.Cmd {
    return tea.Batch(
        // 1. Show spinner immediately
        func() tea.Msg {
            return TaskStartMsg{Task: task.ToTask()}
        },

        // 2. Execute in background
        func() tea.Msg {
            cmd := exec.Command("gh", task.Args...)
            output, err := cmd.CombinedOutput()

            task.State = TaskFinished
            if err != nil {
                task.State = TaskError
                task.Error = err
            }

            // 3a. Return task completion
            finishedMsg := TaskFinishedMsg{Task: task.ToTask()}

            // 3b. Return data update message
            updateMsg := task.Msg(cmd, err)

            return tea.Batch(finishedMsg, updateMsg)
        },
    )
}
```

**Benefits:**

✅ **Immediate feedback** - Spinner shows instantly
✅ **Non-blocking** - UI remains responsive during operations
✅ **Error handling** - Errors displayed in footer with context
✅ **Composability** - Tasks can be batched, chained, cancelled
✅ **Separation** - Task logic separate from UI components

**Application to gbm:**

gbm could use this for:
- `git worktree add` (slow on large repos)
- `git fetch` / `git pull` operations
- JIRA API calls
- Git status updates
- Branch checkout operations

All with consistent spinner feedback and error handling.

### 3.4 Message-Based Section Routing

**Problem Solved:** Multiple sections need independent state updates within single Bubble Tea update loop.

**Solution:** Message wrapping with section ID routing.

```go
// Wrapper message type
type SectionMsg struct {
    Id          int        // Target section
    Type        string     // Section type (for debugging)
    InternalMsg tea.Msg    // Wrapped message
}

// Helper to wrap commands for specific section
func (m *BaseModel) MakeSectionCmd(cmd tea.Cmd) tea.Cmd {
    if cmd == nil {
        return nil
    }

    return func() tea.Msg {
        return SectionMsg{
            Id:          m.Id,
            Type:        m.Type,
            InternalMsg: cmd(),
        }
    }
}
```

**Usage Pattern:**

```go
// Component returns section-specific command
func (m *PrsSectionModel) Update(msg tea.Msg) tea.Cmd {
    switch msg := msg.(type) {
    case key.KeyMsg:
        if key.Matches(msg, m.keys.Refresh) {
            return m.MakeSectionCmd(m.fetchPRs())  // Wraps command
        }
    }
    return nil
}

// Root model routes to correct section
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case SectionMsg:
        // Route to specific section
        return m.updateSection(msg.Id, msg.InternalMsg)
    }

    // Broadcast to all sections
    return m.updateAllSections(msg)
}

func (m Model) updateSection(id int, msg tea.Msg) (tea.Model, tea.Cmd) {
    for i := range m.prs {
        if m.prs[i].GetId() == id {
            return m, m.prs[i].Update(msg)
        }
    }
    // ... check issues, repo sections
    return m, nil
}
```

**Benefits:**

✅ **Isolation** - Sections don't receive irrelevant messages
✅ **Scalability** - Can have many sections without cross-talk
✅ **Debugging** - Clear message flow, easy to trace
✅ **Flexibility** - Easy to add new section types

**Application to gbm:**

This would allow gbm to:
- Have multiple worktree views (active, archived, by branch prefix)
- Update specific worktree without refreshing all
- Support split views (worktrees + branches)
- Maintain independent scroll position per section

### 3.5 Composition Over Inheritance

**Problem Solved:** Components share common behavior but need specific implementations.

**Solution:** Embed `BaseModel` and override specific methods.

```go
// Base provides table, search, prompt functionality
type BaseModel struct {
    Id        int
    Type      string
    Config    config.SectionConfig
    SearchBar search.Model
    Table     table.Model
    Prompt    prompt.Model
    // ... pagination, filtering, etc.
}

// Base implements common methods
func (m *BaseModel) NumRows() int { return m.Table.NumRows() }
func (m *BaseModel) CurrRow() int { return m.Table.CurrRow() }
func (m *BaseModel) NextRow() int { return m.Table.NextRow() }
// ... etc.

// Specific section embeds and extends
type PrsSectionModel struct {
    section.BaseModel  // Composition, not inheritance
    Prs []prrow.Data
}

// Override specific behavior
func (m *PrsSectionModel) FetchNextPage() tea.Cmd {
    // PR-specific GraphQL query
    return m.MakeSectionCmd(fetchPRsCmd(m.Config, m.Prs))
}

func (m *PrsSectionModel) View() string {
    // PR-specific rendering
    rows := make([]table.Row, len(m.Prs))
    for i, pr := range m.Prs {
        rows[i] = prrow.Render(m.Ctx, pr)
    }

    m.Table.SetRows(rows)
    return m.BaseModel.View()  // Delegate to base for layout
}
```

**Benefits:**

✅ **Reuse** - Common functionality implemented once
✅ **Flexibility** - Easy to override specific behavior
✅ **Type safety** - No runtime reflection needed
✅ **Clarity** - Composition makes relationships explicit

**Application to gbm:**

gbm could have:
- `BaseWorklistModel` with table/search/pagination
- `ActiveWorktreesModel` extending base
- `BranchListModel` extending base
- `RecentActivityModel` extending base
- All share navigation, filtering, prompt logic

### 3.6 Declarative Keybindings

**Problem Solved:** Users want customizable keybindings without code changes.

**Solution:** Keybindings defined in YAML, mapped to actions at runtime.

**Configuration** (`.config/gh-dash/config.yaml`):

```yaml
keybindings:
  universal:
    - key: "R"
      builtin: "refreshAll"

    - key: "ctrl+p"
      command: "gh pr checkout {{.PrNumber}}"
      name: "checkout PR"

  prs:
    - key: "m"
      builtin: "mergePR"

    - key: "o"
      command: "open {{.PrUrl}}"
      name: "open in browser"
```

**Implementation:**

```go
// keys/keys.go
type KeyMap struct {
    // Built-in actions
    Refresh   key.Binding
    Merge     key.Binding
    Close     key.Binding
    // ...

    // Custom commands
    Custom    []CustomCommand
}

type CustomCommand struct {
    Key     string
    Command string  // Go template: "gh pr checkout {{.PrNumber}}"
    Name    string
}

// Rebind from config
func (k *KeyMap) Rebind(config config.KeybindingsConfig) {
    for _, kb := range config.Universal {
        if kb.Builtin != "" {
            // Map to built-in action
            switch kb.Builtin {
            case "refreshAll":
                k.Refresh.SetKeys(kb.Key)
            }
        } else if kb.Command != "" {
            // Add custom command
            k.Custom = append(k.Custom, CustomCommand{
                Key:     kb.Key,
                Command: kb.Command,
                Name:    kb.Name,
            })
        }
    }
}
```

**Execution:**

```go
// Check custom commands in Update()
for _, cmd := range m.keys.Custom {
    if key.Matches(msg, cmd.Key) {
        return m, executeCustomCommand(cmd, m.getCurrRow())
    }
}

func executeCustomCommand(cmd CustomCommand, data RowData) tea.Cmd {
    // Execute template with current row data
    tmpl := template.New("cmd").Parse(cmd.Command)
    var buf bytes.Buffer
    tmpl.Execute(&buf, data)

    // Execute shell command
    return tea.ExecProcess(exec.Command("sh", "-c", buf.String()), nil)
}
```

**Benefits:**

✅ **Customization** - Users define keybindings in config
✅ **Extensibility** - Custom commands via templates
✅ **No code changes** - Add new actions without recompiling
✅ **Documentation** - Help shows configured keybindings

**Application to gbm:**

gbm could support:
- Custom git commands: `git worktree remove {{.Path}}`
- Editor opening: `code {{.Path}}`
- JIRA links: `open https://jira.company.com/browse/{{.JiraKey}}`
- Shell commands: `cd {{.Path}} && git pull`

### 3.7 Configurable Layout System

**Problem Solved:** Different users want different column visibility and widths.

**Solution:** Layout configuration per section with flexible width calculation.

**Configuration:**

```yaml
prs:
  - title: "My PRs"
    layout:
      updatedAt: { width: 10 }
      repo:      { hidden: true }
      title:     { width: 50, grow: true }
      author:    { width: 15 }
      lines:     { width: 10 }
```

**Implementation:**

```go
type ColumnConfig struct {
    Width  *int  `yaml:"width,omitempty"`
    Hidden *bool `yaml:"hidden,omitempty"`
    Grow   *bool `yaml:"grow,omitempty"`
}

type Column struct {
    Title     string
    Width     int
    Grow      bool
    Hidden    bool
    FlexGrow  int  // Priority for growing
}

// Compute final widths
func computeColumnWidths(cols []Column, totalWidth int) []Column {
    // 1. Reserve space for fixed-width columns
    fixedWidth := 0
    growCount := 0
    for _, col := range cols {
        if col.Hidden {
            continue
        }
        if col.Grow {
            growCount++
        } else {
            fixedWidth += col.Width
        }
    }

    // 2. Distribute remaining width to growing columns
    remainingWidth := totalWidth - fixedWidth
    if growCount > 0 {
        growWidth := remainingWidth / growCount
        for i := range cols {
            if cols[i].Grow && !cols[i].Hidden {
                cols[i].Width = growWidth
            }
        }
    }

    return cols
}
```

**Benefits:**

✅ **Flexibility** - Users control what they see
✅ **Responsive** - Columns adapt to terminal width
✅ **Defaults** - Sensible defaults, overridable per section
✅ **No hardcoding** - All dimensions from config

**Application to gbm:**

gbm could configure:
- Worktree columns: branch, path, status, last commit, age
- Branch columns: name, tracking, ahead/behind, last commit
- Visibility toggles for advanced users vs. beginners
- Compact mode for small terminals

---

## 4. Libraries & Technologies

### 4.1 Core TUI Stack

| Library | Purpose | Version | Why Used |
|---------|---------|---------|----------|
| **charmbracelet/bubbletea** | Main TUI framework | Latest | Elm Architecture, type-safe, composable |
| **charmbracelet/lipgloss** | Styling and layout | Latest | Declarative styling, adaptive colors, layout primitives |
| **charmbracelet/bubbles** | Pre-built components | Latest | Spinner, textinput, help - battle-tested components |
| **charmbracelet/glamour** | Markdown rendering | Latest | GitHub-flavored markdown with theme support |

**Why This Stack:**

- **Type-safe** - Compile-time checks for messages and state
- **Composable** - Components naturally nest and delegate
- **Testable** - Pure functions, no side effects in core logic
- **Active** - Charm.sh maintains regularly, large community

### 4.2 GitHub Integration

| Library | Purpose | Notes |
|---------|---------|-------|
| **cli/go-gh/v2** | GitHub CLI library | Official gh integration, handles auth |
| **shurcooL/githubv4** | GraphQL client | Efficient queries, pagination support |
| **cli/shurcooL-graphql** | GraphQL builder | Type-safe query construction |

**Data Fetching Pattern:**

```go
// GraphQL query for PRs
type PRQuery struct {
    Viewer struct {
        PullRequests struct {
            Nodes []struct {
                Number int
                Title  string
                Author struct{ Login string }
                // ... 30+ fields
            }
            PageInfo struct {
                HasNextPage bool
                EndCursor   string
            }
        } `graphql:"pullRequests(first: 20, after: $cursor)"`
    }
}

// Execute with caching
client := githubv4.NewClient(httpClient)
err := client.Query(ctx, &query, variables)
```

**Benefits:**
- Fetch only needed fields (vs. REST API over-fetching)
- Single request for related data
- Built-in pagination
- Type-safe responses

### 4.3 Configuration & Validation

| Library | Purpose | Why Used |
|---------|---------|----------|
| **knadh/koanf/v2** | Config management | Supports YAML, env vars, defaults, merging |
| **go-playground/validator/v10** | Struct validation | Declarative validation tags, custom validators |

**Configuration Loading:**

```go
k := koanf.New(".")
k.Load(file.Provider("config.yaml"), yaml.Parser())
k.Load(env.Provider("GH_DASH_", ".", nil), nil)

var config Config
k.Unmarshal("", &config)

validate := validator.New()
if err := validate.Struct(config); err != nil {
    // Handle validation errors
}
```

**Validation Example:**

```go
type SectionConfig struct {
    Title   string `validate:"required"`
    Limit   *int   `validate:"omitempty,min=1,max=50"`
    Filters string `validate:"omitempty"`
}
```

### 4.4 Utilities

| Library | Purpose | Usage |
|---------|---------|-------|
| **lrstanley/bubblezone** | Mouse support | Click zones for interactive elements |
| **atotto/clipboard** | Clipboard ops | Copy URLs, branch names, etc. |
| **gen2brain/beeep** | Desktop notifications | Task completion alerts |
| **maypok86/otter/v2** | Caching | Cache expensive GitHub API calls |

### 4.5 CLI Framework

| Library | Purpose | Notes |
|---------|---------|-------|
| **spf13/cobra** | CLI framework | Standard choice for Go CLIs |
| **spf13/viper** | Flag management | Not heavily used (koanf handles config) |

**CLI Structure:**

```go
rootCmd := &cobra.Command{
    Use:   "gh-dash",
    Short: "GitHub dashboard in your terminal",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Load config
        config := loadConfig()

        // Initialize TUI
        program := tea.NewProgram(
            tui.New(config),
            tea.WithAltScreen(),
            tea.WithMouseCellMotion(),
        )

        return program.Start()
    },
}
```

---

## 5. Component Hierarchy

### 5.1 Visual Component Tree

```
┌─ Model (ui.go) ──────────────────────────────────────────┐
│                                                            │
│  ┌─ Tabs ─────────────────────────────────────┐           │
│  │  [PRs] [Issues] [Repo]                      │           │
│  └─────────────────────────────────────────────┘           │
│                                                            │
│  ┌─ Main Content ────────────────┐  ┌─ Sidebar ────────┐ │
│  │                                │  │                   │ │
│  │  ┌─ Section (PRs) ───────┐    │  │  ┌─ PR View ───┐ │ │
│  │  │                        │    │  │  │ Overview    │ │ │
│  │  │  ┌─ SearchBar ──────┐ │    │  │  │ Checks      │ │ │
│  │  │  └──────────────────┘ │    │  │  │ Activity    │ │ │
│  │  │                        │    │  │  │ Files       │ │ │
│  │  │  ┌─ Table ──────────┐ │    │  │  └─────────────┘ │ │
│  │  │  │ ┌─ ListViewport─┐│ │    │  │                   │ │
│  │  │  │ │ • PR Row      ││ │    │  │  [Markdown]       │ │
│  │  │  │ │ • PR Row      ││ │    │  │  [Comments]       │ │
│  │  │  │ │ • PR Row ✓    ││ │    │  │  [Checks]         │ │
│  │  │  │ │ • PR Row      ││ │    │  │                   │ │
│  │  │  │ └───────────────┘│ │    │  │                   │ │
│  │  │  └──────────────────┘ │    │  │                   │ │
│  │  │                        │    │  │                   │ │
│  │  │  [Pagination: 1/5]     │    │  │                   │ │
│  │  └────────────────────────┘    │  └───────────────────┘ │
│  │                                │                         │
│  └────────────────────────────────┘                         │
│                                                            │
│  ┌─ Footer ────────────────────────────────────────────┐   │
│  │  ⣾ Closing PR #123...  │  ? help │ ← → switch       │   │
│  └────────────────────────────────────────────────────┘   │
│                                                            │
│  ┌─ Prompt ───────────────────────┐                        │
│  │ Confirm merge PR #123?         │                        │
│  │         [Yes]  [No]            │                        │
│  └────────────────────────────────┘                        │
└────────────────────────────────────────────────────────────┘
```

### 5.2 Component Responsibilities

#### **Root Model** (`ui.go`)
- Initialize and coordinate all components
- Handle global keybindings (view switching, quit)
- Route messages to appropriate components
- Manage task lifecycle (start, track, finish)
- Compute and propagate context updates

#### **Section** (`section/`, `prssection/`, etc.)
- Display list of items (PRs, issues, branches)
- Handle navigation (up, down, page up/down)
- Filter/search items
- Fetch data and paginate
- Show confirmation prompts
- Report selection to root model

#### **Detail Views** (`prview/`, `issueview/`)
- Display detailed information about selected item
- Tabs for different aspects (overview, checks, activity, files)
- Markdown rendering for descriptions/comments
- Action buttons (approve, merge, comment)
- Fetch additional data on demand

#### **Table** (`table/`)
- Generic table rendering
- Column management (widths, alignment)
- Row selection and highlighting
- Keyboard/mouse navigation
- Responsive to dimension changes

#### **ListViewport** (`listviewport/`)
- Scrollable container for rows
- Virtual scrolling for large lists
- Auto-centering of selected item
- Smooth scrolling animations
- Keyboard navigation (vim bindings)

#### **Search** (`search/`)
- Filter input field
- Real-time filtering
- Highlight matches
- Clear button

#### **Footer** (`footer/`)
- Display help hints
- Show task spinner and status
- Error messages
- Current view indicator

### 5.3 Data Flow

```
┌─────────────────┐
│  User Input     │
│  (key press)    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────┐
│  Root Model.Update()        │
│  • Check global keys        │
│  • Route SectionMsg         │
│  • Handle TaskStartMsg      │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│  Section.Update()           │
│  • Check section keys       │
│  • Update table/search      │
│  • Return SectionCmd        │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│  Task Execution             │
│  • Show spinner             │
│  • Execute GitHub API       │
│  • Return TaskFinishedMsg   │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│  Data Update                │
│  • Update section data      │
│  • Refresh table            │
│  • Show success/error       │
└────────┬────────────────────┘
         │
         ▼
┌─────────────────────────────┐
│  View Rendering             │
│  • Compute layout           │
│  • Render components        │
│  • Display to terminal      │
└─────────────────────────────┘
```

---

## 6. Data Flow & State Management

### 6.1 Configuration Loading

**Startup Sequence:**

```
main()
  → cmd.Execute()
    → loadConfig()
      ├─ Load defaults
      ├─ Load ~/.config/gh-dash/config.yaml
      ├─ Merge env vars (GH_DASH_*)
      ├─ Validate with validator
      └─ Return Config
    → initTUI(config)
      ├─ Create ProgramContext
      ├─ Initialize components
      └─ tea.NewProgram(model)
```

**Config Structure:**

```go
type Config struct {
    // GitHub settings
    RepoPath string `koanf:"repoPath"`

    // UI preferences
    Theme      ThemeConfig      `koanf:"theme"`
    Preview    PreviewConfig    `koanf:"preview"`
    Keybindings KeybindingsConfig `koanf:"keybindings"`

    // Data sections
    PRs    []SectionConfig `koanf:"prs" validate:"dive"`
    Issues []SectionConfig `koanf:"issues" validate:"dive"`

    // Performance
    RefetchHours int `koanf:"refetchHours" validate:"min=1,max=24"`
}

type SectionConfig struct {
    Title   string        `koanf:"title" validate:"required"`
    Filters string        `koanf:"filters"`
    Limit   *int          `koanf:"limit" validate:"omitempty,min=1,max=50"`
    Layout  LayoutConfig  `koanf:"layout"`
}
```

### 6.2 GitHub Data Fetching

**Incremental Loading Pattern:**

```go
// 1. Initial fetch (minimal data)
type PRListQuery struct {
    Viewer struct {
        PullRequests struct {
            Nodes []struct {
                Number    int
                Title     string
                CreatedAt time.Time
                // ... ~10 fields for list view
            }
            PageInfo PageInfo
        } `graphql:"pullRequests(first: $limit, after: $cursor)"`
    }
}

// 2. Enrichment on selection (detailed data)
type PRDetailQuery struct {
    Repository struct {
        PullRequest struct {
            Number      int
            Body        string
            Comments    []Comment
            Reviews     []Review
            Commits     []Commit
            CheckSuites []CheckSuite
            // ... ~50 fields for detail view
        } `graphql:"pullRequest(number: $number)"`
    }
}
```

**Caching Strategy:**

```go
// Cache with TTL
cache, _ := otter.MustBuilder[string, interface{}](10_000).
    WithTTL(time.Hour).
    Build()

func fetchPRs(ctx context.Context, filters string) ([]PR, error) {
    cacheKey := fmt.Sprintf("prs:%s", filters)

    // Check cache
    if cached, ok := cache.Get(cacheKey); ok {
        return cached.([]PR), nil
    }

    // Fetch from API
    prs, err := queryGitHub(ctx, filters)
    if err != nil {
        return nil, err
    }

    // Cache result
    cache.Set(cacheKey, prs)
    return prs, nil
}
```

**Pagination:**

```go
type Paginator struct {
    HasNextPage bool
    EndCursor   string
}

func (m *Model) fetchNextPage() tea.Cmd {
    return func() tea.Msg {
        variables := map[string]interface{}{
            "cursor": m.paginator.EndCursor,
            "limit":  20,
        }

        var query PRListQuery
        err := client.Query(ctx, &query, variables)

        return FetchedPRsMsg{
            PRs:      query.Viewer.PullRequests.Nodes,
            PageInfo: query.Viewer.PullRequests.PageInfo,
        }
    }
}
```

### 6.3 State Updates

**Update Flow:**

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {

    // Data fetched from GitHub
    case FetchedPRsMsg:
        // Append to existing data
        m.prs = append(m.prs, msg.PRs...)
        m.paginator = msg.PageInfo
        m.isLoading = false
        return m, nil

    // User action completed
    case UpdatePRMsg:
        // Find and update specific PR
        for i := range m.prs {
            if m.prs[i].Number == msg.PRNumber {
                if msg.IsClosed != nil {
                    m.prs[i].IsClosed = *msg.IsClosed
                }
                if msg.IsMerged != nil {
                    m.prs[i].IsMerged = *msg.IsMerged
                }
            }
        }
        return m, nil

    // Error occurred
    case ErrorMsg:
        m.ctx.Error = msg.Err
        return m, nil
    }
}
```

**Optimistic Updates:**

```go
// Update UI immediately, revert on error
func (m *Model) toggleAssignment(pr PR) tea.Cmd {
    // Optimistically update
    pr.IsAssigned = !pr.IsAssigned
    m.updatePR(pr)

    return tea.Batch(
        // Show as complete
        func() tea.Msg {
            return UpdatedPRMsg{PR: pr}
        },

        // Actual API call
        func() tea.Msg {
            err := api.ToggleAssignment(pr.Number)
            if err != nil {
                // Revert on error
                pr.IsAssigned = !pr.IsAssigned
                return UpdatedPRMsg{PR: pr, Error: err}
            }
            return nil
        },
    )
}
```

### 6.4 Error Handling

**Error Display:**

```go
// Global error in context
type ProgramContext struct {
    Error error
}

// Footer shows error
func (m Footer) View() string {
    if m.ctx.Error != nil {
        return lipgloss.NewStyle().
            Foreground(lipgloss.Color("9")).  // Red
            Render(fmt.Sprintf("❌ %s", m.ctx.Error))
    }
    return m.renderHelp()
}

// Clear error on any key
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    if _, ok := msg.(tea.KeyMsg); ok {
        m.ctx.Error = nil
    }
}
```

**Task Errors:**

```go
case TaskFinishedMsg:
    task := msg.Task

    if task.State == TaskError {
        // Show error in footer
        m.ctx.Error = fmt.Errorf(
            "%s failed: %w",
            task.StartText,
            task.Error,
        )

        // Send notification
        beeep.Notify(
            "gh-dash",
            fmt.Sprintf("%s failed", task.StartText),
            "",
        )
    } else {
        // Show success briefly
        m.successMsg = task.FinishedText
        time.AfterFunc(3*time.Second, func() {
            m.successMsg = ""
        })
    }
```

---

## 7. Best Practices Observed

### 7.1 Code Organization

✅ **Feature-based component structure** - Each component in its own package
✅ **Interface segregation** - Minimal interfaces (Identifier, Component, Table)
✅ **Clear naming** - `prssection`, `prview`, `prrow` show hierarchy
✅ **Separate concerns** - Data layer, UI layer, config layer
✅ **Single responsibility** - Each component does one thing well

### 7.2 Bubble Tea Patterns

✅ **No mutable global state** - All state in model
✅ **Pure Update() functions** - No side effects, return commands
✅ **Message types for everything** - Even internal communication
✅ **Batch commands** - Use `tea.Batch()` for multiple operations
✅ **Sequence commands** - Use `tea.Sequence()` for ordered operations
✅ **Context for shared state** - Avoid passing refs between components

### 7.3 Performance

✅ **Virtual scrolling** - Only render visible rows
✅ **Caching** - Cache GitHub API responses
✅ **Incremental loading** - Fetch minimal data initially, enrich on demand
✅ **Debouncing** - Search input debounced to avoid excessive filtering
✅ **Lazy rendering** - Detail views only render when visible
✅ **Style caching** - Compute lipgloss styles once, reuse

### 7.4 User Experience

✅ **Immediate feedback** - Spinner shows instantly for async operations
✅ **Optimistic updates** - UI updates before API confirms
✅ **Error recovery** - Errors displayed, clearable with any key
✅ **Responsive layout** - Adapts to terminal size
✅ **Keyboard + mouse** - Both input methods supported
✅ **Help hints** - Footer shows available actions
✅ **Confirmation prompts** - Destructive actions require confirmation
✅ **Desktop notifications** - Long operations notify when complete

### 7.5 Configuration

✅ **Defaults that work** - Zero config required to start
✅ **Progressive disclosure** - Advanced features in config, not required
✅ **YAML config** - Human-readable, version-controllable
✅ **Validation** - Config validated on load with clear errors
✅ **Hot reload** - Changes take effect on next fetch
✅ **Environment variables** - Override config via `GH_DASH_*`

### 7.6 Testing

✅ **Component tests** - Test components in isolation
✅ **Golden files** - Test rendering output
✅ **Table-driven tests** - Test data transformations
✅ **Mock context** - Test with fake ProgramContext
✅ **No integration tests** - Unit tests cover business logic

### 7.7 Code Quality

✅ **Linting** - `golangci-lint` enforced
✅ **Formatting** - `gofmt` on all files
✅ **Documentation** - Godoc comments on public APIs
✅ **Error handling** - Errors wrapped with context
✅ **Type safety** - Minimal use of `interface{}`
✅ **No reflection** - All types known at compile time

---

## 8. Recommendations for gbm

**Note:** These recommendations focus on **TUI architecture improvements only**. The git service layer, configuration system, errors, and documentation are already excellent.

---

### 8.1 Immediate Wins (Low Effort, High Impact)

#### **1. Extract Base Section Model**

**Current State:** TUI code in `cmd/service/worktree_*.go` has table model but not abstracted for reuse.

**Solution:** Create `section.BaseModel` with common table/search/prompt behavior.

```go
// internal/tui/section/base.go
type BaseModel struct {
    Id       int
    Type     string
    Config   config.SectionConfig
    Ctx      *context.ProgramContext

    Table    table.Model
    Search   search.Model
    Prompt   prompt.Model

    IsLoading      bool
    IsSearching    bool
    CurrPage       int
    TotalPages     int
}

// Common methods
func (m *BaseModel) NumRows() int { return m.Table.NumRows() }
func (m *BaseModel) NextRow() { m.Table.NextRow() }
// ... etc.

// internal/tui/worktrees/worktrees.go
type Model struct {
    section.BaseModel
    Worktrees []Worktree
}
```

**Benefits:**
- Extract ~200 lines of common code
- Enable multiple section types (worktrees, branches, tags)
- Consistent navigation across all views

**Effort:** 4-6 hours

---

#### **2. Implement Context Propagation**

**Current State:** TUI doesn't propagate shared state (dimensions, styles, services) through components. No responsive layout on terminal resize.

**Solution:** Create `ProgramContext` with services, dimensions, and computed styles.

```go
// internal/tui/context/context.go
type ProgramContext struct {
    // Services
    GitService  *git.Service
    JiraService *jira.Service

    // Dimensions
    ScreenWidth  int
    ScreenHeight int

    // State
    RepoRoot string
    Error    error

    // Styling
    Theme  Theme
    Styles Styles

    // Task management
    StartTask func(Task) tea.Cmd
}

// Update all components
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.ctx.ScreenWidth = msg.Width
        m.ctx.ScreenHeight = msg.Height
        m.ctx.Styles = computeStyles(m.ctx)

        // Propagate to all components
        m.worktreeList.UpdateProgramContext(m.ctx)
        m.branchList.UpdateProgramContext(m.ctx)
        m.footer.UpdateProgramContext(m.ctx)
    }
}
```

**Benefits:**
- Decouple components from services
- Easy to mock for testing
- Single source of truth for dimensions/theme
- Enable responsive layout

**Effort:** 6-8 hours

---

#### **3. Add Task-Based Async Operations**

**Current State:** Git operations via `git.Service` are synchronous and block the TUI. No spinner feedback for slow operations.

**Solution:** Implement task tracking system with immediate spinner feedback.

```go
// internal/tui/tasks/worktree.go
func AddWorktree(ctx *context.ProgramContext, branch, path string) tea.Cmd {
    return fireTask(ctx, Task{
        Id:           buildTaskId("worktree_add", branch),
        StartText:    fmt.Sprintf("Creating worktree for %s", branch),
        FinishedText: fmt.Sprintf("Worktree created at %s", path),
        Execute: func() error {
            return ctx.GitService.AddWorktree(branch, path)
        },
        OnComplete: func() tea.Msg {
            return RefreshWorktreesMsg{}
        },
    })
}

// Usage in component
case key.Matches(msg, m.keys.Add):
    return m, tasks.AddWorktree(m.ctx, branch, path)
```

**Benefits:**
- Non-blocking operations
- Spinner feedback
- Consistent error handling
- Better UX for slow operations (large repos)

**Effort:** 8-10 hours

---

### 8.2 Medium-Term Improvements (Moderate Effort, High Value)

#### **4. Refactor to Component-Based Architecture**

**Current State:** TUI code lives in `cmd/service/worktree_*.go` (worktree.go, worktree_tui.go, worktree_table.go, worktree_fsm.go, etc.). Works but not modular.

**Solution:** Move to `internal/tui/` with component-based structure following gh-dash patterns.

**New Structure:**

```
internal/tui/
├── ui.go                      # Root model
├── context/
│   ├── context.go
│   └── styles.go
├── components/
│   ├── section/               # Base section
│   │   └── base.go
│   ├── worktrees/             # Worktree list section
│   │   ├── worktrees.go       # Model
│   │   ├── row.go             # Row rendering
│   │   └── actions.go         # Add/remove/checkout
│   ├── branches/              # Branch list section
│   │   └── branches.go
│   ├── table/                 # Generic table
│   │   └── table.go
│   ├── footer/                # Status footer
│   │   └── footer.go
│   ├── search/                # Search input
│   │   └── search.go
│   └── prompt/                # Confirmation prompt
│       └── prompt.go
├── tasks/                     # Async operations
│   ├── worktree.go
│   └── git.go
└── keys/                      # Keybindings
    └── keys.go
```

**Benefits:**
- Clear separation of concerns
- Components reusable (table, search, prompt)
- Easier to add new views (tags, remotes, stashes)
- Better testability

**Effort:** 16-20 hours

---

#### **5. Add Configurable Keybindings**

**Current State:** Keybindings hardcoded in TUI code. gbm already has `.gbm/config.yaml` for other settings.

**Solution:** Extend config with YAML-based keybindings and custom command templates.

**Config:**

```yaml
# .gbm/config.yaml
keybindings:
  universal:
    - key: "R"
      builtin: "refreshAll"
    - key: "ctrl+o"
      command: "code {{.Path}}"
      name: "open in VS Code"

  worktrees:
    - key: "a"
      builtin: "addWorktree"
    - key: "d"
      builtin: "removeWorktree"
    - key: "enter"
      command: "cd {{.Path}}"
      name: "navigate to worktree"
```

**Benefits:**
- Users customize keybindings
- No code changes for new actions
- Template-based commands (powerful!)
- Help shows configured bindings

**Effort:** 12-16 hours

---

#### **6. Implement Section Interface**

**Current State:** TUI shows only worktree list. To add branches/tags view would require duplicating navigation/table logic.

**Solution:** Abstract common behavior into `Section` interface like gh-dash.

```go
type Section interface {
    GetId() int
    GetType() string
    Update(msg tea.Msg) tea.Cmd
    View() string

    NumRows() int
    CurrRow() int
    NextRow()
    PrevRow()

    SetIsSearching(bool)
    GetFilters() string
}

// Enable multiple views
type Model struct {
    sections      []Section
    currSectionId int

    // Specific implementations
    worktrees  *worktrees.Model   // implements Section
    branches   *branches.Model    // implements Section
    tags       *tags.Model        // implements Section
}
```

**Benefits:**
- Add new views without changing root model
- Polymorphic handling of different data types
- Consistent navigation across all sections
- Support split views (worktrees + branches side-by-side)

**Effort:** 10-12 hours

---

### 8.3 Long-Term Vision (High Effort, Transformative)

#### **7. Multi-Section Dashboard**

**Goal:** Support multiple configured sections like gh-dash.

**Config:**

```yaml
worktrees:
  - title: "Active Worktrees"
    filters: "status:active"
    layout:
      branch:     { width: 30 }
      path:       { width: 40, grow: true }
      status:     { width: 10 }
      lastCommit: { width: 50 }
      age:        { width: 10 }

  - title: "Stale Worktrees"
    filters: "age:>30d"
    layout:
      branch:     { width: 30 }
      path:       { width: 40, grow: true }
      age:        { width: 10 }

branches:
  - title: "Feature Branches"
    filters: "prefix:feature/"
    layout:
      name:     { width: 30 }
      tracking: { width: 20 }
      ahead:    { width: 5 }
      behind:   { width: 5 }
```

**Benefits:**
- Dashboard-style overview
- Multiple views of same data
- User-configured workflows
- Power users can deeply customize

**Effort:** 40-50 hours

---

#### **8. Detail Sidebar with Tabs**

**Goal:** Show rich information about selected worktree.

**Tabs:**
- **Overview** - Branch, path, status, last commit
- **Activity** - Recent commits in this worktree
- **Changes** - `git status` output
- **Remotes** - Tracking info, ahead/behind

**Implementation:**

```go
// internal/tui/components/worktreeview/
type Model struct {
    tabs      tabs.Model
    activeTab int

    // Tab content
    overview  overview.Model
    activity  activity.Model
    changes   changes.Model
    remotes   remotes.Model
}

func (m Model) View() string {
    tabBar := m.tabs.View()

    var content string
    switch m.activeTab {
    case 0:
        content = m.overview.View()
    case 1:
        content = m.activity.View()
    // ...
    }

    return lipgloss.JoinVertical(tabBar, content)
}
```

**Benefits:**
- Rich information display
- Doesn't clutter main list
- Familiar pattern (like gh-dash)
- Can show git log, diff, etc.

**Effort:** 20-24 hours

---

#### **9. JIRA Integration Enhancement**

**Goal:** First-class JIRA support in TUI.

**Features:**
- **JIRA section** - List assigned issues
- **Create worktree from issue** - Select issue, auto-create worktree with proper naming
- **Link worktree to issue** - Show linked issue in worktree list
- **Transition issues** - Mark "In Progress" when checking out worktree

**Implementation:**

```go
// JIRA section
type JiraSection struct {
    section.BaseModel
    Issues []jira.Issue
}

// Link to worktree
type Worktree struct {
    Branch    string
    Path      string
    JiraKey   string      // "PROJ-123"
    JiraTitle string      // "Add new feature"
}

// Action: Create worktree from JIRA
func CreateFromJira(ctx *ProgramContext, issue jira.Issue) tea.Cmd {
    branchName := jira.GenerateBranchName(issue)
    path := filepath.Join("worktrees", branchName)

    return tea.Sequence(
        // Create worktree
        tasks.AddWorktree(ctx, branchName, path),

        // Transition issue
        tasks.TransitionJiraIssue(ctx, issue.Key, "In Progress"),

        // Link in metadata
        tasks.SaveWorktreeMetadata(path, issue.Key),
    )
}
```

**Benefits:**
- Seamless JIRA workflow
- Less context switching
- Auto-naming from issues
- Track progress in JIRA

**Effort:** 30-40 hours

---

### 8.4 Prioritized Roadmap

**Phase 1: TUI Foundation (20-30 hours)**
1. ⏳ Extract Base Section Model (4-6h)
2. ⏳ Implement Context Propagation (6-8h)
3. ⏳ Add Task-Based Async Operations (8-10h)

**Phase 2: TUI Architecture (40-50 hours)**
4. ⏳ Refactor to Component-Based Architecture (16-20h)
5. ⏳ Implement Section Interface (10-12h)
6. ⏳ Add Configurable Keybindings (12-16h)

**Phase 3: TUI Features (60-80 hours)**
7. ⏳ Multi-Section Dashboard (40-50h)
8. ⏳ Detail Sidebar with Tabs (20-24h)

**Phase 4: TUI Integration (30-40 hours)**
9. ⏳ Enhanced JIRA Integration in TUI (30-40h)

**Total Effort:** 150-200 hours for complete TUI transformation

**Note:** The git service layer (organization, errors, testing) is already production-ready and doesn't need this work.

---

### 8.5 Quick Wins to Start

**Week 1:**
- Extract common table code to `internal/tui/components/table/`
- Create `ProgramContext` struct
- Move git operations to async tasks with spinners

**Week 2:**
- Refactor worktree TUI to `internal/tui/components/worktrees/`
- Add search component
- Add confirmation prompt component

**Week 3:**
- Implement section interface
- Add branch list section
- Support switching between sections (tabs)

**Week 4:**
- Add YAML keybindings
- Create footer component with help
- Add error display in footer

---

## 9. Specific gh-dash Features Worth Adopting

Based on gh-dash's TUI architecture, here are the most valuable patterns to adopt for gbm:

### 9.1 Spinner Feedback with Task Tracking

**What gh-dash does:**
- Shows spinner immediately when operation starts
- Displays operation name ("Creating worktree for feature-x...")
- Shows completion message with timing
- Errors displayed in footer with context
- Multiple operations can run concurrently

**How gbm would benefit:**
- `git worktree add` on large repos can take 5-10 seconds
- `git fetch` can be slow with many branches
- Currently blocks TUI with no feedback
- Users don't know if command is working or frozen

**Implementation priority:** **HIGH** - Most noticeable UX improvement

### 9.2 Responsive Layout with Context Propagation

**What gh-dash does:**
- Recomputes all component dimensions on terminal resize
- Styles cached and propagated through `ProgramContext`
- Components re-render with updated widths
- Columns grow/shrink proportionally

**How gbm would benefit:**
- Current TUI doesn't adapt to terminal size changes
- Would enable multi-column layouts (worktrees + sidebar)
- Could show/hide columns based on terminal width
- Better experience on different terminal sizes

**Implementation priority:** **MEDIUM** - Nice to have, not critical

### 9.3 Multiple Section Views

**What gh-dash does:**
- Configure multiple PR sections: "My PRs", "Team PRs", "Needs Review"
- Each section has own filters, layout, keybindings
- Tab navigation between sections
- Sections load independently

**How gbm could use this:**
```yaml
worktrees:
  - title: "Active Worktrees"
    filters: "status:active"
  - title: "Stale Worktrees (>30d)"
    filters: "age:>30d"

branches:
  - title: "Feature Branches"
    filters: "prefix:feature/"
  - title: "All Local Branches"
```

**Implementation priority:** **LOW** - Advanced feature, most users only need one view

### 9.4 Configurable Keybindings with Custom Commands

**What gh-dash does:**
```yaml
keybindings:
  prs:
    - key: "o"
      command: "open {{.PrUrl}}"
    - key: "ctrl+e"
      command: "code {{.RepoPath}}"
```

**How gbm could use this:**
```yaml
keybindings:
  worktrees:
    - key: "o"
      command: "code {{.Path}}"
      name: "open in VS Code"
    - key: "ctrl+j"
      command: "open https://jira.company.com/browse/{{.JiraKey}}"
      name: "open JIRA issue"
```

**Implementation priority:** **MEDIUM** - Power users would love this

### 9.5 Detail Sidebar with Rich Information

**What gh-dash does:**
- Shows PR details in right sidebar
- Tabs for: Overview, Checks, Activity, Files
- Markdown rendering for descriptions
- Real-time check status
- Comment threads

**How gbm could use this:**
- Show worktree details in sidebar
- Tabs: Overview, Git Status, Recent Commits, Remotes
- Display `git log` formatted output
- Show `git status` with syntax highlighting
- Display ahead/behind tracking info

**Implementation priority:** **LOW** - Visual enhancement, current table view sufficient

### 9.6 Search/Filter with Real-time Updates

**What gh-dash does:**
- Press `/` to enter search mode
- Filter list as you type
- Highlight matching text
- ESC to clear search

**How gbm could use this:**
- Filter worktrees by branch name
- Filter by path substring
- Filter by status (active, stale)
- Filter by JIRA key

**Implementation priority:** **MEDIUM** - Helpful with many worktrees

### 9.7 Mouse Support

**What gh-dash does:**
- Click to select row
- Scroll wheel to navigate
- Click buttons in sidebar
- Click tabs to switch sections

**How gbm could use this:**
- Click worktree to select
- Scroll long worktree lists
- Click action buttons (Add, Remove, Sync)

**Implementation priority:** **LOW** - Keyboard navigation sufficient for most users

---

## 10. Priority Recommendations

Based on gbm's current needs and gh-dash patterns:

**🔥 High Priority (Immediate UX Wins):**
1. **Task-based async operations** - Spinner feedback for slow git commands
2. **Search/filter** - Essential with many worktrees
3. **Error display in footer** - Better error visibility than current stderr

**⚡ Medium Priority (Power User Features):**
4. **Configurable keybindings** - Custom commands via templates
5. **Context propagation** - Responsive layout on resize
6. **Enhanced JIRA integration** - Create worktree from TUI-selected issue

**📋 Low Priority (Nice to Have):**
7. **Multiple section views** - Only needed by advanced users
8. **Detail sidebar** - Current table view works well
9. **Mouse support** - Keyboard-first tool doesn't need this

**Recommendation:** Start with High Priority items (tasks + search + error footer) as they provide immediate, noticeable UX improvements with relatively low effort (20-30 hours total).

---

## Conclusion

**gh-dash** is an exceptionally well-designed TUI application that demonstrates best practices for complex Bubble Tea applications. The codebase is highly modular, with clear separation of concerns, strong typing via interfaces, and excellent reusability through composition.

**Key Lessons from gh-dash:**

1. **Interface-based abstractions** enable polymorphic handling of different data types
2. **Context propagation** keeps components decoupled while sharing state
3. **Task-based async operations** provide excellent UX for long-running operations
4. **Component-based architecture** makes code navigable and testable
5. **Configuration-driven UI** enables powerful customization without code changes
6. **Composition over inheritance** through embedded structs reduces duplication

**Current State of gbm (2026-01-03):**

**✅ Excellent Foundation:**
- Git service layer is clean, well-organized, and production-ready
- Configuration system supports templates and file copying
- Comprehensive documentation and testing infrastructure
- Typed errors with rich context
- Universal stdout/stderr pattern for shell integration

**⏳ TUI Architecture Opportunity:**

The TUI implementation in `cmd/service/worktree_*.go` would benefit from gh-dash patterns:

1. **Start with foundation** (context, tasks, base model) - 20-30 hours
2. **Refactor to components** (move to internal/tui/) - 40-50 hours
3. **Add features** (multiple sections, tabs, configurable keys) - 90-120 hours

**Total TUI transformation:** 150-200 hours for a production-ready, extensible TUI.

**Is it Worth It?**

For a tool users interact with daily:
- ✅ **Yes** if you want multiple views (worktrees, branches, tags)
- ✅ **Yes** if you want configurable keybindings
- ✅ **Yes** if you want spinner feedback for slow operations
- ⚠️ **Maybe** if current TUI meets all needs (can defer until needed)

The improved architecture would make future features (git log, stashes, remotes, etc.) much easier to add. The foundation (git service, config, testing) is already excellent, so the TUI refactor can be done incrementally without breaking existing functionality.
