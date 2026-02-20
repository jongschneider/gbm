package service

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	tuiconfig "gbm/pkg/tui/config"
)

func newConfigCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Open the interactive config editor",
		Long: `Open the Config TUI to interactively edit .gbm/config.yaml.

The editor provides tabbed sections for General, JIRA, File Copy, and
Worktrees settings. Changes are validated before saving and the original
file comments are preserved.

Examples:
  # Open the config editor
  gbm config`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigTUI(svc)
		},
	}
	return cmd
}

// runConfigTUI launches the Config TUI in an alternate screen.
func runConfigTUI(svc *Service) error {
	// Open /dev/tty for TUI rendering, same pattern as worktree TUI.
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w (TUI requires an interactive terminal)", err)
	}
	defer func() {
		//nolint:errcheck // Best-effort cleanup
		tty.Close()
	}()

	// Set up color renderer before creating the model.
	renderer := lipgloss.NewRenderer(tty,
		termenv.WithColorCache(true),
		termenv.WithTTY(true),
		termenv.WithProfile(termenv.TrueColor),
	)
	lipgloss.SetDefaultRenderer(renderer)

	// Resolve config file path.
	configPath := resolveConfigPath(svc)

	// Build the TUI model from the config file (or defaults).
	model, err := buildConfigModel(svc, configPath)
	if err != nil {
		return err
	}

	// Run TUI.
	p := tea.NewProgram(model,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running config TUI: %w", err)
	}

	return nil
}

// resolveConfigPath returns the absolute path to .gbm/config.yaml.
// If the service has a known repo root, it uses that; otherwise it
// falls back to the current working directory.
func resolveConfigPath(svc *Service) string {
	root := svc.RepoRoot
	if root == "" {
		// Best-effort fallback for when we're not in a git repo.
		root, _ = os.Getwd() //nolint:errcheck // fallback to cwd
	}
	return filepath.Join(root, ".gbm", "config.yaml")
}

// buildConfigModel creates a ConfigModel populated from the config file
// at configPath. If the file does not exist, a default config is used
// and the model is marked as a new file.
func buildConfigModel(svc *Service, configPath string) (*tuiconfig.ConfigModel, error) {
	cfg := svc.GetConfig()
	adapter := NewConfigAdapter(cfg)

	// Try to load the existing YAML node tree for comment preservation.
	var root *yaml.Node
	var isNew bool
	var modTime time.Time

	cfgFile, err := tuiconfig.LoadConfigFile(configPath)
	if err != nil {
		// File doesn't exist or is unreadable -- start fresh.
		isNew = true
		root = newEmptyYAMLDoc()
	} else {
		root = cfgFile.Root
		modTime = cfgFile.ModTime
	}

	// Seed the dirty tracker with current field values.
	originals := snapshotConfig(adapter)
	dirty := tuiconfig.NewDirtyTracker(originals)

	model := tuiconfig.NewConfigModel(
		tuiconfig.WithAccessor(adapter),
		tuiconfig.WithFilePath(configPath),
		tuiconfig.WithYAMLRoot(root),
		tuiconfig.WithDirtyTracker(dirty),
		tuiconfig.WithNewFile(isNew),
		tuiconfig.WithModTime(modTime),
	)

	return model, nil
}

// snapshotConfig reads all known config keys from the adapter to create
// the initial baseline for dirty tracking.
func snapshotConfig(accessor tuiconfig.ConfigAccessor) map[string]any {
	keys := allConfigKeys()
	snapshot := make(map[string]any, len(keys))
	for _, key := range keys {
		snapshot[key] = accessor.GetValue(key)
	}
	return snapshot
}

// allConfigKeys returns all dot-path config keys that the TUI manages.
func allConfigKeys() []string {
	return []string{
		// General
		"default_branch", "worktrees_dir",
		// JIRA Connection
		"jira.host", "jira.me",
		// JIRA Filters
		"jira.filters.priority", "jira.filters.type", "jira.filters.component",
		"jira.filters.reporter", "jira.filters.assignee", "jira.filters.order_by",
		"jira.filters.status", "jira.filters.labels", "jira.filters.custom_args",
		"jira.filters.reverse",
		// JIRA Markdown
		"jira.markdown.filename_pattern", "jira.markdown.max_depth",
		"jira.markdown.include_comments", "jira.markdown.include_attachments",
		"jira.markdown.use_relative_links", "jira.markdown.include_linked_issues",
		// JIRA Attachments
		"jira.attachments.enabled", "jira.attachments.max_size_mb",
		"jira.attachments.directory", "jira.attachments.download_timeout_seconds",
		"jira.attachments.retry_attempts", "jira.attachments.retry_backoff_ms",
		// File Copy Auto
		"file_copy.auto.enabled", "file_copy.auto.source_worktree",
		"file_copy.auto.copy_ignored", "file_copy.auto.copy_untracked",
		"file_copy.auto.exclude",
	}
}

// newEmptyYAMLDoc creates an empty YAML document node with a mapping child.
func newEmptyYAMLDoc() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode, Tag: "!!map"},
		},
	}
}
