package service

import (
	"fmt"
	"gbm/internal/git"
	"sort"
	"strings"
)

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

// detectRedirects finds 1:1 pairs where a tracked worktree with a pending
// BranchChange happens to sit on the branch a missing config slot wants.
// The lossless resolution: rename the worktree into the missing slot
// (preserves dirty/untracked files), then create a fresh worktree at the
// original slot on its config-desired branch.
//
// Detection only — execution prompts the user. On decline, the BC and
// Missing entries stay in place and the destructive recreate path handles
// them (which still works thanks to topo-sorted BCs + the BC-before-Missing
// ordering).
//
// Only 1:1 matches are taken (one BC and one Missing for a given branch);
// ambiguity falls through.
func detectRedirects(status *SyncStatus, actualMap map[string]git.Worktree, missingMap map[string]string) {
	missingsByBranch := make(map[string][]string)
	for name, branch := range missingMap {
		missingsByBranch[branch] = append(missingsByBranch[branch], name)
	}
	bcsByCurrentBranch := make(map[string][]string)
	for name, bc := range status.BranchChanges {
		bcsByCurrentBranch[bc.CurrentBranch] = append(bcsByCurrentBranch[bc.CurrentBranch], name)
	}

	for bcName, bc := range status.BranchChanges {
		missings := missingsByBranch[bc.CurrentBranch]
		bcs := bcsByCurrentBranch[bc.CurrentBranch]
		if len(missings) != 1 || len(bcs) != 1 {
			continue
		}
		status.WorktreeRedirects = append(status.WorktreeRedirects, WorktreeRedirect{
			FromName:    bcName,
			ToName:      missings[0],
			HeldBranch:  bc.CurrentBranch,
			FreshBranch: bc.DesiredBranch,
			FromPath:    actualMap[bcName].Path,
		})
	}
}

// orderBranchChanges returns BC names in topological order so that any BC
// releasing a branch needed by another BC runs first.
//
// Dependency: bc_a -> bc_b when bc_b.DesiredBranch == bc_a.CurrentBranch
// (bc_a releases the branch bc_b takes, so bc_a must come first).
//
// 2-cycles are detected upstream as swaps; a cycle of length >=3 means
// every involved BC is sitting on a branch some other BC wants, and we
// can't break it via destructive recreate alone. We surface that as an
// error rather than triggering misleading "branch already used by worktree"
// failures mid-run.
func orderBranchChanges(bcs map[string]BranchChange) ([]string, error) {
	indeg := make(map[string]int, len(bcs))
	out := make(map[string][]string, len(bcs))
	for name := range bcs {
		indeg[name] = 0
	}
	for name, bc := range bcs {
		for otherName, otherBC := range bcs {
			if otherName == name {
				continue
			}
			if otherBC.CurrentBranch == bc.DesiredBranch {
				out[otherName] = append(out[otherName], name)
				indeg[name]++
			}
		}
	}

	var ready []string
	for name, d := range indeg {
		if d == 0 {
			ready = append(ready, name)
		}
	}
	sort.Strings(ready)

	order := make([]string, 0, len(bcs))
	for len(ready) > 0 {
		next := ready[0]
		ready = ready[1:]
		order = append(order, next)

		dependents := out[next]
		sort.Strings(dependents)
		var newlyReady []string
		for _, dep := range dependents {
			indeg[dep]--
			if indeg[dep] == 0 {
				newlyReady = append(newlyReady, dep)
			}
		}
		sort.Strings(newlyReady)
		ready = append(ready, newlyReady...)
	}

	if len(order) < len(bcs) {
		seen := make(map[string]bool, len(order))
		for _, n := range order {
			seen[n] = true
		}
		stuck := make([]string, 0, len(bcs)-len(order))
		for name := range bcs {
			if !seen[name] {
				stuck = append(stuck, name)
			}
		}
		sort.Strings(stuck)
		return nil, fmt.Errorf(
			"branch dependency cycle among worktrees [%s]; "+
				"manually move one off its current branch and re-run sync",
			strings.Join(stuck, ", "),
		)
	}

	return order, nil
}
