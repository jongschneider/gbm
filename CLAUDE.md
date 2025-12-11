# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gbm` (Git Branch Manager) is a CLI tool for managing git worktrees based on a bare repository structure. It creates and manages git repositories with:
- A bare `.git` repository at the root
- Worktrees organized under `worktrees/` directory
- Configuration in `.gbm/config.yaml`

The tool uses Cobra for CLI and shell-executes `git` commands via `os/exec`.

## Project Structure

```
cmd/
  main.go              # Application entry point
  service/             # Cobra command definitions
    root.go            # Root command and CLI setup
    init.go            # 'init' command - create new repos
    clone.go           # 'clone' command - clone with worktree structure
    worktree.go        # 'worktree' command group for managing worktrees
internal/
  git/                 # Git operations via exec wrapper
    service.go         # Service struct with runCommand helper
    init.go            # Repository initialization logic
    clone.go           # Repository cloning logic
    worktree.go        # Worktree management logic
  jira/                # JIRA integration (future)
    service.go
```

## Architecture

### Command Flow
1. `cmd/main.go` calls `service.Execute()`
2. `cmd/service/root.go` defines the root Cobra command and registers subcommands
3. Each subcommand delegates to `internal/git.Service` methods
4. `internal/git.Service` wraps git commands using `os/exec`

### Git Service Layer
The `internal/git.Service` executes git commands via shell:
- `runCommand()` helper handles dry-run mode and command formatting
- All git operations use `exec.Command("git", ...)` with appropriate flags
- Supports `--git-dir` for bare repo operations and `-C` for worktree operations

### Key Technologies
- **Cobra**: CLI framework (already integrated)
- **os/exec**: Shell execution wrapper for git commands
- **Bare repositories**: All repos use `git init --bare` with worktrees

## Development Commands

This project uses `just` as the task runner. Run `just` to see all available commands.

### Just Commands

**Quick Development**
```bash
just                  # List all available commands
just build            # Build the gbm binary
just run [ARGS]       # Build and run with optional arguments
just clean            # Clean build artifacts
```

**Validation Pipeline**
```bash
just validate         # Run full validation: format, vet, lint, compile, test-changed
just quick            # Quick validation: format and vet only (for fast feedback)
```

**Code Quality**
```bash
just format           # Format changed Go files with gofmt
just vet              # Run go vet on packages with changes
just lint             # Run golangci-lint on changed packages
just lint-all         # Run golangci-lint on all packages
just compile          # Compile all packages to ensure they build
```

**Testing**
```bash
just test             # Run all tests with 10m timeout
just test-changed     # Run tests only for packages with changes
```

**Utilities**
```bash
just show-changed     # Show what Go files and packages have changed
```

### Direct Go Commands

Standard Go commands also work:
```bash
go build -o gbm cmd/gbm/main.go  # Build directly
go run cmd/gbm/main.go           # Run directly
go test ./...                    # All tests
go test ./internal/git           # Specific package
go test -v -run TestName         # Specific test
go mod tidy                      # Clean up dependencies
```

### Change Detection

The justfile targets use smart change detection that checks:
- Staged changes (`git diff --cached`)
- Unstaged changes (`git diff`)
- Untracked Go files

This allows for faster iteration by only validating what you've changed.

## Implementation Patterns

### Adding New Commands
1. Create command constructor in `cmd/service/` (e.g., `newFooCommand()`)
2. Register it in `cmd/service/root.go` via `rootCmd.AddCommand()`
3. Implement git logic in `internal/git/` (e.g., `Service.Foo()` method)
4. Use `Service.runCommand()` helper for all git exec calls to get dry-run support

### Adding Git Operations
All git operations should:
- Use `exec.Command("git", ...)` for git execution
- Call `s.runCommand(cmd, dryRun)` to execute with dry-run support
- Return formatted errors with command output: `fmt.Errorf("failed to X: %w\nOutput: %s", err, output)`
- Use `--git-dir` flag when operating on bare repos
- Use `-C <path>` flag when operating within worktrees

### Dry-Run Mode
The `runCommand()` method in `internal/git/service.go` handles dry-run:
- Formats commands for display using `formatCommand()`
- Prints `[DRY RUN] <command>` instead of executing
- All commands support `--dry-run` flag to preview operations
- Use FindGitRoot() instead of `git rev-parse --show-toplevel` for bare repo + worktree setups. It correctly finds the repo root even
  when run from inside a worktree by detecting /.git/worktrees/ in the git-dir path.