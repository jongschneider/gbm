package main

import (
	"encoding/json"
	"fmt"
	"gbm/internal/jira"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		status     []string
		labels     []string
		priority   string
		issueType  string
		component  string
		reporter   string
		assignee   string
		orderBy    string
		reverse    bool
		jsonOut    bool
		customArgs []string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List JIRA issues (briefly cached)",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			filters := jira.JiraFilters{
				Priority:   priority,
				Type:       issueType,
				Component:  component,
				Reporter:   reporter,
				Assignee:   assignee,
				OrderBy:    orderBy,
				Status:     status,
				Labels:     labels,
				CustomArgs: customArgs,
				Reverse:    reverse,
			}
			issues, err := svc.GetJiraIssues(filters, flagDryRun)
			if err != nil {
				return err
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(issues)
			}

			if len(issues) == 0 {
				fmt.Fprintln(os.Stderr, "No issues.")
				return nil
			}

			tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(tw, "KEY\tTYPE\tSTATUS\tSUMMARY")
			for _, i := range issues {
				fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", i.Key, i.Type, i.Status, truncate(i.Summary, 80))
			}
			return tw.Flush()
		},
	}

	f := cmd.Flags()
	f.StringSliceVarP(&status, "status", "s", nil, "filter by status (repeatable)")
	f.StringSliceVarP(&labels, "label", "l", nil, "filter by label (repeatable)")
	f.StringVarP(&priority, "priority", "y", "", "filter by priority")
	f.StringVarP(&issueType, "type", "t", "", "filter by issue type")
	f.StringVarP(&component, "component", "C", "", "filter by component")
	f.StringVarP(&reporter, "reporter", "r", "", "filter by reporter")
	f.StringVarP(&assignee, "assignee", "a", "", "filter by assignee (default: current user; \"none\" disables)")
	f.StringVar(&orderBy, "order-by", "", "JQL field to order by")
	f.BoolVar(&reverse, "reverse", false, "reverse order")
	f.BoolVar(&jsonOut, "json", false, "emit JSON instead of a table")
	f.StringArrayVar(&customArgs, "arg", nil, "extra argument to pass through to jira issue list (repeatable)")

	return cmd
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.TrimRight(s[:n-1], " ") + "…"
}
