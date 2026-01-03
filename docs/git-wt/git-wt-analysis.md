# git-wt Analysis: Features, Patterns, and Strategies for gbm

## Executive Summary

**git-wt** is a minimalist Git subcommand that simplifies worktree management with a focus on developer experience through shell integration, automatic directory switching, and intelligent file copying. The tool emphasizes simplicity, configuration via git config, and seamless shell integration.

**Key Takeaway**: git-wt demonstrates a clean, focused approach to worktree management with excellent shell integration and configuration management patterns that could enhance gbm.

---

## What git-wt Does

### Core Functionality

1. **Simple Worktree Management**
   - `git wt` - List all worktrees in a clean table format
   - `git wt <branch>` - Switch to existing worktree or create new one
   - `git wt -d <branch>` - Safe delete (only if merged)
   - `git wt -D <branch>` - Force delete

2. **Automatic Branch Detection**
   - Creates new branch if it doesn't exist
   - Uses existing branch if found (local or remote)
   - Supports both branch names and directory names for switching

3. **Shell Integration**
   - Automatic `cd` to worktree after creation/switch
   - Shell completion for branches and worktree directories
   - Supports bash, zsh, fish, and PowerShell
   - Optional `--no-switch-directory` flag for completion-only

4. **File Copying to New Worktrees**
   - Copy .gitignore'd files (e.g., `.env` files)
   - Copy untracked files
   - Copy modified files
   - Exclude patterns via `.gitignore` syntax

5. **Flexible Configuration**
   - All config via `git config` (e.g., `wt.basedir`)
   - Flag overrides for single invocations
   - Template variables (e.g., `{gitroot}` in paths)

---

## How git-wt Does It

### Architecture & Design Patterns

#### 1. **Command Structure**
```
cmd/
  root.go     # Single Cobra command with all flags
  init.go     # Shell integration scripts
internal/git/
  worktree.go # Worktree operations
  config.go   # Configuration loading
  copy.go     # File copying logic
  branch.go   # Branch operations
  exec.go     # Git command wrapper
```

**Key Pattern**: Flat command structure - everything in root command with flags instead of subcommands

#### 2. **Configuration Management**

**Loading with Defaults**:
```go
func LoadConfig(ctx context.Context) (Config, error) {
    cfg := Config{}

    // BaseDir
    baseDir, err := GitConfig(ctx, configKeyBaseDir)
    if err != nil {
        return cfg, err
    }
    if len(baseDir) == 0 {
        cfg.BaseDir = "../{gitroot}-wt"  // Default value
    } else {
        cfg.BaseDir = baseDir[len(baseDir)-1]  // Use last value
    }

    // Boolean configs
    cfg.CopyIgnored = len(val) > 0 && val[len(val)-1] == "true"

    return cfg, nil
}
```

**Flag Overrides**:
```go
func loadConfig(ctx context.Context, cmd *cobra.Command) (git.Config, error) {
    cfg, err := git.LoadConfig(ctx)
    if err != nil {
        return cfg, err
    }

    // Apply flag overrides using cmd.Flags().Changed()
    if cmd.Flags().Changed("basedir") {
        cfg.BaseDir = basedirFlag
    }
    return cfg, nil
}
```

**Pattern**: Load from git config → Apply flag overrides → Use

#### 3. **Shell Integration Strategy**

**Output Format for Shell Integration**:
- **stdout**: Only the worktree path (for `cd` command)
- **stderr**: All git messages and progress

```go
// In AddWorktree function
cmd.Stdout = os.Stderr  // Git messages go to stderr
cmd.Stderr = os.Stderr

// Then print path to stdout for shell integration
fmt.Println(wtPath)
```

**Shell Wrapper Function** (bash/zsh):
```bash
git() {
    if [[ "$1" == "wt" ]]; then
        shift
        local result
        result=$(command git wt "$@")
        local exit_code=$?
        if [[ $exit_code -eq 0 && -d "$result" ]]; then
            cd "$result"  # Auto-cd if output is a directory
        else
            echo "$result"
            return $exit_code
        fi
    else
        command git "$@"
    fi
}
```

**Pattern**: Wrapper intercepts `git wt` commands, checks if output is directory, auto-cd

#### 4. **Path Resolution & Template Expansion**

```go
// Template expansion
func expandTemplate(ctx context.Context, s string) (string, error) {
    if strings.Contains(s, "{gitroot}") {
        repoName, err := RepoName(ctx)
        if err != nil {
            return "", err
        }
        s = strings.ReplaceAll(s, "{gitroot}", repoName)
    }
    return s, nil
}

// Path expansion (~, relative paths)
func ExpandPath(ctx context.Context, path string) (string, error) {
    // Expand ~
    if strings.HasPrefix(path, "~/") {
        home, err := os.UserHomeDir()
        path = filepath.Join(home, path[2:])
    }

    // Resolve relative path from main repo root (not worktree)
    if !filepath.IsAbs(path) {
        repoRoot, err := MainRepoRoot(ctx)
        return filepath.Join(repoRoot, path), nil
    }

    return filepath.Clean(path), nil
}
```

**Pattern**: Template expansion → Tilde expansion → Relative path resolution (from main repo root)

#### 5. **File Copying with gitignore Pattern Matching**

Uses `go-git/go-git/v5/plumbing/format/gitignore` for pattern matching:

```go
func CopyFilesToWorktree(ctx context.Context, srcRoot, dstRoot string, opts CopyOptions) error {
    var files []string

    if opts.CopyIgnored {
        ignored, err := listIgnoredFiles(ctx, srcRoot)
        files = append(files, ignored...)
    }

    // Build NoCopy matcher using gitignore patterns
    var noCopyMatcher gitignore.Matcher
    if len(opts.NoCopy) > 0 {
        var patterns []gitignore.Pattern
        for _, p := range opts.NoCopy {
            patterns = append(patterns, gitignore.ParsePattern(p, nil))
        }
        noCopyMatcher = gitignore.NewMatcher(patterns)
    }

    // Filter and copy
    for _, file := range files {
        pathComponents := strings.Split(file, string(filepath.Separator))
        if noCopyMatcher != nil && noCopyMatcher.Match(pathComponents, false) {
            continue  // Skip files matching NoCopy patterns
        }
        copyFile(src, dst)
    }
}
```

**Pattern**: Use git commands to list files → Filter with gitignore patterns → Copy

#### 6. **Worktree Discovery**

```go
func FindWorktreeByBranchOrDir(ctx context.Context, query string) (*Worktree, error) {
    worktrees, err := ListWorktrees(ctx)

    // First, try to find by branch name
    for _, wt := range worktrees {
        if wt.Branch == query {
            return &wt, nil
        }
    }

    // Then, try by directory name (relative path from base dir)
    baseDir, err := ExpandBaseDir(ctx, cfg.BaseDir)
    for _, wt := range worktrees {
        relPath, err := filepath.Rel(baseDir, wt.Path)
        if err != nil || strings.HasPrefix(relPath, "..") {
            continue
        }
        if relPath == query {
            return &wt, nil
        }
    }

    return nil, nil
}
```

**Pattern**: Branch name takes priority, then directory name

---

## Technologies & Libraries

### Core Dependencies

1. **Cobra** (`spf13/cobra`) - CLI framework
   - Single root command with flags
   - Shell completion integration

2. **go-git** (`go-git/go-git/v5`) - Git operations in Go
   - **Used only for gitignore pattern matching**
   - All git operations use `git` CLI via `os/exec`

3. **tablewriter** (`olekukonko/tablewriter`) - Table rendering
   - Clean table output with customizable borders
   - Header/footer control

4. **k1LoW/exec** - Enhanced `os/exec` wrapper
   - Better error handling with exit codes
   - Familiar API similar to standard library

### Interesting Library Usage

**Why go-git for gitignore only?**
- Most operations use `git` CLI for reliability
- go-git used specifically for `.gitignore` pattern parsing
- Pattern: Use libraries for specific features, shell out for main operations

---

## Features & Patterns Applicable to gbm

### 1. **Shell Integration Implementation** ⭐⭐⭐

**What**: Auto-cd to worktree after creation, plus completion

**Current gbm**: Has shell integration helper but not full auto-cd wrapper

**Recommendation**: Implement similar shell wrapper pattern
- Print worktree path to stdout for shell integration
- Provide `gbm --init <shell>` command to output wrapper script
- Add `--no-switch-directory` flag for completion-only setup

**Implementation**:
```go
// cmd/service/shell-integration.go
func newShellIntegrationCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "shell-init [shell]",
        Short: "Output shell integration script",
        Args:  cobra.ExactArgs(1),
        RunE:  runShellInit,
    }
    return cmd
}
```

### 2. **Git Config for Configuration** ⭐⭐

**What**: Use `git config` instead of separate config file

**Current gbm**: Uses `.gbm/config.yaml` for configuration

**Recommendation**: Consider hybrid approach
- JIRA config → Keep in `.gbm/config.yaml` (credentials)
- Worktree preferences → Move to `git config` (e.g., `gbm.basedir`)
- Allows per-repo and global configuration

**Benefits**:
- Per-repo config: `git config gbm.basedir "../worktrees"`
- Global config: `git config --global gbm.basedir "~/worktrees/{gitroot}"`
- Follows git conventions

### 3. **Flag Override Pattern** ⭐⭐⭐

**What**: Config from git config with flag overrides using `cmd.Flags().Changed()`

**Current gbm**: Direct flag usage

**Recommendation**: Implement flag override pattern
```go
func loadConfig(ctx context.Context, cmd *cobra.Command) (Config, error) {
    // Load from git config
    cfg, err := LoadGitConfig(ctx)

    // Apply flag overrides only if explicitly set
    if cmd.Flags().Changed("basedir") {
        cfg.BaseDir = basedirFlag
    }

    return cfg, nil
}
```

**Benefits**:
- Clear precedence: flags > git config > defaults
- Single invocation overrides without changing config
- Better UX for experimentation

### 4. **Template Variable System** ⭐⭐

**What**: Variables like `{gitroot}` in configuration paths

**Current gbm**: Hardcoded path logic

**Recommendation**: Add template system for paths
```go
// Support variables in basedir config
// git config gbm.basedir "../{gitroot}-worktrees"
// git config gbm.basedir "~/dev/{gitroot}/branches"
```

**Variables to support**:
- `{gitroot}` - Repository directory name
- `{branch}` - Branch name
- `{issue}` - JIRA issue key

### 5. **File Copying Feature** ⭐⭐⭐

**What**: Copy ignored/untracked/modified files to new worktrees

**Current gbm**: No file copying

**Recommendation**: Implement selective file copying
- Copy `.env` files to new worktrees
- Copy other ignored config files
- Exclude patterns using gitignore syntax

**Use Cases**:
- Development environment files (`.env`, `.env.local`)
- IDE configurations (`.vscode/settings.json`)
- Build artifacts during development

**Implementation**:
```go
type CopyOptions struct {
    CopyIgnored   bool
    CopyUntracked bool
    CopyModified  bool
    NoCopy        []string  // Patterns to exclude
}

// In config
// git config gbm.copyignored true
// git config --add gbm.nocopy "*.log"
// git config --add gbm.nocopy "vendor/"
```

### 6. **Enhanced Table Output** ⭐

**What**: Clean table rendering with tablewriter

**Current gbm**: Uses Bubble Tea for TUI

**Recommendation**: Consider tablewriter for non-TUI list views
- Better for CI/CD and scripting contexts
- Simpler than full TUI for basic listing
- Can coexist with Bubble Tea for interactive mode

**Pattern**:
```bash
gbm worktree list          # Simple table output (tablewriter)
gbm worktree               # Interactive TUI (Bubble Tea)
```

### 7. **Simplified Command Structure** ⭐

**What**: Single root command with flags instead of many subcommands

**Current gbm**: Nested subcommands (`worktree add`, `worktree remove`, etc.)

**Recommendation**: Consider simplifying for common operations
```bash
# Current gbm
gbm worktree add feature-x

# Simplified (like git-wt)
gbm feature-x              # Create or switch
gbm -d feature-x           # Delete
```

**Trade-off**: Less explicit but more ergonomic for frequent use

### 8. **E2E Testing Pattern** ⭐⭐⭐

**What**: Comprehensive E2E tests that build binary and test in real shells

**Current gbm**: Unit tests

**Recommendation**: Add E2E test suite
```go
// testutil/repo.go - Test repository helper
type TestRepo struct {
    t    testing.TB
    Root string
}

func NewTestRepo(t testing.TB) *TestRepo {
    // Creates temp git repo with cleanup
}

func (r *TestRepo) Git(args ...string) string {
    // Execute git command in repo
}

func (r *TestRepo) Commit(message string) {
    // Stage all and commit
}
```

**Test Coverage**:
- Build actual binary
- Test in real bash/zsh/fish shells
- Test shell integration end-to-end
- Test worktree creation from within worktrees

### 9. **Stdout/Stderr Separation** ⭐⭐⭐

**What**: Machine-readable output to stdout, human messages to stderr

**Current gbm**: Mixed output

**Recommendation**: Separate outputs for shell integration
```go
// When creating worktree
cmd.Stdout = os.Stderr  // Git messages
cmd.Stderr = os.Stderr

// Then output path for consumption
fmt.Println(wtPath)     // To stdout
```

**Benefits**:
- Shell scripts can capture paths reliably
- Still shows git progress to user
- Enables better shell integration

### 10. **MainRepoRoot vs RepoRoot** ⭐⭐

**What**: Separate functions for current worktree root vs main repo root

```go
// RepoRoot returns current worktree or repo root
func RepoRoot(ctx context.Context) (string, error) {
    cmd, err := gitCommand(ctx, "rev-parse", "--show-toplevel")
    // ...
}

// MainRepoRoot returns main repo root even from worktree
func MainRepoRoot(ctx context.Context) (string, error) {
    cmd, err := gitCommand(ctx, "rev-parse", "--git-common-dir")
    // Find parent of .git directory
}
```

**Current gbm**: Uses `FindGitRoot()` which might not handle both cases

**Recommendation**: Ensure clear distinction between:
- Current worktree root
- Main repository root
- Use appropriate one for different operations

### 11. **Config with Multiple Values** ⭐

**What**: Support multiple values for array configs

```go
// git config --add gbm.nocopy "*.log"
// git config --add gbm.nocopy "vendor/"

func GitConfig(ctx context.Context, key string) ([]string, error) {
    cmd, err := gitCommand(ctx, "config", "--get-all", key)
    // Returns all values as slice
}
```

**Recommendation**: Use `--get-all` for array-type configs

---

## Not Recommended / Lower Priority

### 1. **Avoid go-git for Main Operations**
- git-wt uses go-git only for gitignore parsing
- All worktree operations use `git` CLI
- **Reason**: Git CLI is battle-tested, handles edge cases better
- **gbm already does this correctly**

### 2. **Skip PowerShell Integration (for now)**
- Complex and platform-specific
- Lower priority unless Windows users request it

### 3. **Don't Oversimplify Commands**
- gbm has more complex workflows (JIRA integration, sync, etc.)
- Full simplification like git-wt may not fit
- Keep subcommands for complex operations

---

## Implementation Priority

### High Priority (Immediate Value)

1. **Shell Integration with Auto-CD** ⭐⭐⭐
   - Biggest UX improvement
   - `gbm shell-init <shell>` command
   - Auto-cd after worktree creation

2. **Flag Override Pattern** ⭐⭐⭐
   - Better config management
   - Use `cmd.Flags().Changed()` pattern

3. **File Copying Feature** ⭐⭐⭐
   - Copy `.env` and config files to new worktrees
   - Major developer experience improvement

4. **E2E Test Suite** ⭐⭐⭐
   - Build confidence in releases
   - Test real shell integration

### Medium Priority (Nice to Have)

5. **Git Config Integration** ⭐⭐
   - Hybrid with YAML for credentials
   - Per-repo worktree preferences

6. **Template Variables** ⭐⭐
   - `{gitroot}`, `{branch}`, `{issue}` in paths

7. **Stdout/Stderr Separation** ⭐⭐
   - Better for scripting and automation

### Low Priority (Future Consideration)

8. **Simplified Command Aliases** ⭐
   - `gbm <branch>` as alias for `gbm worktree add`

9. **tablewriter for Non-TUI** ⭐
   - Alternative to Bubble Tea for simple listing

---

## Code Quality Patterns

### 1. **Testing Utilities**
```go
// testutil/repo.go provides clean test helpers
repo := testutil.NewTestRepo(t)
repo.CreateFile("README.md", "# Test")
repo.Commit("initial commit")
```

### 2. **Error Handling**
```go
// Check for specific exit codes
var exitErr *exec.ExitError
if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
    return nil, nil  // Key not found is not an error
}
```

### 3. **Context Throughout**
- All functions accept `context.Context`
- Enables cancellation and timeouts
- Better for CLI commands

### 4. **Porcelain Parsing**
```go
// Use --porcelain format for git commands
cmd, err := gitCommand(ctx, "worktree", "list", "--porcelain")
// Parse structured output instead of pretty-printed format
```

---

## Summary of Recommendations

### Adopt Now
1. Shell integration with auto-cd wrapper
2. Flag override pattern with `cmd.Flags().Changed()`
3. File copying feature for ignored/untracked files
4. E2E testing framework with testutil helpers
5. Stdout/stderr separation for shell integration

### Evaluate Further
1. Hybrid git config + YAML configuration
2. Template variable system for paths
3. Enhanced table output with tablewriter

### Skip
1. Full command simplification (keep subcommands)
2. go-git for main operations (already avoided)
3. PowerShell support (unless requested)

---

## Architecture Comparison

| Aspect | git-wt | gbm | Recommendation |
|--------|--------|-----|----------------|
| Commands | Flat (flags) | Nested (subcommands) | Keep gbm nested |
| Config | git config only | YAML only | Hybrid approach |
| Shell Integration | Full auto-cd | Helper only | Add auto-cd |
| File Copying | Yes | No | Add feature |
| JIRA | No | Yes | Keep gbm advantage |
| TUI | No | Yes (Bubble Tea) | Keep gbm advantage |
| Testing | E2E with shells | Unit only | Add E2E |

---

## Conclusion

git-wt demonstrates excellent patterns for:
- **Shell integration** with automatic directory switching
- **Configuration management** with git config and flag overrides
- **File copying** for development environment files
- **E2E testing** with real shell environments

The most valuable additions to gbm would be:
1. Full shell integration (auto-cd wrapper)
2. File copying feature
3. Flag override pattern
4. E2E test suite

These would significantly improve the developer experience while maintaining gbm's advantages in JIRA integration and interactive TUI.
