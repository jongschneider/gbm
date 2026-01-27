package service

import (
	"fmt"
	"gbm/pkg/tui"
	"gbm/pkg/tui/config"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

// newConfigCommand creates the 'gbm config' command that launches the interactive TUI.
func newConfigCommand(svc *Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage GBM configuration interactively",
		Long: `Launch an interactive terminal UI for managing .gbm/config.yaml.

The config TUI provides a user-friendly interface to configure GBM settings including:
- Basic settings (default branch, worktrees directory)
- JIRA integration
- File copy rules
- Worktree definitions

All changes are saved to .gbm/config.yaml when you confirm.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigTUI(svc)
		},
	}

	return cmd
}

// runConfigTUI launches the config TUI.
func runConfigTUI(svc *Service) error {
	theme := tui.DefaultTheme()

	// Load initial state from config
	initialState := configToState(svc.config)

	// Create form factory
	formFactory := createFormFactory(initialState)

	// Create save callback
	onSave := func(state *tui.ConfigState) error {
		stateToConfig(state, svc.config)
		return svc.SaveConfig()
	}

	// Create reset callback
	onReset := func() (*tui.ConfigState, error) {
		err := svc.loadConfig()
		if err != nil {
			return nil, err
		}
		return configToState(svc.config), nil
	}

	// Create the root config model with all options
	model := tui.NewConfigModel(
		theme,
		tui.WithInitialState(initialState),
		tui.WithFormFactory(formFactory),
		tui.WithOnSave(onSave),
		tui.WithOnReset(onReset),
	)

	// Create and run the program
	program := tea.NewProgram(model, tea.WithAltScreen())
	_, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}

// configToState converts a Config to a ConfigState.
func configToState(cfg *Config) *tui.ConfigState {
	if cfg == nil {
		return &tui.ConfigState{}
	}

	state := &tui.ConfigState{
		DefaultBranch: cfg.DefaultBranch,
		WorktreesDir:  cfg.WorktreesDir,

		// JIRA - note: we don't have an Enabled field in JiraConfig,
		// so we infer enabled from whether host is set
		JiraEnabled:                    cfg.Jira.Me != "" || cfg.Jira.Filters.Status != nil,
		JiraHost:                       "", // JiraConfig doesn't have Host directly
		JiraUsername:                   cfg.Jira.Me,
		JiraAPIToken:                   "",
		JiraFiltersStatus:              cfg.Jira.Filters.Status,
		JiraFiltersPriority:            cfg.Jira.Filters.Priority,
		JiraFiltersType:                cfg.Jira.Filters.Type,
		JiraAttachmentsEnabled:         cfg.Jira.Attachments.Enabled,
		JiraAttachmentsMaxSize:         int(cfg.Jira.Attachments.MaxSizeMB),
		JiraAttachmentsDir:             cfg.Jira.Attachments.Directory,
		JiraMarkdownIncludeComments:    cfg.Jira.Markdown.IncludeComments,
		JiraMarkdownIncludeAttachments: cfg.Jira.Markdown.IncludeAttachments,
		JiraMarkdownUseRelativeLinks:   cfg.Jira.Markdown.UseRelativeLinks,
		JiraMarkdownFilenamePattern:    cfg.Jira.Markdown.FilenamePattern,
	}

	// Convert FileCopy rules
	state.FileCopyRules = make([]tui.FileCopyRuleState, len(cfg.FileCopy.Rules))
	for i, rule := range cfg.FileCopy.Rules {
		files := make([]string, len(rule.Files))
		copy(files, rule.Files)
		state.FileCopyRules[i] = tui.FileCopyRuleState{
			SourceWorktree: rule.SourceWorktree,
			Files:          files,
		}
	}

	// Convert Worktrees
	if cfg.Worktrees != nil {
		state.Worktrees = make([]tui.WorktreeEntryState, 0, len(cfg.Worktrees))
		for name, wt := range cfg.Worktrees {
			state.Worktrees = append(state.Worktrees, tui.WorktreeEntryState{
				Name:        name,
				Branch:      wt.Branch,
				MergeInto:   wt.MergeInto,
				Description: wt.Description,
			})
		}
	}

	return state
}

// stateToConfig updates a Config from a ConfigState.
func stateToConfig(state *tui.ConfigState, cfg *Config) {
	if state == nil || cfg == nil {
		return
	}

	cfg.DefaultBranch = state.DefaultBranch
	cfg.WorktreesDir = state.WorktreesDir

	// JIRA
	cfg.Jira.Me = state.JiraUsername
	cfg.Jira.Filters.Status = state.JiraFiltersStatus
	cfg.Jira.Filters.Priority = state.JiraFiltersPriority
	cfg.Jira.Filters.Type = state.JiraFiltersType
	cfg.Jira.Attachments.Enabled = state.JiraAttachmentsEnabled
	cfg.Jira.Attachments.MaxSizeMB = int64(state.JiraAttachmentsMaxSize)
	cfg.Jira.Attachments.Directory = state.JiraAttachmentsDir
	cfg.Jira.Markdown.IncludeComments = state.JiraMarkdownIncludeComments
	cfg.Jira.Markdown.IncludeAttachments = state.JiraMarkdownIncludeAttachments
	cfg.Jira.Markdown.UseRelativeLinks = state.JiraMarkdownUseRelativeLinks
	cfg.Jira.Markdown.FilenamePattern = state.JiraMarkdownFilenamePattern

	// FileCopy rules
	cfg.FileCopy.Rules = make([]FileCopyRule, len(state.FileCopyRules))
	for i, rule := range state.FileCopyRules {
		files := make([]string, len(rule.Files))
		copy(files, rule.Files)
		cfg.FileCopy.Rules[i] = FileCopyRule{
			SourceWorktree: rule.SourceWorktree,
			Files:          files,
		}
	}

	// Worktrees
	if len(state.Worktrees) > 0 {
		cfg.Worktrees = make(map[string]WorktreeConfig)
		for _, wt := range state.Worktrees {
			cfg.Worktrees[wt.Name] = WorktreeConfig{
				Branch:      wt.Branch,
				MergeInto:   wt.MergeInto,
				Description: wt.Description,
			}
		}
	}
}

// createFormFactory creates a form factory that builds forms for each section.
func createFormFactory(state *tui.ConfigState) tui.FormFactory {
	return func(section string, currentState *tui.ConfigState, theme *tui.Theme, onUpdate func()) tea.Model {
		switch section {
		case "Basics":
			return createBasicsForm(currentState, theme, onUpdate)
		case "JIRA":
			return createJiraForm(currentState, theme, onUpdate)
		case "FileCopy":
			return createFileCopyForm(currentState, theme, onUpdate)
		case "Worktrees":
			return createWorktreesForm(currentState, theme, onUpdate)
		default:
			return nil
		}
	}
}

// createBasicsForm creates a BasicsForm populated from the current state.
func createBasicsForm(state *tui.ConfigState, theme *tui.Theme, onUpdate func()) tea.Model {
	return config.NewBasicsForm(config.BasicsFormConfig{
		Theme:         theme,
		DefaultBranch: state.DefaultBranch,
		WorktreesDir:  state.WorktreesDir,
		OnSave: func(data map[string]string) error {
			state.DefaultBranch = data["default_branch"]
			state.WorktreesDir = data["worktrees_dir"]
			onUpdate()
			return nil
		},
	})
}

// createJiraForm creates a JiraForm populated from the current state.
func createJiraForm(state *tui.ConfigState, theme *tui.Theme, onUpdate func()) tea.Model {
	return config.NewJiraForm(config.JiraFormConfig{
		Theme:                      theme,
		Enabled:                    state.JiraEnabled,
		Host:                       state.JiraHost,
		Username:                   state.JiraUsername,
		APIToken:                   state.JiraAPIToken,
		FiltersStatus:              state.JiraFiltersStatus,
		FiltersPriority:            state.JiraFiltersPriority,
		FiltersType:                state.JiraFiltersType,
		AttachmentsEnabled:         state.JiraAttachmentsEnabled,
		AttachmentsMaxSize:         state.JiraAttachmentsMaxSize,
		AttachmentsDir:             state.JiraAttachmentsDir,
		MarkdownIncludeComments:    state.JiraMarkdownIncludeComments,
		MarkdownIncludeAttachments: state.JiraMarkdownIncludeAttachments,
		MarkdownUseRelativeLinks:   state.JiraMarkdownUseRelativeLinks,
		MarkdownFilenamePattern:    state.JiraMarkdownFilenamePattern,
		OnSave: func(data map[string]any) error {
			state.JiraEnabled = data["jira_enabled"].(bool)

			if state.JiraEnabled {
				state.JiraHost = getStringOrEmpty(data, "jira_host")
				state.JiraUsername = getStringOrEmpty(data, "jira_username")
				state.JiraAPIToken = getStringOrEmpty(data, "jira_api_token")

				// Filters
				statusStr := getStringOrEmpty(data, "jira_filters_status")
				if statusStr != "" {
					state.JiraFiltersStatus = splitAndTrim(statusStr, ",")
				} else {
					state.JiraFiltersStatus = nil
				}
				state.JiraFiltersPriority = getStringOrEmpty(data, "jira_filters_priority")
				state.JiraFiltersType = getStringOrEmpty(data, "jira_filters_type")

				// Attachments
				state.JiraAttachmentsEnabled = getBoolOrFalse(data, "jira_attachments_enabled")
				state.JiraAttachmentsMaxSize = getIntFromString(data, "jira_attachments_max_size")
				state.JiraAttachmentsDir = getStringOrEmpty(data, "jira_attachments_dir")

				// Markdown
				state.JiraMarkdownIncludeComments = getBoolOrFalse(data, "jira_markdown_include_comments")
				state.JiraMarkdownIncludeAttachments = getBoolOrFalse(data, "jira_markdown_include_attachments")
				state.JiraMarkdownUseRelativeLinks = getBoolOrFalse(data, "jira_markdown_use_relative_links")
				state.JiraMarkdownFilenamePattern = getStringOrEmpty(data, "jira_markdown_filename_pattern")
			}

			onUpdate()
			return nil
		},
	})
}

// createFileCopyForm creates a FileCopyForm populated from the current state.
func createFileCopyForm(state *tui.ConfigState, theme *tui.Theme, onUpdate func()) tea.Model {
	// Convert state rules to form rules
	rules := make([]config.FileCopyRule, len(state.FileCopyRules))
	for i, r := range state.FileCopyRules {
		files := make([]string, len(r.Files))
		copy(files, r.Files)
		rules[i] = config.FileCopyRule{
			SourceWorktree: r.SourceWorktree,
			Files:          files,
		}
	}

	return config.NewFileCopyForm(config.FileCopyFormConfig{
		Theme: theme,
		Rules: rules,
		OnSave: func(formRules []config.FileCopyRule) error {
			// Convert form rules back to state rules
			stateRules := make([]tui.FileCopyRuleState, len(formRules))
			for i, r := range formRules {
				files := make([]string, len(r.Files))
				copy(files, r.Files)
				stateRules[i] = tui.FileCopyRuleState{
					SourceWorktree: r.SourceWorktree,
					Files:          files,
				}
			}
			state.FileCopyRules = stateRules
			onUpdate()
			return nil
		},
	})
}

// createWorktreesForm creates a WorktreesForm populated from the current state.
func createWorktreesForm(state *tui.ConfigState, theme *tui.Theme, onUpdate func()) tea.Model {
	// Convert state worktrees to form worktrees
	worktrees := make([]config.WorktreeEntry, len(state.Worktrees))
	for i, wt := range state.Worktrees {
		worktrees[i] = config.WorktreeEntry{
			Name:        wt.Name,
			Branch:      wt.Branch,
			MergeInto:   wt.MergeInto,
			Description: wt.Description,
		}
	}

	return config.NewWorktreesForm(config.WorktreesFormConfig{
		Theme:     theme,
		Worktrees: worktrees,
		OnSave: func(formWorktrees []config.WorktreeEntry) error {
			// Convert form worktrees back to state worktrees
			stateWorktrees := make([]tui.WorktreeEntryState, len(formWorktrees))
			for i, wt := range formWorktrees {
				stateWorktrees[i] = tui.WorktreeEntryState{
					Name:        wt.Name,
					Branch:      wt.Branch,
					MergeInto:   wt.MergeInto,
					Description: wt.Description,
				}
			}
			state.Worktrees = stateWorktrees
			onUpdate()
			return nil
		},
	})
}

// Helper functions for extracting values from map[string]any.

func getStringOrEmpty(data map[string]any, key string) string {
	if v, ok := data[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func getBoolOrFalse(data map[string]any, key string) bool {
	if v, ok := data[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}

func getIntFromString(data map[string]any, key string) int {
	s := getStringOrEmpty(data, key)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func splitAndTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
