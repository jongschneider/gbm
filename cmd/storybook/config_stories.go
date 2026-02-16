// Package main provides story definitions for config TUI components.
package main

import (
	"gbm/pkg/tui"
	"gbm/pkg/tui/config"
	"gbm/pkg/tui/fields"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	storybook "github.com/jongschneider/storybook-go"
)

// configModelStory creates stories for the ConfigModel two-pane layout.
func configModelStory() storybook.Story {
	return storybook.Story{
		Name:        "ConfigModel",
		Description: "Two-pane config TUI with sidebar and content",
		Variants: []storybook.Variant{
			{
				Name: "BasicsSection",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper("Basics")
				},
			},
			{
				Name: "JIRASection",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper("JIRA")
				},
			},
			{
				Name: "FileCopySection",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper("FileCopy")
				},
			},
			{
				Name: "WorktreesSection",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper("Worktrees")
				},
			},
			{
				Name: "ContentFocused",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper("Basics")
				},
				Play: func(pc *storybook.PlayContext) {
					// Press 'l' to focus content
					pc.Key("l")
					pc.Wait(200 * time.Millisecond)
				},
			},
		},
	}
}

func newConfigModelWrapper(initialSection string) *configModelWrapper {
	theme := tui.DefaultTheme()

	formFactory := func(section string, state *tui.ConfigState, t *tui.Theme, onUpdate func()) tea.Model {
		switch section {
		case "JIRA":
			return config.NewJiraForm(config.JiraFormConfig{
				Theme:    t,
				Enabled:  true,
				Host:     "https://jira.example.com",
				Username: "user@example.com",
			})
		case "FileCopy":
			return config.NewFileCopyForm(config.FileCopyFormConfig{
				Theme: t,
				Rules: []config.FileCopyRule{
					{SourceWorktree: "main", Files: []string{".env", ".env.local"}},
				},
			})
		case "Worktrees":
			return config.NewWorktreesForm(config.WorktreesFormConfig{
				Theme: t,
				Worktrees: []config.WorktreeEntry{
					{Name: "main", Branch: "main", MergeInto: "", Description: "Main branch"},
				},
			})
		default:
			return config.NewBasicsForm(config.BasicsFormConfig{
				Theme:         t,
				DefaultBranch: "main",
				WorktreesDir:  "./worktrees",
			})
		}
	}

	model := tui.NewConfigModel(theme, tui.WithFormFactory(formFactory))

	// Navigate to the desired section
	if initialSection != "Basics" {
		model.Update(tui.SidebarSelectionChangedMsg{Section: initialSection})
	}

	return &configModelWrapper{model: model}
}

type configModelWrapper struct {
	model *tui.ConfigModel
}

func (m *configModelWrapper) Init() tea.Cmd {
	return m.model.Init()
}

func (m *configModelWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, cmd := m.model.Update(msg)
	if cm, ok := newModel.(*tui.ConfigModel); ok {
		m.model = cm
	}
	return m, cmd
}

func (m *configModelWrapper) View() string {
	return m.model.View()
}

// sidebarStory creates stories for the Sidebar component.
func sidebarStory() storybook.Story {
	return storybook.Story{
		Name:        "Sidebar",
		Description: "Config section navigation with error badges",
		Variants: []storybook.Variant{
			{
				Name: "Default",
				Factory: func(_ ...storybook.Args) tea.Model {
					sb := tui.NewSidebar(tui.DefaultTheme())
					return &sidebarModel{sidebar: sb}
				},
			},
			{
				Name: "WithFocus",
				Factory: func(_ ...storybook.Args) tea.Model {
					sb := tui.NewSidebar(tui.DefaultTheme())
					return &sidebarModel{sidebar: sb}
				},
				Play: func(pc *storybook.PlayContext) {
					// Navigate down to highlight different items
					pc.Key("down")
					pc.Wait(200 * time.Millisecond)
					pc.Key("down")
					pc.Wait(200 * time.Millisecond)
					pc.AssertContains("FileCopy")
				},
			},
			{
				Name: "WithValidationErrors",
				Factory: func(_ ...storybook.Args) tea.Model {
					sb := tui.NewSidebar(tui.DefaultTheme())
					sb.SetError("Basics", true)
					sb.SetError("JIRA", true)
					return &sidebarModel{sidebar: sb}
				},
			},
		},
	}
}

// sidebarModel wraps tui.Sidebar for storybook display.
type sidebarModel struct {
	sidebar *tui.Sidebar
}

func (m *sidebarModel) Init() tea.Cmd {
	return m.sidebar.Init()
}

func (m *sidebarModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	sb, cmd := m.sidebar.Update(msg)
	if s, ok := sb.(*tui.Sidebar); ok {
		m.sidebar = s
	}
	return m, cmd
}

func (m *sidebarModel) View() string {
	return m.sidebar.View()
}

// basicsFormStory creates stories for the BasicsForm component.
func basicsFormStory() storybook.Story {
	return storybook.Story{
		Name:        "BasicsForm",
		Description: "Basic configuration form with branch and worktrees directory",
		Variants: []storybook.Variant{
			{
				Name: "Empty",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewBasicsForm(config.BasicsFormConfig{
						Theme: tui.DefaultTheme(),
					})
					return form
				},
			},
			{
				Name: "Populated",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewBasicsForm(config.BasicsFormConfig{
						Theme:         tui.DefaultTheme(),
						DefaultBranch: "main",
						WorktreesDir:  "./worktrees",
					})
					return form
				},
			},
			{
				Name: "WithValidationErrors",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewBasicsForm(config.BasicsFormConfig{
						Theme:         tui.DefaultTheme(),
						DefaultBranch: "", // Empty will trigger validation
						WorktreesDir:  "", // Empty will trigger validation
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					// Try to save to trigger validation
					pc.Key("s")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Validation")
				},
			},
			{
				Name: "DiscardConfirmation",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewBasicsForm(config.BasicsFormConfig{
						Theme:         tui.DefaultTheme(),
						DefaultBranch: "main",
						WorktreesDir:  "./worktrees",
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					// Press q to show discard confirmation
					pc.Key("q")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Discard")
				},
			},
		},
	}
}

// jiraFormStory creates stories for the JiraForm component.
func jiraFormStory() storybook.Story {
	return storybook.Story{
		Name:        "JiraForm",
		Description: "JIRA configuration with enable/disable toggle and subsections",
		Variants: []storybook.Variant{
			{
				Name: "Disabled",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewJiraForm(config.JiraFormConfig{
						Theme:   tui.DefaultTheme(),
						Enabled: false,
					})
					return form
				},
			},
			{
				Name: "EnabledWithSubsections",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewJiraForm(config.JiraFormConfig{
						Theme:                      tui.DefaultTheme(),
						Enabled:                    true,
						Host:                       "https://jira.company.com",
						Username:                   "user@company.com",
						APIToken:                   "secret-token",
						FiltersStatus:              []string{"In Dev", "Open"},
						FiltersPriority:            "High",
						FiltersType:                "Bug",
						AttachmentsEnabled:         true,
						AttachmentsMaxSize:         10,
						AttachmentsDir:             "./attachments",
						MarkdownIncludeComments:    true,
						MarkdownIncludeAttachments: true,
						MarkdownUseRelativeLinks:   false,
						MarkdownFilenamePattern:    "{key}.md",
					})
					return form
				},
			},
			{
				Name: "ServerSection",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewJiraForm(config.JiraFormConfig{
						Theme:   tui.DefaultTheme(),
						Enabled: true,
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					// Tab to navigate to server fields
					pc.Key("tab")
					pc.Wait(200 * time.Millisecond)
					pc.AssertContains("JIRA Host")
				},
			},
			{
				Name: "WithValidationErrors",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewJiraForm(config.JiraFormConfig{
						Theme:   tui.DefaultTheme(),
						Enabled: true,
						// Empty fields will trigger validation
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("s") // Try to save
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("required")
				},
			},
		},
	}
}

// fileCopyFormStory creates stories for the FileCopyForm component.
func fileCopyFormStory() storybook.Story {
	return storybook.Story{
		Name:        "FileCopyForm",
		Description: "File copy rules table with add/edit/delete modals",
		Variants: []storybook.Variant{
			{
				Name: "EmptyRules",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewFileCopyForm(config.FileCopyFormConfig{
						Theme: tui.DefaultTheme(),
						Rules: []config.FileCopyRule{},
					})
					return form
				},
			},
			{
				Name: "WithRules",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewFileCopyForm(config.FileCopyFormConfig{
						Theme: tui.DefaultTheme(),
						Rules: []config.FileCopyRule{
							{SourceWorktree: "main", Files: []string{".env", ".env.local"}},
							{SourceWorktree: "develop", Files: []string{"config.yaml"}},
							{SourceWorktree: "main", Files: []string{".gitignore", "Makefile", "docker-compose.yml"}},
						},
					})
					return form
				},
			},
			{
				Name: "AddModal",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewFileCopyForm(config.FileCopyFormConfig{
						Theme: tui.DefaultTheme(),
						Rules: []config.FileCopyRule{},
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("a") // Open add modal
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Add File Copy Rule")
				},
			},
			{
				Name: "EditModal",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewFileCopyForm(config.FileCopyFormConfig{
						Theme: tui.DefaultTheme(),
						Rules: []config.FileCopyRule{
							{SourceWorktree: "main", Files: []string{".env"}},
						},
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("e") // Open edit modal
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Edit File Copy Rule")
				},
			},
			{
				Name: "DeleteConfirmation",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewFileCopyForm(config.FileCopyFormConfig{
						Theme: tui.DefaultTheme(),
						Rules: []config.FileCopyRule{
							{SourceWorktree: "main", Files: []string{".env"}},
						},
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("d") // Open delete confirmation
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Delete")
				},
			},
		},
	}
}

// worktreesFormStory creates stories for the WorktreesForm component.
func worktreesFormStory() storybook.Story {
	return storybook.Story{
		Name:        "WorktreesForm",
		Description: "Worktrees configuration table with add/edit/delete modals",
		Variants: []storybook.Variant{
			{
				Name: "EmptyTable",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewWorktreesForm(config.WorktreesFormConfig{
						Theme:     tui.DefaultTheme(),
						Worktrees: []config.WorktreeEntry{},
					})
					return form
				},
			},
			{
				Name: "PopulatedTable",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewWorktreesForm(config.WorktreesFormConfig{
						Theme: tui.DefaultTheme(),
						Worktrees: []config.WorktreeEntry{
							{Name: "main", Branch: "main", MergeInto: "", Description: "Main branch"},
							{Name: "feature-auth", Branch: "feature/auth", MergeInto: "main", Description: "Auth feature"},
							{Name: "bugfix-login", Branch: "bugfix/login-redirect", MergeInto: "develop", Description: "Fix redirect"},
						},
					})
					return form
				},
			},
			{
				Name: "AddModal",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewWorktreesForm(config.WorktreesFormConfig{
						Theme:     tui.DefaultTheme(),
						Worktrees: []config.WorktreeEntry{},
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("a")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Add Worktree")
				},
			},
			{
				Name: "EditModal",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewWorktreesForm(config.WorktreesFormConfig{
						Theme: tui.DefaultTheme(),
						Worktrees: []config.WorktreeEntry{
							{Name: "main", Branch: "main", MergeInto: "", Description: "Main branch"},
						},
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("e")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Edit Worktree")
				},
			},
			{
				Name: "DeleteConfirmation",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewWorktreesForm(config.WorktreesFormConfig{
						Theme: tui.DefaultTheme(),
						Worktrees: []config.WorktreeEntry{
							{Name: "main", Branch: "main", MergeInto: "", Description: "Main branch"},
						},
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("d")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Delete")
				},
			},
			{
				Name: "WithValidationError",
				Factory: func(_ ...storybook.Args) tea.Model {
					form := config.NewWorktreesForm(config.WorktreesFormConfig{
						Theme:     tui.DefaultTheme(),
						Worktrees: []config.WorktreeEntry{},
					})
					return form
				},
				Play: func(pc *storybook.PlayContext) {
					// Open add modal and try to confirm with empty fields
					pc.Key("a")
					pc.Wait(200 * time.Millisecond)
					pc.Key("enter") // Try to confirm
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Error")
				},
			},
		},
	}
}

// helpOverlayStory creates stories for the HelpOverlay component.
func helpOverlayStory() storybook.Story {
	return storybook.Story{
		Name:        "HelpOverlay",
		Description: "Keyboard shortcuts help overlay",
		Variants: []storybook.Variant{
			{
				Name: "Default",
				Factory: func(_ ...storybook.Args) tea.Model {
					overlay := tui.NewHelpOverlay().
						WithTheme(tui.DefaultTheme()).
						WithWidth(60).
						WithHeight(30)
					return &helpOverlayModel{overlay: overlay}
				},
			},
			{
				Name: "CustomGroups",
				Factory: func(_ ...storybook.Args) tea.Model {
					overlay := tui.NewHelpOverlay().
						WithTheme(tui.DefaultTheme()).
						WithWidth(60).
						WithHeight(30).
						WithGroups([]tui.ShortcutGroup{
							{
								Name: "Custom Navigation",
								Shortcuts: []tui.Shortcut{
									{Key: "j/k", Description: "Move down/up"},
									{Key: "Enter", Description: "Confirm selection"},
								},
							},
							{
								Name: "Actions",
								Shortcuts: []tui.Shortcut{
									{Key: "c", Description: "Create new item"},
									{Key: "d", Description: "Delete item"},
								},
							},
						})
					return &helpOverlayModel{overlay: overlay}
				},
			},
		},
	}
}

// helpOverlayModel wraps tui.HelpOverlay for storybook display.
type helpOverlayModel struct {
	overlay *tui.HelpOverlay
}

func (m *helpOverlayModel) Init() tea.Cmd {
	return m.overlay.Init()
}

func (m *helpOverlayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newOverlay, cmd := m.overlay.Update(msg)
	if o, ok := newOverlay.(*tui.HelpOverlay); ok {
		m.overlay = o
	}
	return m, cmd
}

func (m *helpOverlayModel) View() string {
	return m.overlay.View()
}

// validationOverlayStory creates stories for the ValidationOverlay component.
func validationOverlayStory() storybook.Story {
	return storybook.Story{
		Name:        "ValidationOverlay",
		Description: "Validation error display overlay",
		Variants: []storybook.Variant{
			{
				Name: "SingleError",
				Factory: func(_ ...storybook.Args) tea.Model {
					overlay := fields.NewValidationOverlay([]string{
						"Default Branch: branch name is required",
					}).
						WithTheme(tui.DefaultTheme()).
						WithWidth(60).
						WithHeight(20)
					return &validationOverlayModel{overlay: overlay}
				},
			},
			{
				Name: "MultipleErrors",
				Factory: func(_ ...storybook.Args) tea.Model {
					overlay := fields.NewValidationOverlay([]string{
						"Default Branch: branch name is required",
						"Worktrees Directory: path is required",
						"JIRA Host: must start with http:// or https://",
						"Username: username is required",
					}).
						WithTheme(tui.DefaultTheme()).
						WithWidth(60).
						WithHeight(25)
					return &validationOverlayModel{overlay: overlay}
				},
			},
			{
				Name: "SaveError",
				Factory: func(_ ...storybook.Args) tea.Model {
					overlay := fields.NewValidationOverlay([]string{
						"Failed to write config file: permission denied",
					}).
						WithTitle("Save Error").
						WithTheme(tui.DefaultTheme()).
						WithWidth(60).
						WithHeight(20)
					return &validationOverlayModel{overlay: overlay}
				},
			},
		},
	}
}

// validationOverlayModel wraps fields.ValidationOverlay for storybook display.
type validationOverlayModel struct {
	overlay *fields.ValidationOverlay
}

func (m *validationOverlayModel) Init() tea.Cmd {
	return m.overlay.Init()
}

func (m *validationOverlayModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newOverlay, cmd := m.overlay.Update(msg)
	if o, ok := newOverlay.(*fields.ValidationOverlay); ok {
		m.overlay = o
	}
	return m, cmd
}

func (m *validationOverlayModel) View() string {
	return m.overlay.View()
}
