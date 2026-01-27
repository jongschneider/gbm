# GBM Config TUI — Product Requirements Document

**Status**: Ready for Implementation  
**Version**: 1.0  
**Date**: January 2026  
**RFC Target**: Phase 1 Approval

---

## Problem Statement

Users currently manage `.gbm/config.yaml` by editing YAML directly, creating friction for:
- **Non-technical users**: Cannot configure GBM without manual YAML editing
- **Configuration discovery**: No guided experience; users must know all available options
- **Error prevention**: Invalid configurations are caught at runtime, not at point of entry
- **Discoverability**: No built-in help or context for what each setting does

**Cost of inaction**: Users abandon GBM configuration or introduce bugs via malformed YAML.

## Proposed Solution

Add an interactive, fullscreen **TUI (Terminal User Interface)** command (`gbm config`) that provides:
- **Lazygit-style sidebar navigation** for section hierarchy (Basics, JIRA, FileCopy, Worktrees)
- **Guided form fields** with real-time validation
- **Visual feedback** on invalid settings before save
- **Consistent UX** with existing GBM commands (`gbm wt ls`, `gbm wt add`)

### Overview

```
┌─────────────────────────────────────────────────────────────┐
│                   GBM Configuration                         │
├──────────────┬──────────────────────────────────────────────┤
│              │                                              │
│  ▸ Basics    │  Default Branch: [main____________]          │
│  ▾ JIRA      │  Worktrees Dir:  [./worktrees____]           │
│    ▸ Server  │                                              │
│    ▸ Filters │  [Help]  [Save]  [Reset]  [Quit]             │
│  ▸ FileCopy  │                                              │
│  ▸ Worktrees │                                              │
│              │                                              │
└──────────────┴──────────────────────────────────────────────┘
```

### Key Features

1. **Navigation**
   - Expandable sidebar (↑↓ to move, ←→ to collapse/expand, Enter to select)
   - Visual indicators (focused section, validation badges)
   - Breadcrumb support for nested sections

2. **Configuration Sections**
   - **Basics**: required fields (default_branch, worktrees_dir)
   - **JIRA**: optional toggle with nested subsections (Server, Filters, Attachments, Markdown)
   - **FileCopy**: Rules list (with filepicker) + Auto settings
   - **Worktrees**: Tracked worktrees with add/edit/delete

3. **Form Fields**
   - Text inputs with optional validators (URLs, branch names, etc.)
   - Dropdowns (Selector) for predefined options
   - Toggles (Confirm) for yes/no decisions
   - Table views for lists (worktrees, file copy rules)
   - File picker for selecting files/directories

4. **Validation & Safety**
   - Real-time field validation with inline errors
   - Block save if validation fails
   - Confirmation before discard/quit
   - Show validation summary before persisting

5. **Accessibility**
   - Dark/light theme support
   - Keyboard-only navigation (no mouse required)
   - Help screen with keyboard cheatsheet (? key)
   - Responsive terminal widths (80–240 columns)

### UX Flows

#### User Journey: Configure JIRA

1. User runs `gbm config`
2. TUI opens with Basics section visible
3. User navigates to JIRA section (↓ key)
4. User expands JIRA (→ key) → sees subsections (Server, Filters, etc.)
5. User selects Server subsection
6. Form fields appear:
   - `host`: Text input (validates HTTPS URL)
   - `username`: Text input
   - `api_token`: Masked text input
7. User fills fields, tabs to next
8. User adjusts other subsections (Filters, Attachments, Markdown)
9. User presses `s` or focuses `[Save]` button
10. TUI validates all sections via `validateConfig()`
11. If valid: writes to `.gbm/config.yaml` and exits
12. If invalid: shows error overlay, returns to editing

#### User Journey: Edit Worktrees

1. User navigates to Worktrees section
2. Table shows current worktrees (name, branch, merge_into)
3. User presses `a` to add worktree → modal opens
4. Modal shows form:
   - `branch`: Text input
   - `merge_into`: Dropdown
   - `description`: Text input
5. User fills form, presses Enter
6. New worktree added to table
7. User presses `d` to delete → confirmation modal
8. User presses `s` to save all changes

### Design Considerations

- **Color palette**: Reuse existing theme (cyan/blue focus, gray blur)
- **Component consistency**: All fields use same styling (TextInput, Selector, Confirm, Table)
- **Focus indicators**: Bold/color for focused fields, muted for blurred
- **Error display**: Red badges + inline messages
- **Modals**: Center-aligned, with borders, escape-to-cancel
- **Keyboard feedback**: Visual response to key presses (cursor movement, field updates)

---

## End State

- [ ] `gbm config` command exists and launches without errors
- [ ] Sidebar navigation works (↑↓←→ keys, Enter selection)
- [ ] All config sections navigable (Basics, JIRA, FileCopy, Worktrees)
- [ ] All config fields editable via form inputs
- [ ] Validation prevents invalid saves (blocks, shows error overlay)
- [ ] Changes persist to `.gbm/config.yaml` on save
- [ ] Storybook covers >80% of components/sections/pages
- [ ] Dark + light themes both functional
- [ ] Unit tests >80% coverage, E2E happy paths pass
- [ ] No breaking changes to existing commands
- [ ] Help text accessible via `?` key
- [ ] Code reviewed and merged

---

## Success Metrics

### Quantitative

- **Storybook coverage**: >80% of components/sections in >3 states
- **Test coverage**: Unit tests >80%, E2E happy paths 100%
- **Performance**: TUI launches in <500ms, keypress response <50ms
- **Code reuse**: 57% of code from existing components (870–1,290 LOC saved)

### Qualitative

- Users report config TUI easier to use than manual YAML editing
- No increase in support requests for config issues
- Developers satisfied with code consistency (same components as `gbm wt` commands)

---

## Acceptance Criteria

### Phase 1: Scaffolding (Week 1)
- [ ] Sidebar component renders correctly (collapsed, expanded, badges)
- [ ] RootModel (ConfigModel) delegates to Navigator correctly
- [ ] Basic storybook stories run without errors
- [ ] `gbm config` command registered and launches TUI

### Phase 2: Basics Section (Week 2)
- [ ] Basics section with default_branch + worktrees_dir fields
- [ ] TextInput validators work (branch names, directory paths)
- [ ] Form validation prevents invalid saves
- [ ] Save/discard flows work correctly

### Phase 3: JIRA Section (Week 2.5)
- [ ] JIRA enable/disable toggle works
- [ ] JIRA subsections (Server, Filters, Attachments, Markdown) render
- [ ] Subsections hidden/shown based on parent toggle state
- [ ] All JIRA fields editable and validatable

### Phase 4: File Copy & Worktrees (Week 3.5)
- [ ] FileCopy Rules table renders with filepicker modal
- [ ] Filepicker navigation works (↑↓, Enter to select, Escape to cancel)
- [ ] Worktrees table with add/edit/delete modals
- [ ] All list operations persist to config

### Phase 5: Integration & Polish (Week 4)
- [ ] All sections integrated in one ConfigModel
- [ ] Save/discard/reset flows work end-to-end
- [ ] Validation error overlays appear on invalid save attempts
- [ ] Dark/light theme both render correctly
- [ ] Help screen shows keyboard shortcuts

### Phase 6: Testing & Merge (Week 5)
- [ ] Unit tests pass (>80% coverage)
- [ ] E2E test: load config → edit → save → verify file
- [ ] Storybook stories pass (>80% coverage)
- [ ] Code review approved
- [ ] Merged to main branch

---

## Technical Context

### Existing Patterns

The GBM project has a well-established TUI framework (`pkg/tui/`) used in `gbm wt ls` and `gbm wt add` commands:

- **Navigator**: Stack-based screen management (push/pop models)
- **Wizard**: Multi-step form orchestration with Field interface
- **Field interface**: All inputs implement same lifecycle (Init/Update/View/Focus/Blur)
- **Theme**: Consistent color palette + styling (Focused/Blurred variants)
- **Table**: Data grid with cursor navigation, filter mode, async rendering

This PRD reuses 10+ production-proven components, reducing new code from ~1,500–2,000 LOC to ~650–950 LOC.

### Key Files

**Reused components** (no changes needed):
- [`pkg/tui/field.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/field.go) — Field interface
- [`pkg/tui/wizard.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/wizard.go) — Multi-step form
- [`pkg/tui/navigator.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/navigator.go) — Screen stack
- [`pkg/tui/fields/textinput.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/fields/textinput.go) — Text input field
- [`pkg/tui/fields/selector.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/fields/selector.go) — Dropdown field
- [`pkg/tui/fields/confirm.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/fields/confirm.go) — Toggle field
- [`pkg/tui/table.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/table.go) — Table component
- [`pkg/tui/theme.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/theme.go) — Styling
- [`pkg/tui/context.go`](file:///Users/jschneider/code/scratch/gbm/pkg/tui/context.go) — Shared state
- [`cmd/service/config.go`](file:///Users/jschneider/code/scratch/gbm/cmd/service/config.go) — `validateConfig()` function

**New files** (to be created):
- `pkg/tui/config/model.go` — RootModel (ConfigModel)
- `pkg/tui/config/sidebar.go` — Sidebar navigation
- `pkg/tui/config/section_builder.go` — Step composition helpers
- `pkg/tui/config/filecopy_rules.go` — FileCopy + filepicker integration
- `pkg/tui/config/stories/*.go` — Storybook stories (components, sections, pages)
- `cmd/service/config_tui.go` — TUI command handler
- `cmd/service/config_test.go` — E2E tests (new file or modifications)

**Modified files**:
- `cmd/service/root.go` — Register `gbm config` command

### Dependencies

**Go packages** (already in go.mod):
- `github.com/charmbracelet/bubbletea` — TUI framework
- `github.com/charmbracelet/bubbles` — Filepicker, table, textinput
- `github.com/charmbracelet/lipgloss` — Terminal styling
- `github.com/stretchr/testify` — Test assertions
- `golang.org/x/sync/errgroup` — Goroutine management

**No new external dependencies required** — all libraries already in use.

### Data Model Changes

**No schema changes** — working with existing config.yaml structure:

```yaml
default_branch: main
worktrees_dir: ./worktrees

jira:
  enabled: false
  server:
    host: https://jira.example.com
    username: user@example.com
    api_token: secret
  filters:
    status: ["In Dev", "Open"]
    priority: High
  attachments: { ... }
  markdown: { ... }

filecopy:
  auto: { ... }
  rules:
    - source_worktree: main
      files: [README.md, go.mod]

worktrees:
  - name: feature-x
    branch: feature/x
    merge_into: main
```

ConfigModel in-memory:
```go
type ConfigModel struct {
    config            *service.Config  // current (editable)
    original          *service.Config  // backup for reset
    dirty             bool             // has unsaved changes
    sidebar           *Sidebar
    sections          map[string]Section
    currentSection    string
    validationErrors  map[string][]string
}
```

---

## Risks & Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|-----------|
| Filepicker in modal doesn't handle Escape correctly | Medium | High | Write storybook + E2E test for modal open/close early (Phase 4 start) |
| Validation logic needs updates for new config fields | Low | Medium | Dry-run validation in storybook before integration |
| Keyboard conflicts between sidebar keys (↑↓←→) and field keys | Low | Medium | Sidebar only consumes keys when focused (not in field) |
| Table slow with large worktree lists (100+) | Low | Low | Table already has filter mode (/), use for large lists |
| JIRA multi-select filters too complex | Medium | Medium | Ship v1 without multi-select; use single Selector fields (v2 upgrade) |
| Theme inconsistency across new components | Low | Low | All components inherit from existing Theme, Context |

---

## Alternatives Considered

### Alternative 1: Web-Based Config UI
**Approach**: Serve config editor via HTTP server  
**Why rejected**: Out-of-scope for a CLI tool; adds deployment complexity; users expect CLI workflows

### Alternative 2: Wizard-Style Linear Form
**Approach**: Single multi-step form (no sidebar), linear progression  
**Why rejected**: Doesn't scale to 4 major sections + nested subsections; harder to navigate large config; users can't jump between sections

### Alternative 3: Custom Component Library
**Approach**: Build custom TextInput, Selector, Table from scratch  
**Why rejected**: 870–1,290 LOC already exists and battle-tested; risk of bugs and inconsistency; slow to ship

---

## Non-Goals

This PRD does **NOT** cover:

- [ ] `gbm config validate` CLI command (separate from TUI)
- [ ] `gbm config export [yaml|json]` command
- [ ] Config file diffs / version history
- [ ] Search/filter sections
- [ ] Config templates / quick-start wizard
- [ ] Multi-user config synchronization
- [ ] Config encryption or secrets management
- [ ] Syntax highlighting or advanced YAML features
- [ ] Plugin/extension system for custom sections
- [ ] Auto-formatting of YAML on save

These are deferred to v2+ or separate RFCs.

---

## Interface Specifications

### CLI Command

```bash
gbm config [options]
```

**Options** (Phase 1: none; Phase 2+ optional):
- `--reset`: Reset config to defaults (with confirmation)
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
            config := svc.GetConfig()
            model := pkg.tui.config.NewConfigModel(config, svc)
            p := tea.NewProgram(model)
            finalModel, err := p.Run()
            if err != nil {
                return fmt.Errorf("config TUI error: %w", err)
            }
            if saved := finalModel.(*ConfigModel).Saved; saved {
                return svc.SaveConfig(finalModel.(*ConfigModel).Config())
            }
            return nil
        },
    }
    return cmd
}
```

### TUI Keyboard Shortcuts

```
Navigation
  ↑↓         Move focus in sidebar / list
  ←→         Collapse/expand section
  Tab        Move to next field (in form)
  Shift+Tab  Move to previous field
  q          Quit
  Ctrl+C     Discard & quit (with confirmation)

Editing
  a          Add item (in lists)
  e          Edit selected item
  d          Delete item
  Space      Toggle option (in Selector)
  /          Filter/search (in Table)

Actions
  s          Save changes
  Ctrl+S     Save (alternative)
  ?          Show help
  Enter      Select / confirm

Form Fields
  ↑↓         Navigate options (in Selector)
  y/n        Quick yes/no (in Confirm)
  Escape     Back / cancel
```

---

## Documentation Requirements

### User Documentation
- [ ] `gbm config` command in CLI help (`gbm config --help`)
- [ ] In-app help screen (? key) with keyboard shortcuts
- [ ] README section on config TUI usage
- [ ] Troubleshooting guide for validation errors

### Developer Documentation
- [ ] Code comments on ConfigModel, Sidebar, custom fields
- [ ] Storybook story descriptions (purpose of each story)
- [ ] Architecture diagram (Navigator → Sidebar → Sections → Fields)
- [ ] Testing guide (how to run storybook, unit tests, E2E tests)
- [ ] Contribution guide for adding new sections

---

## Open Questions

| Question | Owner | Status | Resolution |
|----------|-------|--------|-----------|
| Should JIRA filters support multi-select or single-select in v1? | Product | Resolved | Single-select for v1; upgrade to multi-select in v2 |
| Should users edit worktree branch/merge_into via config TUI, or view-only? | Product | Resolved | View + edit; changes saved to config.yaml |
| Should save show a confirmation of changes before committing? | Product | Resolved | Show validation summary if errors; otherwise auto-save on confirm |
| What's the minimum terminal width we should support? | Eng | Resolved | 80 columns (standard CLI) |
| Help system: inline (?) vs dedicated screen vs both? | Design | Resolved | Both: inline help on field focus + dedicated screen (?) |
| Should we support config import from external sources? | Product | Open | Deferred to v2 |
| Should deleted worktrees be recoverable (soft-delete)? | Product | Open | Deferred to v2 |

---

## Appendix

### Glossary

- **TUI**: Terminal User Interface (fullscreen interactive CLI)
- **Navigator**: Screen stack management (push/pop models)
- **Wizard**: Multi-step form orchestrator
- **Field**: Input component (TextInput, Selector, Confirm, etc.)
- **Sidebar**: Navigation panel showing section hierarchy
- **Storybook**: Visual testing framework for components
- **Focused**: Component with keyboard focus (highlighted)
- **Blurred**: Component without focus (muted)
- **Modal**: Overlay dialog (for add/edit/delete, error messages)

### References

- **Implementation Plan**: [`IMPLEMENTATION_PLAN.md`](./IMPLEMENTATION_PLAN.md) — Phase breakdown, file structure, dev workflow
- **Component Reuse**: [`COMPONENT_REUSE.md`](./COMPONENT_REUSE.md) — Detailed breakdown of reused components
- **Quick Start**: [`QUICK_START.md`](./QUICK_START.md) — 5-minute guide for developers
- **Config Structure**: [`config.example.yaml`](./config.example.yaml) — Example `.gbm/config.yaml`
- **Existing TUI**: [`pkg/tui/`](./pkg/tui/) — Existing components to reuse
- **Test Examples**: [`pkg/tui/wizard_test.go`](./pkg/tui/wizard_test.go) — Test patterns

---

**Status**: Ready for RFC review  
**Next Step**: Phase 1 approval and implementation kickoff

