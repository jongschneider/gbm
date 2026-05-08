// Binary gbm-jira is a standalone CLI exposing gbm's JIRA integration layer
// on top of the `jira` CLI. It reuses gbm/internal/jira for fetching, ADF
// parsing, markdown rendering, and attachment downloads.
package main

import (
	"fmt"
	"gbm/internal/jira"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagDebug  bool
	flagDryRun bool
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gbm-jira",
		Short:         "Standalone JIRA helper (lifted from gbm)",
		Long:          "gbm-jira wraps the `jira` CLI with caching, ADF-aware markdown rendering, and attachment downloads.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "enable debug logging")
	cmd.PersistentFlags().BoolVar(&flagDryRun, "dry-run", false, "print jira commands instead of executing them")

	cmd.AddCommand(
		newMeCmd(),
		newListCmd(),
		newShowCmd(),
		newMarkdownCmd(),
		newAttachmentsCmd(),
	)

	return cmd
}

// newService builds a jira.Service backed by a file-based cache.
func newService() (*jira.Service, error) {
	store, err := newFileCacheStore()
	if err != nil {
		return nil, err
	}
	return jira.NewService(flagDebug, store), nil
}
