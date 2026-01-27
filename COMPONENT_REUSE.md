# Component Reuse Strategy: `gbm config` TUI

**Goal**: Maximize reuse of battle-tested components from `pkg/tui` to minimize new code, ensure consistency, and ship faster.

## High-Level Reuse Map

```
                    ┌─────────────────────────────────┐
                    │     gbm config TUI               │
                    │   (NEW: config_tui.go)           │
                    └──────────────┬──────────────────┘
                                   │
                    ┌──────────────┴──────────────┐
                    │                             │
         ┌──────────▼─────────┐       ┌──────────▼────────┐
         │   Navigator        │       │   Sidebar (NEW)   │
         │ (REUSE: nav.go)    │       │                   │
         └──────────┬─────────┘       │ • Section nav     │
                    │                 │ • Expand/collapse │
         ┌──────────▼──────────────┐  └───────────────────┘
         │ SectionModel (Wizard)   │
         │ (REUSE: wizard.go)      │
         │ + Field steps           │
         └──────┬────────┬────┬────┘
                │        │    │
       ┌────────▼┐  ┌───▼──┐ │
       │TextInput│  │Select├─┴─ (REUSE: fields.go)
       │(REUSE)  │  │Confirm │
       └─────────┘  └────────┘
            │           │
         bubbles/    bubbles/
         textinput   (confirm pattern)
```

## Detailed Reuse Breakdown

### 1. Root Navigation: `Navigator` (REUSE pkg/tui/navigator.go)

**What we get**:
- Stack-based screen management (push/pop current model)
- Automatic delegation to top-of-stack model (Init/Update/View)
- Clean message routing (NavigateMsg for transitions)
- No custom navigation state management needed

**Our usage**:
```go
nav := tui.NewNavigator(sidebarModel)
// Later: user selects section
nav.Push(sectionWizardModel)  // navigator automatically delegates to wizard
// Later: user presses Esc
nav.Pop()  // back to sidebar
```

**Code saved**: ~100-150 lines (screen stack, focus management, delegation)

---

### 2. Form Orchestration: `Wizard` (REUSE pkg/tui/wizard.go)

**What we get**:
- Multi-step form progression (current step tracking)
- Forward/back/skip logic
- Context management (WorkflowState)
- Field lifecycle (Init, Focus, Blur, Update, View)
- Validation hooks (on field completion)
- Escape key handling (back to previous step)

**Our usage**:
```go
steps := []tui.Step{
    {
        Name: "default_branch",
        Field: fields.NewTextInput("default_branch", "Default Branch", ""),
    },
    {
        Name: "worktrees_dir",
        Field: fields.NewTextInput("worktrees_dir", "Worktrees Dir", ""),
    },
}
basicsWizard := tui.NewWizard(steps, ctx)
// Wizard handles navigation, validation, state
```

**Code saved**: ~150-200 lines (step progression, field focus management, cancellation)

---

### 3. Text Input Fields: `fields.TextInput` (REUSE pkg/tui/fields/textinput.go)

**What we get**:
- Extends bubbles/textinput with theme support
- Built-in validator function: `WithValidator(func(string) error)`
- Focus/Blur lifecycle
- Placeholder, default values
- Width/height adaptation
- Focused/Blurred styling

**Our usage**:
```go
branch := fields.NewTextInput("default_branch", "Default Branch", "The primary branch for new worktrees")
branch.WithDefault("main")
branch.WithValidator(func(s string) error {
    if !isValidBranchName(s) {
        return fmt.Errorf("invalid branch: %s", s)
    }
    return nil
})
```

**Code saved**: ~80-120 lines per field (text rendering, validation, theme application)

---

### 4. Dropdown/Selection: `fields.Selector` (REUSE pkg/tui/fields/selector.go)

**What we get**:
- Cursor-based navigation (↑↓ with wrapping)
- Option list rendering
- Selected value tracking
- Theme styling
- Focus/Blur lifecycle

**Our usage**:
```go
statusFilter := fields.NewSelector("status", "Filter by Status", []fields.Option{
    {Label: "In Dev", Value: "In Dev"},
    {Label: "Open", Value: "Open"},
    {Label: "To Do", Value: "To Do"},
})
```

**Code saved**: ~60-100 lines per dropdown (cursor management, rendering, selection)

---

### 5. Toggle/Confirm: `fields.Confirm` (REUSE pkg/tui/fields/confirm.go)

**What we get**:
- Yes/No selection with left/right arrows
- Shortcut keys (y/n for immediate selection)
- Summary text support
- Theme styling
- Complete/cancelled tracking

**Our usage**:
```go
enableJira := fields.NewConfirm("jira_enabled", "Enable JIRA Integration?")
enableJira.WithSummary("Configure JIRA server, filters, attachments, and markdown settings")
```

**Code saved**: ~40-60 lines per toggle (Yes/No rendering, key handling)

---

### 6. Table/List Rendering: `Table` (REUSE pkg/tui/table.go)

**What we get**:
- Bubbles/table integration with theme support
- Cursor navigation (↑↓) with wrapping
- Filter mode (`/` to search, fuzzy matching)
- Async cell rendering (spinners for loading)
- Column headers, scrolling, height management
- Focused/Blurred row styling

**Our usage**:
```go
cols := []tui.Column{
    {Title: "Name", Width: 15},
    {Title: "Branch", Width: 20},
    {Title: "Merge Into", Width: 15},
}
rows := []table.Row{
    {"main", "main", ""},
    {"feature-x", "feature/x", "main"},
}
t := tui.NewTable(cols, rows, ctx, theme)
```

**Code saved**: ~120-180 lines per table (cursor management, filtering, async rendering)

---

### 7. Theming & Styling: `Theme` (REUSE pkg/tui/theme.go)

**What we get**:
- Consistent color palette (Focused/Blurred variants)
- FieldStyles (Title, Description, Input, Error)
- TableStyles (Header, Selected, Cell, Border)
- Lipgloss integration
- Dark/light mode support (via Context)

**Our usage**:
```go
theme := tui.DefaultTheme()
// All fields automatically use theme.Focused / theme.Blurred
// Table uses theme.Table.Selected for highlighted rows
```

**Code saved**: ~30-50 lines (color definitions, style application)

---

### 8. Context & State: `Context` (REUSE pkg/tui/context.go)

**What we get**:
- Shared theme, width, height
- Custom field state (WorkflowState)
- Passed to all fields for responsive design

**Our usage**:
```go
ctx := tui.NewContext()
ctx.Theme = tui.DefaultTheme()
ctx.State = &tui.WorkflowState{}
// Pass to Wizard, which passes to all Fields
```

**Code saved**: ~20-30 lines (state plumbing)

---

### 9. External: Filepicker (REUSE charmbracelet/bubbles/filepicker)

**What we get**:
- Filesystem browser (directory tree navigation)
- File selection with Enter
- Escape to cancel
- Handles symlinks, permissions, hidden files
- Used in production (Glow, many other Bubble Tea apps)

**Our usage**:
```go
picker := filepicker.New()
picker.CurrentDirectory = repoRoot
// User selects files → returned as []string
selectedFiles := picker.SelectedFiles()
```

**Code saved**: ~200-300 lines (filesystem traversal, filtering, rendering)

---

### 10. Config Validation: `validateConfig()` (REUSE cmd/service/config.go)

**What we get**:
- Existing validation logic (required fields, URL format, template vars)
- Error messages with YAML field names
- Reusable on both CLI and TUI

**Our usage**:
```go
// On save in ConfigModel:
if err := validateConfig(m.config); err != nil {
    // Show error overlay, stay in TUI
    return m, cmd
}
```

**Code saved**: ~50-100 lines (validation re-implementation)

---

## Summary: Reuse Scorecard

| Component | Source | Type | Effort Saved | Risk |
|-----------|--------|------|--------------|------|
| Navigator | pkg/tui | Navigation stack | 100-150 LOC | Low (stable) |
| Wizard | pkg/tui | Form orchestration | 150-200 LOC | Low (stable) |
| TextInput | pkg/tui | Form field | 80-120 LOC × N fields | Low |
| Selector | pkg/tui | Form field | 60-100 LOC × N selects | Low |
| Confirm | pkg/tui | Form field | 40-60 LOC × N toggles | Low |
| Table | pkg/tui | Data display | 120-180 LOC | Low (stable) |
| Theme | pkg/tui | Styling | 30-50 LOC | Low (proven) |
| Context | pkg/tui | State | 20-30 LOC | Low |
| Filepicker | bubbles | File selection | 200-300 LOC | Low (production-proven) |
| validateConfig | service | Validation | 50-100 LOC | Low |
| **TOTAL** | | | **~870-1,290 LOC** | **All Low** |

## New Code Required

**Only** these modules need new implementation:

1. **Sidebar** (pkg/tui/config/sidebar.go): ~150-200 LOC
   - Section navigation, expand/collapse state
   - Breadcrumb rendering

2. **FileCopy Rules wrapper** (pkg/tui/config/filecopy_rules.go): ~100-150 LOC
   - Adapt filepicker for multi-file selection
   - Rules table + filepicker modal

3. **Section builders** (pkg/tui/config/section_builder.go): ~50-100 LOC
   - Helper to compose steps from config data
   - Reduce boilerplate in main model

4. **ConfigModel** (pkg/tui/config/model.go): ~150-200 LOC
   - Root model (delegates to Navigator)
   - Config load/save orchestration
   - Global key handling

5. **Storybook stories** (pkg/tui/config/stories/*.go): ~200-300 LOC
   - Component stories (TextInput variants, etc.)
   - Section stories (Basics form, JIRA server, etc.)
   - Page stories (full layouts, dark/light)

**Total New Code**: ~650-950 LOC

**Code Reused**: ~870-1,290 LOC (57% of estimated size!)

---

## Implementation Checklist

- [ ] Verify Wizard + Field interface can handle all our sections
- [ ] Test TextInput validator with complex rules (URLs, template vars)
- [ ] Check Selector for JIRA filters (multi-select or single?)
- [ ] Validate Table filter mode works for Worktrees
- [ ] Prototype Sidebar expand/collapse with existing theme
- [ ] Test filepicker in modal overlay (Escape handling)
- [ ] Ensure Context is passed correctly through all models
- [ ] Verify validateConfig works as-is (no modifications needed)

---

## Risk Assessment

**Low-Risk Reuses**:
- Navigator, Wizard, Table, Theme (all stable, used in production commands)
- TextInput, Selector, Confirm (all proven Field implementations)
- validateConfig (already in use everywhere)

**Potential Issues**:
- Filepicker in modal: need to ensure Escape bubbles correctly (test thoroughly)
- MultiSelect for JIRA filters: Selector is single-select, may need custom or skip multi-filter in v1
- Table filter + keyboard handling: verify no conflicts with config TUI keys

**Mitigation**:
- Write storybook stories early for visual validation
- E2E test modal open/close flow
- Test Escape key propagation in Navigator + nested models

---

## Summary

By reusing 10+ proven components, we reduce custom code from ~1,500-2,000 LOC to ~650-950 LOC.
All reused components are battle-tested and proven stable. 
The config TUI will have consistent look/feel with existing commands (wt ls, wt add).
