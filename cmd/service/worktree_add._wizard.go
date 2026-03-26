package service

import (
	"bufio"
	"errors"
	"fmt"
	"gbm/internal/git"
	"gbm/internal/jira"
	"gbm/pkg/tui"
	"gbm/pkg/tui/workflows"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// addNavigatorAdapter wraps add workflow to work with Navigator.
// It handles type selection and workflow progression via the Navigator stack.
type addNavigatorAdapter struct {
	nav      *tui.Navigator
	stepsMap map[string][]tui.Step
	ctx      *tui.Context
}

// newAddNavigatorAdapter creates a new adapter with Navigator initialized with type selector.
func newAddNavigatorAdapter(ctx *tui.Context, stepsMap map[string][]tui.Step) *addNavigatorAdapter {
	typeSelector := workflows.SelectWorkflowType()
	typeSelectorStep := tui.Step{
		Name:  "workflow_type_selector",
		Field: typeSelector,
	}
	typeWizard := tui.NewWizard([]tui.Step{typeSelectorStep}, ctx)

	return &addNavigatorAdapter{
		nav:      tui.NewNavigator(typeWizard),
		stepsMap: stepsMap,
		ctx:      ctx,
	}
}

// Init delegates to Navigator.
func (a *addNavigatorAdapter) Init() tea.Cmd {
	return a.nav.Init()
}

// Update handles type selection completion and workflow transitions.
func (a *addNavigatorAdapter) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle Ctrl+C to quit
	if km, ok := msg.(tea.KeyMsg); ok && km.Type == tea.KeyCtrlC {
		return a, tea.Quit
	}

	// Handle back navigation from workflow
	if _, ok := msg.(tui.BackBoundaryMsg); ok && a.nav.Depth() > 1 {
		a.nav.Pop()
		// Reset context state when going back to type selector
		a.ctx.State = &tui.WorkflowState{}
		newTypeWizard := a.createTypeWizard()
		a.nav.Pop()
		a.nav.Push(newTypeWizard)
		return a, newTypeWizard.Init()
	}

	// Update through Navigator
	_, cmd := a.nav.Update(msg)
	currentModel := a.nav.Current()

	// Handle type selector completion
	if a.nav.Depth() == 1 {
		if wiz, ok := currentModel.(*tui.Wizard); ok && wiz.IsComplete() {
			cmd = a.transitionToWorkflow(wiz)
			if cmd != nil {
				return a, cmd
			}
		}
	}

	// Handle workflow completion or workflow complete message
	if a.nav.Depth() > 1 {
		if wiz, ok := currentModel.(*tui.Wizard); ok && wiz.IsComplete() {
			return a, tea.Quit
		}
		if _, ok := msg.(tui.WorkflowCompleteMsg); ok {
			return a, tea.Quit
		}
	}

	return a, cmd
}

// transitionToWorkflow creates and pushes workflow wizard for selected type.
func (a *addNavigatorAdapter) transitionToWorkflow(typeWiz *tui.Wizard) tea.Cmd {
	selectedType := typeWiz.State().WorkflowType
	if selectedType == "" || selectedType == "unknown" {
		return nil
	}

	steps, ok := a.stepsMap[selectedType]
	if !ok {
		return nil
	}

	workflowWizard := tui.NewWizard(steps, a.ctx)
	workflowWizard.State().WorkflowType = selectedType
	a.nav.Push(workflowWizard)
	return workflowWizard.Init()
}

// createTypeWizard creates a fresh type selector wizard.
func (a *addNavigatorAdapter) createTypeWizard() *tui.Wizard {
	typeSelector := workflows.SelectWorkflowType()
	typeSelectorStep := tui.Step{
		Name:  "workflow_type_selector",
		Field: typeSelector,
	}
	return tui.NewWizard([]tui.Step{typeSelectorStep}, a.ctx)
}

// View delegates to Navigator.
func (a *addNavigatorAdapter) View() string {
	return a.nav.View()
}

// Context returns the TUI context for accessing services.
func (a *addNavigatorAdapter) Context() *tui.Context {
	return a.ctx
}

// gitServiceAdapter adapts *git.Service to tui.GitService interface,
// converting git.Worktree to tui.WorktreeInfo.
type gitServiceAdapter struct {
	svc *git.Service
}

func (a *gitServiceAdapter) BranchExists(branch string) (bool, error) {
	return a.svc.BranchExists(branch)
}

func (a *gitServiceAdapter) ListBranches(dryRun bool) ([]string, error) {
	return a.svc.ListBranches(dryRun)
}

func (a *gitServiceAdapter) ListWorktrees(dryRun bool) ([]tui.WorktreeInfo, error) {
	wts, err := a.svc.ListWorktrees(dryRun)
	if err != nil {
		return nil, err
	}
	infos := make([]tui.WorktreeInfo, len(wts))
	for i, wt := range wts {
		infos[i] = tui.WorktreeInfo{Name: wt.Name, Branch: wt.Branch}
	}
	return infos, nil
}

// jiraServiceAdapter adapts *jira.Service to tui.JiraService interface.
type jiraServiceAdapter struct {
	jiraService jiraService
}

type jiraService interface {
	GetJiraIssues(filters jira.JiraFilters, dryRun bool) ([]jira.JiraIssue, error)
}

func newJiraServiceAdapter(jiraSvc jiraService) *jiraServiceAdapter {
	return &jiraServiceAdapter{jiraService: jiraSvc}
}

func (a *jiraServiceAdapter) FetchIssues() ([]tui.JiraIssue, error) {
	// Fetch issues using an empty filter and no dry-run
	issues, err := a.jiraService.GetJiraIssues(jira.JiraFilters{}, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JIRA issues: %w", err)
	}

	// Convert internal JiraIssue to tui.JiraIssue
	result := make([]tui.JiraIssue, len(issues))
	for i, issue := range issues {
		result[i] = tui.JiraIssue{
			Key:     issue.Key,
			Summary: issue.Summary,
		}
	}
	return result, nil
}

// runWorktreeAddWizardTUI runs the add wizard using real service dependencies.
// Uses Navigator to manage seamless transitions between:
// 1. Type selector screen - user chooses Feature/Bug/Hotfix/Merge
// 2. Workflow screen - workflow-specific steps for the selected type.
func runWorktreeAddWizardTUI(svc *Service) error {
	// Open /dev/tty for TUI rendering FIRST, before creating any models/styles
	// This is required for shell integration where stdout is captured
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("failed to open /dev/tty: %w (TUI requires an interactive terminal)", err)
	}
	defer func() {
		//nolint:errcheck // Best-effort cleanup
		tty.Close()
	}()

	// Set up color renderer BEFORE creating the model, so styles are created with the correct renderer
	renderer := lipgloss.NewRenderer(tty,
		termenv.WithColorCache(true),
		termenv.WithTTY(true),
		termenv.WithProfile(termenv.TrueColor),
	)
	lipgloss.SetDefaultRenderer(renderer)

	// Resolve worktrees directory for display in review screen
	worktreesDir, err := svc.GetWorktreesPath()
	if err != nil {
		worktreesDir = "" // Non-fatal: review screen will just omit the path
	}

	// Build context with services (after renderer is set up)
	ctx := tui.NewContext().
		WithDimensions(100, 30).
		WithTheme(tui.DefaultTheme()).
		WithGitService(&gitServiceAdapter{svc: svc.Git}).
		WithJiraService(newJiraServiceAdapter(svc.Jira)).
		WithConfig(svc.GetConfig()).
		WithWorktreesDir(worktreesDir)

	// Build stepsMap for all workflow types
	stepsMap, err := buildStepsMap(ctx)
	if err != nil {
		return err
	}

	// Run the adapter program with tty for both input and output
	adapter := newAddNavigatorAdapter(ctx, stepsMap)
	p := tea.NewProgram(adapter,
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithAltScreen(),
	)
	finalModel, err := p.Run()
	if err != nil {
		return fmt.Errorf("testadd error: %w", err)
	}

	// Handle final state
	return handleFinalState(finalModel, svc)
}

// buildStepsMap creates workflow steps for each workflow type.
func buildStepsMap(ctx *tui.Context) (map[string][]tui.Step, error) {
	stepsMap := make(map[string][]tui.Step)
	for _, workflowType := range []string{"feature", "bug", "hotfix", "merge"} {
		steps, err := workflows.GetWorkflowSteps(workflowType, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to get steps for workflow %s: %w", workflowType, err)
		}
		stepsMap[workflowType] = steps
	}
	return stepsMap, nil
}

// handleFinalState processes the final wizard state and creates the worktree.
func handleFinalState(finalModel tea.Model, svc *Service) error {
	state, err := extractWizardState(finalModel)
	if err != nil {
		return err
	}
	if state == nil {
		// Cancelled
		return nil
	}

	// Process merge workflow to populate names from custom fields
	if state.WorkflowType == tui.WorkflowTypeMerge {
		err := processMergeState(state)
		if err != nil {
			return fmt.Errorf("failed to process merge workflow: %w", err)
		}
	}

	// Handle existing worktree replacement
	if err := handleExistingWorktree(svc, state.WorktreeName); err != nil {
		return err
	}

	// Create the worktree
	wt, err := createWorktreeFromState(svc, state)
	if err != nil {
		return err
	}

	// Handle merge workflow post-creation steps
	if state.WorkflowType == tui.WorkflowTypeMerge {
		err := performMergeWorkflow(svc, wt, state)
		if err != nil {
			return err
		}
	}

	// Post-creation tasks (non-fatal)
	if err := svc.CopyFilesToWorktree(state.WorktreeName); err != nil {
		PrintWarning(fmt.Sprintf("failed to copy files to worktree: %v", err))
	}
	if err := svc.CreateJiraMarkdownFile(state.WorktreeName); err != nil {
		PrintWarning(fmt.Sprintf("failed to create JIRA markdown: %v", err))
	}

	return nil
}

// extractWizardState extracts and validates the workflow state from the final model.
// Returns nil state if the wizard was cancelled.
func extractWizardState(finalModel tea.Model) (*tui.WorkflowState, error) {
	adapter, ok := finalModel.(*addNavigatorAdapter)
	if !ok {
		return nil, fmt.Errorf("unexpected model type: %T", finalModel)
	}

	currentModel := adapter.nav.Current()
	if currentModel == nil {
		return nil, errors.New("no wizard found")
	}

	w, ok := currentModel.(*tui.Wizard)
	if !ok {
		return nil, fmt.Errorf("unexpected wizard type: %T", currentModel)
	}

	if !w.IsComplete() {
		fmt.Fprintf(os.Stderr, "Cancelled\n")
		return nil, nil
	}

	return w.State(), nil
}

// handleExistingWorktree checks if worktree exists and prompts for replacement.
// Returns nil if worktree doesn't exist or was successfully removed.
func handleExistingWorktree(svc *Service, worktreeName string) error {
	existingWorktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	var exists bool
	for _, wt := range existingWorktrees {
		if wt.Name == worktreeName {
			exists = true
			break
		}
	}

	if !exists {
		return nil
	}

	if !ShouldAllowInput() {
		return fmt.Errorf("worktree '%s' already exists", worktreeName)
	}

	fmt.Fprintf(os.Stderr, "Worktree '%s' already exists. Replace it? (y/N): ", worktreeName)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read user input: %w", err)
	}
	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Fprintf(os.Stderr, "Cancelled\n")
		return nil
	}

	if _, err := svc.Git.RemoveWorktree(worktreeName, false, false); err != nil {
		return fmt.Errorf("failed to remove existing worktree: %w", err)
	}
	PrintSuccess(fmt.Sprintf("Removed existing worktree '%s'", worktreeName))
	return nil
}

// createWorktreeFromState creates a new worktree using the wizard state.
func createWorktreeFromState(svc *Service, state *tui.WorkflowState) (*git.Worktree, error) {
	worktreesDir, err := svc.GetWorktreesPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktrees directory: %w", err)
	}

	wt, err := svc.Git.AddWorktree(worktreesDir, state.WorktreeName, state.BranchName, true, state.BaseBranch, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create worktree: %w", err)
	}

	fmt.Println(wt.Path)
	PrintSuccess(fmt.Sprintf("Created worktree '%s' for branch '%s'", wt.Name, wt.Branch))
	return wt, nil
}

// performMergeWorkflow handles the merge operation for merge workflows.
func performMergeWorkflow(svc *Service, wt *git.Worktree, state *tui.WorkflowState) error {
	sourceBranch := getCustomField(state, "source_branch")
	if sourceBranch == "" {
		return nil
	}

	// Resolve to origin/ remote ref when available to ensure we merge the latest remote state,
	// not a potentially stale local branch.
	mergeRef := resolveToRemoteRef(svc, wt.Path, sourceBranch)

	targetBranch := getCustomField(state, "target_branch")
	commitMsg := fmt.Sprintf("Merge %s into %s", sourceBranch, targetBranch)
	err := svc.Git.MergeBranchWithCommit(wt.Path, mergeRef, commitMsg, false)
	if err != nil {
		return fmt.Errorf("failed to merge %s into worktree: %w", sourceBranch, err)
	}
	PrintSuccess(fmt.Sprintf("Merged '%s' into '%s'", sourceBranch, targetBranch))
	return nil
}

// resolveToRemoteRef returns "origin/<branch>" if the remote ref exists, otherwise the original branch name.
func resolveToRemoteRef(svc *Service, worktreePath, branch string) string {
	// Don't double-prefix if already a remote ref
	if strings.HasPrefix(branch, "origin/") || strings.HasPrefix(branch, "remotes/") {
		return branch
	}

	originRef := "origin/" + branch
	exists, err := svc.Git.BranchExistsInPath(worktreePath, originRef)
	if err == nil && exists {
		return originRef
	}
	return branch
}

// processMergeState populates WorktreeName, BranchName, and BaseBranch from merge custom fields.
func processMergeState(state *tui.WorkflowState) error {
	sourceBranch := getCustomField(state, "source_branch")
	targetBranch := getCustomField(state, "target_branch")

	if sourceBranch == "" || targetBranch == "" {
		return errors.New("source and target branches are required")
	}

	// Sanitize branch names for use in names
	sourceSanitized := sanitizeForName(sourceBranch)
	targetSanitized := sanitizeForName(targetBranch)

	state.WorktreeName = fmt.Sprintf("MERGE_%s-to-%s", sourceSanitized, targetSanitized)
	state.BranchName = fmt.Sprintf("merge/%s-to-%s", sourceSanitized, targetSanitized)
	state.BaseBranch = targetBranch

	return nil
}

func getCustomField(state *tui.WorkflowState, key string) string {
	if v := state.GetField(key); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func sanitizeForName(name string) string {
	return strings.ReplaceAll(name, "/", "_")
}
