# PRD: `gbm config` Interactive TUI

**Status**: Draft
**Version**: 1.0
**Date**: January 2026

---

## 1. Overview

Add a `gbm config` command that launches a fullscreen, interactive TUI for managing `.gbm/config.yaml`. The TUI uses a lazygit-inspired sidebar navigation pattern, allowing users to explore and edit configuration sections without touching YAML directly.

### Goals
- **Accessibility**: Enable non-technical users to configure GBM without editing YAML
- **Discoverability**: Show all available config options with descriptions/help
- **Validation**: Real-time feedback on invalid settings before save
- **Visual Consistency**: Reuse theme, components, and aesthetic from `gbm wt ls` and `gbm wt add`
- **Testability**: Comprehensive storybook-go coverage for all components, sections, and page layouts

---

## 2. User Experience

### 2.1 Navigation Model: Lazygit-Style Sidebar

```
┌─────────────────────────────────────────────────────────────┐
│                   GBM Configuration                         │
├──────────────┬──────────────────────────────────────────────┤
│              │                                              │
│  ▸ Basics    │  Default Branch: [main]                      │
│  ▸ JIRA      │  Worktrees Dir:  [worktrees]                 │
│  ▸ File Copy │                                              │
│  ▸ Worktrees │  [Help] [Save] [Reset] [Quit]                │
│              │                                              │
└──────────────┴──────────────────────────────────────────────┘
```

**Sidebar Features**:
- Expandable sections (▸ = collapsed, ▾ = expanded)
- Keyboard navigation: `↑↓` to move, `←→` to collapse/expand, `enter` to select subsection
- Visual indicator for current section (highlight/color)
- Badge support for incomplete/invalid sections (e.g., "⚠ JIRA" if required fields missing)

**Content Area**:
- Shows form fields for selected section
- Scrollable if content exceeds viewport
- Footer with contextual actions: `[Help] [Save] [Reset] [Quit]`

---

## 2.2 Section Hierarchy

### Root Sections (in sidebar)

1. **Basics** (required)
   - `default_branch` (string, required)
   - `worktrees_dir` (string, required, with template var help)

2. **JIRA** (optional)
   - Enable/Disable toggle
   - Nested subsections:
     - **Server**: host, username, api_token
     - **Filters**: status[], priority, type, labels
     - **Attachments**: enabled, max_size_mb, directory, timeout, retries, backoff
     - **Markdown**: include_comments, include_attachments, use_relative_links, filename_pattern, linked_issues depth

3. **File Copy** (optional)
   - Nested subsections:
     - **Rules**: list of (source_worktree → files[])
     - **Auto**: enabled, source_worktree, copy_ignored, copy_untracked, exclude[]

4. **Worktrees** (view/edit)
   - List of tracked worktrees
   - Nested subsections per worktree (branch, merge_into, description)

### Interaction Model per Section

#### Simple Fields (Basics)
- TextInput for each field
- Validate on save, show inline errors
- Help text on demand (`?` key)

#### Toggles (JIRA enabled/disabled)
- Confirm toggle, collapse subsections if disabling
- Gray out child fields when parent disabled

#### Lists (File Copy Rules, Worktrees)
- Table-style view with rows
- Actions: `a` (add), `e` (edit selected), `d` (delete), Enter (expand details)
- Modal or inline form for add/edit

#### Nested Forms (JIRA Server, Attachments)
- Rendered as sections within parent
- Breadcrumb or back button for navigation

---

## 2.3 Example User Journey: Configure JIRA

1. User runs `gbm config`
2. TUI opens with Basics section visible
3. User presses `↓` to focus JIRA, presses `→` to expand
4. JIRA subsections appear: Server, Filters, Attachments, Markdown
5. User selects "Server", edits form:
   - `host`: `https://jira.company.com`
   - `username`: `user@company.com`
   - `api_token`: Pastes token (masked display)
6. User tabs through other subsections, adjusting settings
7. User presses `s` (save) or focuses `[Save]` button
8. TUI validates all sections, shows summary of changes
9. On confirm, writes to `.gbm/config.yaml` and exits
10. User sees success message in stderr

---

## 3. Architecture

### 3.1 Code Organization

```
pkg/tui/
├── config/                          # NEW: Config TUI module
│   ├── model.go                     # Root config TUI model (manages sidebar + sections)
│   ├── sidebar.go                   # Sidebar nav component (NEW)
│   ├── section_builder.go           # Helper to compose sections with Wizard + Fields
│   ├── filecopy_rules.go            # File copy rules + filepicker integration (NEW)
│   └── stories/                     # Storybook stories (all at top level)
│       ├── components_storybook.go  # Stories: TextInput, Selector, Confirm variants
│       ├── sections_storybook.go    # Stories: Basics, JIRA Server, FileCopy Rules, etc.
│       ├── pages_storybook.go       # Stories: Full ConfigTUI layouts, dark/light modes
│       └── mocks.go                 # Helper data for storybook (test configs, options)
│
├── (REUSED existing components)
│   ├── fields/textinput.go          # ✓ Text input (branch name, URLs, tokens)
│   ├── fields/selector.go           # ✓ Dropdown (select worktree, filter statuses)
│   ├── fields/confirm.go            # ✓ Yes/No toggle (enable JIRA)
│   ├── wizard.go                    # ✓ Multi-step form orchestration
│   ├── navigator.go                 # ✓ Stack-based screen navigation
│   ├── table.go                     # ✓ Table with async rendering (worktrees list)
│   ├── theme.go                     # ✓ Consistent styling (reuse Focused/Blurred styles)
│   ├── context.go                   # ✓ Shared theme/width/height context
│   └── field.go                     # ✓ Field interface (all fields implement this)
│
└── (external reused)
    ├── charmbracelet/bubbles/filepicker  # ✓ File picker (file copy rule selection)
    ├── charmbracelet/bubbles/textinput   # ✓ Text input (underlying TextInput field)
    ├── charmbracelet/bubbles/table       # ✓ Table component (underlying table.go)
    └── charmbracelet/lipgloss            # ✓ Styling (colors, borders, layouts)

cmd/service/
├── config.go                        # Existing config validation (REUSE validateConfig)
├── config_tui.go                    # NEW: TUI command handler + orchestration
└── root.go                          # Register "gbm config" command
```

### 3.1.1 Component Reuse Matrix

| Component | Reused? | Purpose | Notes |
|-----------|---------|---------|-------|
| `fields.TextInput` | ✓ Yes | Text fields (branch, URLs, tokens) | Extends bubbles/textinput, has validation |
| `fields.Selector` | ✓ Yes | Dropdown lists (statuses, priorities) | Cursor navigation, option selection |
| `fields.Confirm` | ✓ Yes | Toggle enable/disable (JIRA) | Yes/No with v/n shortcuts |
| `Wizard` | ✓ Yes | Multi-step forms per section | Skip logic, forward/back navigation |
| `Navigator` | ✓ Yes | Screen stack (sidebar → section → modal) | Push/pop for nested navigation |
| `Table` | ✓ Yes | Worktrees list, file copy rules preview | Async cell support, filter mode |
| `Theme` | ✓ Yes | Consistent colors & styles | Focused/Blurred FieldStyles |
| `Context` | ✓ Yes | Shared state (width, height, theme) | Passed to fields for responsiveness |
| `filepicker` (bubbles) | ✓ Yes | File/dir selection for rules | Used for rule "files[]" selection |
| `validateConfig()` (service) | ✓ Yes | Config validation | Run on save to block invalid changes |

### 3.2 Component Hierarchy

```
RootModel (tea.Model)
├── Sidebar (navigation state, selected section)
├── SectionStack (stack of active section models)
│   └── [Section interfaces]
│       ├── Basics (Form)
│       ├── JIRA (nested container)
│       │   ├── Server (Form)
│       │   ├── Filters (Form)
│       │   ├── Attachments (Form)
│       │   └── Markdown (Form)
│       ├── FileCopy (nested container)
│       │   ├── Rules (List + Form modal)
│       │   └── Auto (Form)
│       └── Worktrees (List + Form modal)
└── Footer (actions: Save, Reset, Quit)
```

### 3.3 Data Flow

```
Input (keypress, UI interaction)
  ↓
RootModel.Update(msg)
  ↓
Current Section.Update(msg)
  ↓
State change (modify local config copy)
  ↓
Mark "dirty" (unsaved changes)
  ↓
Sidebar badge updates
  ↓
View() re-renders
```

**Config Persistence**:
- Load `.gbm/config.yaml` on startup
- Keep in-memory copy during editing
- On save:
  1. Validate all sections via `validateConfig()`
  2. If valid, write to file and exit
  3. If invalid, show error overlay, return to editing
- On quit without save: confirm discard changes

---

## 4. Detailed Component Specs

### 4.1 Sidebar

**Model**:
```go
type Sidebar struct {
    sections    []SidebarSection
    focused     int            // index of focused section
    expanded    map[string]bool // section → expanded state
}

type SidebarSection struct {
    key      string          // "basics", "jira", etc.
    label    string
    icon     string          // "▸" or "▾"
    children []SidebarSection
    invalid  bool            // true if validation failed
}
```

**Interactions**:
- `↑↓`: Move focus up/down
- `←`: Collapse focused section
- `→`: Expand focused section
- `Enter`: Select section, switch content area
- Mouse click: Select section

**Display**:
- Highlight focused section (color or bg)
- Show "⚠" badge if section has validation errors
- Breadcrumb path at top for nested sections

### 4.2 Form Component (Basics, JIRA Server, etc.)

**Reuse**: Build forms using `Wizard` + `Field` interface (existing components)

Forms are orchestrated via the `Wizard` component, which already supports:
- Multi-step progression (field → field)
- Forward/back navigation (next, prev, skip)
- State management via `Context` & `WorkflowState`
- Validation on completion

**Section Forms** are arrays of `Step`, each with a `Field`:
```go
type Step struct {
    Name  string
    Field Field  // TextInput, Selector, Confirm, or custom
    Skip  func(*WorkflowState) bool
}
```

**Field Types** (all implement `Field` interface):
- `fields.TextInput` (extends bubbles/textinput, has ValidatorFunc)
- `fields.Selector` (cursor-nav dropdown, options[])
- `fields.Confirm` (Yes/No toggle with v/n shortcuts)
- Custom fields for composite inputs (e.g., filepicker wrapper)

**Interactions** (inherited from Wizard + underlying bubbles):
- `Tab` / `Shift+Tab`: Next/prev step
- `↑↓`: Navigate options (in Selector)
- Type: Edit text (in TextInput)
- `Enter`: Confirm field → advance step
- `Esc`: Go back (if not first step) or cancel
- `Ctrl+S`: Save (root model level)

**Validation** (via Field.WithValidator):
- `TextInput.WithValidator(func(s string) error)` runs on Enter
- Show inline error if validation fails
- On root model Save: run `validateConfig()` on entire config object

### 4.3 List Component (Worktrees)

**Reuse**: Use existing `Table` component from `pkg/tui/table.go`

The `Table` component provides:
- `table.Model` from bubbles (cursor navigation, column headers)
- Filter mode (`/` to search)
- Async cell rendering (spinners for loading)
- Theme application (focused/blurred row styles)
- Scrolling with focus tracking

**Worktrees List Model**:
```go
type WorktreesList struct {
    table        *tui.Table
    rows         []table.Row      // worktree name, branch, merge_into
    columns      []table.Column
    selectedIdx  int              // current row index
    modal        tea.Model        // for add/edit (Wizard with WorktreeConfig steps)
}
```

**Interactions** (inherited from Table):
- `↑↓`: Move focus up/down rows
- `/`: Filter rows by name/branch
- `a`: Open modal to add new worktree
- `e`: Edit focused item (open modal with Wizard)
- `d`: Delete focused item (confirm with fields.Confirm)
- `Esc`: Close modal / cancel
- `q`: Quit to sidebar

**Display**:
- Table with columns: Name | Branch | Merge Into
- Current row highlighted (from Theme.Table.Selected style)
- Scrollbar if >height rows
- Footer: `a:add  e:edit  d:delete  /filter  ?:help`

### 4.3.1 File Copy Rules (Using Bubbletea filepicker)

For FileCopy Rules, leverage the battle-tested `charmbracelet/bubbles` **file picker** component:

**Why filepicker**:
- Users browse filesystem to select source files/directories
- Familiar UX (same as `fd`, `find`, terminal tools)
- Handles symlinks, permissions, nested traversal
- Proven in production (used in many Bubble Tea apps)

**Model**:
```go
type FileCopyRuleList struct {
    rules       []FileCopyRule
    focused     int
    modal       *filepicker.Model  // for selecting source files
    currentRule *FileCopyRule      // being edited
}
```

**Interactions**:
- `↑↓`: Move focus up/down rules
- `a`: Open filepicker to add new rule (select source files)
- `e`: Edit rule (show form: source_worktree, files[], validation)
- `d`: Delete rule (confirm)
- `Esc`: Close filepicker / cancel

**Display**:
- Table: source_worktree | files[] (preview, e.g., ".env, config/")
- Filepicker modal when adding/editing
- Footer: `a:add  e:edit  d:delete  q:quit  ?:help`

**Integration**:
- Filepicker starts in repo root (configurable)
- Users select files/dirs, press Enter to confirm
- Selected paths populate `files[]` in rule

### 4.4 Toggle Component (Enable/Disable JIRA)

**Reuse**: Use existing `fields.Confirm` component

The `Confirm` component provides:
- Yes/No selection (with left/right or h/l keys)
- Shortcut keys (y for yes, n for no)
- Inline rendering with optional summary
- Theme support (Focused/Blurred styles)

**Usage**:
```go
enableJira := fields.NewConfirm("jira_enabled", "Enable JIRA Integration?")
enableJira.WithSummary("Allows creating worktrees from JIRA issues")
// Add as a Step in JIRA section Wizard
```

**Interactions** (inherited from Confirm):
- `←→` or `h/l`: Move focus between Yes/No
- `y` / `n`: Immediate submit (shortcut)
- `Space`: Toggle current selection
- `Enter`: Confirm selection → advance step

**Display**:
- `[✓] Enable JIRA Integration   [ ] Disable`
- Styled with Theme.Focused colors when focused
- Summary text (optional) above Yes/No

### 4.5 Root Config Model

**Reuse**: Use `Navigator` to manage sidebar + section stack

The `Navigator` component provides:
- Stack-based screen management (push/pop)
- Delegation to current model (Init/Update/View)
- Message routing (NavigateMsg for screen transitions)

**RootModel**:
```go
type ConfigModel struct {
    nav            *tui.Navigator     // stack: [Sidebar, currentSection, modal?]
    sidebar        *Sidebar           // Always at bottom of stack
    config         *service.Config    // In-memory working copy
    original       *service.Config    // For reset/discard
    dirty          bool               // Unsaved changes
}

func (m *ConfigModel) Init() tea.Cmd {
    return m.nav.Init()
}

func (m *ConfigModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle global keys (Ctrl+S, Ctrl+C)
    // Delegate to Navigator
    return m.nav.Update(msg)
}

func (m *ConfigModel) View() string {
    return m.nav.View()
}
```

**Navigation Flow**:
1. Sidebar at bottom of stack (always accessible via Esc)
2. User selects section → push SectionModel (Wizard + fields)
3. User presses Save → validate + write + quit
4. User presses Esc → pop back to Sidebar
5. User presses q → quit without save (confirm with Confirm field)

---

## 5. Storybook-Go Coverage

### 5.1 Story Structure

Each component should have storybook-go stories covering:

#### **Individual Components** (pkg/tui/config/stories/components.go)
- `TextInput` (empty, filled, error, focused, placeholder)
- `Toggle` (on, off, focused)
- `Section Header` (collapsed, expanded, invalid)
- `FormField` (text, password, number, select)

#### **Complete Sections** (pkg/tui/config/stories/sections.go)
- `BasicsForm` (empty, filled, validation errors)
- `JiraServerForm` (with masked token, empty required fields)
- `FileCopyRulesList` (empty, 3 rules, focused rule, filepicker modal)
- `FilePickerModal` (empty directory, nested dirs, file selection)
- `WorktreesList` (empty, 5 worktrees, one with invalid config)
- `AutoFileCopyForm` (enabled, disabled, with exclusions)

#### **Full Page Layouts** (pkg/tui/config/stories/pages.go)
- `ConfigTUIBasicsSection` (sidebar with Basics focused)
- `ConfigTUIJiraExpanded` (sidebar with JIRA expanded, Server focused)
- `ConfigTUIFileCopyRules` (sidebar with FileCopy focused, Rules list)
- `ConfigTUIWithValidationErrors` (multiple sections with badges)
- `ConfigTUIModalOpen` (modal for adding rule overlaid)

#### **Dark/Light Modes**
- Each story should render in both light and dark theme
- Storybook wrapper controls theme via context

### 5.2 Story Template Example

```go
// components.go
func TextInputEmpty() storybook.Story {
    return storybook.Story{
        Name: "TextInput / Empty",
        Component: NewTextInput("default_branch", "", true),
    }
}

func TextInputWithError() storybook.Story {
    input := NewTextInput("default_branch", "inv@lid", true)
    input.SetError("Branch name cannot contain @")
    return storybook.Story{
        Name: "TextInput / With Error",
        Component: input,
    }
}

// pages.go with theme toggle
func ConfigPageBasicsLight() storybook.Story {
    ctx := &tui.Context{Theme: tui.DefaultTheme()}
    model := NewConfigModel(ctx, &testConfig)
    model.focusedSection = "basics"
    return storybook.Story{
        Name: "ConfigTUI / Basics (Light)",
        Component: model,
    }
}

func ConfigPageBasicsDark() storybook.Story {
    ctx := &tui.Context{Theme: tui.DarkTheme()} // hypothetical
    model := NewConfigModel(ctx, &testConfig)
    model.focusedSection = "basics"
    return storybook.Story{
        Name: "ConfigTUI / Basics (Dark)",
        Component: model,
    }
}
```

---

## 6. Data Models

### 6.1 Config Model (in-memory)

```go
type ConfigModel struct {
    config          *service.Config  // loaded from .gbm/config.yaml
    original        *service.Config  // backup for reset/discard
    dirty           bool             // has unsaved changes
    sidebar         *Sidebar
    sections        map[string]Section
    currentSection  string
    validationErrors map[string][]string
}
```

### 6.2 Validation

Reuse existing [validateConfig()](file:///Users/jschneider/code/scratch/gbm/worktrees/manage_config/cmd/service/config.go#L16) from cmd/service:

```go
func (m *ConfigModel) Validate() error {
    return validateConfig(m.config)
}

func (m *ConfigModel) ValidateSection(sectionKey string) error {
    // Validate single section fields
    // E.g., ValidateSection("jira.server") validates JIRA host, username, token
}
```

---

## 7. Command Integration

### 7.1 CLI Command

```bash
gbm config [options]
```

**Options**:
- `--reset`: Reset config to defaults (confirm)
- `--validate`: Validate config and exit (no TUI)
- `--format [yaml|json]`: Export config to stdout

**Implementation** (cmd/service/config_tui.go):
```go
func newConfigCommand(svc *Service) *cobra.Command {
    cmd := &cobra.Command{
        Use:   "config",
        Short: "Manage configuration interactively",
        Long:  "Open an interactive TUI to edit .gbm/config.yaml",
        Args:  cobra.NoArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Load config
            config := svc.GetConfig()

            // Launch TUI
            model := pkg.tui.config.NewConfigModel(config, svc)
            p := tea.NewProgram(model)
            finalModel, err := p.Run()

            if err != nil {
                return fmt.Errorf("config TUI error: %w", err)
            }

            // Save changes if confirmed
            if saved := finalModel.(*ConfigModel).Saved; saved {
                return svc.SaveConfig(finalModel.(*ConfigModel).Config())
            }
            return nil
        },
    }
    return cmd
}
```

---

## 8. Implementation Phases

### Phase 1: Core Components & Sidebar (Week 1)
- [ ] Sidebar navigation model + view
- [ ] Form component (text inputs, validation)
- [ ] TextInput field component
- [ ] Theme + styling (match existing aesthetic)
- [ ] Basic storybook stories for components

### Phase 2: Sections (Week 2)
- [ ] Basics section (default_branch, worktrees_dir)
- [ ] JIRA section (Server form, toggle)
- [ ] JIRA subsections (Filters, Attachments, Markdown)
- [ ] Storybook stories for each section

### Phase 3: Lists & File Copy (Week 3)
- [ ] List component (table view for Worktrees)
- [ ] FileCopy Rules section (integrate filepicker for file selection)
- [ ] Filepicker wrapper (adapt charmbracelet/bubbles filepicker for config)
- [ ] FileCopy Auto section (form with exclusions list)
- [ ] Modal for add/edit flows
- [ ] Storybook stories for lists, filepicker, and modals

### Phase 4: Worktrees & Polish (Week 4)
- [ ] Worktrees section (view/edit)
- [ ] Root model integration (all sections)
- [ ] Save/discard/reset flows
- [ ] Validation error handling
- [ ] Full page storybook stories

### Phase 5: Testing & Docs (Week 5)
- [ ] Unit tests for models
- [ ] E2E test (load config, edit, save, verify file)
- [ ] Storybook dark/light theme coverage
- [ ] Help text & error messages
- [ ] CLI integration test

---

## 9. Testing Strategy

### 9.1 Storybook-Go

**Purpose**: Visual validation, component isolation, regression testing

**Coverage**:
- All components in all states (empty, filled, error, loading, disabled)
- All sections in isolation
- Full page layouts with different content sizes
- Dark/light themes
- Responsive widths (80col, 120col, 240col terminals)

**Run Stories**:
```bash
go test ./pkg/tui/config/stories -v
go test ./pkg/tui/config -v -run TestStories
```

### 9.2 Unit Tests

**Test Files**:
- `pkg/tui/config/model_test.go`: ConfigModel state transitions
- `pkg/tui/config/form_test.go`: Form validation, field focus
- `pkg/tui/config/list_test.go`: List navigation, add/delete
- `pkg/tui/config/sidebar_test.go`: Sidebar expand/collapse, focus

**Example**:
```go
func TestFormValidation(t *testing.T) {
    form := NewForm([]FormField{
        {key: "host", required: true, validator: validateURL},
    })
    form.SetValue("host", "invalid-url")

    err := form.Validate()
    assert.Error(t, err)
    assert.Equal(t, "invalid URL format", form.GetError("host"))
}
```

### 9.3 E2E Tests

**File**: `cmd/service/config_test.go`

**Scenarios**:
1. Load config, edit Basics, save
2. Enable JIRA, fill Server section, save
3. Discard changes without save
4. Validation error (invalid URL), correct, save
5. Add file copy rule, delete rule, save

---

## 10. Success Criteria

- [ ] `gbm config` launches without errors
- [ ] Users can navigate all sections via sidebar
- [ ] Users can edit all config fields
- [ ] Validation prevents invalid saves
- [ ] Changes persist to `.gbm/config.yaml`
- [ ] Storybook covers 80%+ of components/sections
- [ ] Dark/light theme both work
- [ ] Help text available for all fields
- [ ] Keyboard + mouse navigation work smoothly
- [ ] Exit (quit/save) operations confirmed
- [ ] No breaking changes to existing commands

---

## 11. Out of Scope (v2+)

- [ ] `gbm config validate` CLI command
- [ ] `gbm config export [yaml|json]` command
- [ ] `gbm config diff` (show changes before save)
- [ ] `gbm config reset` (restore to defaults)
- [ ] Config file diffs / version history
- [ ] Search/filter sections
- [ ] Config templates / quick-start wizard

---

## 12. Appendix: Design Tokens (Colors, Spacing)

Reuse from existing theme:

```go
// pkg/tui/theme.go (extend)
Focused: FieldStyles{
    Title:       lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86")).Underline(true),
    Input:       lipgloss.NewStyle().Foreground(lipgloss.Color("255")).Background(lipgloss.Color("238")).Bold(true),
    Error:       lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true),
}
Blurred: FieldStyles{
    // muted versions
}

// Sidebar
SidebarFocused  = color(86)  // cyan
SidebarBlurred  = color(240) // gray
SidebarBadge    = color(196) // red

// Table/List
RowSelected = color(229) on color(57) // yellow on purple
RowDefault  = default
```

---

## Appendix: Keyboard Cheatsheet (Help Screen)

```
Navigation
  ↑↓         Move focus
  ←→         Collapse/expand section
  Tab        Move to next field
  q          Quit

Editing
  a          Add item (in lists)
  e          Edit selected item
  d          Delete item
  Space      Toggle option

Actions
  s/Ctrl+S   Save changes
  Ctrl+C     Discard & quit
  ?          Show help
```
