# CLAUDE.md

Guidance for working with `gbm` (Git Branch Manager) - CLI tool for git worktree management.

## Project Overview

- Bare `.git` repo root, worktrees in `worktrees/` dir
- Interactive Bubble Tea TUI + CLI (Cobra)
- JIRA integration for issue-based worktrees
- Config: `.gbm/config.yaml` (remotes, JIRA, file copying)
- Shell integration for auto-cd

Stack: Cobra (CLI), Bubble Tea (TUI), `os/exec` (git)

## Project Structure

```
cmd/service/          Cobra commands + TUI (Bubble Tea FSM)
internal/
  git/                Git operations (exec wrapper)
  jira/               JIRA API integration
  utils/              Shared helpers
testutil/             Test utilities
```

## Architecture

**Command flow:** CLI (Cobra) → git/jira services → `os/exec` calls

**Git layer:** All commands via `exec.Command("git", ...)`. Use `--git-dir` (bare repo) or `-C` (worktree). Dry-run: call `printDryRun()` before exec.

**TUI layer:** Bubble Tea FSM with states (list, create, delete, checkout, sync). See `worktree_tui.go`, `worktree_fsm.go`, `worktree_table.go`.

**JIRA layer:** Fetch issues, generate branch names, format for display.

## Development Commands

Use `just` (see `justfile`):

| Command | What |
|---------|------|
| `just` | List all |
| `just build` | Build binary → `./gbm` |
| `just run [ARGS]` | Build + run |
| `just validate` | format + vet + lint + compile + test-changed |
| `just quick` | format + vet only |
| `just format`, `just vet`, `just lint`, `just compile` | Specific checks |
| `just test` | All tests (10m timeout) |
| `just test-changed` | Tests for changed packages |
| `just install` | Build + install as `gbm2` to `/usr/local/bin` |

Or use Go directly:
```bash
go build -o gbm ./cmd           # Build
go run ./cmd                    # Run
go test ./...                   # All tests
go test ./internal/git -v       # Specific package
go mod tidy                     # Clean deps
```

Smart change detection: checks staged, unstaged, untracked Go files. Enables fast iteration.

## Implementation Patterns

### New Commands

1. Create constructor in `cmd/service/` (e.g., `newFooCommand()`)
2. Register in `cmd/service/root.go`: `rootCmd.AddCommand(...)`
3. Implement in service: `internal/git.Service`, `internal/jira.Service`, or Bubble Tea TUI
4. Follow dry-run pattern if applicable

### Git Operations

- Use `exec.Command("git", ...)`
- Check `if dryRun` → call `printDryRun(cmd)`
- Execute: `cmd.Output()` (capture) or `cmd.Run()` (stream)
- Errors: `fmt.Errorf("context: %w", err)`
- Flags: `--git-dir` (bare), `-C path` (worktree)

### Output Pattern

**stdout = data, stderr = messages** (no exceptions, no env checks)

```go
fmt.Println(path)                                    // data → stdout
fmt.Fprintf(os.Stderr, "✓ Switched to %s\n", name)  // message → stderr
```

**Why:** Shell integration (`path=$(gbm wt switch x)`), piping (`gbm wt list | xargs`), scripting.

**Examples:**
```bash
gbm wt switch feature-x       # /path/to/worktree (stdout) + message (stderr)
gbm wt add PROJ-123 feat/123  # Same pattern
gbm wt list                   # TUI on /dev/tty, selected path to stdout
```

## Testing

### Commands

```bash
just test              # All tests
just test-changed      # Changed packages only
go test ./internal/git # Specific package
go test -run TestName  # Specific test
go test -short ./...   # Unit tests only (skip E2E)
go test -cover ./...   # With coverage
```

### Structure

- **Unit tests:** `*_test.go` files, fast, isolated
- **E2E tests:** `e2e_test.go`, real commands
- **Utils:** `testutil/` (reusable helpers like `TestRepo`)

### Writing Tests

Use table-driven tests with subtests:

```go
testCases := []struct{
    name      string
    expectErr func(t *testing.T, err error)  // always non-nil, always called
    expect    func(t *testing.T, got T)      // always non-nil, always called
}{
    {"case1", func(t *testing.T, err error) { assert.NoError(t, err) }, ...},
}
for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        got, err := fn()
        tc.expectErr(t, err)
        tc.expect(t, got)
    })
}
```

With mocks: add `assertMocks func(t *testing.T, ...)` field.

Target: >80% unit, >70% overall.

## Configuration

Location: `.gbm/config.yaml`

```yaml
default_branch: main
worktrees_dir: worktrees          # or use templates: ../{gitroot}-wt, ~/dev/{gitroot}/wt

remotes:
  origin:
    url: git@github.com:user/repo.git
  upstream:
    url: git@github.com:org/repo.git

jira:
  enabled: true
  host: https://jira.company.com
  username: user@company.com
  api_token: ${JIRA_API_TOKEN}
  jql: "assignee = currentUser() AND status != Done"
  branch_prefix: feature/

file_copy:
  rules:
    - source: ".env"
      target: ".env"
      create_if_missing: true
  auto:
    enabled: true
    source_worktree: "{default}"
    copy_ignored: true
    copy_untracked: false
    exclude: ["*.log", "node_modules/", "build/"]
```

See `config.example.yaml` for full reference.

Flag overrides: `gbm wt add feat-x feat/x --base devel` (overrides `default_branch`)

## Troubleshooting

**Config missing:**
```bash
mkdir -p .gbm
cp config.example.yaml .gbm/config.yaml
# Edit with your settings
```

**TUI fails (`/dev/tty` error):**
- Requires interactive terminal: `gbm wt list` (not piped/redirected)
- Use `gbm wt switch feature-x` as alternative

**"failed to create worktree":**
```bash
git branch -a | grep <branch>    # Verify branch doesn't exist
git fetch                         # Update remotes
cd main-worktree && git status   # Must be clean
```

**File copy not working:**
```bash
cat .gbm/config.yaml | grep -A5 "auto:"  # Verify enabled
gbm wt switch main && ls .env            # Check source files exist
```

**Permission denied:**
```bash
chmod -R u+w worktrees/      # Fix ownership
ls -l worktrees/             # Verify
```

**Dirty worktree error:**
```bash
cd worktrees/<name>
git status
git add . && git commit -m "msg"  # or git stash
```

Built-in help: `gbm --help`, `gbm wt --help`, `gbm wt add --help`

## Performance

**Large repos:**
- Use TUI filtering: `gbm wt list` (type to filter)
- Worktree path templates spread across filesystem: `worktrees_dir: ../{gitroot}-wt`
- Exclude large dirs from file copy: `.gitignore` patterns in `file_copy.auto.exclude`

**Syncing:**
```bash
gbm wt sync --fetch        # Just fetch, no pull
git fetch origin $(git rev-parse --abbrev-ref HEAD)  # Single branch
```
