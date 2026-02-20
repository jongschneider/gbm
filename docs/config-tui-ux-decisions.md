# Config TUI — UX Decisions

Decisions captured from review of `config-tui-design.md`.

---

## Critical

### 1. `tab` key overloading

**Decision:** Defer. The miller-columns rule picker is cut to v2, which eliminates two of the three `tab` collisions. For v1, `tab`/`shift-tab` is used exclusively for section tab navigation. The rule picker panel-switching and worktree-cycling keybindings will be designed when the picker is built in v2.

---

## Major

### 2. Missing config fields

**Decision:** Add all missing JIRA fields to v1: `api_token`, `username`, `jql`, `branch_prefix`. They go in the JIRA tab's Connection group (alongside `host`).

**Sensitive value display (`api_token`):** Masked (`********`) when the field is unfocused. When the user navigates to the field (focus), the actual value is revealed (including `${ENV_VAR}` references). No extra keybinding needed — focus is the reveal trigger. This is a new `FieldType` variant (`SensitiveString`) or a `Sensitive bool` flag on `FieldMeta`.

### 3. List overlay `enter` ambiguity

**Decision:** `a` to add a new item (focuses the text input, type value, enter to confirm the add). `enter` (when not in the text input) commits all changes and closes the overlay. `esc` cancels and discards changes. `d` deletes the selected item. This matches the `a add rule` / `d delete rule` pattern used in the File Copy rules list for consistency.

### 4. Rule picker complexity

**Decision:** Miller-columns file browser is deferred to v2. For v1, file_copy rules are edited as follows:
- Each rule is a focusable row showing `source_worktree → file1, file2, ...`
- `enter` on a rule opens a sub-form with two fields: `source_worktree` (string) and `files` (string list, using the same list overlay as other string lists)
- `a` adds a new rule, `d` deletes the focused rule
- Users type file paths manually in the string-list overlay

### 5. JIRA tab scrolling

**Decision:** Auto-scroll viewport to keep the focused field visible. Show a scroll position indicator (e.g., `7/16` in the right margin or status bar). Add section-jump keybindings: `{`/`}` to jump between group headers, `g`/`G` for top/bottom of the field list.

### 6. Undo mechanism

**Decision:** Text-input level only. Bubble Tea's `textinput` component supports `ctrl-z` natively while in editing mode — that's sufficient. No field-level undo after committing an edit. Users can press `r` to reset a field to its last-saved value, or `esc` to cancel mid-edit.

**Clarification on `r` (reset):** `r` resets to the *last-saved* value (what was loaded from disk), not the schema default. This is less destructive and more useful.

---

## Minor

### 7. `r` reset confirmation

**Decision:** Confirm only if the field is dirty (has unsaved changes). Show `Reset to saved value? y/n`. If the field is already clean, `r` is a no-op.

### 8. Footer overflow + help overlay

**Decision:** Show only the 3-4 most important keybindings in the footer (context-dependent). Bind `?` to a full keybinding reference overlay, closeable with `?` or `esc`. This avoids footer truncation on narrow terminals and gives users a discoverable reference.

### 9. Bool field display

**Decision:** Follow Surge's pattern — render `yes`/`no` text with color: green (`SuccessAccent`) for yes, red (`ErrorAccent`) for no when focused. Muted/gray when unfocused. No checkbox characters. This matches the existing codebase convention in `fields/confirm.go`.

### 10. Validation overlay — jump to error

**Decision:** Yes. Selecting an error in the validation overlay (via `enter` or arrow keys + enter) navigates directly to the offending field: switches tab, scrolls to field, and focuses it. IDE-style error navigation.

### 11. Overflow handling

**Decision:** Handle all three overflow scenarios in v1:
- **Long field values:** Truncate display with `...` when unfocused. Show full value in description area on focus. Text input scrolls horizontally during editing (Bubble Tea native).
- **Rule summaries:** Truncate with count: `main → .env, config/ (+5 more)`.
- **Terminal resize:** Define minimum terminal size (60x16). Below that, show a "terminal too small" message. Handle `WindowSizeMsg` to reflow layout.

### 12. `d` delete key behavior

**Decision:** `d` always prompts for confirmation: `Delete "X"? y/n`. Applies in all contexts (list overlay items, rules list entries). Consistent and safe.

### 13. `jira.me` vs `jira.username`

**Decision:** Use `jira.username`. Self-documenting, matches common conventions. Update the design doc; `config.example.yaml` already uses this name.

### 14. YAML comment preservation

**Decision:** Use `go-yaml/yaml.v3` node-based API to preserve comments when saving. Read config into `yaml.Node` tree, modify values in-place, write back. This keeps user comments intact across TUI edits.

### 15. Small polish items

**Decision:** Address all three in v1:
- **Footer:** Show `tab/shift-tab` (not just `tab`) for section navigation.
- **Empty rules list:** Show `(no rules configured) — press a to add`.
- **Overlay breadcrumbs:** Overlay titles include context path, e.g., `JIRA > Filters > Status` instead of just `Status`.

---

## Suggestions (all accepted for v1)

### 16. Dirty count in status bar

**Decision:** Show `[N modified]` instead of `[modified]` in the status bar. e.g., `[2 modified]`. Gives users a sense of how many fields they've changed.

### 17. `/` field search

**Decision:** Bind `/` to a field search/filter in browsing mode. Type to filter fields by label text. Clears on `esc`. Useful on the JIRA tab with 16+ fields.

### 18. `?` help overlay as a state

**Decision:** Add a `Help` state to the state machine. `?` in browsing mode opens a full keybinding reference overlay. Closeable with `?` or `esc`. The overlay shows all keybindings grouped by context (browsing, editing, list overlay, etc.).

### 19. File mtime check before save

**Decision:** Before writing config, check the file's modification time against when it was loaded. If the file changed on disk (external edit), show a warning overlay: `Config file was modified externally. Overwrite? y/n`. This prevents silent data loss.

### 20. First-run behavior (no config exists)

**Decision:** `gbm config` on a repo with no `.gbm/config.yaml` creates a default config in memory using detected values (`getDefaultBranch()`, `worktrees_dir: worktrees`). The TUI opens in normal editing mode with these defaults. Saving creates the file. The status bar shows `[new file]` instead of a file path.

