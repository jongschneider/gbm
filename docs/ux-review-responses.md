# UX Review Responses

Responses to the UX review of `config-tui-design.md`, captured during walkthrough.

---

## Round 1 (prior review)

### 1. Config schema mismatch with actual code

**Issue:** The design references fields (`jira.username`, `jira.api_token`, `jira.jql`, `jira.branch_prefix`) that don't exist in the actual `JiraConfig` struct.

**Response:** Design is outdated. Update the design doc to match the actual `JiraConfig` struct (`Host`, `Me`, `Filters`, `Markdown`, `Attachments`). Remove references to `username`, `api_token`, `jql`, `branch_prefix`, `enabled`.

### 2. No way to disable optional sections

**Issue:** Users can set up JIRA via `enter` on the placeholder, but there's no mechanism to un-configure it.

**Response:** Add an "Enabled" boolean toggle as the first field in each optional section (JIRA, File Copy). When toggled off, remaining fields are hidden and the section is omitted from the saved YAML. When toggled on, fields appear below.

### 3. Rule editor footer ambiguity

**Issue:** Footer shows `enter edit • enter done • esc cancel` — `enter` mapped to two actions.

**Response:** Follow Surge's overlay pattern. `enter` always confirms/closes the overlay. Arrow keys navigate between fields. To edit a field, use a different key (e.g., `e` or `space`). Footer: `↑↓ navigate • e edit • enter confirm • esc cancel`. This avoids the ambiguous double-`enter` in the current design.

### 4. Nested overlay behavior unspecified

**Issue:** Rule editor can open a list overlay inside it. `esc` behavior and state preservation across overlay levels not documented.

**Response:** Standard modal stack. `esc` closes only the innermost overlay, returning to the one beneath with its state intact. Each overlay level is independent — closing the inner list overlay returns to the rule editor with its state preserved. `enter` (confirm) in the inner overlay saves changes to the parent overlay's state.

### 5. `/` search has no walkthrough

**Issue:** Search is referenced but has zero interaction examples.

**Response:** Add a walkthrough section to the design doc (Section 16). Show how `/` opens a search bar on the JIRA tab, how typing filters fields by label text, what happens to group headers during filtering (hide empty groups), and how `esc` clears the filter and restores the full list. V1 scope.

### 6. No error badges on tabs

**Issue:** After failed save, no persistent indicator on tab labels showing which tabs have errors.

**Response:** Yes — add error badges on tab labels (e.g., `General (!)` or red marker) after a failed save. Badges persist until the error is fixed, then clear automatically.

### 7. Write failure errors unaddressed

**Issue:** Design covers validation failures but not I/O failures (disk full, permission denied).

**Response:** Show an error overlay (same visual treatment as validation errors) with a distinct title like "Save Failed" and the full error message (file path + OS error). Dismiss with `esc`.

### 8. No "reset all" action

**Issue:** `r` resets one field; no bulk reset.

**Response:** Add `R` (capital) keybinding to reset all dirty fields to their last-saved values. Shows a confirmation overlay listing all fields that will be reset before proceeding.

### 9. Long value display unclear

**Issue:** How full values display for focused fields that already have a description string.

**Response:** When focused and the value is truncated, show the full value on one line above the description. Two lines of hint text appear under the field label: full value first, description second.

### 10. `tab` during Editing state

**Issue:** Behavior of `tab` key during editing not documented.

**Response:** `tab` is ignored (no-op) during Editing state. Prevents accidental section switching. User must `enter` or `esc` to exit editing first.

### 11. Auto-advance on edit commit

**Issue:** `enter` then `down` repeated per field is slow on 20-field tabs.

**Response:** No auto-advance. After committing an edit, focus stays on the current field. User explicitly presses `down` to move. More predictable, lets user verify the value.

### 12. Sensitive field auto-reveals on focus

**Issue:** Navigating past API Token reveals it — screen-sharing risk.

**Response:** Keep current design: auto-reveal on focus. Accepted trade-off for simplicity.

### 13. No pre-save backup

**Issue:** No `.bak` file written before overwriting.

**Response:** Yes — always write `.gbm/config.yaml.bak` before overwriting. Cheap safety net for recovery.

### 14. Unknown-key preservation

**Issue:** Config keys not in the TUI must survive save round-trip.

**Response:** Yes — add as an explicit hard requirement in the design doc. Unknown YAML keys (e.g., `remotes:`, future fields from newer versions) must be preserved on save. This constrains implementation to node-level YAML manipulation.

### 15. Missing sections (remotes, worktrees map)

**Issue:** Not addressed in the design.

**Response:** Add a new **Worktrees** tab (4th tab: General, JIRA, File Copy, Worktrees). Shows a list of worktree entries (like File Copy rules). Each entry is a focusable row showing `name → branch (merge_into)`. `enter` opens an editor overlay with fields for branch, merge_into, and description. `a` adds, `d` deletes. `remotes` remains out of scope for v1 — add a note in the design doc.

### 16. Inconsistent confirmation patterns

**Issue:** Some use inline `y/n`, others use overlay confirmations.

**Response:** State the rule explicitly in the design doc: single-field actions (reset field, delete rule) use inline `y/n` confirmation. Multi-field/global actions (quit with unsaved changes, reset all, external change overwrite) use overlay confirmations.

### 17. Flash duration unspecified

**Issue:** How long does the save success message persist?

**Response:** Timed — 3 seconds. Flash auto-clears after 3 seconds, reverting to the normal status bar content.

---

## Round 2 (UX agent review)

### 18. No handling for corrupt/unparseable config files (Critical)

**Issue:** The design covers "no config" and "valid config" but not what happens when `.gbm/config.yaml` contains invalid YAML (syntax error, wrong types, truncated file).

**Response:** Show the TUI with an error banner and offer to open the raw YAML in `$EDITOR` for manual fixing. The TUI should not silently load defaults over a corrupt file.

### 19. `FieldMeta.Default` conflicts with reset behavior (Critical)

**Issue:** `FieldMeta.Default` is described as "default value for reset" but `r` resets to the last-saved (disk) value via `DirtyTracker.original`. The field name is misleading.

**Response:** Remove `Default` from `FieldMeta` entirely. First-run defaults are handled separately in the config creation logic, not in field metadata. Reset always uses `DirtyTracker.original`.

### 20. `enter` vs `e` inconsistency across contexts (Important)

**Issue:** In browsing mode `enter` edits; in editor overlays `e` edits and `enter` confirms. Users will press `enter` to edit inside an overlay and accidentally close it.

**Response:** Make `e` the universal edit key at all levels. Redefine `enter` in browsing mode as save-and-quit (form submit), with full validation and dirty guard. This creates a consistent model:

| Context | `e` | `enter` |
|---------|-----|---------|
| Browsing | edit focused field | save & quit (with guard) |
| Editing a field | — | commit value |
| Editor overlay | edit focused field | confirm & close overlay |
| List overlay | — | confirm & close overlay |

`enter` always means "I'm done at this level, confirm and go up." `e` always means "dive into this thing."

### 21. No per-field error indicators after save validation (Important)

**Issue:** After save validation fails and the error overlay is dismissed, individual fields show no error markers — only tab-level badges. Users must remember which fields failed or re-trigger save.

**Response:** Show inline error markers on each field that failed validation (red indicator + inline error text). Clears automatically when the value is corrected.

### 22. `esc` in list overlay discards ALL changes (Important)

**Issue:** Adding 3 items, deleting 1, then pressing `esc` loses all 4 operations. Risky for complex lists.

**Response:** If changes were made in the overlay, `esc` shows "Discard N changes? y/n" before closing. If no changes were made, `esc` closes immediately.

### 23. Bool field color accessibility (Important)

**Issue:** Green/red for yes/no is indistinguishable for red-green colorblind users. Muted unfocused state may have low contrast.

**Response:** Use bold weight for `yes` and dim/faint for `no` instead of relying on color alone. Works in monochrome terminals too. Color can still be applied as an additional signal.

### 24. `ctrl-c` during editing unspecified (Important)

**Issue:** If you're mid-edit on a field and press `ctrl-c`, the design doesn't specify whether it cancels the edit or quits the program.

**Response:** Two-step: first `ctrl-c` cancels the current edit (same as `esc`). A second `ctrl-c` triggers the quit guard. Prevents the surprising double-action of discarding an edit and prompting to quit simultaneously.

### 25. Enabled toggle lifecycle across save/reload (Important)

**Issue:** Toggling an optional section's Enabled to `no` and saving omits the section from YAML. But the design says "values reappear when toggled back." This is contradictory across save/reload cycles.

**Response:** Lossy — values are gone after save. Toggling off + saving removes the section from YAML entirely. Reloading shows the empty placeholder again. Simple and predictable. In-memory retention within a single session is fine (toggle off then back on without saving preserves values), but once saved and reloaded, values are lost.

### 26. Sensitive field auto-reveal on focus (Important)

**Issue:** Screen-sharing risk when navigating past API token fields.

**Response:** Keep auto-reveal. Accepted trade-off for simplicity. (Same as round 1 decision 12.)

### 27. Atomic writes not specified (Important)

**Issue:** No atomic write sequence documented. A crash mid-write could corrupt the config file.

**Response:** `.bak` is sufficient. Atomic writes are an implementation detail that doesn't need to be in the design doc. The backup file provides the recovery path.

### 28. Empty values displayed inconsistently (Important)

**Issue:** Worktree entries show `merge: --` for empty `merge_into`, but other empty fields show `(empty)`.

**Response:** Use `--` everywhere for empty/unset values. Short and clean.

### 29. JIRA group ordering (Important)

**Issue:** Connection > Filters > Markdown > Attachments. Review suggested reordering for first-time setup flow.

**Response:** Keep current order. Matches the YAML structure. Users can search with `/` if needed.

### 30. Worktree name validation (Important)

**Issue:** No validation specified for worktree names when adding via `a`. Duplicates, invalid characters, and renaming not addressed.

**Response:** Validate on add: reject duplicates and invalid characters (whitespace, null bytes — anything git doesn't allow in branch names). Allow renaming via an `r` key in the worktree editor overlay to change the map key.

### 31. Keybinding grouping in help overlay (Important)

**Issue:** ~20 keybindings in browsing mode including vim aliases. Help overlay shows a flat list.

**Response:** Group primary keys (up/dn, tab, e, s, q) in the main section. Show vim aliases (j/k, g/G, {/}) in a separate "Shortcuts" section.

### 32. "Me" label in JIRA Connection (Minor)

**Issue:** The label `Me` in the JIRA Connection group is unclear. Previous decision 13 resolved to use `jira.username`.

**Response:** Use "Username" as the label. Clear and matches the config key semantics.

### 33. File Copy section-level Enabled toggle (Minor)

**Issue:** Auto Copy has its own Enabled field, but there's no toggle to disable the entire File Copy section.

**Response:** Keep as-is. Auto Copy has its own Enabled toggle. Rules exist or they don't. No need for a section-level toggle.

### 34. `{`/`}` group jump discoverability (Minor)

**Issue:** Group jump keys are only shown in `?` help, not in the footer. Most useful on the JIRA tab with 4 groups and 24 fields.

**Response:** Add `{/} groups` to the footer when on the JIRA tab specifically.

### 35. `a`/`d` keys on non-list tabs (Minor)

**Issue:** Pressing `a` or `d` on General or JIRA tabs does nothing. Could confuse users.

**Response:** Silent no-op. The footer doesn't show `a`/`d` on those tabs, so there's no implied promise.

### 36. Search filter empty state (Minor)

**Issue:** `/` search doesn't specify what happens when no fields match.

**Response:** Keep empty space. The empty section body itself communicates no results. No placeholder message needed.

### 37. General tab sparseness (Minor)

**Issue:** General tab has only 2 fields and will feel empty.

**Response:** Fine as-is. Two fields is clean and fast. Future fields will fill it naturally.

### 38. Inline confirmations inside overlays (Minor)

**Issue:** When deleting a list item inside an overlay, where does the `y/n` confirmation appear?

**Response:** Within the overlay body, below the focused item. Keeps context local to the active modal.

### 39. Field-type-aware footer hints (Minor)

**Issue:** The footer could show different verbs based on the focused field type (e.g., `e toggle` for bools, `e edit` for strings, `e open` for lists).

**Response:** Yes — change the footer verb based on focused field type. Small touch that improves discoverability.
