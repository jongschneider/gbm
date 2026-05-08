package main

import (
	"fmt"
	"gbm/internal/jira"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

func newMarkdownCmd() *cobra.Command {
	var (
		outDir        string
		noComments    bool
		noAttachments bool
		noLinked      bool
		depth         int
	)

	cmd := &cobra.Command{
		Use:   "markdown <KEY>",
		Short: "Render a JIRA issue to markdown (with optional attachments)",
		Long: "Writes <out>/.jira/<KEY>/<KEY>.md plus an attachments/ subdir " +
			"and, recursively, linked/<LINKED_KEY>/<LINKED_KEY>.md bundles.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}

			root, err := resolveOutDir(outDir)
			if err != nil {
				return err
			}

			opts := jira.DefaultIssueMarkdownOptions(root)
			opts.IncludeComments = !noComments
			opts.DownloadAttachments = !noAttachments
			opts.IncludeLinkedIssues = !noLinked
			opts.MaxDepth = depth

			result, err := svc.GenerateIssueMarkdownFile(args[0], opts, flagDryRun)
			if err != nil {
				return err
			}

			jira.PrintMarkdownResult(result)
			fmt.Println(result.MarkdownPath)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&outDir, "out", "o", ".", "directory under which the .jira/<KEY>/ bundle is written")
	f.BoolVar(&noComments, "no-comments", false, "skip comments")
	f.BoolVar(&noAttachments, "no-attachments", false, "skip attachment downloads")
	f.BoolVar(&noLinked, "no-linked", false, "skip linked/parent/child issues")
	f.IntVar(&depth, "depth", 2, "max recursion depth for linked issues (1 = main ticket only)")

	return cmd
}

func resolveOutDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve out dir: %w", err)
	}
	if err := os.MkdirAll(abs, 0o755); err != nil {
		return "", fmt.Errorf("create out dir: %w", err)
	}
	return abs, nil
}
