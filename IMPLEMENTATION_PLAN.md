# Implementation Plan: `gbm config` TUI

**Status**: Ready to Build  
**Estimated Duration**: 4-5 weeks  
**Code Reuse**: 57% (870-1,290 LOC from existing components)

---

## Quick Links

- [specs/prd-config-tui.md](./specs/prd-config-tui.md) - Full product specification
- [COMPONENT_REUSE.md](./COMPONENT_REUSE.md) - Detailed reuse analysis
- [pkg/tui/](./pkg/tui/) - Existing components to reuse

---

## Build Strategy

### Why This Works

1. **Reuse Proven Code**: Navigator, Wizard, TextInput, Selector, Confirm, Table are all battle-tested in production
2. **Consistency**: Uses same theme, styling, patterns as `gbm wt ls` and `gbm wt add`
3. **Fast Iteration**: Storybook-go enables visual validation without running full CLI
4. **Minimal New Code**: Only ~650-950 LOC of new implementation needed
5. **Low Risk**: All risky patterns already validated in existing commands

### Architecture Overview

```
ConfigTUI (RootModel)
    ↓ (delegates via Navigator)
[Sidebar, Section Wizard, Modal]*
    ↓ (each Section = Wizard orchestrating Steps)
[TextInput, Selector, Confirm, Custom Fields]*
    ↓ (each Field implements Field interface)
[bubbles/textinput, bubbles/table, bubbles/filepicker]
    ↓ (underlying bubbles components)
[lipgloss styling + theme]
```

---

## Phase Breakdown

### Phase 1: Scaffolding & Core (Week 1)

**Deliverables**:
- [ ] Project structure in place
- [ ] Sidebar component (expandable sections)
- [ ] RootModel (Navigator + config load/save)
- [ ] Basic storybook setup
- [ ] Initial stories for components

**New Files**:
- `pkg/tui/config/model.go` (RootModel using Navigator)
- `pkg/tui/config/sidebar.go` (Sidebar navigation)
- `pkg/tui/config/stories/components_storybook.go` (first stories)
- `cmd/service/config_tui.go` (command handler)
- Update `cmd/service/root.go` (register command)

**Tests**:
- Unit: ConfigModel state transitions
- E2E: `gbm config` launches and exits cleanly

**Storybook Coverage**:
- Sidebar: collapsed, expanded, invalid badges

---

### Phase 2: Basics Section (Week 2)

**Deliverables**:
- [ ] Basics section (default_branch, worktrees_dir)
- [ ] TextInput fields with validators
- [ ] Section switching via sidebar
- [ ] Form validation UI

**Reused**:
- `Wizard` + `fields.TextInput` for form
- `Theme` for styling
- `validateConfig()` for validation
- `Navigator` for sidebar → section transition

**New Files**:
- `pkg/tui/config/section_builder.go` (helpers to build steps)
- Tests for Basics form

**Storybook Stories**:
- TextInput: empty, filled, error, masked
- BasicsForm: empty, filled, validation errors
- Full page: Sidebar + Basics focused

---

### Phase 3: JIRA Section (Week 2.5)

**Deliverables**:
- [ ] JIRA toggle (enable/disable)
- [ ] JIRA Server subsection (host, username, api_token)
- [ ] JIRA Filters subsection (status, priority, type, labels using Selector)
- [ ] JIRA Attachments subsection (toggle + numeric fields)
- [ ] JIRA Markdown subsection (toggles + text field)

**Reused**:
- `fields.Confirm` for JIRA enable/disable
- `fields.TextInput` for server details (with URL validator for host)
- `fields.Selector` for filter options
- `Wizard` with skip logic (skip subsections if JIRA disabled)

**New Files**:
- Logic to skip JIRA subsections when disabled

**Storybook Stories**:
- Confirm: enabled/disabled
- JiraServerForm: with token masked
- JiraFiltersForm: with selector examples
- Full page: Sidebar + JIRA expanded showing all subsections

---

### Phase 4: File Copy & Worktrees (Week 3.5)

**Deliverables**:
- [ ] FileCopy Auto section (enabled, source, checkboxes, excludes[])
- [ ] FileCopy Rules section (list + filepicker modal)
- [ ] Worktrees section (list with add/edit/delete)

**Reused**:
- `Table` for rules/worktrees list
- `filepicker` from bubbles for file selection
- `fields.Confirm` for enable auto-copy
- `Wizard` for add/edit modals

**New Files**:
- `pkg/tui/config/filecopy_rules.go` (filepicker integration)
- Logic for multi-file selection from filepicker

**Storybook Stories**:
- FilePickerModal: empty dir, nested dirs, file selection
- FileCopyRulesList: empty, 3 rules, focused rule
- WorktreesList: empty, 5 worktrees, edit modal open
- Full page: Sidebar + FileCopy focused with modal

---

### Phase 5: Integration & Polish (Week 4)

**Deliverables**:
- [ ] All sections integrated in ConfigModel
- [ ] Save/discard/reset flows
- [ ] Validation error overlays
- [ ] Theme support (dark/light)
- [ ] Help text for all fields
- [ ] Comprehensive keyboard shortcuts

**New Files**:
- Error overlay component (modal showing validation errors)
- Help screen (keyboard shortcuts)

**Storybook Stories**:
- All components in dark + light theme
- Full pages with validation errors
- Modal overlays (error, help, add/edit)
- Different terminal widths (80, 120, 240 cols)

---

### Phase 6: Testing & Polish (Week 5)

**Deliverables**:
- [ ] Unit tests for all new models
- [ ] E2E test (load config → edit → save → verify file)
- [ ] Storybook regression suite
- [ ] Documentation

**Tests**:
- `pkg/tui/config/model_test.go` (ConfigModel state transitions, dirty flag, save)
- `pkg/tui/config/sidebar_test.go` (expand/collapse, focus navigation)
- `cmd/service/config_test.go` (E2E: edit all sections, save, verify)
- Storybook: run all stories, verify no render errors

**Storybook Coverage**:
- >80% of components in >3 states
- All sections in isolation + together
- All modals
- Dark/light themes
- Different viewport sizes

---

## File Structure (Final)

```
pkg/tui/
├── config/                           # NEW
│   ├── model.go                      # ConfigModel (RootModel)
│   ├── sidebar.go                    # Sidebar component
│   ├── section_builder.go            # Helpers to build steps
│   ├── filecopy_rules.go             # FileCopy rules + filepicker
│   ├── error_overlay.go              # Error modal
│   ├── help_screen.go                # Help modal
│   └── stories/
│       ├── components_storybook.go   # Component stories
│       ├── sections_storybook.go     # Section stories
│       ├── pages_storybook.go        # Full page stories
│       ├── mocks.go                  # Test data
│       └── _test.go (optional)       # Story tests
│
├── (existing files, no changes)
│
└── field.go, wizard.go, navigator.go, theme.go, context.go, table.go
    fields/textinput.go, selector.go, confirm.go

cmd/service/
├── config.go                         # Existing (no changes)
├── config_tui.go                     # NEW: TUI command
├── root.go                           # MODIFY: add config command
├── config_test.go                    # MODIFY: add E2E tests
└── (others unchanged)
```

---

## Development Workflow

### Day-to-Day

1. **Write storybook story first** (TDD for UI)
   ```bash
   # Edit pkg/tui/config/stories/components_storybook.go
   # Add: func TextInputWithError() storybook.Story { ... }
   ```

2. **Run storybook to visualize**
   ```bash
   go test ./pkg/tui/config/stories -run TextInputWithError
   # See rendered component
   ```

3. **Implement component** (in pkg/tui/config/)
   ```bash
   # Edit pkg/tui/config/sidebar.go or model.go
   # Implement the component from story spec
   ```

4. **Verify story passes**
   ```bash
   go test ./pkg/tui/config -v
   ```

5. **Write unit tests** (optional but recommended)
   ```bash
   # Edit pkg/tui/config/sidebar_test.go
   # Test state transitions, focus navigation
   ```

### Weekly Build Cycle

- **Monday**: Story definitions for section (components_storybook.go)
- **Tuesday-Thursday**: Implementation + story verification
- **Friday**: Tests + integration with previous sections

### Testing Commands

```bash
# Run all config TUI tests
go test ./pkg/tui/config/... -v

# Run storybook only
go test ./pkg/tui/config/stories -v

# Run specific story
go test ./pkg/tui/config -run TextInputWithError

# Run E2E test
go test ./cmd/service -run TestConfigTUI -v

# Run all checks
just validate
```

---

## Milestones & Success Criteria

### Milestone 1: Phase 1-2 (End of Week 2)
- [ ] `gbm config` command exists
- [ ] Sidebar navigates between sections
- [ ] Basics section editable
- [ ] Can save and exit
- [ ] Storybook stories run

### Milestone 2: Phase 3 (End of Week 3)
- [ ] JIRA section complete with all subsections
- [ ] Validation prevents invalid saves
- [ ] Dark/light theme works

### Milestone 3: Phase 4 (End of Week 4)
- [ ] File copy rules with filepicker
- [ ] Worktrees list
- [ ] All sections integrated

### Milestone 4: Phase 5-6 (End of Week 5)
- [ ] Full config TUI works end-to-end
- [ ] Storybook coverage >80%
- [ ] Tests pass (>70% overall coverage)
- [ ] Ready for user testing

---

## Known Risks & Mitigations

### Risk 1: Filepicker in Modal
**Issue**: Filepicker might not handle Escape correctly when nested in Navigator
**Mitigation**: Write storybook story + E2E test for this flow early (Phase 4 start)

### Risk 2: Complex Validation
**Issue**: validateConfig() might need updates for new config structures
**Mitigation**: Dry-run validation in storybook before full integration

### Risk 3: Keyboard Conflicts
**Issue**: Sidebar keys (↑↓←→) might conflict with field keys
**Mitigation**: Design sidebar to consume keys only when focused (not in field)

### Risk 4: Performance (Large Lists)
**Issue**: Table might be slow with 100+ worktrees
**Mitigation**: Table already has filter mode (/), use for large lists

### Risk 5: JIRA Filters Complexity
**Issue**: Multi-select dropdowns for JIRA filters might be complex
**Mitigation**: Ship v1 without multi-select, use single Selector fields (can upgrade in v2)

---

## Success Criteria (Definition of Done)

- [ ] `gbm config` launches without errors
- [ ] All sections navigable via sidebar
- [ ] All config fields editable
- [ ] Validation prevents invalid saves
- [ ] Changes persist to `.gbm/config.yaml`
- [ ] Storybook covers >80% of components/sections
- [ ] Tests pass: unit >80%, E2E happy paths
- [ ] No breaking changes to existing commands
- [ ] Dark + light theme both work
- [ ] Keyboard navigation smooth (no missed keys)
- [ ] Help text accessible on demand
- [ ] Code reviewed and merged

---

## Next Steps

1. **Review this plan** with team
2. **Approve Phase 1 scope** (sidebar + RootModel + first stories)
3. **Begin Phase 1 implementation** (target: end of this week)
4. **Daily standups** on storybook stories + test progress

---

## Questions for Team

1. **JIRA Filters**: Multi-select or single-select in v1?
2. **Worktrees Edit**: Should users be able to edit branch/merge_into via config TUI, or view-only?
3. **Preview Mode**: Should save show a confirmation of changes before committing?
4. **Terminal Width**: Any known minimum/maximum widths we should test?
5. **Help System**: Inline help (?) vs dedicated help screen (?) or both?
