# Shell Integration Simplification Summary

## Decision: Remove Temp File Fallback

Based on your feedback, we're **eliminating temp files entirely** and relying solely on `/dev/tty` for TUI rendering.

---

## Before vs After

### Current Implementation (Complex)
```bash
# 60+ lines of shell code with temp file logic
gbm2() {
    if [ "$1" = "worktree" ] && [ "$2" = "switch" ] && [ $# -gt 2 ]; then
        local cmd_output=$(command gbm2 "$@" 2>/dev/null)
        # ... extract cd command with grep ...
    elif [ "$1" = "wt" ] && [ "$2" = "switch" ] && [ $# -gt 2 ]; then
        # Duplicate logic for wt alias
    elif [ "$1" = "worktree" ] && [ "$2" = "list" ]; then
        command gbm2 "$@"
        local switch_file="$TMPDIR/.gbm-switch-$$"
        # ... read temp file, cleanup ...
    elif [ "$1" = "wt" ] && ([ "$2" = "list" ] || ...); then
        # More temp file logic
    else
        command gbm2 "$@"
    fi
}
```

**Go code:**
- Creates temp file: `$TMPDIR/.gbm-switch-{PPID}`
- Writes path to temp file
- Relies on shell to cleanup

### New Implementation (Simple)
```bash
# 20 lines - single, clean pattern
gbm2() {
    if [[ ("$1" = "worktree" || "$1" = "wt") && \
          ("$2" = "switch" || "$2" = "sw" || "$2" = "s" || \
           "$2" = "add" || "$2" = "a" || \
           "$2" = "list" || "$2" = "ls" || "$2" = "l") ]]; then

        local result
        result=$(command gbm2 "$@" 2>/dev/stderr)
        local exit_code=$?

        [ $exit_code -eq 0 ] && [ -d "$result" ] && cd "$result"
        return $exit_code
    else
        command gbm2 "$@"
    fi
}
```

**Go code:**
- TUI renders to `/dev/tty`
- Outputs path to stdout
- **No temp files**

---

## What Gets Deleted

### Shell Integration (`cmd/service/shell-integration.go`)
**Remove:**
- All temp file path construction (`$TMPDIR/.gbm-switch-$$`)
- All temp file reading logic
- Duplicate handling for `wt` vs `worktree`
- Duplicate handling for command aliases
- Temp file cleanup logic

**Reduce:**
- 97 lines → ~40 lines (60% reduction)

### TUI Table (`cmd/service/worktree_table.go`)
**Remove:**
- `os.Getppid()` calls
- Temp file writing: `os.WriteFile(tmpFile, ...)`
- Temp file path construction
- Temp file cleanup: `os.Remove(tmpFile)`

**Add:**
- `/dev/tty` opening: `os.OpenFile("/dev/tty", os.O_RDWR, 0)`
- Bubble Tea options: `tea.WithInput(tty)`, `tea.WithOutput(tty)`

---

## Benefits

### Simplicity
- ✅ **67% less shell code**: 60+ lines → 20 lines
- ✅ **Single code path**: No special cases for TUI vs non-TUI
- ✅ **No file I/O**: Eliminated entirely from shell integration

### Consistency
- ✅ **All commands use stdout**: switch, add, list - same pattern
- ✅ **Uniform shell wrapper**: One condition handles all aliases

### Maintainability
- ✅ **Easier to understand**: Single stdout capture pattern
- ✅ **Fewer edge cases**: No temp file permissions, cleanup, stale files
- ✅ **Less state**: No files to manage between shell and Go

### User Experience
- ✅ **Same behavior**: Auto-cd works exactly as before
- ✅ **Cleaner**: No temp files cluttering `$TMPDIR`
- ✅ **Faster**: No file I/O overhead

---

## Technical Details

### How /dev/tty Works

**The Problem:**
When shell captures stdout: `result=$(command gbm2 wt list)`
- The TUI can't render because stdout is redirected to `result`
- User sees nothing

**The Solution:**
```go
// Open the controlling terminal directly
tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)

// TUI renders to /dev/tty (bypasses stdout capture)
program := tea.NewProgram(model,
    tea.WithInput(tty),
    tea.WithOutput(tty),
)

// After TUI exits, stdout is still available
fmt.Println(selectedWorktreePath)  // Goes to stdout for shell capture
```

**Result:**
- TUI renders normally (user sees the interface)
- Shell captures the path from stdout
- Best of both worlds!

### What About Non-Interactive Environments?

In CI/CD, scripts without TTY, containers, etc., `/dev/tty` won't be available.

**Our stance:**
- TUI commands (list) **require** an interactive terminal
- If `/dev/tty` fails, return an error
- This is **acceptable** - TUI isn't meant for non-interactive use

**Alternatives for non-interactive:**
- Use `gbm wt switch <name>` directly (no TUI)
- Parse `gbm worktree list` output (future: add `--format=json`)

---

## Task Updates

### Task 1.1.2: Shell Integration Wrapper
**Time:** 3-4 hours → **2 hours** (simpler!)
**Change:**
- Delete all temp file logic
- Single unified stdout capture
- 20 lines instead of 60+

### Task 1.1.4: TUI /dev/tty Rendering
**Time:** 3-4 hours → **2-3 hours**
**Change:**
- Remove temp file fallback
- Error if `/dev/tty` unavailable
- Delete all temp file code

### Total Time Saved
**Before:** ~7-8 hours for shell integration tasks
**After:** ~4-5 hours for shell integration tasks
**Savings:** ~3 hours + ongoing maintenance burden removed

---

## Migration Path

For users with existing shell integration:

1. **Update shell integration:**
   ```bash
   eval "$(gbm2 shell-integration)"
   ```

2. **Reload shell:**
   ```bash
   source ~/.zshrc  # or ~/.bashrc
   ```

3. **Test:**
   ```bash
   gbm2 wt list       # Should work, auto-cd after selection
   gbm2 wt switch foo # Should auto-cd
   gbm2 wt add bar    # Should auto-cd after creation
   ```

**No manual cleanup needed** - temp files stop being created automatically.

---

## Conclusion

By removing the temp file fallback and relying on `/dev/tty`:

✅ **Simpler code** - 67% less shell code
✅ **Single pattern** - unified stdout capture
✅ **No file I/O** - eliminated temp file overhead
✅ **Same UX** - auto-cd works identically
✅ **Less maintenance** - fewer edge cases

The trade-off (requiring `/dev/tty` for TUI) is acceptable because:
- Interactive TUI requires an interactive terminal anyway
- Non-interactive use cases can use non-TUI commands
- Simpler implementation = fewer bugs
