package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newMeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "me",
		Short: "Print the current JIRA user",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := newService()
			if err != nil {
				return err
			}
			user, _, err := svc.GetJiraUser("", flagDryRun)
			if err != nil {
				return err
			}
			fmt.Println(user)
			return nil
		},
	}
}
