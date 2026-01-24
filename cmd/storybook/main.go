// Package main provides a storybook browser for gbm TUI components.
package main

import (
	"errors"
	"log"
	"time"

	"gbm/pkg/tui"
	"gbm/pkg/tui/fields"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	storybook "github.com/jongschneider/storybook-go"
)

func main() {
	registry := storybook.NewRegistry()

	// Register Table stories
	registry.Register(tableStory())

	// Register Field stories
	registry.Register(textInputStory())
	registry.Register(selectorStory())
	registry.Register(confirmStory())
	registry.Register(filterableStory())

	// Register Wizard story
	registry.Register(wizardStory())

	// Run the storybook
	sb := storybook.New(registry)
	p := tea.NewProgram(sb, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// tableStory creates stories for the Table component.
func tableStory() storybook.Story {
	return storybook.Story{
		Name:        "Table",
		Description: "Filterable table with async cell support and cycling navigation",
		Variants: []storybook.Variant{
			{
				Name: "Default",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newTableModel(false, false)
				},
			},
			{
				Name: "WithFilter",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newTableModel(true, false)
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Wait(500 * time.Millisecond)
					pc.Key("/")
					pc.Wait(200 * time.Millisecond)
					pc.Type("feat")
					pc.Wait(500 * time.Millisecond)
					pc.AssertContains("feature")
				},
			},
			{
				Name: "WithCycling",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newTableModel(false, true)
				},
				Play: func(pc *storybook.PlayContext) {
					// Navigate down past the last item to wrap to top
					pc.Key("down")
					pc.Key("down")
					pc.Key("down")
					pc.Key("down")
					pc.Wait(300 * time.Millisecond)
				},
			},
			{
				Name: "WithFilterAndCycling",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newTableModel(true, true)
				},
			},
		},
	}
}

// tableModel wraps tui.Table for storybook display.
type tableModel struct {
	table *tui.Table
	ctx   *tui.Context
}

func newTableModel(filterable, cycling bool) *tableModel {
	ctx := tui.NewContext().WithDimensions(80, 24)

	columns := []tui.Column{
		{Title: "Name", Width: 20},
		{Title: "Branch", Width: 30},
		{Title: "Kind", Width: 10},
		{Title: "Status", Width: 10},
	}

	rows := []table.Row{
		{"* main", "main", "tracked", "✓"},
		{"feature-auth", "feature/auth", "ad hoc", "↑ 2"},
		{"feature-ui", "feature/ui-updates", "ad hoc", "↓ 1"},
		{"bugfix-123", "bug/fix-123", "ad hoc", "↕ 1↑2↓"},
	}

	tbl := tui.NewTable(ctx).
		WithColumns(columns).
		WithRows(rows).
		WithHeight(8).
		WithFocused(true).
		WithFilterable(filterable).
		WithCycling(cycling).
		Build()

	return &tableModel{table: tbl, ctx: ctx}
}

func (m *tableModel) Init() tea.Cmd {
	return m.table.Init()
}

func (m *tableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.ctx = m.ctx.WithDimensions(sizeMsg.Width, sizeMsg.Height)
	}
	_, cmd := m.table.Update(msg)
	return m, cmd
}

func (m *tableModel) View() string {
	return m.table.View()
}

// textInputStory creates stories for the TextInput field.
func textInputStory() storybook.Story {
	return storybook.Story{
		Name:        "TextInput",
		Description: "Text entry field with optional validation",
		Variants: []storybook.Variant{
			{
				Name: "Default",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newTextInputModel("branch_name", "Branch Name", "Enter the name for your new branch")
				},
			},
			{
				Name: "WithPlaceholder",
				Factory: func(_ ...storybook.Args) tea.Model {
					ti := fields.NewTextInput("worktree", "Worktree Name", "Name for the worktree directory").
						WithPlaceholder("e.g., feature-auth")
					return newFieldModel(ti)
				},
			},
			{
				Name: "WithValidation",
				Factory: func(_ ...storybook.Args) tea.Model {
					ti := fields.NewTextInput("name", "Required Field", "This field cannot be empty").
						WithValidator(func(s string) error {
							if s == "" {
								return errors.New("field is required")
							}
							return nil
						})
					return newFieldModel(ti)
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("enter")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("required")
				},
			},
			{
				Name: "WithDefault",
				Factory: func(_ ...storybook.Args) tea.Model {
					ti := fields.NewTextInput("branch", "Branch Name", "Pre-filled with a default value")
					ti.WithDefault("feature/my-feature")
					return newFieldModel(ti)
				},
			},
		},
	}
}

func newTextInputModel(key, title, description string) tea.Model {
	ti := fields.NewTextInput(key, title, description)
	return newFieldModel(ti)
}

// selectorStory creates stories for the Selector field.
func selectorStory() storybook.Story {
	return storybook.Story{
		Name:        "Selector",
		Description: "List of options for single selection",
		Variants: []storybook.Variant{
			{
				Name: "WorkflowType",
				Factory: func(_ ...storybook.Args) tea.Model {
					options := []fields.Option{
						{Label: "Feature - New functionality", Value: "feature"},
						{Label: "Bug Fix - Fix an issue", Value: "bug"},
						{Label: "Hotfix - Urgent production fix", Value: "hotfix"},
					}
					sel := fields.NewSelector("workflow_type", "Select Workflow Type", options)
					return newFieldModel(sel)
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("down")
					pc.Wait(200 * time.Millisecond)
					pc.Key("down")
					pc.Wait(200 * time.Millisecond)
					pc.AssertContains("▸")
				},
			},
			{
				Name: "BaseBranch",
				Factory: func(_ ...storybook.Args) tea.Model {
					options := []fields.Option{
						{Label: "main", Value: "main"},
						{Label: "develop", Value: "develop"},
						{Label: "release/v2.0", Value: "release/v2.0"},
					}
					sel := fields.NewSelector("base_branch", "Select Base Branch", options)
					return newFieldModel(sel)
				},
			},
		},
	}
}

// confirmStory creates stories for the Confirm field.
func confirmStory() storybook.Story {
	return storybook.Story{
		Name:        "Confirm",
		Description: "Yes/No confirmation dialog",
		Variants: []storybook.Variant{
			{
				Name: "Simple",
				Factory: func(_ ...storybook.Args) tea.Model {
					conf := fields.NewConfirm("confirm", "Create worktree?")
					return newFieldModel(conf)
				},
			},
			{
				Name: "WithSummary",
				Factory: func(_ ...storybook.Args) tea.Model {
					conf := fields.NewConfirm("confirm", "Ready to create worktree?").
						WithSummary("Worktree: feature-auth\nBranch: feature/auth\nBase: main")
					return newFieldModel(conf)
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("right")
					pc.Wait(300 * time.Millisecond)
					pc.Key("left")
					pc.Wait(300 * time.Millisecond)
				},
			},
		},
	}
}

// filterableStory creates stories for the Filterable field.
func filterableStory() storybook.Story {
	return storybook.Story{
		Name:        "Filterable",
		Description: "Filterable list with text input for filtering or custom values",
		Variants: []storybook.Variant{
			{
				Name: "JiraIssues",
				Factory: func(_ ...storybook.Args) tea.Model {
					options := []fields.Option{
						{Label: "PROJ-123: Add user authentication", Value: "PROJ-123"},
						{Label: "PROJ-124: Fix login redirect", Value: "PROJ-124"},
						{Label: "PROJ-125: Update dashboard UI", Value: "PROJ-125"},
						{Label: "PROJ-126: Implement dark mode", Value: "PROJ-126"},
						{Label: "PROJ-127: Add export feature", Value: "PROJ-127"},
					}
					f := fields.NewFilterable("jira_issue", "Select JIRA Issue", "Type to filter or enter custom value", options)
					return newFieldModel(f)
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Type("auth")
					pc.Wait(500 * time.Millisecond)
					pc.AssertContains("authentication")
				},
			},
			{
				Name: "Branches",
				Factory: func(_ ...storybook.Args) tea.Model {
					options := []fields.Option{
						{Label: "main", Value: "main"},
						{Label: "develop", Value: "develop"},
						{Label: "feature/auth", Value: "feature/auth"},
						{Label: "feature/ui-updates", Value: "feature/ui-updates"},
						{Label: "bugfix/login-redirect", Value: "bugfix/login-redirect"},
						{Label: "release/v2.0", Value: "release/v2.0"},
					}
					f := fields.NewFilterable("branch", "Select Branch", "Filter branches or enter a new name", options)
					return newFieldModel(f)
				},
			},
			{
				Name: "Empty",
				Factory: func(_ ...storybook.Args) tea.Model {
					f := fields.NewFilterable("custom", "Enter Value", "No predefined options - type your own", []fields.Option{})
					return newFieldModel(f)
				},
			},
		},
	}
}

// wizardStory creates stories for the Wizard component.
func wizardStory() storybook.Story {
	return storybook.Story{
		Name:        "Wizard",
		Description: "Multi-step form orchestrator",
		Variants: []storybook.Variant{
			{
				Name: "FeatureWorkflow",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newWizardModel()
				},
				Play: func(pc *storybook.PlayContext) {
					// Select feature workflow
					pc.Key("enter")
					pc.Wait(300 * time.Millisecond)
					// Enter worktree name
					pc.Type("auth-feature")
					pc.Key("enter")
					pc.Wait(300 * time.Millisecond)
				},
			},
		},
	}
}

func newWizardModel() tea.Model {
	ctx := tui.NewContext().WithDimensions(80, 24)

	steps := []tui.Step{
		{
			Name: "workflow_type",
			Field: fields.NewSelector("workflow_type", "Select Workflow Type", []fields.Option{
				{Label: "Feature - New functionality", Value: "feature"},
				{Label: "Bug Fix - Fix an issue", Value: "bug"},
				{Label: "Hotfix - Urgent production fix", Value: "hotfix"},
			}),
		},
		{
			Name: "worktree_name",
			Field: fields.NewTextInput("worktree_name", "Worktree Name", "Enter a name for the worktree directory"),
		},
		{
			Name: "branch_name",
			Field: fields.NewTextInput("branch_name", "Branch Name", "Enter the branch name"),
		},
		{
			Name: "confirm",
			Field: fields.NewConfirm("confirm", "Create worktree?"),
		},
	}

	return tui.NewWizard(steps, ctx)
}

// fieldModel wraps a tui.Field for storybook display.
type fieldModel struct {
	field tui.Field
}

func newFieldModel(f tui.Field) *fieldModel {
	return &fieldModel{field: f}
}

func (m *fieldModel) Init() tea.Cmd {
	initCmd := m.field.Init()
	focusCmd := m.field.Focus()
	return tea.Batch(initCmd, focusCmd)
}

func (m *fieldModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.field, cmd = m.field.Update(msg)
	return m, cmd
}

func (m *fieldModel) View() string {
	return m.field.View()
}
