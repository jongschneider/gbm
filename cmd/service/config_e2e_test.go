package service

import (
	"gbm/internal/jira"
	"gbm/pkg/tui"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// TestConfigTUI_E2E_HappyPath tests the complete config TUI flow:
// load config → navigate to Basics → edit → save → verify file.
// This test exercises the actual ConfigModel with service layer callbacks.
func TestConfigTUI_E2E_HappyPath(t *testing.T) {
	// Create temp directory for test repo
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "test-repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	// Create .gbm directory and initial config
	gbmDir := filepath.Join(repoRoot, ".gbm")
	require.NoError(t, os.MkdirAll(gbmDir, 0o755))

	configPath := filepath.Join(gbmDir, "config.yaml")
	initialConfig := &Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
	}
	initialData, err := yaml.Marshal(initialConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, initialData, 0o644))

	// Create a mock service with the config
	svc := &Service{
		config:   initialConfig,
		RepoRoot: repoRoot,
	}

	// Create theme
	theme := tui.DefaultTheme()

	// Create initial state from config (same as runConfigTUI)
	initialState := configToState(svc.config)

	// Create form factory (same as runConfigTUI)
	formFactory := createFormFactory(initialState)

	// Track if save was called
	saveCalled := false
	var savedState *tui.ConfigState

	// Create save callback
	onSave := func(state *tui.ConfigState) error {
		saveCalled = true
		savedState = state
		stateToConfig(state, svc.config)
		return svc.SaveConfig()
	}

	// Create reset callback
	onReset := func() (*tui.ConfigState, error) {
		return configToState(svc.config), nil
	}

	// Create ConfigModel (same as runConfigTUI)
	model := tui.NewConfigModel(
		theme,
		tui.WithInitialState(initialState),
		tui.WithFormFactory(formFactory),
		tui.WithOnSave(onSave),
		tui.WithOnReset(onReset),
	)

	// Initialize model
	model.Init()

	// Verify initial state
	assert.Equal(t, "main", model.GetState().DefaultBranch)
	assert.Equal(t, "worktrees", model.GetState().WorktreesDir)
	assert.False(t, model.IsDirty())

	// === Step 1: Navigate to Basics section ===
	// Send SidebarSelectionMsg to select Basics (this focuses the content pane)
	model.Update(tui.SidebarSelectionMsg{Section: "Basics"})

	// Verify focus moved to content pane
	assert.Equal(t, tui.ContentFocused, model.GetPaneFocus())

	// === Step 2: Edit the state directly (simulating form edits) ===
	// In a real TUI, the BasicsForm would update state via its fields.
	// We simulate this by directly modifying the state. We also clear the
	// form cache so the save flow's flushAllForms() doesn't overwrite our
	// direct state changes with the forms' stale initial values.
	for k := range model.GetFormCache() {
		delete(model.GetFormCache(), k)
	}
	model.GetState().DefaultBranch = "develop"
	model.GetState().WorktreesDir = "my-worktrees"

	// Mark as dirty (this would happen via the onUpdate callback in real use)
	model.GetState().MarkDirty()
	assert.True(t, model.IsDirty())

	// === Step 3: Save via Ctrl+S + confirm ===
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Verify save was called
	assert.True(t, saveCalled)
	assert.NotNil(t, savedState)
	assert.Equal(t, "develop", savedState.DefaultBranch)
	assert.Equal(t, "my-worktrees", savedState.WorktreesDir)

	// Verify dirty flag was cleared
	assert.False(t, model.IsDirty())

	// === Step 5: Verify file was written correctly ===
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var savedConfig Config
	require.NoError(t, yaml.Unmarshal(fileData, &savedConfig))

	assert.Equal(t, "develop", savedConfig.DefaultBranch)
	assert.Equal(t, "my-worktrees", savedConfig.WorktreesDir)
}

// TestConfigTUI_E2E_EditJira tests editing JIRA settings.
func TestConfigTUI_E2E_EditJira(t *testing.T) {
	// Create temp directory for test repo
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "test-repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	// Create .gbm directory and initial config
	gbmDir := filepath.Join(repoRoot, ".gbm")
	require.NoError(t, os.MkdirAll(gbmDir, 0o755))

	configPath := filepath.Join(gbmDir, "config.yaml")
	initialConfig := &Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
	}
	initialData, err := yaml.Marshal(initialConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, initialData, 0o644))

	// Create mock service
	svc := &Service{
		config:   initialConfig,
		RepoRoot: repoRoot,
	}

	// Create state and model
	theme := tui.DefaultTheme()
	initialState := configToState(svc.config)
	formFactory := createFormFactory(initialState)

	onSave := func(state *tui.ConfigState) error {
		stateToConfig(state, svc.config)
		return svc.SaveConfig()
	}

	model := tui.NewConfigModel(
		theme,
		tui.WithInitialState(initialState),
		tui.WithFormFactory(formFactory),
		tui.WithOnSave(onSave),
	)
	model.Init()

	// Navigate to JIRA section
	model.Update(tui.SidebarSelectionMsg{Section: "JIRA"})
	assert.Equal(t, tui.ContentFocused, model.GetPaneFocus())

	// Edit JIRA settings (simulating form edits by setting state directly).
	// Clear form cache so flushAllForms() doesn't overwrite direct state changes.
	for k := range model.GetFormCache() {
		delete(model.GetFormCache(), k)
	}
	state := model.GetState()
	state.JiraEnabled = true
	state.JiraUsername = "test@example.com"
	state.JiraHost = "https://jira.example.com"
	state.JiraFiltersStatus = []string{"Open", "In Progress"}
	state.JiraAttachmentsEnabled = true
	state.JiraAttachmentsMaxSize = 10
	state.JiraMarkdownIncludeComments = true
	state.MarkDirty()

	// Save via Ctrl+S + confirm
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Verify file was written correctly
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var savedConfig Config
	require.NoError(t, yaml.Unmarshal(fileData, &savedConfig))

	// Verify basic fields preserved
	assert.Equal(t, "main", savedConfig.DefaultBranch)
	assert.Equal(t, "worktrees", savedConfig.WorktreesDir)

	// Verify JIRA settings
	assert.Equal(t, "https://jira.example.com", savedConfig.Jira.Host)
	assert.Equal(t, "test@example.com", savedConfig.Jira.Me)
	assert.Equal(t, []string{"Open", "In Progress"}, savedConfig.Jira.Filters.Status)
	assert.True(t, savedConfig.Jira.Attachments.Enabled)
	assert.Equal(t, int64(10), savedConfig.Jira.Attachments.MaxSizeMB)
	assert.True(t, savedConfig.Jira.Markdown.IncludeComments)
}

// TestConfigTUI_E2E_EditFileCopyRules tests editing file copy rules.
func TestConfigTUI_E2E_EditFileCopyRules(t *testing.T) {
	// Create temp directory for test repo
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "test-repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	// Create .gbm directory and initial config
	gbmDir := filepath.Join(repoRoot, ".gbm")
	require.NoError(t, os.MkdirAll(gbmDir, 0o755))

	configPath := filepath.Join(gbmDir, "config.yaml")
	initialConfig := &Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
	}
	initialData, err := yaml.Marshal(initialConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, initialData, 0o644))

	// Create mock service
	svc := &Service{
		config:   initialConfig,
		RepoRoot: repoRoot,
	}

	// Create state and model
	theme := tui.DefaultTheme()
	initialState := configToState(svc.config)
	formFactory := createFormFactory(initialState)

	onSave := func(state *tui.ConfigState) error {
		stateToConfig(state, svc.config)
		return svc.SaveConfig()
	}

	model := tui.NewConfigModel(
		theme,
		tui.WithInitialState(initialState),
		tui.WithFormFactory(formFactory),
		tui.WithOnSave(onSave),
	)
	model.Init()

	// Navigate to FileCopy section
	model.Update(tui.SidebarSelectionMsg{Section: "FileCopy"})
	assert.Equal(t, tui.ContentFocused, model.GetPaneFocus())

	// Add file copy rules (simulating form edits by setting state directly).
	// Clear form cache so flushAllForms() doesn't overwrite direct state changes.
	for k := range model.GetFormCache() {
		delete(model.GetFormCache(), k)
	}
	state := model.GetState()
	state.FileCopyRules = []tui.FileCopyRuleState{
		{
			SourceWorktree: "main",
			Files:          []string{".env", "config.local.yaml"},
		},
		{
			SourceWorktree: "develop",
			Files:          []string{"test-fixtures/"},
		},
	}
	state.MarkDirty()

	// Save via Ctrl+S + confirm
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Verify file was written correctly
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var savedConfig Config
	require.NoError(t, yaml.Unmarshal(fileData, &savedConfig))

	// Verify file copy rules
	require.Len(t, savedConfig.FileCopy.Rules, 2)
	assert.Equal(t, "main", savedConfig.FileCopy.Rules[0].SourceWorktree)
	assert.Equal(t, []string{".env", "config.local.yaml"}, savedConfig.FileCopy.Rules[0].Files)
	assert.Equal(t, "develop", savedConfig.FileCopy.Rules[1].SourceWorktree)
}

// TestConfigTUI_E2E_EditWorktrees tests editing worktree entries.
func TestConfigTUI_E2E_EditWorktrees(t *testing.T) {
	// Create temp directory for test repo
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "test-repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	// Create .gbm directory and initial config
	gbmDir := filepath.Join(repoRoot, ".gbm")
	require.NoError(t, os.MkdirAll(gbmDir, 0o755))

	configPath := filepath.Join(gbmDir, "config.yaml")
	initialConfig := &Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
	}
	initialData, err := yaml.Marshal(initialConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, initialData, 0o644))

	// Create mock service
	svc := &Service{
		config:   initialConfig,
		RepoRoot: repoRoot,
	}

	// Create state and model
	theme := tui.DefaultTheme()
	initialState := configToState(svc.config)
	formFactory := createFormFactory(initialState)

	onSave := func(state *tui.ConfigState) error {
		stateToConfig(state, svc.config)
		return svc.SaveConfig()
	}

	model := tui.NewConfigModel(
		theme,
		tui.WithInitialState(initialState),
		tui.WithFormFactory(formFactory),
		tui.WithOnSave(onSave),
	)
	model.Init()

	// Navigate to Worktrees section
	model.Update(tui.SidebarSelectionMsg{Section: "Worktrees"})
	assert.Equal(t, tui.ContentFocused, model.GetPaneFocus())

	// Add worktree entries (simulating form edits by setting state directly).
	// Clear form cache so flushAllForms() doesn't overwrite direct state changes.
	for k := range model.GetFormCache() {
		delete(model.GetFormCache(), k)
	}
	state := model.GetState()
	state.Worktrees = []tui.WorktreeEntryState{
		{
			Name:        "feature-x",
			Branch:      "feature/feature-x",
			MergeInto:   "develop",
			Description: "Feature X branch",
		},
		{
			Name:        "hotfix-y",
			Branch:      "hotfix/hotfix-y",
			MergeInto:   "main",
			Description: "Hotfix Y branch",
		},
	}
	state.MarkDirty()

	// Save via Ctrl+S + confirm
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Verify file was written correctly
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var savedConfig Config
	require.NoError(t, yaml.Unmarshal(fileData, &savedConfig))

	// Verify worktree entries
	require.Len(t, savedConfig.Worktrees, 2)
	assert.Equal(t, "feature/feature-x", savedConfig.Worktrees["feature-x"].Branch)
	assert.Equal(t, "develop", savedConfig.Worktrees["feature-x"].MergeInto)
	assert.Equal(t, "hotfix/hotfix-y", savedConfig.Worktrees["hotfix-y"].Branch)
	assert.Equal(t, "main", savedConfig.Worktrees["hotfix-y"].MergeInto)
}

// TestConfigTUI_E2E_Reset tests the reset functionality.
func TestConfigTUI_E2E_Reset(t *testing.T) {
	// Create temp directory for test repo
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "test-repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	// Create .gbm directory and initial config
	gbmDir := filepath.Join(repoRoot, ".gbm")
	require.NoError(t, os.MkdirAll(gbmDir, 0o755))

	configPath := filepath.Join(gbmDir, "config.yaml")
	initialConfig := &Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
	}
	initialData, err := yaml.Marshal(initialConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, initialData, 0o644))

	// Create mock service
	svc := &Service{
		config:   initialConfig,
		RepoRoot: repoRoot,
	}

	// Create state and model
	theme := tui.DefaultTheme()
	initialState := configToState(svc.config)
	formFactory := createFormFactory(initialState)

	onSave := func(state *tui.ConfigState) error {
		stateToConfig(state, svc.config)
		return svc.SaveConfig()
	}

	onReset := func() (*tui.ConfigState, error) {
		// Simulate reloading config from file
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, err
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		return configToState(&cfg), nil
	}

	model := tui.NewConfigModel(
		theme,
		tui.WithInitialState(initialState),
		tui.WithFormFactory(formFactory),
		tui.WithOnSave(onSave),
		tui.WithOnReset(onReset),
	)
	model.Init()

	// Make unsaved changes
	state := model.GetState()
	state.DefaultBranch = "develop"
	state.MarkDirty()
	assert.Equal(t, "develop", model.GetState().DefaultBranch)
	assert.True(t, model.IsDirty())

	// Press 'r' to reset (shows discard confirmation when dirty)
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Confirm discard of unsaved changes
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Verify state was reset to original
	assert.Equal(t, "main", model.GetState().DefaultBranch)
}

// TestConfigTUI_E2E_PreservesOriginalFields tests that saving doesn't lose unrelated config fields.
func TestConfigTUI_E2E_PreservesOriginalFields(t *testing.T) {
	// Create temp directory for test repo
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "test-repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	// Create .gbm directory and initial config with many fields
	gbmDir := filepath.Join(repoRoot, ".gbm")
	require.NoError(t, os.MkdirAll(gbmDir, 0o755))

	configPath := filepath.Join(gbmDir, "config.yaml")
	initialConfig := &Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
		Jira: JiraConfig{
			Host: "https://jira.original.com",
			Me:   "original@example.com",
			Filters: jira.JiraFilters{
				Status:   []string{"Open"},
				Priority: "High",
			},
		},
		FileCopy: FileCopyConfig{
			Rules: []FileCopyRule{
				{SourceWorktree: "main", Files: []string{".env"}},
			},
		},
		Worktrees: map[string]WorktreeConfig{
			"existing": {Branch: "existing-branch", MergeInto: "main"},
		},
	}
	initialData, err := yaml.Marshal(initialConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, initialData, 0o644))

	// Create mock service
	svc := &Service{
		config:   initialConfig,
		RepoRoot: repoRoot,
	}

	// Create state and model
	theme := tui.DefaultTheme()
	initialState := configToState(svc.config)
	formFactory := createFormFactory(initialState)

	onSave := func(state *tui.ConfigState) error {
		stateToConfig(state, svc.config)
		return svc.SaveConfig()
	}

	model := tui.NewConfigModel(
		theme,
		tui.WithInitialState(initialState),
		tui.WithFormFactory(formFactory),
		tui.WithOnSave(onSave),
	)
	model.Init()

	// Only modify the basics (simulating form edits by setting state directly).
	// Clear form cache so flushAllForms() doesn't overwrite direct state changes.
	for k := range model.GetFormCache() {
		delete(model.GetFormCache(), k)
	}
	state := model.GetState()
	state.DefaultBranch = "develop"
	state.MarkDirty()

	// Save via Ctrl+S + confirm
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Verify file was written correctly
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var savedConfig Config
	require.NoError(t, yaml.Unmarshal(fileData, &savedConfig))

	// Verify edited field
	assert.Equal(t, "develop", savedConfig.DefaultBranch)

	// Verify original fields are preserved
	assert.Equal(t, "worktrees", savedConfig.WorktreesDir)
	assert.Equal(t, "https://jira.original.com", savedConfig.Jira.Host)
	assert.Equal(t, "original@example.com", savedConfig.Jira.Me)
	assert.Equal(t, []string{"Open"}, savedConfig.Jira.Filters.Status)
	require.Len(t, savedConfig.FileCopy.Rules, 1)
	assert.Equal(t, "main", savedConfig.FileCopy.Rules[0].SourceWorktree)
	require.Contains(t, savedConfig.Worktrees, "existing")
	assert.Equal(t, "existing-branch", savedConfig.Worktrees["existing"].Branch)
}

// TestConfigTUI_E2E_FlushToState_BasicsForm tests the real save flow where
// field values are typed into the BasicsForm and then saved via Ctrl+S.
// This exercises the FlushToState path (forms → state → config → disk)
// rather than bypassing it by editing state directly.
func TestConfigTUI_E2E_FlushToState_BasicsForm(t *testing.T) {
	// Create temp directory for test repo
	tmpDir := t.TempDir()
	repoRoot := filepath.Join(tmpDir, "test-repo")
	require.NoError(t, os.MkdirAll(repoRoot, 0o755))

	// Create .gbm directory and initial config
	gbmDir := filepath.Join(repoRoot, ".gbm")
	require.NoError(t, os.MkdirAll(gbmDir, 0o755))

	configPath := filepath.Join(gbmDir, "config.yaml")
	initialConfig := &Config{
		DefaultBranch: "main",
		WorktreesDir:  "worktrees",
	}
	initialData, err := yaml.Marshal(initialConfig)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, initialData, 0o644))

	// Create mock service
	svc := &Service{
		config:   initialConfig,
		RepoRoot: repoRoot,
	}

	// Create state and model (same as runConfigTUI)
	theme := tui.DefaultTheme()
	initialState := configToState(svc.config)
	formFactory := createFormFactory(initialState)

	onSave := func(state *tui.ConfigState) error {
		stateToConfig(state, svc.config)
		return svc.SaveConfig()
	}

	model := tui.NewConfigModel(
		theme,
		tui.WithInitialState(initialState),
		tui.WithFormFactory(formFactory),
		tui.WithOnSave(onSave),
	)
	model.Init()

	// Send a WindowSizeMsg so the form can render
	model.Update(tea.WindowSizeMsg{Width: 120, Height: 40})

	// Navigate to Basics section via sidebar selection (Enter on Basics)
	model.Update(tui.SidebarSelectionMsg{Section: "Basics"})
	assert.Equal(t, tui.ContentFocused, model.GetPaneFocus())

	// The Basics form is now focused. Field 0 (default_branch) should be focused.
	// The text input should contain "main" from the initial config.
	// Clear the text field by selecting all and typing the new value.
	// Use Ctrl+A to select all, then type the new value.
	// Actually, TextInput from bubbles doesn't support Ctrl+A. We need to
	// delete the existing text character by character and then type new text.

	// First, clear the existing "main" text (4 chars) by pressing backspace
	for i := 0; i < 10; i++ { // extra backspaces to be safe
		model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	}

	// Type "develop"
	for _, ch := range "develop" {
		model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}

	// Now press Ctrl+S to trigger save flow
	model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})

	// Confirm save (press 'y')
	model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	// Verify file was written correctly
	fileData, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var savedConfig Config
	require.NoError(t, yaml.Unmarshal(fileData, &savedConfig))

	assert.Equal(t, "develop", savedConfig.DefaultBranch,
		"FlushToState should capture the typed value from the text input field")
	assert.Equal(t, "worktrees", savedConfig.WorktreesDir,
		"Unmodified fields should preserve their original values")
}
