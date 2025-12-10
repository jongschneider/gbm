# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gbm` (Git Branch Manager) is a CLI tool for managing git branches. The project is in early development stages.

## Project Structure

```
cmd/gbm/        # Main application entry point
internal/git/   # Git service layer (interface to git operations)
```

The main.go imports a `cmd` package that doesn't exist yet. This package will contain the CLI command definitions and execution logic (likely using cobra or a similar CLI framework).

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

## Architecture Notes

### Missing cmd Package
The main.go references `gbm/cmd` package which doesn't exist yet. This package should contain:
- CLI command definitions (root command, subcommands)
- Command execution logic
- Logging setup (CloseLogFile() and PrintError() functions)
- Execute() function to run the CLI

### Git Service Layer
The `internal/git` package provides a service abstraction for git operations. Currently just a skeleton struct - will need methods for branch operations like:
- Listing branches
- Creating/deleting branches
- Switching branches
- Branch status/information

### Expected Dependencies
Based on the structure, likely dependencies will include:
- CLI framework (cobra/urfave/cli)
- Git library (go-git or exec wrapper)
- Logging library
