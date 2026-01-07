# Constants Plan for pkg/tui - Magic Strings Refactor

## Overview
This document identifies magic strings in the `pkg/tui` package that are **used multiple times** and should be extracted into named constants for improved maintainability, consistency, and testability.

## Key Principle
**Only extract constants for values used 2+ times.** Single-use values (like field labels/descriptions) should remain as literal strings to avoid unnecessary abstraction layers.

## 1. Workflow Type Constants
**Location**: Multiple files (fields, workflows, wizard)  
**Usage Count**: feature: 3x, bug: 4x, hotfix: 4x, merge: 3x (all 2+, extract these)

### 1.1 Workflow Types
- `"feature"` - Feature workflow type (used 3 times)
- `"bug"` - Bug fix workflow type (used 4 times)
- `"hotfix"` - Hotfix workflow type (used 4 times)
- `"merge"` - Merge workflow type (used 3 times)

**Impact**: Used in:
- `workflows/workflows_selector.go` - SelectWorkflowType() field options
- `workflows/router.go` - GetWorkflowSteps() switch statement
- `pkg/tui/wizard.go` - storeFieldValue() and calculateDefaultBranchName() switch statements
- Tests in multiple test files

**Proposed Constants**:
```
const (
    WorkflowTypeFeature = "feature"
    WorkflowTypeBug     = "bug"
    WorkflowTypeHotfix  = "hotfix"
    WorkflowTypeMerge   = "merge"
)
```

---

## 2. Field Key Constants
**Location**: Multiple files (fields, wizard, workflows)  
**Usage Count**: worktree_name: 7x, branch_name: 10x, base_branch: 8x, confirm: 8x (all 2+, extract these) / workflow_type: 2x (extract) / jira_issue: 1x (skip)

### 2.1 Standard Field Keys
- `"worktree_name"` - Worktree/issue selection field (used 7 times) ✓ EXTRACT
- `"branch_name"` - Branch name input field (used 10 times) ✓ EXTRACT
- `"base_branch"` - Base branch selection field (used 8 times) ✓ EXTRACT
- `"jira_issue"` - JIRA issue field (used 1 time) ✗ SKIP - single use
- `"workflow_type"` - Workflow type selector field (used 2 times) ✓ EXTRACT
- `"confirm"` - Confirmation field (used 8 times) ✓ EXTRACT
- `"source_branch"` - Source branch for merge (used in router only) ✗ SKIP - single location
- `"target_branch"` - Target branch for merge (used in router only) ✗ SKIP - single location

**Impact**: Used in:
- `wizard.go` - storeFieldValue() method for storing field values in state
- `wizard.go` - applyFieldDefaults() and calculateDefaultBranchName() for conditional logic
- `workflows/router.go` - Step definitions
- `fields/confirm.go`, `fields/filterable.go`, `fields/selector.go`, `fields/textinput.go` - Constructor calls

**Proposed Constants** (only multi-use):
```
const (
    FieldKeyWorkflowType = "workflow_type"   // 2 uses
    FieldKeyWorktreeName = "worktree_name"   // 7 uses
    FieldKeyBranchName   = "branch_name"     // 10 uses
    FieldKeyBaseBranch   = "base_branch"     // 8 uses
    FieldKeyConfirm      = "confirm"         // 8 uses
)
```

---

## 3. Branch Name Prefixes
**Location**: wizard.go, workflows/workflows.go, tests  
**Current Usage**: Hard-coded prefix strings for generating branch names

### 3.1 Branch Prefixes
- `"feature/"` - Feature branch prefix (wizard.go:304, workflows.go:38, 45)
- `"hotfix/"` - Hotfix branch prefix (wizard.go:314, workflows.go:47)
- `"bug/"` - Bug fix branch prefix (wizard.go:309, workflows.go:57)
- `"merge/"` - Merge branch prefix (workflows.go:279)

**Impact**: Used in:
- `wizard.go` - calculateDefaultBranchName() method
- `workflows/workflows.go` - Branch name generation functions
- Multiple test files

**Proposed Constants**:
```
const (
    BranchPrefixFeature = "feature/"
    BranchPrefixBug     = "bug/"
    BranchPrefixHotfix  = "hotfix/"
    BranchPrefixMerge   = "merge/"
)
```

---

## 4. Worktree Name Prefixes
**Location**: workflows/workflows.go  
**Current Usage**: Hard-coded prefixes for special worktree names

### 4.1 Worktree Prefixes
- `"HOTFIX_"` - Hotfix worktree prefix (workflows.go:161)
- `"MERGE_"` - Merge worktree prefix (workflows.go:276)

**Impact**: Used in:
- `ProcessHotfixWorkflow()` function
- `ProcessMergeWorkflow()` function
- Potentially in state checking logic

**Proposed Constants**:
```
const (
    WorktreePrefixHotfix = "HOTFIX_"
    WorktreePrefixMerge  = "MERGE_"
)
```

---

## 6. UI/Display Constants
**Location**: confirm.go, fields files, wizard.go  
**Current Usage**: Hard-coded labels, placeholders, and UI strings

### 6.1 Button Labels
- `"Yes"` - Confirm yes button (confirm.go:149)
- `"No"` - Confirm no button (confirm.go:150)
- `"(suggested from config)"` - Label suffix (workflows/merge_suggestion.go:77)

**Impact**: Used in:
- Confirm field rendering
- Option sorting display

### 6.2 Placeholder Texts
- `"Type to filter or enter custom value..."` - Filterable placeholder (fields/filterable.go:46)
- `"Enter value..."` - TextInput placeholder (fields/textinput.go:37)
- `"feature/KEY-description"` - TextInput placeholder (workflows/router.go:45)
- `"bug/KEY-description"` - TextInput placeholder (workflows/router.go:129)
- `"hotfix/KEY-description"` - TextInput placeholder (workflows/router.go:239)

**Impact**: Used in:
- Field constructor calls
- User guidance text

### 6.3 UI Element Labels
- `"▸ "` - Cursor symbol (selector.go:106, filterable.go:256)
- `"  "` - Non-selected cursor spacing (selector.go:104, filterable.go:254)

**Impact**: Used in:
- List item rendering for selection

### 6.4 Loading/Status Messages
- `"Loading options..."` - Loading indicator text (fields/filterable.go:233)
- `"Error loading options: %v"` - Error message template (fields/filterable.go:239)
- `"No matches. Press Enter to use: %q"` - No matches message (fields/filterable.go:247)
- `"No matches"` - Empty filtered list text (fields/filterable.go:249)

**Impact**: Used in:
- Filterable field View() method

**Proposed Constants**:
```
const (
    // Button labels
    ButtonLabelYes = "Yes"
    ButtonLabelNo  = "No"
    
    // Placeholders
    PlaceholderFilterable      = "Type to filter or enter custom value..."
    PlaceholderTextInput       = "Enter value..."
    PlaceholderFeatureBranch   = "feature/KEY-description"
    PlaceholderBugBranch       = "bug/KEY-description"
    PlaceholderHotfixBranch    = "hotfix/KEY-description"
    
    // UI Elements
    CursorSymbol   = "▸ "
    CursorSpacing  = "  "
    
    // Status Messages
    MessageLoading      = "Loading options..."
    MessageNoMatches    = "No matches"
    SuggestedSuffix     = " (suggested from config)"
)
```

---

## 7. Validation and Error Messages
**Location**: workflows/workflows.go, fields/textinput.go  
**Current Usage**: Hard-coded validation error messages

### 7.1 Error Messages
- `"branch name cannot be empty"` - Empty branch validation (workflows.go:171)
- `"branch name contains invalid character: %c"` - Invalid character validation (workflows.go:178)

**Impact**: Used in:
- `validateBranchName()` function
- Field validation during user input

**Proposed Constants**:
```
const (
    ErrorBranchEmpty        = "branch name cannot be empty"
    ErrorBranchInvalidChar  = "branch name contains invalid character: %c"
)
```

---

## 8. Step Names
**Location**: workflows/router.go, tests  
**Current Usage**: Hard-coded step identifier strings

### 8.1 Step Names
- `"worktree_name"` - Worktree selection step (same as FieldKeyWorktreeName)
- `"branch_name"` - Branch input step (same as FieldKeyBranchName)
- `"base_branch"` - Base branch step (same as FieldKeyBaseBranch)
- `"source_branch"` - Source branch step (same as FieldKeySourceBranch)
- `"target_branch"` - Target branch step (same as FieldKeyTargetBranch)
- `"confirm"` - Confirmation step (same as FieldKeyConfirm)

**Impact**: Used in:
- Step definitions in router.go
- Skip logic comparison (wizard.go:343, 346)
- Test assertions

**Note**: These are primarily the same as FieldKey constants, so they could reuse those constants or have aliases.

---

## 9. Color/Style Constants
**Location**: theme.go, confirm.go, selector.go, filterable.go  
**Current Usage**: Hard-coded lipgloss color codes

### 9.1 Color Codes
- `"86"` - Cyan highlight (theme.go:25)
- `"246"` - Dark gray (theme.go:26, selector.go:116, filterable.go:266)
- `"255"` - White text (theme.go:27, confirm.go:131, 140)
- `"238"` - Dark background (theme.go:27)
- `"196"` - Red (theme.go:28, confirm.go:141)
- `"240"` - Muted gray (theme.go:32, confirm.go:134, 145, 146)
- `"235"` - Very dark gray (theme.go:33)
- `"243"` - Medium gray (theme.go:34)
- `"124"` - Dark red (theme.go:35)
- `"212"` - Magenta/pink (selector.go:44, filterable.go:63)
- `"244"` - Light gray (selector.go:116, filterable.go:266)
- `"62"` - Blue/teal (confirm.go:132)

**Impact**: Used in:
- Default theme creation
- Cursor styling
- Button styling in confirm field

**Proposed Constants**:
```
const (
    // Text colors
    ColorCyan       = "86"
    ColorWhite      = "255"
    ColorRed        = "196"
    ColorMagenta    = "212"
    ColorDarkRed    = "124"
    
    // Gray tones (from light to dark)
    ColorLightGray  = "244"
    ColorGray       = "243"
    ColorDarkGray   = "246"
    ColorVeryDark   = "240"
    ColorDarker     = "235"
    ColorDarkest    = "238"
    
    // Backgrounds
    ColorBlueTeal   = "62"
)
```

---

## 10. Format and Separator Patterns
**Location**: wizard.go, workflows/workflows.go  
**Current Usage**: Hard-coded string formatting patterns

### 10.1 Format Strings
- `"%s - %s"` - JIRA issue label format (workflows/router.go:33, 117, 200)
- `"feature/%s_%s"` - Branch name format with underscore (wizard.go:359)
- `"feature/%s"` - Branch name prefix format (wizard.go:364)
- `"JIRA-%s"` (inferred from tests) - JIRA issue key format

**Impact**: Used in:
- JIRA issue option label formatting
- Branch name generation

**Proposed Constants**:
```
const (
    FormatJiraOptionLabel = "%s - %s"
    FormatBranchNameFull  = "%s_%s"
    FormatBranchNameBase  = "%s"
)
```

---

## 11. Keyboard Input/Navigation Constants
**Location**: confirm.go, selector.go, filterable.go, textinput.go, wizard.go  
**Current Usage**: Hard-coded key strings and Bubble Tea KeyType  
**Note**: Bubble Tea provides two different systems for handling keyboard input:
1. `tea.KeyType` enum constants - for special keys (Enter, Escape, Ctrl+C, etc.)
2. `bubbles/key.Binding` system - for user-definable key bindings with help text

### 11.1 Key Mappings - Bubble Tea KeyType Usage (Best Practice - Continue Using)
The code currently uses Bubble Tea's built-in `tea.KeyType` constants in several places:
- `tea.KeyCtrlC` - Control+C exit (wizard.go:71)
- `tea.KeyEsc` - Escape key (wizard.go:75)
- `tea.KeyEnter` - Enter/return key (tests use this)

**Recommendation**: Continue using Bubble Tea's KeyType constants for these special keys as they're the idiomatic way to handle keyboard input in Bubble Tea. Do NOT change to string constants. These are the proper approach and should remain as-is.

### 11.2 Keyboard String Comparisons - Extract Only Multi-Use
Usage counts for string-based key comparisons via `keyMsg.String()`:
- `"enter"` - used 6 times ✓ EXTRACT
- `"down"` - used 3 times ✓ EXTRACT
- `"ctrl+c"` - used 3 times ✓ EXTRACT
- `"up"` - used 2 times ✓ EXTRACT
- `"ctrl+k"` - used 2 times ✓ EXTRACT
- `"ctrl+j"` - used 2 times ✓ EXTRACT
- `"left"` / `"right"` / `"tab"` / `"y"` / `"Y"` / `"n"` / `"N"` / `"q"` - all used 1 time each ✗ SKIP

**Proposed Constants** (multi-use only):
```
const (
    // Navigation (multi-use)
    KeyEnter     = "enter"
    KeyUp        = "up"
    KeyDown      = "down"
    KeyCtrlUp    = "ctrl+k"
    KeyCtrlDown  = "ctrl+j"
    KeyCtrlC     = "ctrl+c"
)
```

**Alternative Recommendation**: Consider refactoring to use Bubble Tea's `key.Binding` system from the `bubbles/key` package instead of string comparisons. This would:
- Eliminate all 6 keyboard string constants
- Provide type-safe key handling
- Enable automatic help text generation
- Support configurable keybindings
- Better align with Bubble Tea patterns

This would be a larger refactor but provides more benefits than simple string constants.

---

## 12. Configuration/Label Text
**Location**: workflows/router.go  
**Usage Count**: Most labels used 1x only (some used 2-3x)

### 12.1 Multi-Use Field Labels (Extract Only These)
- `"Select JIRA Issue or Enter Worktree Name"` - Label for worktree selection (used 3 times - once per workflow)
- `"Enter Branch Name"` - Label for branch name input (used 3 times - once per workflow)
- `"Select Base Branch"` - Label for base branch selection (used 3 times - once per workflow)

**Recommendation**: Extract only these 3 multi-use labels. All other descriptions/labels are single-use and should remain as literals.

**Proposed Constants** (multi-use only):
```
const (
    LabelWorktreeSelection = "Select JIRA Issue or Enter Worktree Name"
    LabelBranchName        = "Enter Branch Name"
    LabelBaseBranch        = "Select Base Branch"
)
```

### 12.2 Single-Use Field Descriptions (Skip - Not Worth Abstracting)
The following descriptions are used only once and should remain as inline strings:
- Description for worktree selection
- Description for feature branch name
- Description for bug branch name
- Description for hotfix branch name
- Description for feature base branch
- Description for bug base branch
- Description for hotfix base branch
- Confirmation prompts for each workflow type
- Workflow type selector label
- Source/target branch labels and descriptions for merge workflow

---

## Summary Statistics

| Category | Total Count | Multi-Use | Extract? | Severity |
|----------|------------|-----------|----------|----------|
| Workflow Types | 4 | 4 (3-4x each) | ✓ YES | High |
| Field Keys | 8 | 5 (2-10x each) | ✓ YES | High |
| Branch Prefixes | 4 | 4 (3-40x each) | ✓ YES | High |
| Configuration Labels | 19 | 3 (3x each) | ✓ YES (3 only) | Medium |
| Keyboard Mappings (String-based) | 13 | 6 (2-6x each) | ✓ YES (6 only) | Medium |
| Worktree Prefixes | 2 | 1 (1x each) | ✗ NO (single use) | Low |
| Format Patterns | 3 | 0 | ✗ NO | Low |
| UI/Display Strings | 16 | 2 (2-6x) | ✓ YES (2 only) | Low |
| Validation Messages | 2 | 0 | ✗ NO (not found) | Low |
| Color Codes | 12 | 0 | ✗ SKIP | Low |
| **Bubble Tea KeyType** | 3 | 3 | ✗ NO (continue using) | N/A |
| | | | | |
| **TOTAL TO EXTRACT** | | | **~22-25** | |

---

## Recommended Extraction Strategy

### Phase 1: High Impact (Immediate)
1. **Workflow types** - 4 constants (used 3-4x each)
2. **Field keys** - 5 constants (used 2-10x each)
3. **Branch prefixes** - 4 constants (used 3-40x each)

These should go in `pkg/tui/constants.go`

### Phase 2: Medium Impact (Soon)
1. **Configuration labels** - 3 constants (reused 3x each across workflows)
    - `LabelWorktreeSelection`, `LabelBranchName`, `LabelBaseBranch`
2. **Keyboard mappings** - 6 constants (used 2-6x each)
    - `KeyEnter`, `KeyUp`, `KeyDown`, `KeyCtrlUp`, `KeyCtrlDown`, `KeyCtrlC`

These can go in `pkg/tui/fields/constants.go`

### Phase 3: NOT Worth Extracting
The following are single-use and should remain as inline literals:
- Most field descriptions
- Worktree prefixes (single use)
- Separator constants (underscore, hyphen - too trivial)
- Format patterns (no multi-use patterns found)
- Color codes (used via hardcoded values, not worth extracting)
- Validation messages (not found in actual code)
- Single-use keyboard keys (left, right, tab, y, Y, n, N, q)
- Single-use configuration prompts/labels
- Bubble Tea KeyType constants (continue using as-is, idiomatic approach)

### Alternative: Consider Bubble Tea key.Binding Refactor
Instead of extracting 6 keyboard string constants, consider refactoring to use `bubbles/key.Binding` system. This would provide better benefits (type safety, help text, configurability) with more effort.

### File Organization Recommendation

**Create: `pkg/tui/constants.go`** (High-impact constants)
```go
// Workflow types
const (
    WorkflowTypeFeature = "feature"
    WorkflowTypeBug     = "bug"
    WorkflowTypeHotfix  = "hotfix"
    WorkflowTypeMerge   = "merge"
)

// Field keys
const (
    FieldKeyWorkflowType = "workflow_type"
    FieldKeyWorktreeName = "worktree_name"
    FieldKeyBranchName   = "branch_name"
    FieldKeyBaseBranch   = "base_branch"
    FieldKeyConfirm      = "confirm"
)

// Branch prefixes
const (
    BranchPrefixFeature = "feature/"
    BranchPrefixBug     = "bug/"
    BranchPrefixHotfix  = "hotfix/"
    BranchPrefixMerge   = "merge/"
)
```

**Create: `pkg/tui/fields/constants.go`** (Field-specific constants)
```go
// Field labels (reused across workflows)
const (
    LabelWorktreeSelection = "Select JIRA Issue or Enter Worktree Name"
    LabelBranchName        = "Enter Branch Name"
    LabelBaseBranch        = "Select Base Branch"
)

// Keyboard keys (multi-use)
const (
    KeyEnter     = "enter"
    KeyUp        = "up"
    KeyDown      = "down"
    KeyCtrlUp    = "ctrl+k"
    KeyCtrlDown  = "ctrl+j"
    KeyCtrlC     = "ctrl+c"
)
```

**Note**: Do NOT create color codes or theme constants - the color values in `theme.go` are fine as-is since they're only used once in their respective contexts.

---

## Testing Considerations

When extracting constants, consider:
1. All test files that currently use these strings will benefit from constant references
2. Tests can be updated to use constants for consistency
3. New tests can validate constant combinations (e.g., prefix + issue-key combinations)
4. Configuration tests can validate that all required labels/descriptions are present

---

## Backward Compatibility Notes

- The constants refactor will be internal to the package
- No public API changes
- All functionality remains identical
- Tests will need updates to use new constants
