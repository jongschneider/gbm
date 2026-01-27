# Quick Start: `gbm config` TUI Development

## TL;DR

This project adds an interactive TUI for managing `.gbm/config.yaml` using **lazygit-style sidebar navigation**.

### Key Files to Know

| File | Purpose |
|------|---------|
| [PRD_CONFIG_TUI.md](./PRD_CONFIG_TUI.md) | Full specification |
| [COMPONENT_REUSE.md](./COMPONENT_REUSE.md) | What we're reusing (57% code reuse!) |
| [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) | Week-by-week build plan |

### Reuse Strategy

**Don't write new components!** Reuse:
- `Navigator` - screen stack
- `Wizard` - multi-step forms
- `TextInput`, `Selector`, `Confirm` - form fields
- `Table` - data display
- `Theme`, `Context` - styling & state
- `filepicker` (bubbles) - file selection
- `validateConfig()` (service) - validation

**See [COMPONENT_REUSE.md](./COMPONENT_REUSE.md) for full matrix**

---

## Architecture at a Glance

```
RootModel (ConfigModel)
    ↓ (uses Navigator)
Sidebar (navigation)
    ↓ (selects section)
SectionModel = Wizard + Fields
    ↓ (field types)
TextInput | Selector | Confirm | FilePicker
    ↓ (uses)
bubbles components + Theme
```

---

## Storybook-First Development

**Write stories BEFORE implementing**:

```go
// In pkg/tui/config/stories/components_storybook.go
func TextInputWithError() storybook.Story {
    input := fields.NewTextInput("host", "JIRA Host", "")
    input.SetError("Invalid URL format")
    return storybook.Story{
        Name: "TextInput / With Error",
        Component: input,
    }
}

// Run it:
// go test ./pkg/tui/config -run TextInputWithError
```

---

## Section Structure

Each section (Basics, JIRA, FileCopy, Worktrees) is a `Wizard`:

```go
steps := []tui.Step{
    {
        Name: "jira_host",
        Field: fields.NewTextInput("jira_host", "JIRA Host", "https://jira.example.com"),
    },
    {
        Name: "jira_username",
        Field: fields.NewTextInput("jira_username", "Username", "user@example.com"),
    },
    {
        Name: "jira_token",
        Field: fields.NewTextInput("jira_token", "API Token", "")
            .WithValidator(validateToken),
    },
}
jiraServerWizard := tui.NewWizard(steps, ctx)
```

---

## Testing Checklist per Component

### ✅ Component Works
```bash
# 1. Write storybook story
# 2. Run storybook
go test ./pkg/tui/config -run YourStory

# 3. Verify visual
# 4. Implement component
# 5. Re-run storybook
go test ./pkg/tui/config -run YourStory -v
```

### ✅ Unit Test
```bash
# pkg/tui/config/mycomponent_test.go
go test ./pkg/tui/config -run TestMyComponent -v
```

### ✅ E2E Test
```bash
# cmd/service/config_test.go
go test ./cmd/service -run TestConfig -v
```

---

## Key Implementation Tips

### 1. Use Wizard for Forms
```go
// ✅ Good: reuse Wizard
wizard := tui.NewWizard(steps, ctx)

// ❌ Bad: custom form management
type MyForm struct {
    step int
    fields []Field
    // ... 50 lines of navigation logic
}
```

### 2. Use Selector for Dropdowns
```go
// ✅ Good: reuse Selector
options := []fields.Option{
    {Label: "In Dev", Value: "In Dev"},
    {Label: "Done", Value: "Done"},
}
selector := fields.NewSelector("status", "Status", options)

// ❌ Bad: custom dropdown
type MyDropdown struct {
    options []string
    cursor int
    // ... custom rendering
}
```

### 3. Validators on TextInput
```go
// ✅ Good: validator function
host := fields.NewTextInput("host", "Host", "")
host.WithValidator(func(s string) error {
    if !strings.Contains(s, "https://") {
        return fmt.Errorf("host must be HTTPS")
    }
    return nil
})

// ❌ Bad: validate in Update()
func (f *TextInput) Update(msg tea.Msg) (tui.Field, tea.Cmd) {
    // ... validation logic mixed in
}
```

### 4. Filepicker for File Selection
```go
// ✅ Good: reuse filepicker
picker := filepicker.New()
picker.CurrentDirectory = repoRoot

// ❌ Bad: custom filesystem browser
type FileBrowser struct {
    // ... custom dir traversal
}
```

### 5. Table for Lists
```go
// ✅ Good: reuse Table
t := tui.NewTable(cols, rows, ctx, theme)

// ❌ Bad: custom list rendering
type MyList struct {
    items []Item
    cursor int
    // ... custom rendering + filtering
}
```

---

## Dark/Light Theme Support

All storybook stories should test both themes:

```go
func JiraServerFormLight() storybook.Story {
    ctx := &tui.Context{Theme: tui.DefaultTheme()}
    // ... story setup
}

func JiraServerFormDark() storybook.Story {
    ctx := &tui.Context{Theme: tui.DarkTheme()} // hypothetical
    // ... story setup
}
```

---

## Phase Guidance

### Phase 1: Build Infrastructure
- [ ] Sidebar component (section list, expand/collapse)
- [ ] RootModel (configModel using Navigator)
- [ ] Command registration (gbm config)
- [ ] First storybook stories (sidebar variants)

**Success**: `gbm config` launches, shows sidebar, can navigate to section

### Phase 2: Basics Section
- [ ] TextInput fields (default_branch, worktrees_dir)
- [ ] Validators
- [ ] Form stories
- [ ] Save flow

**Success**: Edit Basics, save to file, verify with `cat .gbm/config.yaml`

### Phase 3: JIRA & Filters
- [ ] JIRA toggle (Confirm field)
- [ ] JIRA Server form
- [ ] JIRA Filters (Selector dropdowns)
- [ ] Skip logic (hide filters if JIRA disabled)

**Success**: Enable/disable JIRA, edit all subsections

### Phase 4: File Copy & Worktrees
- [ ] FileCopy Auto form
- [ ] FileCopy Rules (Table + filepicker modal)
- [ ] Worktrees list (Table)
- [ ] Add/edit/delete modals

**Success**: Add file copy rule with filepicker, add worktree

### Phase 5: Integration & Polish
- [ ] Error overlays
- [ ] Help screen
- [ ] Dark/light theme storybook stories
- [ ] Comprehensive tests

**Success**: All sections work, storybook >80%, tests pass

---

## Commands You'll Use

```bash
# Build the binary
just build
# or: go build -o gbm ./cmd

# Run (will open TUI)
just run config
# or: ./gbm config

# Test
just test
# or: go test ./...

# Test with coverage
just test-coverage

# Lint
just lint

# Run specific test
go test ./pkg/tui/config -run TestSidebar -v

# Run specific storybook story
go test ./pkg/tui/config -run TextInputWithError -v
```

---

## Questions?

Refer to:
- **Architecture**: [PRD_CONFIG_TUI.md § 3](./PRD_CONFIG_TUI.md#3-architecture)
- **Components**: [COMPONENT_REUSE.md](./COMPONENT_REUSE.md)
- **Timeline**: [IMPLEMENTATION_PLAN.md § Phase Breakdown](./IMPLEMENTATION_PLAN.md)
- **Code**: Check `pkg/tui/wizard.go`, `pkg/tui/fields/textinput.go`, etc.

---

## Checklist: Ready to Start?

- [ ] Read [PRD_CONFIG_TUI.md](./PRD_CONFIG_TUI.md)
- [ ] Review [COMPONENT_REUSE.md](./COMPONENT_REUSE.md)
- [ ] Understand [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md) timeline
- [ ] Know your section assignment (Phase 1, 2, 3, 4, or 5)
- [ ] Familiar with existing `pkg/tui/fields/` components
- [ ] Ready to write storybook stories first?

**Ready? Let's build!** 🚀
