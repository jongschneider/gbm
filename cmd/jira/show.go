package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "show <KEY>",
		Short: "Show details for a JIRA issue",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			details, err := svc.GetJiraIssue(args[0], flagDryRun)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(details)
			}

			fmt.Print(details.Display())
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "emit JSON instead of human-readable output")
	return cmd
}
