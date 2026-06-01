package service

import (
	"gbm/internal/git"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildDetectInputs mirrors the bookkeeping getSyncStatus does before running
// the detect passes: it derives the missing/orphaned/branch-change maps from a
// desired config (name -> branch) and the actual on-disk worktrees.
func buildDetectInputs(config map[string]string, actual []git.Worktree) (
	status *SyncStatus,
	actualMap map[string]git.Worktree,
	missingMap, orphanedMap map[string]string,
	orphanedByBranch map[string][]string,
) {
	actualMap = make(map[string]git.Worktree)
	for _, wt := range actual {
		actualMap[wt.Name] = wt
	}

	status = &SyncStatus{
		OrphanedWorktrees: make(map[string]string),
		BranchChanges:     make(map[string]BranchChange),
		InSync:            true,
	}
	missingMap = make(map[string]string)
	orphanedMap = make(map[string]string)

	for name, branch := range config {
		actualWt, exists := actualMap[name]
		if !exists {
			missingMap[name] = branch
			continue
		}
		if actualWt.Branch != branch {
			status.BranchChanges[name] = BranchChange{
				WorktreeName:  name,
				CurrentBranch: actualWt.Branch,
				DesiredBranch: branch,
			}
		}
	}
	for name, wt := range actualMap {
		if _, ok := config[name]; !ok {
			orphanedMap[name] = wt.Branch
		}
	}
	orphanedByBranch = make(map[string][]string)
	for name, branch := range orphanedMap {
		orphanedByBranch[branch] = append(orphanedByBranch[branch], name)
	}
	return status, actualMap, missingMap, orphanedMap, orphanedByBranch
}

func TestDetectShifts(t *testing.T) {
	testCases := []struct {
		// desired config: worktree name -> branch
		config map[string]string
		assert func(t *testing.T, status *SyncStatus, missingMap, orphanedMap map[string]string)
		name   string
		// actual on-disk worktrees
		actual []git.Worktree
	}{
		{
			// The reported scenario: production(04-1, WIP) should rotate to GS,
			// and the orphan preview(05-1) should fill production. Lossless.
			name: "lossless rotation is detected and consumes the BC, missing, and orphan",
			config: map[string]string{
				"production": "production-2026-05-1",
				"GS":         "production-2026-04-1",
			},
			actual: []git.Worktree{
				{Name: "production", Branch: "production-2026-04-1", Path: "/wt/production"},
				{Name: "preview", Branch: "production-2026-05-1", Path: "/wt/preview"},
			},
			assert: func(t *testing.T, status *SyncStatus, missingMap, orphanedMap map[string]string) {
				t.Helper()
				require.Len(t, status.WorktreeShifts, 1)
				s := status.WorktreeShifts[0]
				assert.Equal(t, "production", s.FromName)
				assert.Equal(t, "GS", s.ToName)
				assert.Equal(t, "production-2026-04-1", s.HeldBranch)
				assert.Equal(t, "preview", s.OrphanName)
				assert.Equal(t, "/wt/preview", s.OrphanPath)
				assert.Equal(t, "production-2026-05-1", s.DesiredBranch)
				assert.Equal(t, "/wt/production", s.FromPath)

				// The shift consumes all three entries so no lossy/duplicate
				// pass picks them up afterward.
				assert.Empty(t, status.BranchChanges, "BC should be consumed")
				assert.NotContains(t, missingMap, "GS", "missing slot should be consumed")
				assert.NotContains(t, orphanedMap, "preview", "orphan should be consumed")
			},
		},
		{
			// When the desired branch is NOT held by any orphan, this is a plain
			// redirect (create-fresh), not a shift. detectShifts must abstain so
			// the later redirect pass handles it.
			name: "no orphan on desired branch -> no shift (leaves work for redirect)",
			config: map[string]string{
				"production": "production-2026-05-1",
				"GS":         "production-2026-04-1",
			},
			actual: []git.Worktree{
				{Name: "production", Branch: "production-2026-04-1", Path: "/wt/production"},
				{Name: "preview", Branch: "INGSVC-5692", Path: "/wt/preview"},
			},
			assert: func(t *testing.T, status *SyncStatus, missingMap, orphanedMap map[string]string) {
				t.Helper()
				assert.Empty(t, status.WorktreeShifts)
				assert.Contains(t, status.BranchChanges, "production", "BC left intact for redirect")
				assert.Contains(t, missingMap, "GS")
				assert.Contains(t, orphanedMap, "preview")
			},
		},
		{
			// The BC's current branch fills no missing slot, so there is nothing
			// to rotate into — fall through (adoption territory).
			name: "current branch fills no missing slot -> no shift",
			config: map[string]string{
				"production": "production-2026-05-1",
			},
			actual: []git.Worktree{
				{Name: "production", Branch: "production-2026-04-1", Path: "/wt/production"},
				{Name: "preview", Branch: "production-2026-05-1", Path: "/wt/preview"},
			},
			assert: func(t *testing.T, status *SyncStatus, missingMap, orphanedMap map[string]string) {
				t.Helper()
				assert.Empty(t, status.WorktreeShifts)
				assert.Contains(t, status.BranchChanges, "production")
			},
		},
		{
			// Two orphans on the desired branch -> ambiguous which fills the
			// vacated slot, so abstain.
			name: "ambiguous orphan (two on desired branch) -> no shift",
			config: map[string]string{
				"production": "production-2026-05-1",
				"GS":         "production-2026-04-1",
			},
			actual: []git.Worktree{
				{Name: "production", Branch: "production-2026-04-1", Path: "/wt/production"},
				{Name: "preview", Branch: "production-2026-05-1", Path: "/wt/preview"},
				{Name: "preview2", Branch: "production-2026-05-1", Path: "/wt/preview2"},
			},
			assert: func(t *testing.T, status *SyncStatus, missingMap, orphanedMap map[string]string) {
				t.Helper()
				assert.Empty(t, status.WorktreeShifts)
				assert.Contains(t, status.BranchChanges, "production")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			status, actualMap, missingMap, orphanedMap, orphanedByBranch := buildDetectInputs(tc.config, tc.actual)
			detectShifts(status, actualMap, missingMap, orphanedMap, orphanedByBranch)
			tc.assert(t, status, missingMap, orphanedMap)
		})
	}
}

// TestDetectShiftsBeatsAdoption verifies the pipeline ordering: when both a
// lossless shift and a lossy adoption could apply to the same BranchChange, the
// shift must win so the tracked worktree's WIP is never trashed.
func TestDetectShiftsBeatsAdoption(t *testing.T) {
	config := map[string]string{
		"production": "production-2026-05-1",
		"GS":         "production-2026-04-1",
	}
	actual := []git.Worktree{
		{Name: "production", Branch: "production-2026-04-1", Path: "/wt/production"},
		{Name: "preview", Branch: "production-2026-05-1", Path: "/wt/preview"},
	}
	status, actualMap, missingMap, orphanedMap, orphanedByBranch := buildDetectInputs(config, actual)

	// Same order as getSyncStatus.
	detectShifts(status, actualMap, missingMap, orphanedMap, orphanedByBranch)
	detectAdoptions(status, actualMap, orphanedMap, orphanedByBranch)

	require.Len(t, status.WorktreeShifts, 1, "rotation should be chosen")
	assert.Empty(t, status.WorktreeAdoptions, "adoption must not fire once the shift consumed the BC")
}
