package service

import (
	"bufio"
	"fmt"
	"gbm/internal/git"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// SyncStatus represents the synchronization status between config and actual worktrees.
type SyncStatus struct {
	MissingWorktrees  []string                // Worktrees in config but not on disk
	OrphanedWorktrees map[string]string       // Worktrees on disk but not in config (name -> branch)
	BranchChanges     map[string]BranchChange // Worktrees where branch differs from config
	WorktreeMoves     []WorktreeMove          // Worktrees that can be moved/renamed
	InSync            bool                    // True if everything matches
}

// BranchChange represents a worktree where the branch needs to change.
type BranchChange struct {
	WorktreeName  string
	CurrentBranch string
	DesiredBranch string
}

// WorktreeMove represents a worktree that can be renamed/moved.
type WorktreeMove struct {
	OldName string
	NewName string
	Branch  string
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

	// Detect rename/move opportunities (orphaned -> missing with same branch)
	// Build reverse index: branch -> orphaned worktree names
	orphanedByBranch := make(map[string][]string)
	for name, branch := range orphanedMap {
		orphanedByBranch[branch] = append(orphanedByBranch[branch], name)
	}

	// Find missing worktrees that have matching orphaned worktrees
	for missingName, missingBranch := range missingMap {
		orphanedNames := orphanedByBranch[missingBranch]

		// Only create a move if there's exactly one orphaned worktree with this branch
		// (avoids ambiguity about which one to move)
		if len(orphanedNames) == 1 {
			orphanedName := orphanedNames[0]

			// Create move entry
			status.WorktreeMoves = append(status.WorktreeMoves, WorktreeMove{
				OldName: orphanedName,
				NewName: missingName,
				Branch:  missingBranch,
			})

			// Remove from missing and orphaned since they'll be handled by move
			delete(missingMap, missingName)
			delete(orphanedMap, orphanedName)
		}
	}

	// Convert remaining missing and orphaned to final lists
	for name := range missingMap {
		status.MissingWorktrees = append(status.MissingWorktrees, name)
	}
	status.OrphanedWorktrees = orphanedMap

	return status, nil
}

// performSync synchronizes actual worktrees with configured worktrees.
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

	// Handle worktree moves FIRST (rename/move worktrees)
	// This preserves uncommitted work when adopting ad-hoc worktrees into config
	if err := performWorktreeMoves(svc, status.WorktreeMoves, dryRun, confirmFunc); err != nil {
		return err
	}

	// Handle missing worktrees (create them)
	if err := createMissingWorktrees(svc, status.MissingWorktrees, worktreesDir, dryRun); err != nil {
		return err
	}

	// Handle branch changes (destructive - requires confirmation)
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

		if _, err := svc.Git.RemoveWorktree(name, true, dryRun); err != nil {
			return fmt.Errorf("failed to remove worktree '%s': %w", name, err)
		}

		configEntry := config.Worktrees[name]
		_, err := svc.Git.AddWorktree(worktreesDir, name, configEntry.Branch, false, "", dryRun)
		if err != nil {
			baseBranch := config.DefaultBranch
			if _, err = svc.Git.AddWorktree(worktreesDir, name, configEntry.Branch, true, baseBranch, dryRun); err != nil {
				return fmt.Errorf("failed to recreate worktree '%s': %w", name, err)
			}
		}

		PrintSuccess(fmt.Sprintf("Updated worktree '%s' to branch '%s'", name, configEntry.Branch))
	}
	return nil
}

func showSyncStatus(svc *Service, status *SyncStatus) error {
	PrintInfo("[DRY RUN] Showing what would be changed")

	config := svc.GetConfig()

	if len(status.WorktreeMoves) > 0 {
		PrintInfo("Worktree renames/moves (will prompt for confirmation):")
		for _, move := range status.WorktreeMoves {
			fmt.Printf("  ↔ %s → %s (branch: %s)\n", move.OldName, move.NewName, move.Branch)
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
