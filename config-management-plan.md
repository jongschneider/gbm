# Configuration Management Improvement Plan

**Date:** 2026-01-04
**Scope:** Implement configuration validation and generation for GBM
**Estimated Effort:** 5-6 hours total
**Priority:** High (addresses config errors early, improves onboarding)

---

## Executive Summary

This plan addresses two critical configuration management improvements:

1. **Schema Validation** - Prevent runtime errors from malformed config (3 hours, Priority 2)
2. **Config Generation** - Improve onboarding with example config (2-3 hours, Priority 2)

**Impact:**
- Catches config errors early with clear messages
- Reduces support burden and improves onboarding

---

## Table of Contents

1. [Current State](#current-state)
2. [Goals & Success Criteria](#goals--success-criteria)
3. [Implementation Tasks](#implementation-tasks)
4. [Testing Strategy](#testing-strategy)
5. [Risk Assessment](#risk-assessment)

---

## Current State

### Existing Implementation

**Location:** `cmd/service/service.go` (lines ~100-200)

**Current Config Loading:**
```go
// Loads YAML directly without validation
data, err := os.ReadFile(configPath)
yaml.Unmarshal(data, &cfg)
// No validation
```

### Problems

1. **No Schema Validation**
   - Invalid YAML passes silently
   - Runtime errors from malformed config
   - No validation of required fields
   - Poor error messages ("field X is nil")

2. **No Config Generation**
   - Users must manually create .gbm/config.yaml
   - No example config provided
   - Trial-and-error to find valid config structure
   - Increases support burden

### Impact

- **User Experience:** Confusing errors, manual config creation
- **Support:** High support burden for config issues

---

## Goals & Success Criteria

### Goal 1: Schema Validation

**Objective:** Validate config structure at load time with clear error messages

**Success Criteria:**
- ✅ All config fields have validation rules
- ✅ Invalid config fails at load with actionable error
- ✅ Validation includes: required fields, types, URL format, file paths
- ✅ Error messages specify field name and reason
- ✅ Unit tests cover all validation rules

**Example Success:**
```bash
$ gbm wt add test test -b
Error: invalid config (.gbm/config.yaml):
  - field 'default_branch': required field is empty
  - field 'remotes.origin.url': must be valid URL or git@ format

Fix these issues and try again.
```

### Goal 2: Config Generation

**Objective:** Provide `gbm init-config` command for example config

**Success Criteria:**
- ✅ Command: `gbm init-config` creates .gbm/config.yaml
- ✅ Generated config includes comments explaining all options
- ✅ Shows examples for JIRA, file copying, remotes
- ✅ Error if .gbm/config.yaml already exists (unless --force)
- ✅ Help text explains next steps
- ✅ E2E test validates command

**Example Success:**
```bash
$ gbm init-config
✓ Created example config at .gbm/config.yaml

Configured with:
  • Default branch: main (detected from git config)

Edit the file to configure:
  • Git remotes
  • JIRA integration (optional)
  • File copying rules (optional)

Next steps:
  1. Edit .gbm/config.yaml with your settings
  2. Run: gbm init
```

**How default branch detection works:**
1. First checks `git config init.defaultBranch`
2. Falls back to `master` if not configured

This respects the user's git configuration preferences.

---

## Implementation Tasks

### Phase 1: Schema Validation (Priority 2, 3 hours)

#### Task 1.1: Add validator dependency
**Effort:** 15 min

```bash
go get github.com/go-playground/validator/v10
```

#### Task 1.2: Add validation tags to Config struct
**Effort:** 30 min
**File:** `cmd/service/service.go`

```go
type Config struct {
    DefaultBranch string                    `yaml:"default_branch" validate:"required,min=1"`
    WorktreesDir  string                    `yaml:"worktrees_dir" validate:"required"`
    Remotes       map[string]RemoteConfig   `yaml:"remotes" validate:"dive"`
    JIRA          *JIRAConfig               `yaml:"jira,omitempty" validate:"omitempty,dive"`
    FileCopy      *FileCopyConfig           `yaml:"file_copy,omitempty" validate:"omitempty,dive"`
}

type RemoteConfig struct {
    URL string `yaml:"url" validate:"required,url|startswith=git@"`
}

type JIRAConfig struct {
    Enabled      bool   `yaml:"enabled"`
    Host         string `yaml:"host" validate:"required_if=Enabled true,omitempty,url"`
    Username     string `yaml:"username" validate:"required_if=Enabled true,omitempty,email"`
    APIToken     string `yaml:"api_token" validate:"required_if=Enabled true"`
    JQL          string `yaml:"jql"`
    BranchPrefix string `yaml:"branch_prefix"`
}

type FileCopyConfig struct {
    Rules []FileCopyRule  `yaml:"rules" validate:"dive"`
    Auto  *AutoCopyConfig `yaml:"auto,omitempty" validate:"omitempty,dive"`
}

type FileCopyRule struct {
    Source          string `yaml:"source" validate:"required"`
    Target          string `yaml:"target" validate:"required"`
    CreateIfMissing bool   `yaml:"create_if_missing"`
}

type AutoCopyConfig struct {
    Enabled         bool     `yaml:"enabled"`
    SourceWorktree  string   `yaml:"source_worktree" validate:"required_if=Enabled true"`
    CopyIgnored     bool     `yaml:"copy_ignored"`
    CopyUntracked   bool     `yaml:"copy_untracked"`
    Exclude         []string `yaml:"exclude"`
}
```

#### Task 1.3: Create validation helper
**Effort:** 30 min
**File:** `cmd/service/config.go` (new file, split from service.go)

```go
package service

import (
    "fmt"
    "github.com/go-playground/validator/v10"
)

var validate = validator.New()

func validateConfig(cfg *Config) error {
    if err := validate.Struct(cfg); err != nil {
        return formatValidationError(err)
    }

    // Custom validations
    if err := validateTemplateVars(cfg.WorktreesDir); err != nil {
        return fmt.Errorf("invalid worktrees_dir template: %w", err)
    }

    return nil
}

func formatValidationError(err error) error {
    if validationErrs, ok := err.(validator.ValidationErrors); ok {
        var messages []string
        for _, e := range validationErrs {
            msg := fmt.Sprintf("field '%s': %s", e.Field(), translateValidationTag(e.Tag()))
            messages append(messages, msg)
        }
        return fmt.Errorf("validation failed:\n  - %s", strings.Join(messages, "\n  - "))
    }
    return err
}

func translateValidationTag(tag string) string {
    switch tag {
    case "required":
        return "required field is empty"
    case "url":
        return "must be valid URL"
    case "email":
        return "must be valid email address"
    case "min":
        return "value too short"
    default:
        return tag
    }
}

func validateTemplateVars(path string) error {
    // Allowed: {gitroot}, {branch}, {issue}
    // TODO: Implement regex validation
    return nil
}
```

#### Task 1.4: Integrate validation into LoadConfig
**Effort:** 15 min
**File:** `cmd/service/config.go`

```go
func LoadConfig(ctx context.Context) (*Config, error) {
    // ... existing YAML loading ...

    // Validate structure
    if err := validateConfig(&cfg); err != nil {
        return nil, &ConfigError{
            Path:    configPath,
            Message: "configuration validation failed",
            Err:     err,
        }
    }

    return &cfg, nil
}
```

#### Task 1.5: Write validation tests
**Effort:** 1 hour
**File:** `cmd/service/config_test.go` (new)

```go
func TestValidateConfig(t *testing.T) {
    tests := []struct {
        name    string
        config  Config
        wantErr bool
        errMsg  string
    }{
        {
            name: "valid config",
            config: Config{
                DefaultBranch: "main",
                WorktreesDir:  "worktrees",
            },
            wantErr: false,
        },
        {
            name: "missing default_branch",
            config: Config{
                WorktreesDir: "worktrees",
            },
            wantErr: true,
            errMsg:  "default_branch",
        },
        {
            name: "invalid remote URL",
            config: Config{
                DefaultBranch: "main",
                WorktreesDir:  "worktrees",
                Remotes: map[string]RemoteConfig{
                    "origin": {URL: "not-a-url"},
                },
            },
            wantErr: true,
            errMsg:  "url",
        },
        {
            name: "JIRA enabled but missing host",
            config: Config{
                DefaultBranch: "main",
                WorktreesDir:  "worktrees",
                JIRA: &JIRAConfig{
                    Enabled: true,
                    // Missing Host, Username, APIToken
                },
            },
            wantErr: true,
            errMsg:  "host",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateConfig(&tt.config)
            if tt.wantErr {
                require.Error(t, err)
                assert.Contains(t, err.Error(), tt.errMsg)
            } else {
                require.NoError(t, err)
            }
        })
    }
}
```

---

### Phase 2: Config Generation (Priority 2, 2-3 hours)

#### Task 2.1: Create example config template
**Effort:** 30 min
**File:** `cmd/service/config.go`

**Implementation notes:**
- Detects default branch from `git config init.defaultBranch`
- Falls back to `master` if not configured
- Generates config dynamically with detected branch

```go
func getDefaultBranch() string {
    // Try git config init.defaultBranch first
    cmd := exec.Command("git", "config", "--get", "init.defaultBranch")
    if output, err := cmd.Output(); err == nil {
        branch := strings.TrimSpace(string(output))
        if branch != "" {
            return branch
        }
    }

    // Fall back to master
    return "master"
}

func generateExampleConfigYAML() string {
    defaultBranch := getDefaultBranch()

    return fmt.Sprintf(`# GBM Configuration
# Generated by: gbm init-config

# Default base branch for new branches (used with -b flag)
default_branch: %s

# Worktrees directory (relative to repo root)
# Supports templates: {gitroot}, {branch}, {issue}
worktrees_dir: worktrees

# Git remotes (optional)
# Configure remotes to set up in new repositories
#remotes:
#  origin:
#    url: git@github.com:user/repo.git
#  upstream:
#    url: git@github.com:org/repo.git

# JIRA integration (optional)
# Enables creating worktrees from JIRA issues
#jira:
#  enabled: true
#  host: https://jira.company.com
#  username: user@company.com
#  api_token: ${JIRA_API_TOKEN}  # Or paste token directly
#  jql: "assignee = currentUser() AND status != Done"
#  branch_prefix: feature/

# File copying (optional)
# Automatically copy files when creating worktrees
#file_copy:
#  # Rule-based copying
#  rules:
#    - source: ".env"
#      target: ".env"
#      create_if_missing: true
#    - source: "config/"
#      target: "config/"
#      create_if_missing: false
#
#  # Automatic copying
#  auto:
#    enabled: false
#    source_worktree: "{default}"  # {default}, {current}, or worktree name
#    copy_ignored: true            # Copy .gitignored files
#    copy_untracked: false         # Copy untracked files
#    exclude:                      # Exclude patterns (gitignore syntax)
#      - "*.log"
#      - "node_modules/"
#      - ".DS_Store"
`, defaultBranch)
}

func GenerateExampleConfig(path string) error {
    // Check if config already exists
    if _, err := os.Stat(path); err == nil {
        return fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
    }

    // Create directory
    dir := filepath.Dir(path)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create directory: %w", err)
    }

    // Generate config with detected default branch
    configYAML := generateExampleConfigYAML()

    // Write example config
    if err := os.WriteFile(path, []byte(configYAML), 0644); err != nil {
        return fmt.Errorf("failed to write config: %w", err)
    }

    return nil
}
```

#### Task 2.2: Implement init-config command
**Effort:** 30 min
**File:** `cmd/service/init_config.go` (new)

```go
package service

import (
    "fmt"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
)

func newInitConfigCommand() *cobra.Command {
    var force bool

    cmd := &cobra.Command{
        Use:   "init-config",
        Short: "Generate example configuration file",
        Long: `Generate an example .gbm/config.yaml file with comments explaining all options.

The generated config includes examples for:
  • Git remotes
  • JIRA integration
  • File copying rules
  • Path templates

By default, fails if config already exists. Use --force to overwrite.`,
        Example: `  # Generate example config
  gbm init-config

  # Overwrite existing config
  gbm init-config --force`,
        RunE: func(cmd *cobra.Command, args []string) error {
            // Find git root
            gitRoot, err := git.FindGitRoot()
            if err != nil {
                return fmt.Errorf("not in a git repository: %w", err)
            }

            configPath := filepath.Join(gitRoot, ".gbm", "config.yaml")

            // Check if exists (unless --force)
            if !force {
                if _, err := os.Stat(configPath); err == nil {
                    return fmt.Errorf("config already exists at %s\nUse --force to overwrite", configPath)
                }
            }

            // Generate config
            if err := GenerateExampleConfig(configPath); err != nil {
                return err
            }

            // Success message
            defaultBranch := getDefaultBranch()
            fmt.Fprintf(os.Stderr, "✓ Created example config at %s\n\n", configPath)
            fmt.Fprintf(os.Stderr, "Configured with:\n")
            fmt.Fprintf(os.Stderr, "  • Default branch: %s (detected from git config)\n\n", defaultBranch)
            fmt.Fprintf(os.Stderr, "Edit the file to configure:\n")
            fmt.Fprintf(os.Stderr, "  • Git remotes\n")
            fmt.Fprintf(os.Stderr, "  • JIRA integration (optional)\n")
            fmt.Fprintf(os.Stderr, "  • File copying rules (optional)\n\n")
            fmt.Fprintf(os.Stderr, "Next steps:\n")
            fmt.Fprintf(os.Stderr, "  1. Edit %s\n", configPath)
            fmt.Fprintf(os.Stderr, "  2. Run: gbm init (if creating new repo)\n")

            return nil
        },
    }

    cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config")

    return cmd
}
```

#### Task 2.3: Register command
**Effort:** 5 min
**File:** `cmd/service/root.go`

```go
func Execute() error {
    rootCmd := newRootCommand()

    // ... existing commands ...
    rootCmd.AddCommand(newInitConfigCommand())  // Add this

    return rootCmd.Execute()
}
```

#### Task 2.4: Write command tests
**Effort:** 1 hour
**File:** `cmd/service/init_config_test.go` (new)

```go
func TestInitConfigCommand(t *testing.T) {
    tmpDir := t.TempDir()
    gitDir := filepath.Join(tmpDir, ".git")
    err := os.MkdirAll(gitDir, 0755)
    require.NoError(t, err)

    configPath := filepath.Join(tmpDir, ".gbm", "config.yaml")

    // Run init-config
    cmd := newInitConfigCommand()
    cmd.SetArgs([]string{})

    // Change to tmpDir
    oldWd, _ := os.Getwd()
    defer os.Chdir(oldWd)
    os.Chdir(tmpDir)

    err = cmd.Execute()
    require.NoError(t, err)

    // Verify config created
    assert.FileExists(t, configPath)

    // Verify content
    data, err := os.ReadFile(configPath)
    require.NoError(t, err)
    content := string(data)

    // Should contain a default_branch (either from git config or "master" fallback)
    assert.Contains(t, content, "default_branch:")
    assert.Contains(t, content, "worktrees_dir: worktrees")
    assert.Contains(t, content, "# JIRA integration")
    assert.Contains(t, content, "# File copying")
}

func TestGetDefaultBranch(t *testing.T) {
    // This test validates the default branch detection logic
    // Note: Actual value depends on user's git config
    branch := getDefaultBranch()

    // Should return a non-empty string
    assert.NotEmpty(t, branch, "default branch should not be empty")

    // Should be either user's configured branch or "master" fallback
    // (we can't assert the exact value since it depends on git config)
}

func TestInitConfigCommand_AlreadyExists(t *testing.T) {
    tmpDir := t.TempDir()
    gitDir := filepath.Join(tmpDir, ".git")
    err := os.MkdirAll(gitDir, 0755)
    require.NoError(t, err)

    configDir := filepath.Join(tmpDir, ".gbm")
    err = os.MkdirAll(configDir, 0755)
    require.NoError(t, err)

    configPath := filepath.Join(configDir, "config.yaml")
    err = os.WriteFile(configPath, []byte("existing"), 0644)
    require.NoError(t, err)

    // Run init-config (should fail)
    cmd := newInitConfigCommand()
    cmd.SetArgs([]string{})

    oldWd, _ := os.Getwd()
    defer os.Chdir(oldWd)
    os.Chdir(tmpDir)

    err = cmd.Execute()
    require.Error(t, err)
    assert.Contains(t, err.Error(), "already exists")

    // Verify original content unchanged
    data, err := os.ReadFile(configPath)
    require.NoError(t, err)
    assert.Equal(t, "existing", string(data))
}

func TestInitConfigCommand_Force(t *testing.T) {
    tmpDir := t.TempDir()
    gitDir := filepath.Join(tmpDir, ".git")
    err := os.MkdirAll(gitDir, 0755)
    require.NoError(t, err)

    configDir := filepath.Join(tmpDir, ".gbm")
    err = os.MkdirAll(configDir, 0755)
    require.NoError(t, err)

    configPath := filepath.Join(configDir, "config.yaml")
    err = os.WriteFile(configPath, []byte("existing"), 0644)
    require.NoError(t, err)

    // Run init-config --force
    cmd := newInitConfigCommand()
    cmd.SetArgs([]string{"--force"})

    oldWd, _ := os.Getwd()
    defer os.Chdir(oldWd)
    os.Chdir(tmpDir)

    err = cmd.Execute()
    require.NoError(t, err)

    // Verify config overwritten
    data, err := os.ReadFile(configPath)
    require.NoError(t, err)
    assert.Contains(t, string(data), "default_branch:")  // Should have default_branch set
}
```

#### Task 2.5: Add E2E test
**Effort:** 30 min
**File:** `e2e_test.go`

```go
func TestE2E_InitConfig(t *testing.T) {
    binPath := buildBinary(t)
    repo := setupGBMRepo(t)

    configPath := filepath.Join(repo.Root, ".gbm", "config.yaml")

    // Remove existing config
    os.Remove(configPath)

    // Run init-config
    _, stderr, err := runGBMStdout(t, binPath, repo.Root, "init-config")
    require.NoError(t, err)

    // Verify success message
    assert.Contains(t, stderr, "✓ Created example config")
    assert.Contains(t, stderr, "Edit the file to configure")

    // Verify config created
    assert.FileExists(t, configPath)

    // Verify content is valid YAML
    data, err := os.ReadFile(configPath)
    require.NoError(t, err)

    var cfg Config
    err = yaml.Unmarshal(data, &cfg)
    require.NoError(t, err)

    // Verify basic structure
    assert.NotEmpty(t, cfg.DefaultBranch, "default_branch should be set")
    assert.Equal(t, "worktrees", cfg.WorktreesDir)
}
```

---

## Testing Strategy

### Unit Tests

**Coverage Goal:** 90%+ for config management code

**Test Files:**
- `cmd/service/config_test.go` - Config loading and validation
- `cmd/service/init_config_test.go` - init-config command

**Test Cases:**

1. **Validation Tests** (15-20 test cases)
   - Valid config passes
   - Missing required fields fail
   - Invalid URL format fails
   - Invalid email format fails
   - JIRA enabled without credentials fails
   - Template variables validate correctly

2. **Config Generation Tests** (5-10 test cases)
   - init-config creates valid config
   - Fails if config exists (without --force)
   - --force overwrites existing config
   - Generated config passes validation
   - All sections present in example

### E2E Tests

**Test Scenarios:**

1. **Config Validation E2E**
   - Invalid config shows clear error
   - Valid config loads successfully

2. **Config Generation E2E**
   - init-config creates config
   - Generated config is valid
   - init command uses generated config

### Manual Testing Checklist

**Before Merge:**
- [ ] Invalid config shows clear error message
- [ ] Valid config loads without issues
- [ ] init-config creates valid example
- [ ] init-config --force overwrites
- [ ] Help text clear and accurate
- [ ] Error messages actionable

---

## Risk Assessment

### Risks

1. **Validation Too Strict**
   - **Mitigation:** Only validate actual requirements
   - **Recovery:** Can disable validation if needed
   - **Testing:** Test valid use cases, not just errors

2. **Performance Impact from Validation**
   - **Mitigation:** Validation is one-time at load
   - **Impact:** ~1ms overhead (negligible)
   - **Testing:** Benchmark if needed

3. **init-config Overwrites Existing Configs Accidentally**
   - **Mitigation:** Requires --force flag to overwrite
   - **Recovery:** User's existing config is preserved unless --force
   - **Testing:** Test failure mode without --force

### Rollback Plan

If issues arise:
1. Revert changes in git
2. Document issues for fix in next release

---

## Success Metrics

### Quantitative

- [ ] 90%+ test coverage for config management code
- [ ] Zero config-related bugs in first week after release

### Qualitative

- [ ] Error messages are clear and actionable
- [ ] init-config reduces support burden
- [ ] Config validation catches errors early

### User Feedback

After release, monitor:
- GitHub issues related to config
- Support requests for config help
- Feedback on error message clarity

---

## Timeline

**Total Effort:** 5-6 hours

**Phase 1: Schema Validation (3 hours)**
- Add validator dependency (15 min)
- Add validation tags to Config struct (30 min)
- Create validation helper functions (30 min)
- Integrate validation into LoadConfig (15 min)
- Write validation tests (1 hour)

**Phase 2: Config Generation (2-3 hours)**
- Create example config template (30 min)
- Implement init-config command (30 min)
- Register command (5 min)
- Write command tests (1 hour)
- Add E2E test (30 min)

**Integration & Testing (30 min)**
- Manual testing
- E2E validation
- Documentation updates

---

## Next Steps

1. Review and approve this plan
2. Create tasks/issues for each phase
3. Start with Phase 1 (Schema Validation)
4. Iterate based on feedback

**Dependencies:**
- None (all changes are self-contained)

**Blockers:**
- None identified
