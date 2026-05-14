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
	detectAdoptions(status, actualMap, orphanedMap, orphanedByBranch)

	for name := range missingMap {
		status.MissingWorktrees = append(status.MissingWorktrees, name)
	}
	status.OrphanedWorktrees = orphanedMap

	return status, nil
}

// detectMoves finds orphans whose branch matches a missing config entry (1:1).
// Lossless: orphan keeps its working tree, just renamed to fill the missing slot.
func detectMoves(status *SyncStatus, missingMap, orphanedMap map[string]string, orphanedByBranch map[string][]string) {
	for missingName, missingBranch := range missingMap {
		orphanedNames := orphanedByBranch[missingBranch]

		// Only create a move if there's exactly one orphaned worktree with this branch
		// (avoids ambiguity about which one to move)
		if len(orphanedNames) != 1 {
			continue
		}
		orphanedName := orphanedNames[0]

		status.WorktreeMoves = append(status.WorktreeMoves, WorktreeMove{
			OldName: orphanedName,
			NewName: missingName,
			Branch:  missingBranch,
		})

		delete(missingMap, missingName)
		delete(orphanedMap, orphanedName)
	}
}

// detectSwaps finds pairs of BranchChanges where each holds what the other wants.
// Lossless: both worktrees preserve dirty/untracked files via three git worktree
// moves through a temp path. Only 2-cycles are detected; N-cycles fall through.
func detectSwaps(status *SyncStatus, actualMap map[string]git.Worktree) {
	claimed := make(map[string]bool)
	names := make([]string, 0, len(status.BranchChanges))
	for name := range status.BranchChanges {
		names = append(names, name)
	}
	for _, nameA := range names {
		if claimed[nameA] {
			continue
		}
		changeA, ok := status.BranchChanges[nameA]
		if !ok {
			continue
		}
		partner := findSwapPartner(nameA, changeA, names, claimed, status.BranchChanges)
		if partner == "" {
			continue
		}
		changeB := status.BranchChanges[partner]
		status.WorktreeSwaps = append(status.WorktreeSwaps, WorktreeSwap{
			NameA: nameA, BranchA: changeA.CurrentBranch, PathA: actualMap[nameA].Path,
			NameB: partner, BranchB: changeB.CurrentBranch, PathB: actualMap[partner].Path,
		})
		claimed[nameA] = true
		claimed[partner] = true
		delete(status.BranchChanges, nameA)
		delete(status.BranchChanges, partner)
	}
}

func findSwapPartner(nameA string, changeA BranchChange, names []string, claimed map[string]bool, changes map[string]BranchChange) string {
	for _, nameB := range names {
		if nameB == nameA || claimed[nameB] {
			continue
		}
		changeB, ok := changes[nameB]
		if !ok {
			continue
		}
		if changeA.DesiredBranch == changeB.CurrentBranch &&
			changeB.DesiredBranch == changeA.CurrentBranch {
			return nameB
		}
	}
	return ""
}

// detectAdoptions finds BranchChanges whose desired branch is held by exactly
// one live orphan. Lossy for the tracked worktree being repointed (its dirty
// work goes to Trash), but lossless for the orphan (renamed via worktree move).
func detectAdoptions(status *SyncStatus, actualMap map[string]git.Worktree, orphanedMap map[string]string, orphanedByBranch map[string][]string) {
	for name, change := range status.BranchChanges {
		liveOrphan, matched := singleLiveOrphan(orphanedByBranch[change.DesiredBranch], orphanedMap)
		if !matched {
			continue
		}

		status.WorktreeAdoptions = append(status.WorktreeAdoptions, WorktreeAdoption{
			Name:          name,
			CurrentBranch: change.CurrentBranch,
			OrphanName:    liveOrphan,
			OrphanPath:    actualMap[liveOrphan].Path,
			DesiredBranch: change.DesiredBranch,
		})
		delete(status.BranchChanges, name)
		delete(orphanedMap, liveOrphan)
	}
}

func singleLiveOrphan(candidates []string, orphanedMap map[string]string) (string, bool) {
	var found string
	matches := 0
	for _, candidate := range candidates {
		if _, stillOrphan := orphanedMap[candidate]; stillOrphan {
			found = candidate
			matches++
		}
	}
	return found, matches == 1
}

// performSync synchronizes actual worktrees with configured worktrees.
//
// Order is intentional: cheapest/safest operations first, destructive last.
//  1. Moves      — orphan renamed into missing slot (lossless)
//  2. Swaps      — two tracked worktrees exchange branches via temp (lossless)
//  3. Adoptions  — orphan adopted to satisfy a branch change (orphan lossless,
//     old tracked worktree's dirty work trashed)
//  4. Missing    — fresh worktrees created from config
//  5. Branch changes — destructive remove-and-recreate fallback
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

	if err := performWorktreeAdoptions(svc, status.WorktreeAdoptions, worktreesDir, dryRun, confirmFunc); err != nil {
		return err
	}

	if err := createMissingWorktrees(svc, status.MissingWorktrees, worktreesDir, dryRun); err != nil {
		return err
	}

	if err := handleBranchChanges(svc, status.BranchChanges, worktreesDir, dryRun, confirmFunc); err != nil {
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

	for name, change := range changes {
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

	if len(status.WorktreeMoves) > 0 {
		PrintInfo("Worktree renames/moves (will prompt for confirmation):")
		for _, move := range status.WorktreeMoves {
			fmt.Printf("  → %s → %s (branch: %s)\n", move.OldName, move.NewName, move.Branch)
		}
		fmt.Println()
	}

	if len(status.WorktreeSwaps) > 0 {
		PrintInfo("Worktree swaps (will prompt for confirmation):")
		for _, swap := range status.WorktreeSwaps {
			fmt.Printf("  ↔ %s (%s) ↔ %s (%s) — dirty files preserved in both\n",
				swap.NameA, swap.BranchA, swap.NameB, swap.BranchB)
		}
		fmt.Println()
	}

	if len(status.WorktreeAdoptions) > 0 {
		PrintInfo("Orphan adoptions (will prompt for confirmation):")
		for _, adopt := range status.WorktreeAdoptions {
			fmt.Printf("  ⇐ %s ← orphan %s (branch: %s → %s); '%s' work goes to Trash\n",
				adopt.Name, adopt.OrphanName, adopt.CurrentBranch, adopt.DesiredBranch, adopt.Name)
		}
		fmt.Println()
	}

	if len(status.MissingWorktrees) > 0 {
		PrintInfo("Missing worktrees (will be created):")
		for _, name := range status.MissingWorktrees {
			branch := config.Worktrees[name].Branch
			fmt.Printf("  + %s (branch: %s)\n", name, branch)
		}
		fmt.Println()
	}

	if len(status.BranchChanges) > 0 {
		PrintInfo("Branch changes needed (will remove and recreate):")
		for name, change := range status.BranchChanges {
			fmt.Printf("  ~ %s: %s → %s\n", name, change.CurrentBranch, change.DesiredBranch)
		}
		fmt.Println()
	}

	// Always show ad-hoc worktrees if present (informational only)
	if len(status.OrphanedWorktrees) > 0 {
		PrintInfo("Ad-hoc worktrees (not tracked in config):")
		for name, branch := range status.OrphanedWorktrees {
			fmt.Printf("  • %s (branch: %s)\n", name, branch)
		}
		fmt.Println()
	}

	// Check if config-tracked worktrees are in sync
	if status.InSync {
		PrintSuccess("All tracked worktrees are in sync with config")
		return nil
	}

	return nil
}
