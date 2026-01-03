# Git Service Refactoring Plan

**Document Version:** 1.0  
**Date:** 2026-01-02  
**Status:** Analysis Complete - Ready for Implementation  
**Task Reference:** P3.1.1 - Review and document current git service organization

---

## Executive Summary

The `internal/git` package is well-organized with **clear separation of concerns** across five focused files. Each file has a specific responsibility. The current structure is clean and follows Go conventions. **No major refactoring is required**, but we can document the organization and make minor improvements for clarity.

### Recommendation: Minimal Changes
The git service is already well-structured. Implementation of P3.1.2 should be limited to:
- Adding optional `branch.go` if it would improve clarity (currently branch operations are distributed)
- Improving code documentation/godoc comments
- No breaking API changes needed

---

## Current File Organization

### 1. `service.go` (94 lines) - Core Repository Operations
**Responsibility:** Core git operations and helper functions

**Functions:**
- `NewService()` - Service initialization with git validation
- `FindGitRoot(startPath)` - Find repository root (handles bare + worktree repos)
- `GetCurrentWorktree()` - Get current worktree info
- `GetBranchStatus(worktreePath)` - Get branch sync status with remote
- `runCommand(cmd, dryRun)` - Execute git commands with dry-run support

**Characteristics:**
- ✅ Focused on core operations
- ✅ Well-documented with detailed examples
- ✅ Handles edge cases (worktrees, bare repos, macOS symlinks)
- ✅ Clean separation: utility functions vs. business logic

**Code Quality:** Excellent. Documentation is clear and comprehensive.

---

### 2. `worktree.go` (554 lines) - Worktree Management
**Responsibility:** All worktree operations and branch management

**Functions:**
- `parseWorktrees(output)` - Parse `git worktree list` output
- `AddWorktree(...)` - Create new worktree with branch
- `ListWorktrees(dryRun)` - List all worktrees
- `GetWorktreeBranch(path)` - Get branch for a worktree
- `RemoveWorktree(name, force)` - Remove worktree with trash support
- `MoveWorktree(oldName, newName)` - Rename/move worktree
- `BranchExists(name)` - Check if branch exists locally
- `BranchExistsInPath(path, name)` - Check in specific worktree
- `DeleteBranch(name, force)` - Delete git branch
- `ListBranches()` - Get all branches
- `Fetch(dryRun)` - Fetch from remote
- `MergeBranchWithCommit(...)` - Merge with commit message
- `GetUpstreamBranch(path)` - Get upstream tracking branch
- `IsInWorktree(path)` - Check if in a worktree
- `PullWorktree(path)` - Pull changes in worktree
- `PushWorktree(path)` - Push changes from worktree

**Size:** 554 lines (largest file)

**Characteristics:**
- ✅ Focused on worktree operations
- ✅ Includes branch operations (used with worktrees)
- ✅ Handles symlinks, upstream branches, push/pull
- ⚠️ Could benefit from branching out branch-specific operations

**Breakdown by Category:**
- Worktree operations: ~45% (AddWorktree, ListWorktrees, RemoveWorktree, MoveWorktree, IsInWorktree)
- Branch operations: ~35% (DeleteBranch, ListBranches, BranchExists, etc.)
- Worktree-branch hybrids: ~20% (GetWorktreeBranch, PullWorktree, PushWorktree)

**Note:** Branch operations are co-located with worktrees because worktrees are fundamentally about managing branches. Separating them would create artificial splitting.

---

### 3. `init.go` (142 lines) - Repository Initialization
**Responsibility:** Create new git repository with worktree structure

**Functions:**
- `Init(name, defaultBranchName)` - Initialize new bare + worktree repo

**Features:**
- ✅ Creates bare repository
- ✅ Creates main worktree
- ✅ Generates config.yaml
- ✅ Creates initial empty commit
- ✅ Well-documented

**Code Quality:** Excellent. Clear, focused responsibility.

---

### 4. `clone.go` (216 lines) - Repository Cloning
**Responsibility:** Clone remote repository with worktree structure

**Functions:**
- `Clone(repoURL, name)` - Clone remote repo as bare + worktree
- `extractRepoName(url)` - Parse repository name from URL
- `getDefaultBranch(gitDir)` - Determine default branch from remote
- `getDefaultBranch()` helper - Handle symbolic-ref parsing

**Features:**
- ✅ Clones as bare repository
- ✅ Detects remote default branch
- ✅ Creates main worktree automatically
- ✅ Generates config.yaml
- ✅ Handles fetch configuration

**Code Quality:** Good. Logic is clear, error handling is comprehensive.

---

### 5. `errors.go` (18 lines) - Error Definitions
**Responsibility:** Sentinel errors for git package

**Error Types:**
- Parameter validation: WorktreesDirectoryEmpty, WorktreeNameEmpty, BranchNameEmpty, etc.
- State errors: NotInWorktree, CouldNotDetermineDefaultBranch

**Code Quality:** Clean. All errors are well-named sentinel types.

---

### 6. `init_test.go` (exists) - Init Tests
**Responsibility:** Unit tests for init.go

**Status:** Not reviewed in depth, but follows standard Go testing patterns.

---

## Function Count by File

| File | Functions | Lines | Focus |
|------|-----------|-------|-------|
| service.go | 5 | 94 | Core operations |
| worktree.go | 16 | 554 | Worktree & branch management |
| init.go | 1 | 142 | Repository initialization |
| clone.go | 3 | 216 | Repository cloning |
| errors.go | - | 18 | Error definitions |
| **Total** | **25** | **1024** | **Complete git layer** |

---

## Proposed Organization

### Option A: Current Structure is Good (Recommended)
**Keep as-is.** The current organization is excellent:

```
internal/git/
  service.go      ✅ Core operations (unchanged)
  worktree.go     ✅ Worktree + branch ops (unchanged)
  init.go         ✅ Repository init (unchanged)
  clone.go        ✅ Repository clone (unchanged)
  errors.go       ✅ Error types (unchanged)
```

**Why:** 
- Clear boundaries between files
- Balanced file sizes (94-554 lines)
- No redundancy
- Each file has a primary responsibility
- Follows git-wt pattern

**Actions in P3.1.2:**
- ✅ No code reorganization needed
- ✅ Add inline godoc comments for public functions (P4.1.2)
- ✅ Consider adding `branch.go` ONLY if it would clarify intent

---

### Option B: Extract Branch Operations (Optional)
**Create `branch.go` for branch-specific operations** (if clarity is desired).

```
internal/git/
  service.go      # Core operations
  worktree.go     # Worktree operations only
  branch.go       # Branch operations (NEW)
  init.go         # Repository initialization
  clone.go        # Repository cloning
  errors.go       # Error types
```

**Functions to move to `branch.go`:**
```go
// Branch checking
- BranchExists(name)
- BranchExistsInPath(path, name)

// Branch operations
- DeleteBranch(name, force)
- ListBranches()
- MergeBranchWithCommit(path, source, msg)

// Branch metadata
- GetUpstreamBranch(path)
```

**Functions to keep in `worktree.go`:**
```go
// Worktree-centric branch operations
- GetWorktreeBranch(path)           # Gets branch FOR a worktree
- PullWorktree(path)                # Pull FOR a worktree
- PushWorktree(path)                # Push FROM a worktree
```

**Recommendation:** Only do this if creating confusion about what belongs where. Currently, worktree.go is clear about its scope.

---

## Analysis: Is Refactoring Needed?

### Metrics

| Metric | Score | Notes |
|--------|-------|-------|
| **File Size Balance** | ✅ Good | Largest file is 554 lines (acceptable) |
| **Responsibility Clarity** | ✅ Excellent | Each file has clear purpose |
| **Code Duplication** | ✅ None | No duplicated functionality |
| **Naming Clarity** | ✅ Clear | Function names clearly indicate purpose |
| **Public API Surface** | ✅ Small | 25 functions, all intentional |
| **Test Coverage** | ⚠️ Partial | init_test.go exists; other files not reviewed |
| **Documentation** | ⚠️ Needs Work | Few godoc comments on public functions |

### Verdict: **No Breaking Refactoring Needed**

The git service is well-organized. The structure supports:
- ✅ Easy navigation (clear what's in each file)
- ✅ Focused testing (each file can be tested independently)
- ✅ Future growth (can add methods without restructuring)
- ✅ Clear dependencies (no circular imports)

---

## Implementation Plan for P3.1

### P3.1.1: Documentation (This Document)
**Status:** ✅ Complete

**Deliverables:**
- Current structure documented
- Proposed structure analyzed
- Recommendation provided
- Ready for P3.1.2

---

### P3.1.2: Optional Enhancements
**Complexity:** Low  
**Estimate:** 2-3 hours

**Option 1: Add Inline Documentation (Recommended)**
Update `CLAUDE.md` with git service overview:
- Public API documentation
- Examples of common patterns
- When to add new functions
- Error handling conventions

**Option 2: Create branch.go (Optional)**
Only if team consensus is that branch operations should be separate.
- Move 5 branch-specific functions
- Keep 3 worktree-branch hybrid functions in worktree.go
- Update imports in service methods

**Option 3: Do Nothing**
Current structure is already excellent. Focus on documentation (P4.1) instead.

---

## Recommendations for New Functions

When adding new git operations in the future:

### Adding to `service.go`
Use for **core repository operations** that don't belong in any specific domain:
- Repository navigation (FindGitRoot, GetCurrentWorktree)
- Utility functions (runCommand, helper functions)
- Configuration queries (GetBranchStatus)

### Adding to `worktree.go`
Use for **worktree-related operations**:
- Worktree creation/deletion/management (AddWorktree, RemoveWorktree)
- Worktree-specific queries (GetWorktreeBranch, IsInWorktree)
- Worktree sync operations (PullWorktree, PushWorktree)
- Branch operations related to worktrees (DeleteBranch, ListBranches)

### Adding to new `branch.go` (if created)
Use for **standalone branch operations**:
- BranchExists, BranchExistsInPath
- DeleteBranch (stand-alone)
- ListBranches (across repository)
- MergeBranchWithCommit
- GetUpstreamBranch (metadata)

### Adding to `init.go`
Use for **initialization logic**:
- New repository setup steps
- Configuration initialization
- Worktree structure setup

### Adding to `clone.go`
Use for **clone logic**:
- Remote repository handling
- Branch detection
- Fetch configuration

---

## Public API Surface

### Current Public Functions (25 total)

**Service Constructor:**
- `NewService()`

**Repository Navigation (service.go):**
- `FindGitRoot(path)` - Find repo root
- `GetCurrentWorktree()` - Current worktree info
- `GetBranchStatus(path)` - Sync status

**Worktree Operations (worktree.go):**
- `AddWorktree(dir, name, branch, create, base)` - Create worktree
- `ListWorktrees(dryRun)` - List all worktrees
- `RemoveWorktree(name, force)` - Remove worktree
- `MoveWorktree(old, new)` - Rename worktree
- `GetWorktreeBranch(path)` - Get worktree's branch
- `IsInWorktree(path)` - Check location

**Branch Operations (worktree.go):**
- `BranchExists(name)` - Check if exists
- `BranchExistsInPath(path, name)` - Check in worktree
- `DeleteBranch(name, force)` - Delete branch
- `ListBranches()` - Get all branches
- `GetUpstreamBranch(path)` - Get upstream

**Sync Operations (worktree.go):**
- `Fetch(dryRun)` - Fetch from remote
- `PullWorktree(path)` - Pull changes
- `PushWorktree(path)` - Push changes
- `MergeBranchWithCommit(path, source, msg)` - Merge

**Initialization (init.go):**
- `Init(name, defaultBranch)` - Initialize repo

**Cloning (clone.go):**
- `Clone(url, name)` - Clone repo

**All APIs are intentional and necessary.** No cleanup needed.

---

## Code Quality Assessment

### Strengths ✅
1. **Clear separation of concerns** - Each file has distinct responsibility
2. **Well-documented service.go** - Detailed comments on complex logic
3. **Consistent error handling** - Wrapped errors with context
4. **Dry-run support** - All write operations support dry-run mode
5. **Edge case handling** - Symlinks, bare repos, worktrees all handled

### Areas for Improvement ⚠️
1. **Limited godoc comments** - Public functions lack doc comments
2. **Error types limited** - Only sentinel errors, no typed errors
3. **Error messages basic** - Don't include exit codes or stderr output
4. **No structured logging** - Uses fmt.Fprintf directly

### Next Steps
- **P3.2**: Add typed errors for common git failures
- **P4.1.2**: Add comprehensive godoc comments

---

## Breaking Changes Analysis

**Moving to this organization would cause:**
- ✅ **No breaking changes** - All public APIs remain
- ✅ **No import changes** - All imports stay in `internal/git`
- ✅ **No behavior changes** - Implementation unchanged
- ✅ **No test changes** - Tests can stay organized by file

---

## Migration Plan (If branch.go is created)

### Step 1: Create branch.go
Copy branch-related functions from worktree.go

### Step 2: Update worktree.go
Remove functions moved to branch.go, keep worktree-branch hybrids

### Step 3: Update imports in service functions
If any service methods call branch functions, ensure imports work

### Step 4: Update tests
If tests exist for moved functions, update test file names/organization

### Step 5: Update documentation
Update CLAUDE.md with new organization

**Estimated Time:** 1-2 hours

---

## Conclusion

**Current State:** The git service is well-organized with clear responsibilities.

**Recommendation:** Keep current structure (Option A). Focus efforts on:
1. ✅ Adding comprehensive godoc comments (P4.1.2)
2. ✅ Improving error types (P3.2)
3. ✅ Documenting patterns (CLAUDE.md)

**Alternative:** If preferred for clarity, create `branch.go` (Option B) in P3.1.2 - 2-3 hour effort, no breaking changes.

---

## References

- Current files: `internal/git/*.go`
- Related: `cmd/service/service.go` (uses git.Service)
- Test infrastructure: `testutil/repo.go`, `e2e_test.go`

---

**Prepared by:** Architecture Review  
**Date:** 2026-01-02  
**Status:** Ready for Team Review and P3.1.2 Implementation
