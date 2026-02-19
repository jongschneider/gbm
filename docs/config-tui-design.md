# Config TUI Design

Design document for the `gbm config` interactive TUI for editing `.gbm/config.yaml`.

## Problem

The config file has nested, structured data (JIRA settings with sub-objects, file copy rules as arrays, template variables). Surge's flat key-value settings approach doesn't map well here. We need something that handles:

- **Nested objects** — `jira.attachments`, `jira.markdown`, `file_copy.auto`
- **Lists** — `file_copy.rules[].files`, `filters.status[]`, `auto.exclude[]`
- **Mixed types** — booleans, strings, integers, string lists
- **Sensitive values** — could contain tokens or credentials
- **Template variables** — `{gitroot}`, `{branch}`, `{issue}` in `worktrees_dir`
- **Optional sections** — JIRA and file_copy are entirely optional
- **Map structures** — `worktrees` is a map of name to config

## Config Structure Reference

```yaml
default_branch: main              # required, string
worktrees_dir: worktrees          # required, string (supports templates)

jira:                             # optional section
  host: https://jira.example.com  # string
  me: user@example.com            # string
  filters:                        # optional sub-section
    priority: "High"              # string
    type: "Bug"                   # string
    component: "Backend"          # string
    reporter: ""                  # string
    assignee: ""                  # string
    order_by: "priority"          # string
    status: ["In Dev.", "Open"]   # string list
    labels: ["backend"]           # string list
    custom_args: []               # string list
    reverse: false                # bool
  markdown:                       # sub-section
    filename_pattern: "{key}.md"  # string (template)
    max_depth: 2                  # int
    include_comments: true        # bool
    include_attachments: true     # bool
    use_relative_links: true      # bool
    include_linked_issues: false  # bool
  attachments:                    # sub-section
    enabled: true                 # bool
    directory: ".jira/attachments"# string
    max_size_mb: 50               # int
    download_timeout_seconds: 30  # int
    retry_attempts: 3             # int
    retry_backoff_ms: 1000        # int

file_copy:                        # optional section
  auto:                           # sub-section
    enabled: true                 # bool
    source_worktree: "{default}"  # string (template)
    copy_ignored: true            # bool
    copy_untracked: false         # bool
    exclude: ["*.log", ".DS_Store"] # string list
  rules:                          # list of objects
    - source_worktree: main       # string
      files: [".env", "config/"]  # string list

worktrees:                        # optional map
  main:
    branch: main                  # string
    merge_into: ""                # string
    description: "Primary"        # string
  feature/auth:
    branch: feature/auth          # string
    merge_into: main              # string
    description: "Auth feature"   # string
```

## Design Direction: Section-Based Form

A **tabbed section view** where each tab shows a scrollable form for one config area. This balances discoverability (tabs show what's configurable) with depth (each section can have its own sub-groupings rendered inline).

### Layout

```
+-- gbm config ----------------------------------------------------------+
|  General    JIRA    File Copy    Worktrees                              |
+------------------------------------------------------------------------+
|                                                                        |
|  Default Branch                                                        |
|  +----------------------------------+                                  |
|  | main                             |                                  |
|  +----------------------------------+                                  |
|                                                                        |
|  Worktrees Directory                                                   |
|  +----------------------------------+                                  |
|  | worktrees                        |                                  |
|  +----------------------------------+                                  |
|  Supports templates: {gitroot}, {branch}, {issue}                      |
|                                                                        |
|                                                                        |
|                                                                        |
+------------------------------------------------------------------------+
|  tab/S-tab section . up/dn navigate . ? help . s save                  |
+------------------------------------------------------------------------+
```

The footer shows only the 3-4 most relevant keybindings for the current
state. Press `?` to open a full keybinding reference overlay.

### JIRA Tab (with sub-groups)

```
+-- gbm config ----------------------------------------------------------+
|  General   [JIRA]   File Copy    Worktrees                             |
+------------------------------------------------------------------------+
|                                                                        |
|  -- Connection -----------------------------------------               |
|  Host              https://jira.example.com                            |
|  Me                user@example.com                                    |
|                                                                        |
|  -- Filters --------------------------------------------               |
|  Priority          High                                                |
|  Type              Bug                                                 |
|  Component         Backend                                             |
|  Reporter          (empty)                                             |
|  Assignee          (empty)                                             |
|  Order By          priority                                            |
|  Status            In Dev., Open                                       |
|  Labels            backend                                             |
|  Custom Args       (empty)                                             |
|  Reverse           no                                                  |
|                                                                        |
|  -- Markdown -------------------------------------------               |
|  Filename Pattern  {key}.md                                            |
|  Max Depth         2                                                   |
|  Include Comments  yes                                                 |
|  Include Attach.   yes                                                 |
|  Relative Links    yes                                                 |
|  Linked Issues     no                                              4/24|
|                                                                        |
|  -- Attachments ----------------------------------------               |
|  Enabled           yes                                                 |
|  Max Size (MB)     50                                                  |
|  Directory         .jira/attachments                                   |
|  Timeout (sec)     30                                                  |
|  Retry Attempts    3                                                   |
|  Retry Backoff     1000ms                                              |
|                                                                        |
+------------------------------------------------------------------------+
|  tab/S-tab section . up/dn navigate . ? help . s save                  |
+------------------------------------------------------------------------+
```

The JIRA tab has 24 fields across 4 groups. On a standard 24-line
terminal, scrolling is required. The viewport auto-scrolls to keep the
focused field visible, and a position indicator (`4/24`) appears in the
right margin.

Groups within a tab are visual-only separators. `up`/`dn` skips over them
-- you can't "focus" a group header.

### File Copy Tab (with list management)

```
+-- gbm config ----------------------------------------------------------+
|  General    JIRA   [File Copy]    Worktrees                            |
+------------------------------------------------------------------------+
|                                                                        |
|  -- Auto Copy ------------------------------------------               |
|  Enabled           yes                                                 |
|  Source Worktree   {default}                                           |
|  Copy Ignored      yes                                                 |
|  Copy Untracked    no                                                  |
|  Exclude           *.log, node_modules/, .DS_Store                     |
|                                                                        |
|  -- Rules ----------------------------------------------               |
|  > 1. main -> .env, config/                                            |
|    2. develop -> .vscode/settings.json                                 |
|                                                                        |
+------------------------------------------------------------------------+
|  a add rule . d delete . enter edit . ? help                           |
+------------------------------------------------------------------------+
```

Rules are focusable rows. `enter` opens a rule editor overlay (see
example 9). `a` adds a new rule. `d` deletes the focused rule (with
confirmation). If no rules exist, the section shows:

```
|  -- Rules ----------------------------------------------               |
|    (no rules configured) -- press a to add                             |
```

Long rule summaries are truncated with a count:

```
|  > 1. main -> .env, config/ (+5 more)                                 |
```

### Worktrees Tab

```
+-- gbm config ----------------------------------------------------------+
|  General    JIRA    File Copy   [Worktrees]                            |
+------------------------------------------------------------------------+
|                                                                        |
|  > 1. main -> branch: main, merge: --                                 |
|    2. feature/auth -> branch: feature/auth, merge: main               |
|    3. bugfix/login -> branch: bugfix/login, merge: main               |
|                                                                        |
+------------------------------------------------------------------------+
|  a add . d delete . enter edit . ? help                                |
+------------------------------------------------------------------------+
```

Worktrees display as a list of entries, one per line. Each entry shows the
worktree name, branch, and merge target. The map key (worktree name) is
the primary identifier.

Worktree entries are focusable rows. `enter` opens a worktree editor
overlay (see example 17). `a` adds a new worktree entry. `d` deletes the
focused entry (with `y/n` confirmation). If no worktrees exist, the
section shows:

```
|    (no worktrees configured) -- press a to add                         |
```

## Interaction Walkthrough

Step-by-step examples of what the user sees at each point.

### 1. Opening the TUI

User runs `gbm config`. The General tab is active, cursor on the first field.
The `>` marker shows which field has focus.

```
+-- gbm config ----------------------------------------------------------+
|  [General]    JIRA    File Copy    Worktrees                           |
+------------------------------------------------------------------------+
|                                                                        |
|  > Default Branch      main                                            |
|    Worktrees Dir       worktrees                                       |
|                                                                        |
|                                                                        |
|                                                                        |
|                                                                        |
|                                                                        |
+------------------------------------------------------------------------+
|  tab/S-tab section . up/dn navigate . ? help . s save                  |
+------------------------------------------------------------------------+
```

If no `.gbm/config.yaml` exists, the TUI creates a default config in
memory using detected values (`getDefaultBranch()`, `worktrees_dir:
worktrees`) and opens in normal editing mode. The status bar shows
`[new file]` instead of a file path. Saving creates the file.

### 2. Pressing `dn` -- move focus down

Nothing opens, nothing changes. Just the cursor moves.

```
|                                                                        |
|    Default Branch      main                                            |
|  > Worktrees Dir       worktrees                                       |
|                                                                        |
```

### 3. Pressing `enter` on a string field -- inline editing

The value becomes an editable text input. The cursor blinks inside.
A description hint appears below if the field has one.

```
|                                                                        |
|    Default Branch      main                                            |
|  > Worktrees Dir       +----------------------------+                  |
|                        | worktrees_                  |                  |
|                        +----------------------------+                  |
|    Supports: {gitroot}, {branch}, {issue}                              |
|                                                                        |
+------------------------------------------------------------------------+
|  enter confirm . esc cancel                                            |
+------------------------------------------------------------------------+
```

The user types to change the value. The footer keybindings change to show
editing-specific keys.

### 4. Pressing `enter` to commit the edit

The input closes. The value updates. If it changed, a `*` marker appears
to indicate it's been modified (dirty).

```
|                                                                        |
|    Default Branch      main                                            |
|  > Worktrees Dir     * ../{gitroot}-wt                                 |
|                                                                        |
+------------------------------------------------------------------------+
|  [1 modified] tab/S-tab . up/dn navigate . ? help . s save            |
+------------------------------------------------------------------------+
```

The status bar shows `[N modified]` (e.g., `[1 modified]`) when any
fields are dirty, giving the user a count of changes.

### 5. Pressing `esc` to cancel an edit

The input closes, the original value is restored, no dirty marker.

```
|                                                                        |
|    Default Branch      main                                            |
|  > Worktrees Dir       worktrees                                       |
|                                                                        |
```

### 6. Pressing `enter` on a bool field -- immediate toggle

No text input opens. The value flips instantly and the field stays focused.
Boolean values are colored: green (`SuccessAccent`) for `yes`, red
(`ErrorAccent`) for `no` when focused. Muted/gray when unfocused. This
follows the Surge `fields/confirm.go` pattern.

Before:
```
|  > Enabled             yes                                             |
```

After pressing `enter`:
```
|  > Enabled           * no                                              |
```

Press `enter` again to flip it back.

### 7. Pressing `tab` -- switch to the next section tab

The entire form body changes to show the new section's fields.
Cursor resets to the first field in the new section.

Before (General tab):
```
|  [General]    JIRA    File Copy    Worktrees                           |
+------------------------------------------------------------------------+
|    Default Branch      main                                            |
|  > Worktrees Dir     * ../{gitroot}-wt                                 |
```

After pressing `tab` (JIRA tab):
```
|  General    [JIRA]    File Copy    Worktrees                           |
+------------------------------------------------------------------------+
|                                                                        |
|  -- Connection -----------------------------------------               |
|  > Host                https://jira.example.com                        |
|    Me                  user@example.com                                |
|                                                                        |
|  -- Filters --------------------------------------------               |
|    Priority            High                                            |
|    Type                Bug                                             |
|    Component           Backend                                         |
|    ...                                                          3/24   |
```

### 8. Pressing `enter` on a string list field -- list overlay

The form dims behind and a centered overlay appears. The overlay title
includes the breadcrumb path (e.g., `JIRA > Filters > Status`). The
overlay has its own cursor for selecting items.

```
+-- gbm config ----------------------------------------------------------+
|  General    [JIRA]    File Copy    Worktrees                           |
+--##################################################################--+
|####################################################################---|
|#####+-  JIRA > Filters > Status ----------+########################--|
|#####|                                      |########################--|
|#####|  > 1. In Dev.                        |########################--|
|#####|    2. Open                           |########################--|
|#####|                                      |########################--|
|#####|  a add . d delete . enter done . esc |########################--|
|#####+-----------------------------------------+########################|
|####################################################################---|
+------------------------------------------------------------------------+
|  [1 modified] editing list...                                          |
+------------------------------------------------------------------------+
```

The `#` represents the dimmed background (Surge does this with
`lipgloss.Place()` + `WithWhitespaceChars`).

Inside the overlay:
- `up`/`dn` selects existing items
- `a` opens a text input to add a new item (type value, press `enter` to confirm the add)
- `d` deletes the selected item (with confirmation: `Delete "X"? y/n`)
- `enter` commits all changes and closes the overlay
- `esc` cancels and discards all changes made in this overlay session

### 9. Pressing `enter` on a rule -- rule editor overlay

Rules have a `source_worktree` and a `files` list. The rule editor
is an overlay with two fields: a string field for the source worktree
and a string list (reusing the list overlay) for file paths.

```
+-- gbm config ----------------------------------------------------------+
|  General    JIRA   [File Copy]    Worktrees                            |
+--##################################################################--+
|####################################################################---|
|#####+-  File Copy > Rules > Edit Rule --------+########################|
|#####|                                          |########################|
|#####|  > Source Worktree   main                |########################|
|#####|    Files             .env, config/       |########################|
|#####|                                          |########################|
|#####|  up/dn navigate . e edit . enter confirm . esc cancel            |
|#####+-----------------------------------------+########################|
|####################################################################---|
+------------------------------------------------------------------------+
|  [1 modified] editing rule...                                          |
+------------------------------------------------------------------------+
```

- `up`/`dn` navigates between the two fields.
- `e` edits the focused field: `Source Worktree` opens an inline text
  input; `Files` opens the list overlay as a nested modal.
- `enter` (when not editing a field) confirms changes and closes the overlay.
- `esc` discards changes and closes the overlay.

**Nested overlay behavior:** Pressing `e` on the `Files` field opens the
list overlay on top of the rule editor overlay. This creates a modal stack.
`esc` closes only the innermost overlay, returning to the one beneath with
its state intact. Each overlay level is independent -- closing the inner
list overlay returns to the rule editor with its state preserved. `enter`
(confirm) in the inner overlay saves changes to the parent overlay's state.

> **V2: Miller-columns file picker.** A yazi-style miller-columns
> browser will replace the manual path entry for the `Files` field.
> The browser will show the source worktree's actual file tree with
> three columns (parent | current | preview), space to toggle
> selection, and lazy directory loading. See `v2-file-picker.md`
> for the full design.

### 10. Validation error on edit commit

User edits `worktrees_dir`, types `../{invalid}`, presses `enter`.
The input stays open and an error appears below it in red.

```
|  > Worktrees Dir       +----------------------------+                  |
|                        | ../{invalid}_               |                  |
|                        +----------------------------+                  |
|    x unknown template variable: {invalid}                              |
|    Supports: {gitroot}, {branch}, {issue}                              |
|                                                                        |
+------------------------------------------------------------------------+
|  enter confirm . esc cancel                                            |
+------------------------------------------------------------------------+
```

The error clears when the user fixes the value or presses `esc` to cancel.

### 11. Pressing `s` to save

Before writing, the TUI writes a backup to `.gbm/config.yaml.bak`. This
provides a recovery path if the save produces incorrect values.

Validation runs on all fields. Before writing, the TUI checks the
file's modification time against when it was loaded. If the file changed
on disk (external edit), a warning overlay appears:

```
|#####+-  External Changes -------------------+########################|
|#####|                                        |########################|
|#####|  Config file was modified externally.  |########################|
|#####|  Overwrite?                            |########################|
|#####|                                        |########################|
|#####|  [Overwrite]    Cancel                 |########################|
|#####|                                        |########################|
|#####+-----------------------------------------+########################|
```

If everything passes, the file is written (using `go-yaml/yaml.v3`
node-based API to preserve user comments) and a success flash appears
in the status bar.

```
+------------------------------------------------------------------------+
|  ok saved .gbm/config.yaml                                             |
+------------------------------------------------------------------------+
```

The success flash auto-clears after 3 seconds, reverting to the normal
status bar content.

If validation fails, errors are shown as a navigable summary. Pressing
`enter` on an error jumps directly to the offending field (switches tab,
scrolls to field, focuses it):

```
|#####+-  Cannot Save ----------------------+########################--|
|#####|                                      |########################--|
|#####|  2 errors:                           |########################--|
|#####|                                      |########################--|
|#####|  > General > Default Branch          |########################--|
|#####|      required field is empty         |########################--|
|#####|                                      |########################--|
|#####|    General > Worktrees Dir           |########################--|
|#####|      unknown template: {bad}         |########################--|
|#####|                                      |########################--|
|#####|  enter go to error . esc close       |########################--|
|#####+-----------------------------------------+########################|
```

After the error overlay is closed, tab labels with validation errors show
a badge indicator (e.g., `General (!)`, `JIRA (!)`). Badges persist until
the corresponding errors are fixed, then clear automatically. This gives
persistent visual feedback about which sections need attention.

#### Write failure handling

If validation passes but the file write fails (disk full, permission
denied, etc.), an error overlay appears with the OS error message:

```
|#####+-  Save Failed -----------------------+########################--|
|#####|                                      |########################--|
|#####|  Failed to write .gbm/config.yaml:   |########################--|
|#####|  permission denied                   |########################--|
|#####|                                      |########################--|
|#####|  esc close                           |########################--|
|#####+-----------------------------------------+########################|
```

Same visual treatment as validation errors. Dismiss with `esc`.

### 12. Pressing `q` to quit -- dirty guard

If there are unsaved changes, a confirmation overlay appears.

```
|#####+-  Unsaved Changes -------------------+########################--|
|#####|                                      |########################--|
|#####|  You have unsaved changes:           |########################--|
|#####|    . Worktrees Dir                   |########################--|
|#####|    . JIRA > Host                     |########################--|
|#####|                                      |########################--|
|#####|  [Save & Quit]    Discard    Cancel  |########################--|
|#####|                                      |########################--|
|#####+-----------------------------------------+########################|
```

- `Save & Quit` -- validates, saves, exits
- `Discard` -- exits without saving
- `Cancel` -- returns to editing

If there are no unsaved changes, `q` exits immediately.

### 13. Pressing `r` to reset a field

Resets the focused field to its **last-saved value** (what was loaded
from disk), not the schema default. If the field is dirty, a
confirmation prompt appears:

```
|  > Worktrees Dir     * ../{gitroot}-wt                                 |
|    Reset to saved value? y/n                                           |
```

After pressing `y`:
```
|  > Worktrees Dir       worktrees                                       |
```

If the field is not dirty (current value matches last-saved), `r` is
a no-op.

#### Pressing `R` to reset all fields

`R` (capital) resets all dirty fields across all tabs to their last-saved
values. A confirmation overlay appears listing every field that will be
reset:

```
|#####+-  Reset All Changes -----------------+########################--|
|#####|                                      |########################--|
|#####|  Reset 3 fields to saved values?     |########################--|
|#####|    . Worktrees Dir                   |########################--|
|#####|    . JIRA > Host                     |########################--|
|#####|    . File Copy > Source Worktree     |########################--|
|#####|                                      |########################--|
|#####|  [Reset]    Cancel                   |########################--|
|#####|                                      |########################--|
|#####+-----------------------------------------+########################|
```

Selecting `Reset` reverts all listed fields. Selecting `Cancel` returns to
editing with no changes. If no fields are dirty, `R` is a no-op.

### 14. File Copy tab -- rules list

Rules display as a summary list. Each rule is one focusable line showing
`source -> files`. Long summaries are truncated with a count.

```
|  General    JIRA   [File Copy]    Worktrees                            |
+------------------------------------------------------------------------+
|                                                                        |
|  -- Auto Copy ------------------------------------------               |
|  > Enabled             yes                                             |
|    Source Worktree      {default}                                      |
|    Copy Ignored         yes                                            |
|    Copy Untracked       no                                             |
|    Exclude              *.log, node_modules/, .DS_Store                |
|                                                                        |
|  -- Rules ----------------------------------------------               |
|  > 1. main -> .env, config/                                            |
|    2. develop -> .vscode/settings.json                                 |
|                                                                        |
|                                                                        |
+------------------------------------------------------------------------+
|  a add rule . d delete . enter edit . ? help                           |
+------------------------------------------------------------------------+
```

When the cursor is on a rule, `enter` opens the rule editor overlay
(shown in example 9) with source worktree and files fields. `a` adds
a blank new rule. `d` deletes the focused rule (with confirmation:
`Delete rule "main"? y/n`).

### 15. Empty optional section

If JIRA isn't configured at all (no keys set), the tab shows a
placeholder instead of a wall of empty fields.

```
|  General    [JIRA]    File Copy    Worktrees                           |
+------------------------------------------------------------------------+
|                                                                        |
|                                                                        |
|                                                                        |
|         JIRA integration is not configured.                            |
|                                                                        |
|         Press enter to set up, or skip this tab.                       |
|                                                                        |
|                                                                        |
|                                                                        |
+------------------------------------------------------------------------+
|  tab/S-tab section . enter set up . q quit                             |
+------------------------------------------------------------------------+
```

Pressing `enter` populates the section with all fields (empty/default)
and focuses the first one.

Once configured, optional sections (JIRA, File Copy) have an "Enabled"
toggle as a conceptual first field. When the "Enabled" toggle is set to
`no`, the remaining fields in that section are hidden and the entire
section is omitted from the saved YAML. When toggled back to `yes`, all
fields reappear with their previous values. This provides a way to
un-configure a section without deleting field values.

### 16. Pressing `/` to search fields

On tabs with many fields (e.g., JIRA with 24 fields), pressing `/` opens
a search bar at the top of the section. Typing filters fields by label
text (case-insensitive substring match). Group headers with no matching
fields are hidden. `esc` clears the filter and closes the search bar,
restoring the full field list.

Before (full JIRA tab):
```
|  General    [JIRA]    File Copy    Worktrees                           |
+------------------------------------------------------------------------+
|  -- Connection -----------------------------------------               |
|  > Host              https://jira.example.com                          |
|    Me                user@example.com                                  |
|  -- Filters --------------------------------------------               |
|    Priority          High                                              |
|    ...                                                                 |
```

After pressing `/` and typing `attach`:
```
|  General    [JIRA]    File Copy    Worktrees                           |
+------------------------------------------------------------------------+
|  / attach_                                                             |
|                                                                        |
|  -- Markdown -------------------------------------------               |
|  > Include Attach.   yes                                               |
|                                                                        |
|  -- Attachments ----------------------------------------               |
|    Enabled           yes                                               |
|    ...                                                          1/2    |
|                                                                        |
+------------------------------------------------------------------------+
|  esc clear search . up/dn navigate . enter edit                        |
+------------------------------------------------------------------------+
```

Only fields matching "attach" are shown. Groups without matches
(Connection, Filters) are hidden. The position indicator reflects the
filtered count.

### 17. Pressing `enter` on a worktree entry -- worktree editor overlay

Worktree entries are edited via an overlay, similar to the rule editor
(example 9). The overlay has three fields: Branch (string), Merge Into
(string), and Description (string).

```
+-- gbm config ----------------------------------------------------------+
|  General    JIRA    File Copy   [Worktrees]                            |
+--##################################################################--+
|####################################################################---|
|#####+-  Worktrees > Edit "feature/auth" --+########################--|
|#####|                                      |########################--|
|#####|  > Branch          feature/auth      |########################--|
|#####|    Merge Into       main             |########################--|
|#####|    Description      Auth feature     |########################--|
|#####|                                      |########################--|
|#####|  up/dn navigate . e edit . enter confirm . esc cancel            |
|#####+-----------------------------------------+########################|
|####################################################################---|
+------------------------------------------------------------------------+
|  editing worktree...                                                   |
+------------------------------------------------------------------------+
```

- `up`/`dn` navigates between the three fields.
- `e` edits the focused field inline (opens a text input).
- `enter` (when not editing a field) confirms changes and closes the overlay.
- `esc` discards changes and closes the overlay.

When adding a new worktree (`a`), a text input first prompts for the
worktree name (the map key), then opens the editor overlay with empty
fields.

---

## Architecture

### Principles

1. **Metadata-driven rendering** -- fields defined as data, not hardcoded views
2. **Type-aware editing** -- each field type gets the right editor
3. **Section isolation** -- each tab is its own model; only the active one receives updates
4. **Edit-in-place** -- no separate "edit mode" screen; inline editing within the form
5. **Dirty tracking** -- know what changed (with count), warn on quit with unsaved changes
6. **Minimum terminal size** -- 60x16 minimum; show "terminal too small" below that. Handle `WindowSizeMsg` to reflow layout on resize.

### Confirmation Patterns

Two confirmation styles are used, chosen by the scope of the action:

- **Inline `y/n` confirmation** -- for single-field or single-item actions: reset field (`r`),
  delete rule (`d`), delete worktree entry (`d`). A prompt appears directly below the focused
  item. Press `y` to confirm, `n` or `esc` to cancel.

- **Overlay confirmation** -- for multi-field or global actions: quit with unsaved changes (`q`),
  reset all fields (`R`), external change overwrite (during save). A centered modal overlay
  appears with buttons (e.g., `[Save & Quit]  Discard  Cancel`). Arrow keys or tab to
  select, `enter` to confirm, `esc` to cancel.

### Field Types

| Type | Display | Edit Behavior |
|------|---------|---------------|
| `string` | value text | inline text input |
| `sensitive_string` | `********` when unfocused, revealed on focus | inline text input |
| `int` | numeric value | inline text input with numeric validation |
| `bool` | colored `yes` / `no` (green/red) | toggle on enter (no text input needed) |
| `string_list` | comma-separated preview | overlay for list editing |
| `object_list` | summary line per item | overlay for item editing |

Long display values are truncated with `...`. When a field is focused and
its value is truncated, the full value is shown on a line above the
description. If the field has both a long value and a description, both
lines appear below the field label: full value first, description second.
Text inputs scroll horizontally during editing (Bubble Tea native behavior).

### Component Hierarchy

```
ConfigModel (root)
|
+-- TabBar
|   renders:   [General]    JIRA    File Copy    Worktrees
|   keys:      tab / shift-tab to switch
|   badges:    (!) on tabs with validation errors after failed save
|
+-- SectionModel[]     (one per tab, only active one receives updates)
|   |
|   +-- ScrollViewport  (auto-scrolls to focused field, shows position)
|   |   renders:        field list with scroll indicator (e.g., 4/24)
|   |
|   +-- GroupHeader     (visual only, not focusable)
|   |   renders:        -- Connection --------------------------
|   |
|   +-- FieldRow[]      (focusable, one per config field)
|   |   |
|   |   |  BROWSING state:
|   |   |  renders:     > Host              https://jira.example.com
|   |   |  (sensitive): > Secret Field      ********
|   |   |  (sensitive, focused): > Secret Field  actual-value
|   |   |
|   |   |  EDITING state (string/int/sensitive_string):
|   |   |  renders:     > Host              +------------------+
|   |   |                                   | https://jira..._ |
|   |   |                                   +------------------+
|   |   |
|   |   |  EDITING state (bool):
|   |   |  no text input -- toggles value on enter, colored yes/no
|   |   |
|   |   +-- ListOverlay   (string_list / object_list fields)
|   |       renders:       centered modal over dimmed form
|   |       title:         breadcrumb path (e.g., JIRA > Filters > Status)
|   |
|   +-- EntryList        (Worktrees tab, File Copy rules)
|       renders:          summary rows (name -> branch, merge)
|       keys:             a add, d delete, enter edit overlay
|
+-- SearchFilter        (activated by /, filters fields by label text)
|   renders:            / search bar at top of section
|   keys:              type to filter, esc to clear and close
|
+-- HelpOverlay         (activated by ?)
|   renders:            full keybinding reference grouped by context
|   keys:              ? or esc to close
|
+-- StatusBar
    renders:   [N modified] tab/S-tab . up/dn . ? help . s save
    changes:   keybindings shown depend on current state (top 3-4 only)
    shows:     [new file] when no config exists on disk
```

### State Machine

The state determines which keys are active and what the view looks like.

```
                         +--------------------+
           tab/shift-tab |                    |
                         |    +----------+    |
                         +--->| Browsing |<---+
                              +----+-----+
                     enter on      |       enter on
                     string/int    |       bool
                         +---------+----------+
                         |                    |
                         v                    v
                  +------------+        (toggle value,
                  |  Editing   |         stay in Browsing)
                  +------+-----+
                         |
              +----------+----------+
              | enter    |          | esc
              | (valid)  |          | (cancel)
              |          |          |
              v          |          v
         save value      |     discard edit
         mark dirty      |     restore value
              |          |          |
              +----------+----------+
                         |
                         v
                  +----------+
              +-->| Browsing |<------------------------+
              |   +-+-+-+-+-++                         |
              |    | | | | |                           |
          esc |  s | |q | | ?           /              |
              |    | |  | |                            |
              |    v |  v v                            |
              |+------+|+----------------+             |
              ||Saving||| Quit Guard     |             |
              |+--+---+|| (if dirty)     |             |
              |   |    |+---+-------+----+             |
              |   |    |    |       |                   |
              |   v    |save+quit  discard              |
              |validate|    |       |                   |
              |   |    |    v       v                   |
              |   +-ok-+-> write --> exit               |
              |   |    |                                |
              |   +-err-+-> errors overlay              |
              |        |   up/dn select error           |
              |        |   enter -> jump to field ------+
              +--------+   esc -> close ----------------+


    Browsing --?--> +----------+ --?/esc--> Browsing
                    |  Help    |
                    | Overlay  |
                    +----------+

    Browsing --/--> +----------+ --esc----> Browsing
                    | Search   |
                    | Filter   |
                    | type to  |
                    | filter   |
                    +----------+

    enter on
    string_list
         |
         v
  +--------------+
  | List Overlay |
  |              |
  | up/dn select |
  | a  add item  |
  | d  delete    |
  |    (confirm) |
  +------+-------+
         |
         +-- enter ---- (confirm, persist changes)
         |
         +-- esc ------ (cancel, discard changes)
         |
         v
    +----------+
    | Browsing  |
    +----------+
```

**Note on `tab` during Editing state:** `tab` and `shift-tab` are ignored
during Editing state (no-op). The user must press `enter` or `esc` to
return to Browsing before switching sections. This prevents accidental
section switches while editing a field value.

### Field Metadata Definition

Each field is described by metadata. The section model iterates over metadata to render and edit fields.

```go
type FieldMeta struct {
    Key         string        // yaml path: "jira.attachments.max_size_mb"
    Label       string        // display: "Max Size (MB)"
    Type        FieldType     // String, SensitiveString, Int, Bool, StringList, ObjectList
    Group       string        // visual group: "Connection", "Filters", etc.
    Description string        // hint text shown below field when editing
    Default     any           // default value for reset
    Validate    func(any) error // field-level validation (optional)
}
```

The `SensitiveString` type displays as `********` when the field is
unfocused. When the user navigates to the field (focus), the actual
value is revealed. This avoids needing a separate toggle keybinding.

Sections are defined declaratively:

```go
var generalFields = []FieldMeta{
    {Key: "default_branch", Label: "Default Branch", Type: String, Group: ""},
    {Key: "worktrees_dir", Label: "Worktrees Directory", Type: String,
     Description: "Supports templates: {gitroot}, {branch}, {issue}"},
}

var jiraFields = []FieldMeta{
    // Connection
    {Key: "jira.host", Label: "Host", Type: String, Group: "Connection"},
    {Key: "jira.me", Label: "Me", Type: String, Group: "Connection"},
    // Filters
    {Key: "jira.filters.priority", Label: "Priority", Type: String, Group: "Filters"},
    {Key: "jira.filters.type", Label: "Type", Type: String, Group: "Filters"},
    {Key: "jira.filters.component", Label: "Component", Type: String, Group: "Filters"},
    {Key: "jira.filters.reporter", Label: "Reporter", Type: String, Group: "Filters"},
    {Key: "jira.filters.assignee", Label: "Assignee", Type: String, Group: "Filters"},
    {Key: "jira.filters.order_by", Label: "Order By", Type: String, Group: "Filters"},
    {Key: "jira.filters.status", Label: "Status", Type: StringList, Group: "Filters"},
    {Key: "jira.filters.labels", Label: "Labels", Type: StringList, Group: "Filters"},
    {Key: "jira.filters.custom_args", Label: "Custom Args", Type: StringList, Group: "Filters"},
    {Key: "jira.filters.reverse", Label: "Reverse", Type: Bool, Group: "Filters"},
    // Markdown
    {Key: "jira.markdown.filename_pattern", Label: "Filename Pattern", Type: String, Group: "Markdown"},
    {Key: "jira.markdown.max_depth", Label: "Max Depth", Type: Int, Group: "Markdown"},
    {Key: "jira.markdown.include_comments", Label: "Include Comments", Type: Bool, Group: "Markdown"},
    {Key: "jira.markdown.include_attachments", Label: "Include Attach.", Type: Bool, Group: "Markdown"},
    {Key: "jira.markdown.use_relative_links", Label: "Relative Links", Type: Bool, Group: "Markdown"},
    {Key: "jira.markdown.include_linked_issues", Label: "Linked Issues", Type: Bool, Group: "Markdown"},
    // Attachments
    {Key: "jira.attachments.enabled", Label: "Enabled", Type: Bool, Group: "Attachments"},
    {Key: "jira.attachments.max_size_mb", Label: "Max Size (MB)", Type: Int, Group: "Attachments"},
    {Key: "jira.attachments.directory", Label: "Directory", Type: String, Group: "Attachments"},
    {Key: "jira.attachments.download_timeout_seconds", Label: "Timeout (sec)", Type: Int, Group: "Attachments"},
    {Key: "jira.attachments.retry_attempts", Label: "Retry Attempts", Type: Int, Group: "Attachments"},
    {Key: "jira.attachments.retry_backoff_ms", Label: "Retry Backoff", Type: Int, Group: "Attachments"},
}

var worktreeFields = []FieldMeta{
    {Key: "branch", Label: "Branch", Type: String, Group: ""},
    {Key: "merge_into", Label: "Merge Into", Type: String, Group: ""},
    {Key: "description", Label: "Description", Type: String, Group: ""},
}
```

### Reading and Writing Config Values

The metadata `Key` field uses dot-path notation (`"jira.attachments.max_size_mb"`). A pair of accessor functions get/set values on the `Config` struct:

```go
func GetConfigValue(cfg *Config, key string) any
func SetConfigValue(cfg *Config, key string, value any) error
```

These use a switch on the key (not reflection) to keep it type-safe and explicit. Adding a new config field means adding one `FieldMeta` entry and one case in each accessor.

### Dirty Tracking

When a field is edited, the new value is compared against the original loaded value. A simple `map[string]bool` tracks which keys have been modified. The status bar shows a dirty count (`[N modified]`, e.g., `[2 modified]`) when any fields have changed.

```go
type DirtyTracker struct {
    original map[string]any  // snapshot at load time
    current  map[string]any  // current values
}

func (d *DirtyTracker) IsDirty() bool
func (d *DirtyTracker) DirtyCount() int    // for [N modified] display
func (d *DirtyTracker) DirtyKeys() []string
func (d *DirtyTracker) ResetKey(key string) // restore to original (last-saved) value
func (d *DirtyTracker) ResetAll()           // restore all keys to original (last-saved) values
func (d *DirtyTracker) MarkClean()          // after save
```

### Validation

Two levels:

1. **Field-level** -- runs on edit commit (enter). If invalid, shows inline error and keeps focus.

2. **Save-level** -- runs before writing. Checks cross-field constraints. Shows error overlay.

### Keybindings

| Context | Key | Action |
|---------|-----|--------|
| Browsing | `tab` / `shift-tab` | next/prev section tab |
| Browsing | `up` / `dn` / `j` / `k` | navigate fields |
| Browsing | `{` / `}` | jump to prev/next group header |
| Browsing | `g` / `G` | jump to first/last field |
| Browsing | `enter` | edit field (or toggle bool) |
| Browsing | `s` | save config |
| Browsing | `r` | reset field to last-saved value (confirm if dirty) |
| Browsing | `R` | reset all fields to last-saved values (confirm overlay) |
| Browsing | `/` | open field search/filter |
| Browsing | `?` | open help overlay |
| Browsing | `a` | add entry (File Copy rules / Worktrees context) |
| Browsing | `d` | delete entry (File Copy rules / Worktrees context, with confirmation) |
| Browsing | `q` / `ctrl-c` | quit (with dirty guard) |
| Editing | `enter` | commit edit |
| Editing | `esc` | cancel edit |
| Editing | `tab` / `shift-tab` | (ignored -- no-op) |
| Editing | `ctrl-z` | undo within text input (Bubble Tea native) |
| Search | _(type)_ | filter fields by label text |
| Search | `esc` | clear filter and close search |
| List overlay | `up` / `dn` | select item |
| List overlay | `a` | add new item (opens text input) |
| List overlay | `d` | delete selected item (with confirmation) |
| List overlay | `enter` | confirm and persist changes |
| List overlay | `esc` | cancel and discard changes |
| Editor overlay | `up` / `dn` | navigate fields |
| Editor overlay | `e` | edit focused field |
| Editor overlay | `enter` | confirm and close overlay |
| Editor overlay | `esc` | cancel and discard changes |
| Help overlay | `?` / `esc` | close help |
| Errors overlay | `up` / `dn` | select error |
| Errors overlay | `enter` | jump to error field |
| Errors overlay | `esc` | close overlay |

## What We Take From Surge

- **Metadata-driven fields** -- define once, render everywhere
- **Type-aware editing** -- bools toggle, strings get inputs, paths could get pickers
- **Tab bar for categories** -- clear top-level navigation
- **Centralized keybindings** -- per-state key maps to avoid conflicts
- **Responsive layout** -- calculate widths at render time
- **Modal overlays** -- `lipgloss.Place()` with dimmed background for list editing
- **Overlay interaction pattern** -- `enter` confirms/closes, `e` edits, arrows navigate

## What We Do Differently

- **Dot-path keys** instead of flat settings -- supports nested config naturally
- **Grouped fields within tabs** -- visual separators for sub-sections (Surge uses separate tabs for everything)
- **List/object editing overlays** -- Surge doesn't have array-typed config values
- **Edit-in-place** -- field becomes editable inline rather than a separate input modal
- **Dirty tracking with per-field count** -- Surge saves on close; we show `[N modified]` and list changed fields
- **Validation at two levels** -- field-level on edit, cross-field on save (with jump-to-error navigation)
- **YAML comment preservation** -- uses `go-yaml/yaml.v3` node API to read/write, keeping user comments intact
- **Sensitive field masking** -- API tokens masked when unfocused, revealed on focus
- **Scrolling with section jumps** -- auto-scroll viewport + `{`/`}` section jump + `g`/`G` top/bottom
- **Field search** -- `/` to filter fields by label on tabs with many fields
- **Map-based entry editing** -- Worktrees tab edits a `map[string]WorktreeConfig` via entry list + overlay

## File Structure (proposed)

```
cmd/service/
  config_cmd.go            -- cobra command, wires config model to tea.Program
  tui_config_adapter.go    -- adapter between service.Config and TUI

pkg/tui/config/
  model.go                 -- ConfigModel (root): tabs + active section + status bar
  section.go               -- SectionModel: scrollable form for one tab
  field_meta.go            -- FieldMeta definition + field type enum (incl. SensitiveString)
  field_row.go             -- FieldRow: label + value + inline editor + sensitive masking
  list_overlay.go          -- StringList editing overlay (a/d/enter/esc)
  rule_overlay.go          -- Rule editor overlay (source worktree + files sub-form)
  worktree_overlay.go      -- Worktree editor overlay (branch + merge_into + description)
  help_overlay.go          -- Full keybinding reference overlay (?)
  search.go                -- Field search/filter (/)
  accessors.go             -- GetConfigValue / SetConfigValue by dot-path key
  dirty.go                 -- DirtyTracker (with DirtyCount for [N modified])
  keys.go                  -- keybinding definitions per state
  sections.go              -- field metadata declarations (generalFields, jiraFields, worktreeFields, etc.)
  validate.go              -- field-level + save-level validation (with jump-to-error)
  yaml_node.go             -- go-yaml/yaml.v3 node-based read/write (comment preservation)
```

## Resolved Questions

1. **Empty optional sections show a placeholder.** If JIRA isn't configured, show "JIRA integration is not configured. Press enter to set up." Pressing enter populates with all fields at defaults.

2. **Config file comments are preserved.** Use `go-yaml/yaml.v3` node-based API. Read into `yaml.Node` tree, modify values in-place, write back. User comments survive TUI edits.

3. **Environment variable references shown raw.** The TUI shows `${JIRA_API_TOKEN}` as-is (not resolved). Sensitive fields are masked when unfocused but reveal the raw value (including `${...}`) on focus.

4. **Create mode on first run.** `gbm config` with no `.gbm/config.yaml` creates a default config in memory using detected values (`getDefaultBranch()`, `worktrees_dir: worktrees`). Saving creates the file. Status bar shows `[new file]`.

5. **Unknown YAML keys are preserved on save.** Config keys not represented in the TUI (e.g., `remotes:`, future fields from newer versions) must survive a save round-trip. This constrains the implementation to node-level YAML manipulation -- never unmarshal into a struct and re-marshal, as that would drop unknown keys.

## Out of Scope (v1)

- **Remotes configuration** -- the `remotes:` section is not editable in the config TUI in v1. Remotes are preserved as-is during save (see resolved question 5).

## V2 Roadmap

Features deferred from v1:

- **Miller-columns file picker** -- A yazi-style browser for selecting files in file_copy rules. Three-column layout (parent | current | preview), lazy directory loading, space to toggle selection, `h`/`l` navigation. Replaces manual path entry in the rule editor.
- **Terminal size graceful degradation** -- Collapse layouts on narrow terminals (hide descriptions, truncate labels, two-column miller browser on <100 cols). V1 shows "terminal too small" below 60x16.
