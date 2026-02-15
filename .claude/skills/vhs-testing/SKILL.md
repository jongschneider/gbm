---
name: vhs-testing
description: VHS-based TUI testing workflow using .tape files for visual analysis. Use when testing interactive TUI commands, verifying visual layout/colors, or debugging TUI rendering with screenshots.
disable-model-invocation: false
---

# VHS-Based TUI Testing

Use this skill when:
- Testing interactive TUI components (selection lists, filters, prompts, forms)
- Verifying visual layout, colors, alignment, or cursor positioning
- Testing keyboard navigation flows and multi-step interactions
- Debugging TUI issues that require visual inspection
- Creating reproducible test cases for TUI bugs

---

## Workflow

```
1. Write .tape file     ->  Define interaction sequence + Screenshot commands
         |
2. Run `vhs <tape>`     ->  Execute in virtual terminal, capture screenshots
         |
3. Read PNG files       ->  Use Read tool to visually analyze TUI state
         |
4. Analyze & verify     ->  Check layout, colors, content, cursor position
         |
5. Iterate if needed    ->  Modify tape or fix code, repeat
```

### When to use VHS vs other testing approaches

| Approach | Use When | Example |
|----------|----------|---------|
| **VHS + Screenshot** | Visual verification needed (layout, colors, alignment) | Verifying highlight follows cursor |
| **testscript (txtar)** | Text output verification | Checking stdout prints expected value |
| **Unit tests** | Model state transitions | Testing Bubble Tea `Update()` logic |

## Prerequisites

VHS must be installed and available in PATH along with `ffmpeg` and `ttyd`:

```bash
vhs --version
# Install if needed: go install github.com/charmbracelet/vhs@latest
```

## Writing a Tape File

A tape file defines the terminal setup, interaction sequence, and screenshot capture points. See [vhs-reference.md](vhs-reference.md) for the complete VHS command reference.

### Minimal template

Use the template at [templates/basic-test.tape](templates/basic-test.tape) as a starting point:

```tape
Output test-output.gif
Set Shell "bash"
Set Width 800
Set Height 400
Set TypingSpeed 50ms

Require <your-binary>

# Launch the TUI
Type "<command>"
Enter
Sleep 300ms
Screenshot screenshots/initial.png

# Interact
Down
Sleep 100ms
Screenshot screenshots/afteraction.png

# Complete
Enter
Sleep 200ms
Screenshot screenshots/result.png
```

### Screenshot placement strategy

Place `Screenshot` commands at key points:

1. **After launch** - Verify initial render is correct
2. **After each interaction** - Verify state changed as expected
3. **After completion** - Verify final output/result

### Screenshot naming

**VHS parser limitation:** VHS tokenizes paths at numbers followed by non-alphanumeric characters. Use **number suffixes** (not prefixes) for sequential ordering:

```
screenshots/initial01.png
screenshots/navigation02.png
screenshots/filtered03.png
screenshots/selected04.png
```

Or use descriptive names without numbers:

```
screenshots/initial.png
screenshots/afterdown.png
screenshots/filtered.png
screenshots/result.png
```

### Timing guidelines

| Action | Sleep duration |
|--------|---------------|
| After starting a TUI component | `Sleep 300ms` |
| After navigation (Up/Down/Left/Right) | `Sleep 100ms` |
| After selection/confirmation | `Sleep 200ms` |
| After spawning a process | `Sleep 500ms` |

Use generous sleep values. It is better to wait too long than to capture an intermediate render state.

## Analyzing Screenshots

After running `vhs <tape>`, use the Read tool to view each captured PNG.

### What to verify

1. **Layout** - Items aligned? Cursor on expected item? Borders rendering?
2. **Color** - Selected item highlighted? Error states in red? Sufficient contrast?
3. **Content** - All expected options visible? Truncation correct? Counts accurate?
4. **State** - Selection indicator on right item? Checkboxes correct? Input placeholder visible?

### Analysis loop

```
1. Run: vhs test.tape
2. For each screenshot:
   - Read PNG with Read tool
   - Verify expected visual state
   - Note any issues
3. If issues found:
   - Identify root cause in TUI code
   - Fix the code
   - Re-run tape to verify fix
```

## Common Pitfalls

### Timing issues

Screenshots may capture intermediate states if Sleep is too short:

```tape
# BAD: May capture before TUI renders
Type "./my-tui"
Enter
Screenshot early.png          # May be blank or partial

# GOOD: Wait for full render
Type "./my-tui"
Enter
Sleep 300ms
Screenshot good.png
```

### Screenshot overwrites

```tape
# BAD: Overwrites same file
Screenshot output.png
Down
Screenshot output.png         # Overwrites previous!

# GOOD: Unique names
Screenshot screenshots/initial.png
Down
Screenshot screenshots/afterdown.png
```

### Binary not found

```tape
# BAD: Assumes binary is on PATH
Type "my-tui"

# GOOD: Use explicit path and guard
Require ./bin/my-tui
Type "./bin/my-tui"
```

### Terminal size

```tape
# BAD: Default size may truncate
Set Width 400
Type "./my-tui --long-option 'Very long text'"

# GOOD: Size terminal appropriately
Set Width 1000
Set Height 600
```

**Recommended sizes:**
- Simple TUIs (lists, prompts): 800x400
- Complex TUIs (tables, pagers): 1200x800
- Full-screen TUIs: 1920x1080

### Non-deterministic timing

VHS tests are timing-dependent. Mitigation:
1. Use generous `Sleep` values
2. For CI, consider VHS for demo generation only, not automated testing
3. Use `Wait` command if available in newer VHS versions

## Directory Structure

Organize VHS test artifacts consistently:

```
vhs/
+-- tui-tests/
|   +-- <component>/
|   |   +-- test-basic.tape
|   |   +-- screenshots/
|   +-- <component>/
|       +-- test-basic.tape
|       +-- screenshots/
+-- output/               # Generated output (gitignored)
```

## Additional Resources

- For complete VHS tape syntax, see [vhs-reference.md](vhs-reference.md)
- For reusable test recipes, see [recipes.md](recipes.md)
- For a starter tape file, see [templates/basic-test.tape](templates/basic-test.tape)

## Running Tests

```bash
# Build your binary first (e.g., make build, go build, etc.)

# Run a single tape
vhs path/to/test.tape

# Validate tape syntax without running
vhs validate path/to/*.tape
```
