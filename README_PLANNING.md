# `gbm config` TUI - Planning & Design Complete ✅

## Project Overview

Add an interactive, fullscreen TUI for managing `.gbm/config.yaml` using a **lazygit-style sidebar navigation pattern**.

### Status: Ready to Implement

All planning documents complete and reviewed. Ready to move to Phase 1 implementation.

---

## Documentation Index

### 📋 Start Here
- **[QUICK_START.md](./QUICK_START.md)** ← Start here for 5-minute overview

### 📐 Detailed Specs
- **[PRD_CONFIG_TUI.md](./PRD_CONFIG_TUI.md)** - Full product specification
  - User experience (UX flows)
  - Section hierarchy
  - Component specifications
  - Storybook coverage plan
  - Command integration
  - Success criteria

### ♻️ Code Reuse Strategy
- **[COMPONENT_REUSE.md](./COMPONENT_REUSE.md)** - What we're reusing (57% code reuse!)
  - Component reuse matrix
  - Detailed breakdown of each reuse
  - Risk assessment
  - **Effort: ~870-1,290 LOC saved**

### 🛣️ Implementation Timeline
- **[IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)** - Week-by-week build plan
  - Phase 1-6 deliverables
  - File structure
  - Development workflow
  - Testing strategy
  - Milestones & success criteria

---

## Key Decisions

### ✅ Lazygit-Style Sidebar
- **Why**: Familiar navigation pattern (used in lazygit, github-cli, etc.)
- **Structure**: Expandable/collapsible sections with nested subsections
- **Benefits**: Clear visual hierarchy, mouse + keyboard support

### ✅ Reuse Existing Components (57% code reuse)
**Reused from `pkg/tui`**:
- `Navigator` - screen stack management
- `Wizard` - multi-step form orchestration
- `TextInput`, `Selector`, `Confirm` - form fields
- `Table` - data display (lists, tables)
- `Theme`, `Context` - styling & state

**External reuse**:
- `charmbracelet/bubbles/filepicker` - file selection
- `cmd/service/validateConfig()` - validation logic

### ✅ Storybook-First Development
- **Why**: Visual validation before running full CLI, isolated component testing
- **Coverage**: Components + sections + full pages + dark/light themes
- **Tools**: storybook-go (already in go.mod)

### ✅ Save on Exit (not auto-save)
- **Validation**: Run `validateConfig()` on save
- **UX**: Block save if validation fails, show error overlay
- **Safety**: Users review all changes before persisting

### ✅ Filepicker for File Copy Rules
- **Why**: Familiar filesystem browsing UX
- **Reuse**: charmbracelet/bubbles/filepicker (production-proven)
- **Benefits**: Handles symlinks, permissions, nested traversal

---

## Architecture Overview

```
ConfigTUI (RootModel)
    │
    ├─ Uses: Navigator (screen stack)
    │
    ├─ Top of stack: Sidebar (section navigation)
    │   ├─ Expandable sections: Basics, JIRA, FileCopy, Worktrees
    │   ├─ Shows current section
    │   ├─ Escape key → quit
    │   └─ Badge support (⚠ for validation errors)
    │
    └─ Sections are Wizards + Fields:
        │
        ├─ Basics (form):
        │   ├─ TextInput: default_branch
        │   └─ TextInput: worktrees_dir
        │
        ├─ JIRA (toggle + nested):
        │   ├─ Confirm: Enable JIRA?
        │   ├─ Server (if enabled):
        │   │   ├─ TextInput: host
        │   │   ├─ TextInput: username
        │   │   └─ TextInput: api_token (masked)
        │   ├─ Filters (if enabled):
        │   │   ├─ Selector: status (multi-option)
        │   │   ├─ Selector: priority
        │   │   └─ Selector: type
        │   ├─ Attachments (if enabled):
        │   │   ├─ Confirm: download attachments?
        │   │   ├─ TextInput: max_size_mb
        │   │   ├─ TextInput: directory
        │   │   ├─ TextInput: timeout_seconds
        │   │   ├─ TextInput: retry_attempts
        │   │   └─ TextInput: retry_backoff_ms
        │   └─ Markdown (if enabled):
        │       ├─ Confirm: include_comments
        │       ├─ Confirm: include_attachments
        │       └─ TextInput: filename_pattern
        │
        ├─ FileCopy (nested):
        │   ├─ Auto (form):
        │   │   ├─ Confirm: enable auto-copy?
        │   │   ├─ Selector: source_worktree
        │   │   ├─ Confirm: copy_ignored?
        │   │   ├─ Confirm: copy_untracked?
        │   │   └─ TextInput[]: exclude patterns
        │   └─ Rules (list + modal):
        │       ├─ Table: source | files[]
        │       └─ Modal (filepicker):
        │           └─ User selects files → added to rule
        │
        └─ Worktrees (list + modal):
            ├─ Table: name | branch | merge_into
            └─ Modal (Wizard):
                ├─ TextInput: branch
                ├─ Selector: merge_into
                └─ TextInput: description
```

---

## File Structure (Final)

```
pkg/tui/
├── config/                      # NEW
│   ├── model.go                 # RootModel (ConfigModel)
│   ├── sidebar.go               # Sidebar component
│   ├── section_builder.go       # Helpers for step composition
│   ├── filecopy_rules.go        # FileCopy rules + filepicker
│   ├── error_overlay.go         # Error modal
│   ├── help_screen.go           # Help modal
│   └── stories/                 # Storybook stories
│       ├── components_storybook.go
│       ├── sections_storybook.go
│       ├── pages_storybook.go
│       ├── mocks.go
│       └── (optional) _test.go
│
├── fields/textinput.go          # REUSE
├── fields/selector.go           # REUSE
├── fields/confirm.go            # REUSE
├── wizard.go                    # REUSE
├── navigator.go                 # REUSE
├── table.go                     # REUSE
├── theme.go                     # REUSE
├── context.go                   # REUSE
└── field.go                     # REUSE

cmd/service/
├── config.go                    # Existing (REUSE validateConfig)
├── config_tui.go                # NEW: TUI command handler
├── root.go                      # MODIFY: register "gbm config"
└── config_test.go               # MODIFY: add E2E tests
```

---

## Phase Timeline

| Phase | Duration | Focus | Deliverable |
|-------|----------|-------|-------------|
| **Phase 1** | Week 1 | Scaffolding | `gbm config` launches, sidebar navigable |
| **Phase 2** | Week 2 | Basics section | Edit default_branch, worktrees_dir, save |
| **Phase 3** | Week 2.5 | JIRA section | Complete JIRA configuration |
| **Phase 4** | Week 3.5 | FileCopy & Worktrees | Lists, modals, filepicker |
| **Phase 5** | Week 4 | Integration | All sections together, error handling |
| **Phase 6** | Week 5 | Testing & Polish | Unit tests, E2E tests, storybook coverage |

---

## Success Criteria

- [ ] `gbm config` launches without errors
- [ ] Sidebar navigates all sections (↑↓←→ keys)
- [ ] All config fields editable
- [ ] Validation prevents invalid saves (blocks save, shows errors)
- [ ] Changes persist to `.gbm/config.yaml`
- [ ] Storybook covers >80% of components/sections (visual regression testing)
- [ ] Unit tests >80%, E2E happy paths pass
- [ ] Dark + light themes both work
- [ ] No breaking changes to existing commands
- [ ] Keyboard + mouse navigation smooth
- [ ] Help text accessible via `?` key
- [ ] Code reviewed and merged

---

## Quick Links

### Existing Components to Reuse
- [pkg/tui/field.go](./pkg/tui/field.go) - Field interface
- [pkg/tui/wizard.go](./pkg/tui/wizard.go) - Multi-step forms
- [pkg/tui/navigator.go](./pkg/tui/navigator.go) - Screen stack
- [pkg/tui/fields/textinput.go](./pkg/tui/fields/textinput.go) - Text fields
- [pkg/tui/fields/selector.go](./pkg/tui/fields/selector.go) - Dropdowns
- [pkg/tui/fields/confirm.go](./pkg/tui/fields/confirm.go) - Yes/No toggles
- [pkg/tui/table.go](./pkg/tui/table.go) - Lists/tables
- [pkg/tui/theme.go](./pkg/tui/theme.go) - Styling
- [cmd/service/config.go](./cmd/service/config.go) - Config validation

### Config Structure Reference
- [Config struct](./cmd/service/service.go#L115) - Root config
- [JiraConfig struct](./cmd/service/service.go#L18)
- [FileCopyConfig struct](./cmd/service/service.go#L76)
- [WorktreeConfig struct](./cmd/service/service.go#L83)
- [Config example](./config.example.yaml)

---

## Questions Before Starting?

1. **JIRA Filters**: Should status/priority/labels support multi-select, or single?
2. **Worktrees Edit**: View-only or editable (branch/merge_into)?
3. **Save Confirmation**: Preview changes before saving?
4. **Terminal Width**: Min/max widths to support?
5. **Help System**: Inline (`?` key) vs dedicated screen vs both?

**Answers documented in [PRD_CONFIG_TUI.md](./PRD_CONFIG_TUI.md) § 2**

---

## Next Steps

1. ✅ Read [QUICK_START.md](./QUICK_START.md) (5 min)
2. ✅ Review [PRD_CONFIG_TUI.md](./PRD_CONFIG_TUI.md) (30 min)
3. ✅ Study [COMPONENT_REUSE.md](./COMPONENT_REUSE.md) (15 min)
4. ✅ Understand [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) (20 min)
5. 📋 **Ready to start Phase 1?**
6. 📋 Any questions? Refer to docs above

---

**Status**: Planning Complete ✅  
**Next**: Phase 1 Implementation 🚀

---

*Generated: January 2026*
