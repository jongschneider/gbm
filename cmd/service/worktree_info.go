package service

import (
	"errors"
	"fmt"
	"gbm/internal/git"
	"os"

	"github.com/spf13/cobra"
)

func newWorktreeInfoCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "info [worktree-name]",
		Short: "Show detailed info about a worktree",
		Long: `Show detailed info about a worktree, including the base branch it was created
from (recorded at 'wt add' time), the upstream tracking branch, divergence
against each, and working-tree status.

Usage:
  gbm wt info                    # Info for the current worktree
  gbm wt info <worktree-name>    # Info for a specific worktree
  gbm wt info --json             # Machine-readable output`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 && args[0] != "." {
				target = args[0]
			}
			return runWorktreeInfo(svc, target)
		},
	}

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		completions, err := generateWorktreeCompletions(svc)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}

	return cmd
}

func runWorktreeInfo(svc *Service, target string) error {
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}
	_, currentName, err := svc.Git.IsInWorktree(wd)
	if err != nil {
		return fmt.Errorf("failed to determine current worktree: %w", err)
	}

	name := target
	if name == "" {
		name = currentName
	}
	if name == "" {
		return errors.New("not currently in a worktree; pass a worktree name")
	}

	var wt *git.Worktree
	for i := range worktrees {
		if worktrees[i].Name == name {
			wt = &worktrees[i]
			break
		}
	}
	if wt == nil {
		return fmt.Errorf("worktree '%s' not found", name)
	}
	if wt.IsBare {
		return errors.New("cannot show info for bare repository")
	}

	info, err := buildWorktreeInfo(svc, wt, currentName)
	if err != nil {
		return err
	}

	if ShouldUseJSON() {
		return OutputJSON(info)
	}
	printWorktreeInfo(info)
	return nil
}

func buildWorktreeInfo(svc *Service, wt *git.Worktree, currentName string) (*WorktreeInfoResponse, error) {
	branch, err := svc.Git.GetWorktreeBranch(wt.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}

	head, err := svc.Git.GetHeadCommit(wt.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	upstream, err := svc.Git.GetUpstreamBranch(wt.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to get upstream: %w", err)
	}

	base, err := svc.Git.GetGbmBase(wt.Path, branch)
	if err != nil {
		return nil, fmt.Errorf("failed to get base: %w", err)
	}

	clean, err := svc.Git.IsWorktreeClean(wt.Path)
	if err != nil {
		return nil, fmt.Errorf("failed to check status: %w", err)
	}

	info := &WorktreeInfoResponse{
		Name:     wt.Name,
		Path:     wt.Path,
		Branch:   branch,
		Head:     head,
		Base:     base,
		Upstream: upstream,
		Clean:    clean,
		Current:  wt.Name == currentName,
	}

	if base != "" {
		if exists, _ := svc.Git.BranchExistsInPath(wt.Path, base); exists {
			ahead, behind, err := svc.Git.CountAheadBehind(wt.Path, base)
			if err == nil {
				info.BaseStatus = &Divergence{Ahead: ahead, Behind: behind}
			}
		}
	}

	if upstream != "" {
		if exists, _ := svc.Git.BranchExistsInPath(wt.Path, upstream); exists {
			ahead, behind, err := svc.Git.CountAheadBehind(wt.Path, upstream)
			if err == nil {
				info.UpstreamStatus = &Divergence{Ahead: ahead, Behind: behind}
			}
		}
	}

	remoteCandidate := "origin/" + branch
	if exists, _ := svc.Git.BranchExistsInPath(wt.Path, remoteCandidate); exists {
		info.RemoteBranch = remoteCandidate
	}

	return info, nil
}

func printWorktreeInfo(info *WorktreeInfoResponse) {
	fmt.Printf("worktree: %s\n", info.Name)
	fmt.Printf("path:     %s\n", info.Path)
	fmt.Printf("branch:   %s\n", info.Branch)
	fmt.Printf("head:     %s\n", info.Head)

	if info.Base != "" {
		fmt.Printf("base:     %s%s\n", info.Base, formatDivergence(info.BaseStatus))
	} else {
		fmt.Println("base:     (not recorded)")
	}

	if info.Upstream != "" {
		fmt.Printf("upstream: %s%s\n", info.Upstream, formatDivergence(info.UpstreamStatus))
	} else {
		fmt.Println("upstream: (not set)")
	}

	if info.RemoteBranch != "" {
		fmt.Printf("remote:   %s\n", info.RemoteBranch)
	} else {
		fmt.Println("remote:   (not pushed)")
	}

	status := "clean"
	if !info.Clean {
		status = "dirty"
	}
	if info.Current {
		status += " (current)"
	}
	fmt.Printf("status:   %s\n", status)
}

func formatDivergence(d *Divergence) string {
	if d == nil {
		return ""
	}
	if d.Ahead == 0 && d.Behind == 0 {
		return " (up to date)"
	}
	return fmt.Sprintf(" (%d ahead, %d behind)", d.Ahead, d.Behind)
}
