package main

import (
	"fmt"
	"gbm/internal/jira"
	"os"

	"github.com/spf13/cobra"
)

func newAttachmentsCmd() *cobra.Command {
	var outDir string

	cmd := &cobra.Command{
		Use:   "attachments <KEY>",
		Short: "Download attachments for a JIRA issue",
		Long:  "Downloads to <out>/.jira/<KEY>/attachments/.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			svc, err := newService()
			if err != nil {
				return err
			}

			details, err := svc.GetJiraIssue(key, flagDryRun)
			if err != nil {
				return err
			}

			if len(details.Attachments) == 0 {
				fmt.Fprintf(os.Stderr, "No attachments on %s.\n", key)
				return nil
			}

			root, err := resolveOutDir(outDir)
			if err != nil {
				return err
			}

			attachmentDir := jira.GenerateAttachmentPath(root, key)

			if flagDryRun {
				fmt.Fprintf(os.Stderr, "[DRY RUN] Would download %d attachments into %s\n", len(details.Attachments), attachmentDir)
				return nil
			}

			service := jira.NewAttachmentService(jira.DefaultAttachmentConfig())
			results, err := service.DownloadAllAttachments(details.Attachments, attachmentDir, root)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
			}

			var downloaded, skipped, failed int
			for _, r := range results {
				switch {
				case r.Skipped:
					skipped++
					fmt.Fprintf(os.Stderr, "  ⚠ %s — skipped (%s)\n", r.Attachment.Filename, r.SkipReason)
				case r.Error != nil:
					failed++
					fmt.Fprintf(os.Stderr, "  ✗ %s — failed: %v\n", r.Attachment.Filename, r.Error)
				default:
					downloaded++
					fmt.Println(r.LocalPath)
				}
			}
			fmt.Fprintf(os.Stderr, "%d downloaded, %d skipped, %d failed\n", downloaded, skipped, failed)
			return nil
		},
	}

	cmd.Flags().StringVarP(&outDir, "out", "o", ".", "directory under which the .jira/<KEY>/attachments/ folder is created")
	return cmd
}
