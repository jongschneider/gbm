package service

import (
	"bufio"
	"fmt"
	"gbm/internal/git"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// SyncStatus represents the synchronization status between config and actual worktrees.
type SyncStatus struct {
	MissingWorktrees  []string                // Worktrees in config but not on disk
	OrphanedWorktrees map[string]string       // Worktrees on disk but not in config (name -> branch)
	BranchChanges     map[string]BranchChange // Worktrees where branch differs from config
	WorktreeMoves     []WorktreeMove          // Orphans adopted to satisfy a missing config entry
	WorktreeSwaps     []WorktreeSwap          // Two tracked worktrees swapping branches
	WorktreeAdoptions []WorktreeAdoption      // Branch change satisfied by adopting an orphan on the desired branch
	WorktreeRedirects []WorktreeRedirect      // Tracked BC worktree renamed into a missing slot whose branch it already holds
	WorktreeShifts    []WorktreeShift         // Lossless rotation: BC worktree fills a missing slot, orphan fills the BC's vacated slot
	InSync            bool                    // True if everything matches
}

// BranchChange represents a worktree where the branch needs to change.
type BranchChange struct {
	WorktreeName  string
	CurrentBranch string
	DesiredBranch string
}

// WorktreeMove represents an orphan worktree being renamed to fill a missing config slot.
type WorktreeMove struct {
	OldName string
	NewName string
	Branch  string
}

// WorktreeSwap represents two tracked worktrees that hold each other's desired branch.
// Resolved by moving both through a temp path so dirty/untracked files are preserved
// in both worktrees.
type WorktreeSwap struct {
	NameA, BranchA string // worktree A currently holds BranchA; config wants BranchB
	PathA          string
	NameB, BranchB string // worktree B currently holds BranchB; config wants BranchA
	PathB          string
}

// WorktreeAdoption represents a branch change whose desired branch is already
// checked out in an orphan (ad-hoc) worktree. Resolved by tearing down the
// existing tracked worktree (its dirty work goes to Trash) and renaming the
// orphan into its slot via git worktree move (preserves orphan's dirty work).
type WorktreeAdoption struct {
	Name          string // tracked worktree being repointed
	CurrentBranch string // branch the tracked worktree is on now (will be torn down)
	OrphanName    string // ad-hoc worktree to adopt
	OrphanPath    string
	DesiredBranch string // == orphan's branch
}

// WorktreeRedirect represents a tracked worktree (with a pending BranchChange)
// whose CurrentBranch happens to match the DesiredBranch of a missing config
// slot. Resolved losslessly by renaming the worktree into the missing slot
// via git worktree move (preserves dirty/untracked files), then creating a
// fresh worktree at the original slot on its config-desired branch.
//
// Without this pass, sync would destructively trash the worktree's WIP and
// then create both slots from scratch.
type WorktreeRedirect struct {
	FromName    string // current slot name (will end up empty + then recreated fresh)
	ToName      string // missing slot name being filled by the rename
	HeldBranch  string // branch being preserved (BC.CurrentBranch == Missing's desired)
	FreshBranch string // branch the fresh worktree at FromName will check out (BC.DesiredBranch)
	FromPath    string // current path of the worktree to move
}

// WorktreeShift represents a fully lossless rotation: a tracked worktree (with a
// pending BranchChange) whose CURRENT branch fills a missing config slot, and
// whose DESIRED branch is already held by an orphan (ad-hoc) worktree. Resolved
// with two git worktree moves and zero data loss:
//
//  1. FromName (on HeldBranch, + WIP) → ToName   — fills the missing slot
//  2. OrphanName (on DesiredBranch)   → FromName — fills the now-vacated slot
//
// This strictly dominates the alternatives: adoption would trash FromName's WIP,
// and a redirect would try to create FromName fresh on DesiredBranch and fail
// because the orphan already has that branch checked out.
type WorktreeShift struct {
	FromName      string // BC slot: vacated by step 1, refilled by the orphan in step 2
	ToName        string // missing slot filled by FromName's dir (preserves its WIP)
	HeldBranch    string // branch preserved by moving FromName → ToName (BC.CurrentBranch)
	OrphanName    string // orphan moved into FromName's vacated slot
	OrphanPath    string // current path of the orphan
	DesiredBranch string // branch the orphan holds (== BC.DesiredBranch)
	FromPath      string // current path of the worktree being moved into ToName
}

func newSyncCommand(svc *Service) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Synchronize worktrees with config.yaml definitions",
		Long: `Synchronize worktrees with config.yaml definitions.

Creates missing worktrees defined in config.yaml and updates worktrees that
are on the wrong branch. Ad-hoc worktrees (not in config) are left alone.

Destructive operations (removing/recreating worktrees) will prompt for
confirmation unless --force is specified.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Fetch from remote first to ensure we have latest refs
			PrintMessage("Fetching from remote...\n")
			if err := svc.Git.Fetch(ShouldUseDryRun()); err != nil {
				return fmt.Errorf("failed to fetch from remote: %w", err)
			}

			// Get sync status
			status, err := getSyncStatus(svc, ShouldUseDryRun())
			if err != nil {
				return err
			}

			// In dry-run mode, just show what would happen
			if ShouldUseDryRun() {
				return showSyncStatus(svc, status)
			}

			// Check if everything is in sync
			if status.InSync {
				PrintSuccess("All worktrees are in sync with config.yaml")
				return nil
			}

			// Create confirmation function
			var confirmFunc func(string) bool
			if force {
				confirmFunc = func(message string) bool {
					PrintMessage("%s [forced: yes]\n", message)
					return true
				}
			} else {
				confirmFunc = func(message string) bool {
					fmt.Print(message + " (y/N): ")
					reader := bufio.NewReader(os.Stdin)
					response, err := reader.ReadString('\n')
					if err != nil {
						return false
					}
					response = strings.TrimSpace(strings.ToLower(response))
					return response == "y" || response == "yes"
				}
			}

			// Perform sync
			return performSync(svc, status, ShouldUseDryRun(), confirmFunc)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Skip confirmation prompts for destructive operations")

	return cmd
}

// getSyncStatus compares configured worktrees with actual worktrees and returns the differences.
func getSyncStatus(svc *Service, _ bool) (*SyncStatus, error) {
	config := svc.GetConfig()

	// Get actual worktrees from git (always read real state, even in dry-run)
	actualWorktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Build maps for easier comparison
	actualMap := make(map[string]git.Worktree)
	for _, wt := range actualWorktrees {
		if !wt.IsBare {
			actualMap[wt.Name] = wt
		}
	}

	status := &SyncStatus{
		MissingWorktrees:  []string{},
		OrphanedWorktrees: make(map[string]string),
		BranchChanges:     make(map[string]BranchChange),
		WorktreeMoves:     []WorktreeMove{},
		WorktreeSwaps:     []WorktreeSwap{},
		WorktreeAdoptions: []WorktreeAdoption{},
		WorktreeRedirects: []WorktreeRedirect{},
		WorktreeShifts:    []WorktreeShift{},
		InSync:            true,
	}

	// Temporary maps for missing and orphaned before move detection
	missingMap := make(map[string]string)  // name -> branch
	orphanedMap := make(map[string]string) // name -> branch

	// Find missing worktrees (in config but not on disk)
	for name, configEntry := range config.Worktrees {
		actual, exists := actualMap[name]
		if !exists {
			missingMap[name] = configEntry.Branch
			status.InSync = false
			continue
		}

		// Check if branch matches
		if actual.Branch != configEntry.Branch {
			status.BranchChanges[name] = BranchChange{
				WorktreeName:  name,
				CurrentBranch: actual.Branch,
				DesiredBranch: configEntry.Branch,
			}
			status.InSync = false
		}
	}

	// Find orphaned worktrees (on disk but not in config)
	// Note: Orphaned worktrees are OK - they're ad-hoc worktrees created manually
	// They don't affect InSync status, but we track them for:
	// - Move detection (can be renamed to match config)
	// - Display to user for informational purposes
	for name, wt := range actualMap {
		if _, exists := config.Worktrees[name]; !exists {
			orphanedMap[name] = wt.Branch
			// Don't set InSync = false - orphaned worktrees are intentional
		}
	}

	orphanedByBranch := make(map[string][]string)
	for name, branch := range orphanedMap {
		orphanedByBranch[branch] = append(orphanedByBranch[branch], name)
	}

	detectMoves(status, missingMap, orphanedMap, orphanedByBranch)
	detectSwaps(status, actualMap)
	detectShifts(status, actualMap, missingMap, orphanedMap, orphanedByBranch)
	detectAdoptions(status, actualMap, orphanedMap, orphanedByBranch)
	detectRedirects(status, actualMap, missingMap)

	for name := range missingMap {
		status.MissingWorktrees = append(status.MissingWorktrees, name)
	}
	status.OrphanedWorktrees = orphanedMap

	return status, nil
}

// performSync synchronizes actual worktrees with configured worktrees.
//
// Order is intentional: lossless passes first, then destructive ones.
// Branch changes run before Missings because a branch change can RELEASE
// a branch that a missing worktree needs to take. (See orderBranchChanges
// for the intra-BC ordering — chains like A releases X that B needs.)
//  1. Moves          — orphan renamed into missing slot (lossless)
//  2. Swaps          — two tracked worktrees exchange branches via temp (lossless)
//  3. Shifts         — BC worktree fills a missing slot, orphan fills the BC's
//     vacated slot (two moves, fully lossless; runs before Adoptions so the
//     no-trash rotation wins over the lossy adoption for the same BC)
//  4. Adoptions      — orphan adopted to satisfy a branch change (orphan lossless,
//     old tracked worktree's dirty work trashed)
//  5. Redirects      — tracked BC worktree renamed into a missing slot whose
//     branch it already holds (lossless; on decline, falls through to step 6+7)
//  6. Branch changes — destructive remove-and-recreate, ordered so releasers run
//     before takers
//  7. Missing        — fresh worktrees created from config (now safe to take
//     branches released by step 5 or 6)
func performSync(
	svc *Service,
	status *SyncStatus,
	dryRun bool,
	confirmFunc func(message string) bool,
) error {
	worktreesDir, err := svc.GetWorktreesPath()
	if err != nil {
		return err
	}

	if err := performWorktreeMoves(svc, status.WorktreeMoves, dryRun, confirmFunc); err != nil {
		return err
	}

	if err := performWorktreeSwaps(svc, status.WorktreeSwaps, worktreesDir, dryRun, confirmFunc); err != nil {
		return err
	}

	if err := performWorktreeShifts(svc, status.WorktreeShifts, worktreesDir, dryRun, confirmFunc); err != nil {
		return err
	}

	if err := performWorktreeAdoptions(svc, status.WorktreeAdoptions, worktreesDir, dryRun, confirmFunc); err != nil {
		return err
	}

	if err := performWorktreeRedirects(svc, status, worktreesDir, dryRun, confirmFunc); err != nil {
		return err
	}

	if err := handleBranchChanges(svc, status.BranchChanges, worktreesDir, dryRun, confirmFunc); err != nil {
		return err
	}

	if err := createMissingWorktrees(svc, status.MissingWorktrees, worktreesDir, dryRun); err != nil {
		return err
	}

	return nil
}

// performWorktreeMoves handles moving/renaming worktrees.
func performWorktreeMoves(svc *Service, moves []WorktreeMove, dryRun bool, confirmFunc func(string) bool) error {
	for _, move := range moves {
		message := fmt.Sprintf("Move worktree '%s' to '%s' (branch: %s)?", move.OldName, move.NewName, move.Branch)

		if !dryRun && confirmFunc != nil && !confirmFunc(message) {
			PrintMessage("Skipped moving worktree '%s'\n", move.OldName)
			PrintInfo(fmt.Sprintf("The ad-hoc worktree '%s' will remain, and '%s' will not be created", move.OldName, move.NewName))
			continue
		}

		if dryRun {
			fmt.Printf("[DRY RUN] Would move worktree '%s' to '%s' (branch: %s)\n", move.OldName, move.NewName, move.Branch)
			continue
		}

		err := svc.Git.MoveWorktree(move.OldName, move.NewName, dryRun)
		if err != nil {
			return fmt.Errorf("failed to move worktree '%s' to '%s': %w", move.OldName, move.NewName, err)
		}

		PrintSuccess(fmt.Sprintf("Moved worktree '%s' → '%s' (branch: %s)", move.OldName, move.NewName, move.Branch))
	}
	return nil
}

// performWorktreeSwaps swaps two tracked worktrees that hold each other's desired branch.
//
// Executes three git worktree moves through a temp path:
//  1. A → tmp (frees A's slot, A still on BranchA)
//  2. B → A's slot (B still on BranchB)
//  3. tmp → B's slot
//
// Dirty/untracked files in both worktrees are preserved. On partial failure,
// rolls back completed moves so the user is left with the original layout.
func performWorktreeSwaps(svc *Service, swaps []WorktreeSwap, worktreesDir string, dryRun bool, confirmFunc func(string) bool) error {
	for _, swap := range swaps {
		message := fmt.Sprintf(
			"Swap branches between '%s' (%s) and '%s' (%s)? "+
				"After: '%s' will hold '%s', '%s' will hold '%s'. "+
				"Working-tree contents move with the worktree (dirty files preserved).",
			swap.NameA, swap.BranchA, swap.NameB, swap.BranchB,
			swap.NameA, swap.BranchB, swap.NameB, swap.BranchA,
		)

		if !dryRun && confirmFunc != nil && !confirmFunc(message) {
			PrintMessage("Skipped swap between '%s' and '%s'\n", swap.NameA, swap.NameB)
			continue
		}

		tmpName := fmt.Sprintf("__gbm_swap_%d", time.Now().UnixNano())
		tmpPath := filepath.Join(worktreesDir, tmpName)

		if dryRun {
			fmt.Printf("[DRY RUN] git worktree move %s %s\n", swap.PathA, tmpPath)
			fmt.Printf("[DRY RUN] git worktree move %s %s\n", swap.PathB, swap.PathA)
			fmt.Printf("[DRY RUN] git worktree move %s %s\n", tmpPath, swap.PathB)
			continue
		}

		// Step 1: park A at temp.
		if err := svc.Git.MoveWorktreeByPath(swap.PathA, tmpPath, false); err != nil {
			return fmt.Errorf("swap step 1 (park '%s' at tmp): %w", swap.NameA, err)
		}

		// Step 2: move B into A's slot.
		if err := svc.Git.MoveWorktreeByPath(swap.PathB, swap.PathA, false); err != nil {
			// Roll back step 1.
			if rbErr := svc.Git.MoveWorktreeByPath(tmpPath, swap.PathA, false); rbErr != nil {
				return fmt.Errorf("swap step 2 (move '%s' → '%s'): %w; rollback also failed: %w",
					swap.NameB, swap.NameA, err, rbErr)
			}
			return fmt.Errorf("swap step 2 (move '%s' → '%s'): %w (rolled back)", swap.NameB, swap.NameA, err)
		}

		// Step 3: move parked A into B's now-empty slot.
		if err := svc.Git.MoveWorktreeByPath(tmpPath, swap.PathB, false); err != nil {
			// Roll back steps 2 and 1 in reverse order.
			rb1 := svc.Git.MoveWorktreeByPath(swap.PathA, swap.PathB, false)
			rb2 := svc.Git.MoveWorktreeByPath(tmpPath, swap.PathA, false)
			if rb1 != nil || rb2 != nil {
				return fmt.Errorf("swap step 3 (move tmp → '%s'): %w; rollback failed: rb1=%w rb2=%w",
					swap.NameB, err, rb1, rb2)
			}
			return fmt.Errorf("swap step 3 (move tmp → '%s'): %w (rolled back)", swap.NameB, err)
		}

		PrintSuccess(fmt.Sprintf("Swapped '%s' ↔ '%s' (branches: %s ↔ %s, dirty files preserved)",
			swap.NameA, swap.NameB, swap.BranchA, swap.BranchB))
	}
	return nil
}

// performWorktreeShifts performs the lossless rotation: move the tracked worktree
// (with its WIP) into the missing slot, then move the orphan into the slot the
// tracked worktree just vacated. Two git worktree moves, nothing trashed.
//
// Rollback-safe: if step 2 fails, step 1 is reversed so the user is left with the
// original layout.
func performWorktreeShifts(svc *Service, shifts []WorktreeShift, worktreesDir string, dryRun bool, confirmFunc func(string) bool) error {
	for _, s := range shifts {
		message := fmt.Sprintf(
			"Rotate losslessly: move '%s' (on '%s', preserves WIP) -> '%s', then move orphan '%s' (on '%s') -> '%s'? Nothing is trashed.",
			s.FromName, s.HeldBranch, s.ToName, s.OrphanName, s.DesiredBranch, s.FromName,
		)

		if !dryRun && confirmFunc != nil && !confirmFunc(message) {
			PrintMessage("Skipped shift '%s' -> '%s' / '%s' -> '%s'\n", s.FromName, s.ToName, s.OrphanName, s.FromName)
			continue
		}

		toPath := filepath.Join(worktreesDir, s.ToName)
		fromPath := filepath.Join(worktreesDir, s.FromName)

		if dryRun {
			fmt.Printf("[DRY RUN] git worktree move %s %s   # preserve WIP on '%s'\n", s.FromPath, toPath, s.HeldBranch)
			fmt.Printf("[DRY RUN] git worktree move %s %s   # move orphan on '%s' into vacated slot\n", s.OrphanPath, fromPath, s.DesiredBranch)
			continue
		}

		// Step 1: move the tracked worktree into the missing slot (keeps its WIP).
		if err := svc.Git.MoveWorktreeByPath(s.FromPath, toPath, false); err != nil {
			return fmt.Errorf("shift: failed to move '%s' -> '%s': %w", s.FromName, s.ToName, err)
		}

		// Step 2: move the orphan into the now-vacated original slot.
		if err := svc.Git.MoveWorktreeByPath(s.OrphanPath, fromPath, false); err != nil {
			// Roll back step 1 so the user keeps the original layout.
			if rbErr := svc.Git.MoveWorktreeByPath(toPath, s.FromPath, false); rbErr != nil {
				return fmt.Errorf("shift: move orphan '%s' -> '%s' failed: %w; rollback also failed: %w",
					s.OrphanName, s.FromName, err, rbErr)
			}
			return fmt.Errorf("shift: move orphan '%s' -> '%s' failed: %w (rolled back)", s.OrphanName, s.FromName, err)
		}

		PrintSuccess(fmt.Sprintf("Shifted '%s' -> '%s' (preserved WIP on '%s'), '%s' -> '%s' (on '%s')",
			s.FromName, s.ToName, s.HeldBranch, s.OrphanName, s.FromName, s.DesiredBranch))

		if err := svc.CopyFilesToWorktree(s.ToName); err != nil {
			PrintWarning(fmt.Sprintf("File copy failed for worktree '%s': %v", s.ToName, err))
		}
		if err := svc.CopyFilesToWorktree(s.FromName); err != nil {
			PrintWarning(fmt.Sprintf("File copy failed for worktree '%s': %v", s.FromName, err))
		}
	}
	return nil
}

// performWorktreeAdoptions handles branch changes whose desired branch is already
// held by an orphan (ad-hoc) worktree.
//
// Three steps, rollback-safe:
//  1. Park existing tracked worktree at a temp path (git worktree move; preserves
//     its dirty files). If this fails, no state has changed.
//  2. Move orphan into the now-empty slot. On failure, roll back step 1 by
//     moving the parked worktree back to its original path.
//  3. Trash the parked worktree (its branch is being repointed; its working
//     tree goes to macOS Trash). On failure here the adoption is already done;
//     emit a warning so the user knows a stray temp worktree exists.
func performWorktreeAdoptions(svc *Service, adoptions []WorktreeAdoption, worktreesDir string, dryRun bool, confirmFunc func(string) bool) error {
	for _, adopt := range adoptions {
		message := fmt.Sprintf(
			"Worktree '%s' is on '%s' but config wants '%s'. Adopt orphan '%s' (already on '%s') as the new '%s'? "+
				"'%s' (with its dirty files) will be moved to Trash; orphan's dirty files preserved.",
			adopt.Name, adopt.CurrentBranch, adopt.DesiredBranch,
			adopt.OrphanName, adopt.DesiredBranch, adopt.Name, adopt.Name,
		)

		if !dryRun && confirmFunc != nil && !confirmFunc(message) {
			PrintMessage("Skipped adoption of '%s' as '%s'\n", adopt.OrphanName, adopt.Name)
			continue
		}

		oldPath := filepath.Join(worktreesDir, adopt.Name)
		// Include the original worktree name in the tmp slot so the eventual
		// Trash entry (which timestamps this name further) is still discoverable.
		tmpName := fmt.Sprintf("__gbm_adopt_%s_%d", adopt.Name, time.Now().UnixNano())
		tmpPath := filepath.Join(worktreesDir, tmpName)

		if dryRun {
			fmt.Printf("[DRY RUN] git worktree move %s %s   # park existing\n", oldPath, tmpPath)
			fmt.Printf("[DRY RUN] git worktree move %s %s   # adopt orphan\n", adopt.OrphanPath, oldPath)
			fmt.Printf("[DRY RUN] trash worktrees/%s        # discard parked old worktree\n", tmpName)
			continue
		}

		// Step 1: park the existing tracked worktree at tmp.
		if err := svc.Git.MoveWorktreeByPath(oldPath, tmpPath, false); err != nil {
			return fmt.Errorf("adoption: failed to park '%s' at tmp: %w", adopt.Name, err)
		}

		// Step 2: move orphan into the now-empty slot.
		if err := svc.Git.MoveWorktreeByPath(adopt.OrphanPath, oldPath, false); err != nil {
			// Roll back step 1.
			if rbErr := svc.Git.MoveWorktreeByPath(tmpPath, oldPath, false); rbErr != nil {
				return fmt.Errorf("adoption: move orphan '%s' → '%s' failed: %w; rollback also failed: %w (parked at %s)",
					adopt.OrphanName, adopt.Name, err, rbErr, tmpPath)
			}
			return fmt.Errorf("adoption: move orphan '%s' → '%s' failed: %w (rolled back)",
				adopt.OrphanName, adopt.Name, err)
		}

		// Step 3: discard the parked worktree (its dirty files → Trash, branch
		// ref untouched). On failure, roll back steps 2 and 1 so the user is
		// left with the original layout: orphan back at its original path,
		// existing tracked worktree back in its slot.
		if _, err := svc.Git.RemoveWorktree(tmpName, true, false); err != nil {
			if rbErr := rollbackAdoption(svc, adopt.OrphanPath, oldPath, tmpPath); rbErr != nil {
				return fmt.Errorf("adoption: discard of parked '%s' failed: %w; rollback also failed: %w",
					adopt.Name, err, rbErr)
			}
			return fmt.Errorf("adoption: discard of parked '%s' failed: %w (rolled back)", adopt.Name, err)
		}

		PrintSuccess(fmt.Sprintf("Adopted '%s' as '%s' (branch: %s)", adopt.OrphanName, adopt.Name, adopt.DesiredBranch))

		if err := svc.CopyFilesToWorktree(adopt.Name); err != nil {
			PrintWarning(fmt.Sprintf("File copy failed for worktree '%s': %v", adopt.Name, err))
		}
	}
	return nil
}

// rollbackAdoption reverses the two git-worktree-moves performed during adoption
// before the trash step. orphanPath is where the orphan started, oldPath is the
// slot that originally held the tracked worktree, tmpPath is where the tracked
// worktree was parked.
//
// If the parked worktree is no longer at tmpPath (e.g. Trash partially succeeded
// before git metadata cleanup failed), we cannot fully restore — return an error
// describing what couldn't be reversed so the caller can surface it.
func rollbackAdoption(svc *Service, orphanPath, oldPath, tmpPath string) error {
	// Reverse step 2: move orphan back to its original path.
	if err := svc.Git.MoveWorktreeByPath(oldPath, orphanPath, false); err != nil {
		return fmt.Errorf("restore orphan to %s: %w", orphanPath, err)
	}

	// Reverse step 1: move parked worktree back into its original slot.
	// If the parked dir is gone (trashed before metadata cleanup failed), we
	// cannot restore it — leave it to the user to recover from ~/.Trash.
	if _, statErr := os.Stat(tmpPath); statErr != nil {
		return fmt.Errorf("parked worktree at %s no longer exists (likely already in Trash); orphan restored but old slot remains empty: %w", tmpPath, statErr)
	}
	if err := svc.Git.MoveWorktreeByPath(tmpPath, oldPath, false); err != nil {
		return fmt.Errorf("restore parked worktree to %s: %w", oldPath, err)
	}
	return nil
}

// performWorktreeRedirects renames a tracked worktree (sitting on a branch a
// missing slot wants) into that missing slot via git worktree move. This
// preserves dirty/untracked files. The original slot is then queued as a
// fresh Missing so createMissingWorktrees creates a clean worktree on the
// BC's originally desired branch.
//
// On user decline, leaves the BC and Missing entries in place so the
// downstream destructive recreate path can handle them instead (which still
// works thanks to the topo-sorted BC ordering and BC-before-Missing flow).
func performWorktreeRedirects(svc *Service, status *SyncStatus, worktreesDir string, dryRun bool, confirmFunc func(string) bool) error {
	for _, r := range status.WorktreeRedirects {
		message := fmt.Sprintf(
			"Rename worktree '%s' (on '%s', preserves any WIP) -> '%s' to satisfy the missing config slot, then create a fresh '%s' on '%s'?",
			r.FromName, r.HeldBranch, r.ToName, r.FromName, r.FreshBranch,
		)

		if !dryRun && confirmFunc != nil && !confirmFunc(message) {
			PrintMessage("Skipped redirect '%s' -> '%s' (will fall back to destructive recreate)\n", r.FromName, r.ToName)
			continue
		}

		newPath := filepath.Join(worktreesDir, r.ToName)
		if dryRun {
			fmt.Printf("[DRY RUN] git worktree move %s %s   # preserve WIP on '%s'\n", r.FromPath, newPath, r.HeldBranch)
			fmt.Printf("[DRY RUN] create fresh worktree '%s' on '%s'\n", r.FromName, r.FreshBranch)
			continue
		}

		if err := svc.Git.MoveWorktreeByPath(r.FromPath, newPath, false); err != nil {
			return fmt.Errorf("redirect: failed to move '%s' to '%s': %w", r.FromName, r.ToName, err)
		}

		PrintSuccess(fmt.Sprintf("Redirected '%s' -> '%s' (preserved work on '%s')", r.FromName, r.ToName, r.HeldBranch))

		if err := svc.CopyFilesToWorktree(r.ToName); err != nil {
			PrintWarning(fmt.Sprintf("File copy failed for worktree '%s': %v", r.ToName, err))
		}

		// Consume the underlying BC + Missing now that the move succeeded,
		// and queue a fresh Missing for the now-empty original slot.
		delete(status.BranchChanges, r.FromName)
		status.MissingWorktrees = removeStringFromSlice(status.MissingWorktrees, r.ToName)
		status.MissingWorktrees = append(status.MissingWorktrees, r.FromName)
	}
	return nil
}

func removeStringFromSlice(s []string, v string) []string {
	for i, x := range s {
		if x == v {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

// createMissingWorktrees creates worktrees that are defined in config but don't exist on disk.
func createMissingWorktrees(svc *Service, missing []string, worktreesDir string, dryRun bool) error {
	config := svc.GetConfig()

	for _, name := range missing {
		configEntry := config.Worktrees[name]

		if dryRun {
			fmt.Printf("[DRY RUN] Would create worktree '%s' for branch '%s'\n", name, configEntry.Branch)
			continue
		}

		branchExists, err := svc.Git.BranchExists(configEntry.Branch)
		if err != nil {
			return fmt.Errorf("failed to check if branch '%s' exists: %w", configEntry.Branch, err)
		}

		createBranch := !branchExists
		baseBranch := config.DefaultBranch
		if _, err = svc.Git.AddWorktree(worktreesDir, name, configEntry.Branch, createBranch, baseBranch, dryRun); err != nil {
			return fmt.Errorf("failed to create worktree '%s': %w", name, err)
		}

		PrintSuccess(fmt.Sprintf("Created worktree '%s' for branch '%s'", name, configEntry.Branch))

		if err := svc.CopyFilesToWorktree(name); err != nil {
			PrintWarning(fmt.Sprintf("File copy failed for worktree '%s': %v", name, err))
		}
	}
	return nil
}

// handleBranchChanges handles worktrees that need to be recreated with a different branch.
func handleBranchChanges(svc *Service, changes map[string]BranchChange, worktreesDir string, dryRun bool, confirmFunc func(string) bool) error {
	config := svc.GetConfig()

	order, err := orderBranchChanges(changes)
	if err != nil {
		return err
	}

	for _, name := range order {
		change := changes[name]
		message := fmt.Sprintf("Worktree '%s' is on branch '%s' but config specifies '%s'. Remove and recreate?",
			name, change.CurrentBranch, change.DesiredBranch)

		if !dryRun && confirmFunc != nil && !confirmFunc(message) {
			PrintMessage("Skipped updating worktree '%s'\n", name)
			continue
		}

		if dryRun {
			fmt.Printf("[DRY RUN] Would remove worktree '%s' (branch '%s') and recreate with branch '%s'\n",
				name, change.CurrentBranch, change.DesiredBranch)
			continue
		}

		configEntry := config.Worktrees[name]

		branchExists, err := svc.Git.BranchExists(configEntry.Branch)
		if err != nil {
			return fmt.Errorf("failed to check if branch '%s' exists: %w", configEntry.Branch, err)
		}

		if _, err := svc.Git.RemoveWorktree(name, true, dryRun); err != nil {
			return fmt.Errorf("failed to remove worktree '%s': %w", name, err)
		}

		createBranch := !branchExists
		baseBranch := ""
		if createBranch {
			baseBranch = config.DefaultBranch
		}
		if _, err := svc.Git.AddWorktree(worktreesDir, name, configEntry.Branch, createBranch, baseBranch, dryRun); err != nil {
			return fmt.Errorf("failed to recreate worktree '%s': %w", name, err)
		}

		PrintSuccess(fmt.Sprintf("Updated worktree '%s' to branch '%s'", name, configEntry.Branch))

		if err := svc.CopyFilesToWorktree(name); err != nil {
			PrintWarning(fmt.Sprintf("File copy failed for worktree '%s': %v", name, err))
		}
	}
	return nil
}

func showSyncStatus(svc *Service, status *SyncStatus) error {
	PrintInfo("[DRY RUN] Showing what would be changed")

	config := svc.GetConfig()

	printSyncSection(len(status.WorktreeMoves), "Worktree renames/moves (will prompt for confirmation):", func() {
		for _, move := range status.WorktreeMoves {
			fmt.Printf("  → %s → %s (branch: %s)\n", move.OldName, move.NewName, move.Branch)
		}
	})
	printSyncSection(len(status.WorktreeSwaps), "Worktree swaps (will prompt for confirmation):", func() {
		for _, swap := range status.WorktreeSwaps {
			fmt.Printf("  ↔ %s (%s) ↔ %s (%s) — dirty files preserved in both\n",
				swap.NameA, swap.BranchA, swap.NameB, swap.BranchB)
		}
	})
	printSyncSection(len(status.WorktreeShifts), "Lossless rotations (two moves, nothing trashed; will prompt for confirmation):", func() {
		for _, s := range status.WorktreeShifts {
			fmt.Printf("  ⮌ %s (on %s) → %s; orphan %s (on %s) → %s\n",
				s.FromName, s.HeldBranch, s.ToName, s.OrphanName, s.DesiredBranch, s.FromName)
		}
	})
	printSyncSection(len(status.WorktreeAdoptions), "Orphan adoptions (will prompt for confirmation):", func() {
		for _, adopt := range status.WorktreeAdoptions {
			fmt.Printf("  ⇐ %s ← orphan %s (branch: %s → %s); '%s' work goes to Trash\n",
				adopt.Name, adopt.OrphanName, adopt.CurrentBranch, adopt.DesiredBranch, adopt.Name)
		}
	})
	printSyncSection(len(status.WorktreeRedirects), "Worktree redirects (lossless rename; will prompt for confirmation):", func() {
		for _, r := range status.WorktreeRedirects {
			fmt.Printf("  → %s (on %s) renames to %s; fresh %s created on %s\n",
				r.FromName, r.HeldBranch, r.ToName, r.FromName, r.FreshBranch)
		}
	})
	printSyncSection(len(status.MissingWorktrees), "Missing worktrees (will be created):", func() {
		for _, name := range status.MissingWorktrees {
			fmt.Printf("  + %s (branch: %s)\n", name, config.Worktrees[name].Branch)
		}
	})
	printSyncSection(len(status.BranchChanges), "Branch changes needed (will remove and recreate):", func() {
		for name, change := range status.BranchChanges {
			fmt.Printf("  ~ %s: %s → %s\n", name, change.CurrentBranch, change.DesiredBranch)
		}
	})
	printSyncSection(len(status.OrphanedWorktrees), "Ad-hoc worktrees (not tracked in config):", func() {
		for name, branch := range status.OrphanedWorktrees {
			fmt.Printf("  • %s (branch: %s)\n", name, branch)
		}
	})

	if status.InSync {
		PrintSuccess("All tracked worktrees are in sync with config")
	}
	return nil
}

func printSyncSection(count int, header string, body func()) {
	if count == 0 {
		return
	}
	PrintInfo(header)
	body()
	fmt.Println()
}
