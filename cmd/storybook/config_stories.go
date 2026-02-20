// Package main provides story definitions for config TUI components.
package main

import (
	"gbm/pkg/tui"
	"gbm/pkg/tui/config"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	storybook "github.com/jongschneider/storybook-go"
)

// --- ConfigModel stories ---.

// configModelStory creates stories for the full ConfigModel two-pane layout.
func configModelStory() storybook.Story {
	return storybook.Story{
		Name:        "ConfigModel",
		Description: "Root config TUI with tab bar, content area, and status bar",
		Variants: []storybook.Variant{
			{
				Name: "DefaultBrowsing",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper(config.TabGeneral, false)
				},
			},
			{
				Name: "JIRATab",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper(config.TabJira, false)
				},
			},
			{
				Name: "NewFile",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper(config.TabGeneral, true)
				},
			},
			{
				Name: "TabNavigation",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper(config.TabGeneral, false)
				},
				Play: func(pc *storybook.PlayContext) {
					// Cycle through all tabs with Tab key
					pc.Key("tab")
					pc.Wait(200 * time.Millisecond)
					pc.AssertContains("JIRA")
					pc.Key("tab")
					pc.Wait(200 * time.Millisecond)
					pc.Key("tab")
					pc.Wait(200 * time.Millisecond)
					pc.Key("shift+tab")
					pc.Wait(200 * time.Millisecond)
				},
			},
			{
				Name: "HelpOverlay",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newConfigModelWrapper(config.TabGeneral, false)
				},
				Play: func(pc *storybook.PlayContext) {
					pc.Key("?")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Keybinding Reference")
				},
			},
		},
	}
}

// configModelWrapper wraps ConfigModel to implement tea.Model for the storybook.
type configModelWrapper struct {
	model *config.ConfigModel
}

func newConfigModelWrapper(tab config.SectionTab, isNew bool) *configModelWrapper {
	opts := []config.ConfigModelOption{
		config.WithTheme(tui.DefaultTheme()),
		config.WithNewFile(isNew),
	}

	m := config.NewConfigModel(opts...)

	// Simulate terminal size so the model renders content
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Navigate to the desired tab
	for m.ActiveTab() != tab {
		m.Update(tea.KeyMsg{Type: tea.KeyTab})
	}

	return &configModelWrapper{model: m}
}

func (w *configModelWrapper) Init() tea.Cmd {
	return w.model.Init()
}

func (w *configModelWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	newModel, cmd := w.model.Update(msg)
	if cm, ok := newModel.(*config.ConfigModel); ok {
		w.model = cm
	}
	return w, cmd
}

func (w *configModelWrapper) View() string {
	return w.model.View()
}

// --- SectionModel stories ---.

// sectionModelStory creates stories for the SectionModel scrollable field list.
func sectionModelStory() storybook.Story {
	return storybook.Story{
		Name:        "SectionModel",
		Description: "Scrollable field list with groups, entries, and viewport",
		Variants: []storybook.Variant{
			{
				Name: "GeneralTab",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newSectionModelWrapper(generalSectionFields(), nil)
				},
			},
			{
				Name: "JIRATab",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newSectionModelWrapper(jiraSectionFields(), nil)
				},
			},
			{
				Name: "WithEntryList",
				Factory: func(_ ...storybook.Args) tea.Model {
					entries := []string{
						"main: .env, .env.local",
						"develop: config.yaml",
					}
					return newSectionModelWrapper(
						fileCopyAutoSectionFields(),
						entries,
					)
				},
			},
			{
				Name: "EmptySection",
				Factory: func(_ ...storybook.Args) tea.Model {
					sm := config.NewSectionModel(
						jiraSectionFields(),
						config.WithSectionTheme(tui.DefaultTheme()),
						config.WithViewportHeight(16),
						config.WithWidth(72),
						config.WithEmptyState(config.NewEmptyState(true, jiraSectionFields())),
					)
					return &sectionWrapper{section: sm}
				},
			},
			{
				Name: "Navigation",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newSectionModelWrapper(jiraSectionFields(), nil)
				},
				Play: func(pc *storybook.PlayContext) {
					// Navigate down through fields
					pc.Key("down")
					pc.Wait(150 * time.Millisecond)
					pc.Key("down")
					pc.Wait(150 * time.Millisecond)
					pc.Key("down")
					pc.Wait(150 * time.Millisecond)
					// Jump to next group
					pc.Key("}")
					pc.Wait(200 * time.Millisecond)
					pc.AssertContains("Filters")
				},
			},
		},
	}
}

// sectionWrapper wraps SectionModel as tea.Model for the storybook.
type sectionWrapper struct {
	section *config.SectionModel
}

func newSectionModelWrapper(fields []config.FieldMeta, entries []string) *sectionWrapper {
	opts := []config.SectionOption{
		config.WithSectionTheme(tui.DefaultTheme()),
		config.WithViewportHeight(16),
		config.WithWidth(72),
	}
	if entries != nil {
		opts = append(opts, config.WithEntryList("Rules", entries, "(no rules configured)"))
	}

	sm := config.NewSectionModel(fields, opts...)

	// Populate sample values
	for i, f := range fields {
		sm.SetFieldValue(i, sampleValue(f))
	}

	return &sectionWrapper{section: sm}
}

func (w *sectionWrapper) Init() tea.Cmd { return nil }

func (w *sectionWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "down", "j":
			w.section.MoveFocusDown()
		case "up", "k":
			w.section.MoveFocusUp()
		case "g":
			w.section.JumpToFirst()
		case "G":
			w.section.JumpToLast()
		case "}":
			w.section.JumpToNextGroup()
		case "{":
			w.section.JumpToPrevGroup()
		}
	}
	return w, nil
}

func (w *sectionWrapper) View() string {
	return w.section.View()
}

// --- FieldRow stories ---.

// fieldRowStory creates stories for the FieldRow component in various states.
func fieldRowStory() storybook.Story {
	return storybook.Story{
		Name:        "FieldRow",
		Description: "Single config field in browsing and editing modes",
		Variants: []storybook.Variant{
			{
				Name: "StringField",
				Factory: func(_ ...storybook.Args) tea.Model {
					fr := config.NewFieldRow(config.FieldMeta{
						Key: "default_branch", Label: "Default Branch", Type: config.String,
					}, tui.DefaultTheme())
					fr.SetValue("main")
					fr.SetFocused(true)
					fr.SetLabelWidth(20)
					fr.SetWidth(72)
					return &fieldRowWrapper{row: fr}
				},
			},
			{
				Name: "BoolField",
				Factory: func(_ ...storybook.Args) tea.Model {
					fr := config.NewFieldRow(config.FieldMeta{
						Key: "jira.attachments.enabled", Label: "Enabled", Type: config.Bool,
					}, tui.DefaultTheme())
					fr.SetValue(true)
					fr.SetFocused(true)
					fr.SetLabelWidth(20)
					fr.SetWidth(72)
					return &fieldRowWrapper{row: fr}
				},
				Play: func(pc *storybook.PlayContext) {
					// Toggle the bool value
					pc.Key("e")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Enabled")
				},
			},
			{
				Name: "SensitiveField",
				Factory: func(_ ...storybook.Args) tea.Model {
					fr := config.NewFieldRow(config.FieldMeta{
						Key: "jira.api_token", Label: "API Token", Type: config.SensitiveString,
					}, tui.DefaultTheme())
					fr.SetValue("secret-token-12345")
					fr.SetFocused(false)
					fr.SetLabelWidth(20)
					fr.SetWidth(72)
					return &fieldRowWrapper{row: fr}
				},
			},
			{
				Name: "DirtyField",
				Factory: func(_ ...storybook.Args) tea.Model {
					fr := config.NewFieldRow(config.FieldMeta{
						Key: "worktrees_dir", Label: "Worktrees Dir", Type: config.String,
					}, tui.DefaultTheme())
					fr.SetValue("./worktrees")
					fr.SetFocused(true)
					fr.SetDirty(true)
					fr.SetLabelWidth(20)
					fr.SetWidth(72)
					return &fieldRowWrapper{row: fr}
				},
			},
			{
				Name: "ErrorField",
				Factory: func(_ ...storybook.Args) tea.Model {
					fr := config.NewFieldRow(config.FieldMeta{
						Key: "default_branch", Label: "Default Branch", Type: config.String,
						Validate: config.ValidateRequired,
					}, tui.DefaultTheme())
					fr.SetValue("")
					fr.SetFocused(true)
					fr.SetHasError(true)
					fr.SetLabelWidth(20)
					fr.SetWidth(72)
					return &fieldRowWrapper{row: fr}
				},
			},
			{
				Name: "Editing",
				Factory: func(_ ...storybook.Args) tea.Model {
					fr := config.NewFieldRow(config.FieldMeta{
						Key:         "jira.host",
						Label:       "Host",
						Type:        config.String,
						Description: "JIRA server URL (e.g., https://jira.company.com)",
					}, tui.DefaultTheme())
					fr.SetValue("https://jira.example.com")
					fr.SetFocused(true)
					fr.SetLabelWidth(20)
					fr.SetWidth(72)
					fr.EnterEditing()
					return &fieldRowWrapper{row: fr}
				},
			},
			{
				Name: "StringListField",
				Factory: func(_ ...storybook.Args) tea.Model {
					fr := config.NewFieldRow(config.FieldMeta{
						Key: "jira.filters.status", Label: "Status", Type: config.StringList,
					}, tui.DefaultTheme())
					fr.SetValue([]string{"In Dev", "Open", "In Review"})
					fr.SetFocused(true)
					fr.SetLabelWidth(20)
					fr.SetWidth(72)
					return &fieldRowWrapper{row: fr}
				},
			},
		},
	}
}

// fieldRowWrapper wraps FieldRow as tea.Model for the storybook.
type fieldRowWrapper struct {
	row *config.FieldRow
}

func (w *fieldRowWrapper) Init() tea.Cmd { return nil }

func (w *fieldRowWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch w.row.State() {
		case config.FieldEditing:
			switch keyMsg.String() {
			case "enter":
				//nolint:errcheck // storybook: visual-only, errors shown inline
				w.row.CommitEditing()
			case "esc":
				w.row.CancelEditing()
			default:
				cmd := w.row.UpdateInput(msg)
				return w, cmd
			}
		case config.FieldBrowsing:
			if keyMsg.String() == "e" {
				if w.row.Meta().Type == config.Bool {
					//nolint:errcheck // storybook: visual-only toggle
					w.row.ToggleBool()
				} else {
					cmd := w.row.EnterEditing()
					return w, cmd
				}
			}
		}
	}
	return w, nil
}

func (w *fieldRowWrapper) View() string {
	return w.row.View()
}

// --- ListOverlay stories ---.

// listOverlayStory creates stories for the ListOverlay modal editor.
func listOverlayStory() storybook.Story {
	return storybook.Story{
		Name:        "ListOverlay",
		Description: "Modal editor for string list fields (add, delete, browse)",
		Variants: []storybook.Variant{
			{
				Name: "WithItems",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newListOverlayWrapper(
						"JIRA > Filters > Status",
						[]string{"In Dev", "Open", "In Review", "Done"},
					)
				},
			},
			{
				Name: "EmptyList",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newListOverlayWrapper(
						"File Copy > Exclude",
						[]string{},
					)
				},
			},
			{
				Name: "AddItem",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newListOverlayWrapper(
						"JIRA > Filters > Labels",
						[]string{"bug", "feature"},
					)
				},
				Play: func(pc *storybook.PlayContext) {
					// Press 'a' to add a new item
					pc.Key("a")
					pc.Wait(300 * time.Millisecond)
					pc.Type("enhancement")
					pc.Wait(200 * time.Millisecond)
					pc.Key("enter")
					pc.Wait(200 * time.Millisecond)
				},
			},
			{
				Name: "DeleteConfirmation",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newListOverlayWrapper(
						"Auto Copy > Exclude",
						[]string{"*.log", "node_modules/", "build/", ".cache/"},
					)
				},
				Play: func(pc *storybook.PlayContext) {
					// Navigate to an item and press 'd' to delete
					pc.Key("down")
					pc.Wait(150 * time.Millisecond)
					pc.Key("d")
					pc.Wait(300 * time.Millisecond)
					pc.AssertContains("Delete")
				},
			},
		},
	}
}

// listOverlayWrapper wraps ListOverlay as tea.Model for the storybook.
type listOverlayWrapper struct {
	overlay *config.ListOverlay
	width   int
	height  int
}

func newListOverlayWrapper(title string, items []string) *listOverlayWrapper {
	overlay := config.NewListOverlay(title, items, tui.DefaultTheme())
	overlay.SetSize(70, 20)
	return &listOverlayWrapper{
		overlay: overlay,
		width:   70,
		height:  20,
	}
}

func (w *listOverlayWrapper) Init() tea.Cmd { return nil }

func (w *listOverlayWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		w.width = sizeMsg.Width
		w.height = sizeMsg.Height
		w.overlay.SetSize(w.width, w.height)
		return w, nil
	}

	result, cmd := w.overlay.Update(msg)
	if result != nil {
		// Overlay wants to close; in the storybook we just ignore it
		return w, cmd
	}
	return w, cmd
}

func (w *listOverlayWrapper) View() string {
	return w.overlay.View(w.width, w.height)
}

// --- HelpOverlay stories ---.

// helpOverlayStory creates stories for the HelpOverlay keybinding reference.
func helpOverlayStory() storybook.Story {
	return storybook.Story{
		Name:        "HelpOverlay",
		Description: "Scrollable keybinding reference overlay",
		Variants: []storybook.Variant{
			{
				Name: "Default",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newHelpOverlayWrapper()
				},
			},
			{
				Name: "Scrolled",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newHelpOverlayWrapper()
				},
				Play: func(pc *storybook.PlayContext) {
					// Scroll down to see more sections
					pc.Key("down")
					pc.Wait(100 * time.Millisecond)
					pc.Key("down")
					pc.Wait(100 * time.Millisecond)
					pc.Key("down")
					pc.Wait(100 * time.Millisecond)
					pc.Key("down")
					pc.Wait(100 * time.Millisecond)
					pc.Key("down")
					pc.Wait(200 * time.Millisecond)
					pc.AssertContains("Keybinding Reference")
				},
			},
		},
	}
}

// helpOverlayWrapper wraps HelpOverlay as tea.Model for the storybook.
type helpOverlayWrapper struct {
	overlay *config.HelpOverlay
	width   int
	height  int
}

func newHelpOverlayWrapper() *helpOverlayWrapper {
	return &helpOverlayWrapper{
		overlay: config.NewHelpOverlay(tui.DefaultTheme()),
		width:   80,
		height:  24,
	}
}

func (w *helpOverlayWrapper) Init() tea.Cmd { return nil }

func (w *helpOverlayWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		w.width = sizeMsg.Width
		w.height = sizeMsg.Height
		return w, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		vpHeight := max(w.height-6, 1)
		closed := w.overlay.HandleKey(keyMsg, vpHeight)
		if closed {
			// In storybook we don't actually close; reset scroll instead
			w.overlay.ResetScroll()
		}
	}
	return w, nil
}

func (w *helpOverlayWrapper) View() string {
	return w.overlay.View(w.width, w.height)
}

// --- ErrorOverlay stories ---.

// errorOverlayStory creates stories for the ErrorOverlay validation display.
func errorOverlayStory() storybook.Story {
	return storybook.Story{
		Name:        "ErrorOverlay",
		Description: "Validation errors overlay with navigation and jump-to-field",
		Variants: []storybook.Variant{
			{
				Name: "SingleError",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newErrorOverlayWrapper([]config.ValidationError{
						{Tab: config.TabGeneral, FieldLabel: "Default Branch", Message: "this field is required", FieldIndex: 0},
					})
				},
			},
			{
				Name: "MultipleErrors",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newErrorOverlayWrapper([]config.ValidationError{
						{Tab: config.TabGeneral, FieldLabel: "Default Branch", Message: "this field is required", FieldIndex: 0},
						{Tab: config.TabGeneral, FieldLabel: "Worktrees Directory", Message: "this field is required", FieldIndex: 1},
						{Tab: config.TabJira, FieldLabel: "Host", Message: "must start with http:// or https://", FieldIndex: 0},
						{Tab: config.TabJira, FieldLabel: "Max Depth", Message: "must be zero or a positive integer", FieldIndex: 5},
					})
				},
				Play: func(pc *storybook.PlayContext) {
					// Navigate through errors
					pc.Key("down")
					pc.Wait(200 * time.Millisecond)
					pc.Key("down")
					pc.Wait(200 * time.Millisecond)
					pc.AssertContains("Validation Errors")
				},
			},
			{
				Name: "NoErrors",
				Factory: func(_ ...storybook.Args) tea.Model {
					return newErrorOverlayWrapper(nil)
				},
			},
		},
	}
}

// errorOverlayWrapper wraps ErrorOverlay as tea.Model for the storybook.
type errorOverlayWrapper struct {
	overlay *config.ErrorOverlay
	width   int
	height  int
}

func newErrorOverlayWrapper(errs []config.ValidationError) *errorOverlayWrapper {
	return &errorOverlayWrapper{
		overlay: config.NewErrorOverlay(errs, tui.DefaultTheme()),
		width:   80,
		height:  24,
	}
}

func (w *errorOverlayWrapper) Init() tea.Cmd { return nil }

func (w *errorOverlayWrapper) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		w.width = sizeMsg.Width
		w.height = sizeMsg.Height
		return w, nil
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		w.overlay.HandleKey(keyMsg)
	}
	return w, nil
}

func (w *errorOverlayWrapper) View() string {
	return w.overlay.View(w.width, w.height)
}

// --- Helpers ---.

// sampleValue returns a representative sample value for a given FieldMeta.
func sampleValue(f config.FieldMeta) string {
	switch f.Type {
	case config.Bool:
		return "yes"
	case config.Int:
		return "10"
	case config.StringList:
		return "item1, item2"
	case config.SensitiveString:
		return "********"
	default:
		switch f.Key {
		case "default_branch":
			return "main"
		case "worktrees_dir":
			return "./worktrees"
		case "jira.host":
			return "https://jira.company.com"
		case "jira.me":
			return "user@company.com"
		default:
			if f.Label != "" {
				return f.Label + " value"
			}
			return "--"
		}
	}
}

// generalSectionFields returns the General tab fields.
func generalSectionFields() []config.FieldMeta {
	return []config.FieldMeta{
		{
			Key: "default_branch", Label: "Default Branch", Type: config.String,
			Validate: config.ValidateRequired,
		},
		{
			Key: "worktrees_dir", Label: "Worktrees Directory", Type: config.String,
			Description: "Supports templates: {gitroot}, {branch}, {issue}",
			Validate:    config.ValidateRequired,
		},
	}
}

// jiraSectionFields returns a representative subset of JIRA tab fields with groups.
func jiraSectionFields() []config.FieldMeta {
	return []config.FieldMeta{
		{Key: "jira.host", Label: "Host", Type: config.String, Group: "Connection"},
		{Key: "jira.me", Label: "Username", Type: config.String, Group: "Connection"},
		{Key: "jira.filters.priority", Label: "Priority", Type: config.String, Group: "Filters"},
		{Key: "jira.filters.type", Label: "Type", Type: config.String, Group: "Filters"},
		{Key: "jira.filters.status", Label: "Status", Type: config.StringList, Group: "Filters"},
		{Key: "jira.filters.reverse", Label: "Reverse", Type: config.Bool, Group: "Filters"},
		{Key: "jira.markdown.filename_pattern", Label: "Filename Pattern", Type: config.String, Group: "Markdown"},
		{Key: "jira.markdown.include_comments", Label: "Include Comments", Type: config.Bool, Group: "Markdown"},
		{Key: "jira.attachments.enabled", Label: "Enabled", Type: config.Bool, Group: "Attachments"},
		{Key: "jira.attachments.max_size_mb", Label: "Max Size (MB)", Type: config.Int, Group: "Attachments"},
		{Key: "jira.attachments.directory", Label: "Directory", Type: config.String, Group: "Attachments"},
	}
}

// fileCopyAutoSectionFields returns the File Copy tab's auto-copy fields.
func fileCopyAutoSectionFields() []config.FieldMeta {
	return []config.FieldMeta{
		{Key: "file_copy.auto.enabled", Label: "Enabled", Type: config.Bool, Group: "Auto Copy"},
		{
			Key: "file_copy.auto.source_worktree", Label: "Source Worktree", Type: config.String, Group: "Auto Copy",
			Description: "Supports template: {default}",
		},
		{Key: "file_copy.auto.copy_ignored", Label: "Copy Ignored", Type: config.Bool, Group: "Auto Copy"},
		{Key: "file_copy.auto.copy_untracked", Label: "Copy Untracked", Type: config.Bool, Group: "Auto Copy"},
		{Key: "file_copy.auto.exclude", Label: "Exclude", Type: config.StringList, Group: "Auto Copy"},
	}
}
