package service

import (
	"fmt"
	"regexp"
	"strings"

	"gbm/internal/git"
	"gbm/internal/jira"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

// ErrGoBack is a special error that signals to go back to the previous screen
var ErrGoBack = fmt.Errorf("go back to previous screen")

// generateMergeCommitMessage creates a commit message for a merge
// If both branches are tracked in config, uses worktree names and descriptions
// Otherwise uses branch names
func generateMergeCommitMessage(svc *Service, sourceBranch, targetBranch string) string {
	config := svc.GetConfig()

	// Find config entries for source and target branches
	var sourceWorktreeName, sourceDescription string
	var targetWorktreeName, targetDescription string
	var sourceTracked, targetTracked bool

	for wtName, wtConfig := range config.Worktrees {
		if wtConfig.Branch == sourceBranch {
			sourceWorktreeName = wtName
			sourceDescription = wtConfig.Description
			sourceTracked = true
		}
		if wtConfig.Branch == targetBranch {
			targetWorktreeName = wtName
			targetDescription = wtConfig.Description
			targetTracked = true
		}
	}

	// If both are tracked, use worktree names and descriptions
	if sourceTracked && targetTracked {
		fromPart := sourceWorktreeName
		if sourceDescription != "" {
			fromPart = fmt.Sprintf("%s - %s", sourceWorktreeName, sourceDescription)
		}

		toPart := targetWorktreeName
		if targetDescription != "" {
			toPart = fmt.Sprintf("%s - %s", targetWorktreeName, targetDescription)
		}

		return fmt.Sprintf("merge: FROM [%s], TO [%s]", fromPart, toPart)
	}

	// Otherwise use branch names
	return fmt.Sprintf("merge: FROM [%s], TO [%s]", sourceBranch, targetBranch)
}

// runWorktreeAddTUI launches the interactive TUI workflow for creating worktrees
func runWorktreeAddTUI(cmd *cobra.Command, svc *Service) error {
	// Loop to allow going back from worktree type-specific flows to type selection
	for {
		var worktreeType string

		// Step 1: Ask what type of worktree to create
		typeForm := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("What type of worktree do you want to create?").
					Options(
						huh.NewOption("Feature", "feature"),
						huh.NewOption("Bug Fix", "bug"),
						huh.NewOption("Hotfix", "hotfix"),
						huh.NewOption("Mergeback", "mergeback"),
					).
					Value(&worktreeType),
			),
		)

		// Run the type selection wizard
		wizard := NewWizard([]WizardStep{{form: typeForm}})
		completed, cancelled, err := wizard.Run()
		if err != nil {
			return err
		}
		if cancelled {
			// User pressed Ctrl+C - cancel entirely
			return fmt.Errorf("cancelled")
		}
		if !completed {
			// User pressed ESC on first screen - cancel entirely
			return fmt.Errorf("cancelled")
		}

		// Step 2: Run type-specific flow
		var flowErr error
		switch worktreeType {
		case "feature":
			flowErr = createFeatureWorktree(svc, "feature")
		case "bug":
			flowErr = createFeatureWorktree(svc, "bug") // Same flow, different prefix
		case "hotfix":
			flowErr = createHotfixWorktree(svc)
		case "mergeback":
			flowErr = createMergebackWorktree(svc)
		default:
			return fmt.Errorf("unknown worktree type: %s", worktreeType)
		}

		// Check if user wants to go back to type selection
		if flowErr == ErrGoBack {
			continue // Loop back to step 1
		}

		// Any other error or success - return it
		return flowErr
	}
}

// Helper functions

// createSortedBranchItems creates sorted and labeled branch items for selection
// Sorting order: 1) Tracked by config, 2) Ad hoc worktree branches, 3) Other branches
func createSortedBranchItems(svc *Service) ([]FilterableItem, error) {
	// Get all branches
	branches, err := svc.Git.ListBranches(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	// Get config to see tracked worktrees
	config := svc.GetConfig()

	// Get worktrees to see which branches have worktrees
	worktrees, err := svc.Git.ListWorktrees(false)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Create maps for quick lookup
	branchToWorktreeName := make(map[string]string) // branch -> worktree name
	trackedBranches := make(map[string]bool)        // branches tracked in config

	// Build worktree lookup map
	for _, wt := range worktrees {
		if !wt.IsBare {
			branchToWorktreeName[wt.Branch] = wt.Name
		}
	}

	// Build tracked branches map
	for wtName, wtConfig := range config.Worktrees {
		trackedBranches[wtConfig.Branch] = true
		// Also track the worktree name -> branch mapping
		if _, exists := branchToWorktreeName[wtConfig.Branch]; !exists {
			branchToWorktreeName[wtConfig.Branch] = wtName
		}
	}

	// Categorize branches
	var trackedItems []FilterableItem
	var adHocItems []FilterableItem
	var otherItems []FilterableItem

	for _, branch := range branches {
		wtName, hasWorktree := branchToWorktreeName[branch]
		isTracked := trackedBranches[branch]

		var label string
		if isTracked && hasWorktree {
			label = fmt.Sprintf("%s (tracked: %s)", branch, wtName)
		} else if isTracked {
			label = fmt.Sprintf("%s (tracked, no worktree)", branch)
		} else if hasWorktree {
			label = fmt.Sprintf("%s (worktree: %s)", branch, wtName)
		} else {
			label = branch
		}

		item := FilterableItem{
			Label: label,
			Value: branch,
		}

		if isTracked {
			trackedItems = append(trackedItems, item)
		} else if hasWorktree {
			adHocItems = append(adHocItems, item)
		} else {
			otherItems = append(otherItems, item)
		}
	}

	// Combine in priority order
	result := make([]FilterableItem, 0, len(branches))
	result = append(result, trackedItems...)
	result = append(result, adHocItems...)
	result = append(result, otherItems...)

	return result, nil
}

// validateWorktreeName ensures name is valid directory name
func validateWorktreeName(name string) error {
	if name == "" {
		return fmt.Errorf("worktree name cannot be empty")
	}
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("invalid characters in worktree name")
	}
	return nil
}

// createWorktreeNameValidator creates a validator that checks both format and existence
func createWorktreeNameValidator(svc *Service) func(string) error {
	return func(name string) error {
		// First check basic validation
		if err := validateWorktreeName(name); err != nil {
			return err
		}

		// Check if worktree already exists
		worktrees, err := svc.Git.ListWorktrees(false)
		if err != nil {
			// Don't fail validation on list error, just skip existence check
			return nil
		}

		for _, wt := range worktrees {
			if wt.Name == name {
				return fmt.Errorf("worktree '%s' already exists", name)
			}
		}

		return nil
	}
}

// validateBranchName ensures branch name is valid
func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}
	// Git branch name rules
	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".") {
		return fmt.Errorf("invalid branch name format")
	}
	return nil
}

// createBranchNameValidator creates a validator that checks both format and existence
func createBranchNameValidator(svc *Service) func(string) error {
	return func(name string) error {
		// First check basic validation
		if err := validateBranchName(name); err != nil {
			return err
		}

		// Check if branch already exists
		exists, err := svc.Git.BranchExists(name)
		if err != nil {
			// Don't fail validation on check error, just skip existence check
			return nil
		}

		if exists {
			return fmt.Errorf("branch '%s' already exists", name)
		}

		return nil
	}
}

// filterWorktreesByPrefix filters worktrees by name prefix
func filterWorktreesByPrefix(wts []git.Worktree, prefix string) []git.Worktree {
	result := []git.Worktree{}
	for _, wt := range wts {
		if strings.HasPrefix(wt.Name, prefix) {
			result = append(result, wt)
		}
	}
	return result
}

// sanitizeBranchName converts branch name to valid directory name
func sanitizeBranchName(branch string) string {
	// Remove common prefixes
	branch = strings.TrimPrefix(branch, "origin/")
	branch = strings.TrimPrefix(branch, "refs/heads/")
	// Replace invalid directory chars
	replacer := strings.NewReplacer("/", "-", "\\", "-")
	return replacer.Replace(branch)
}

// Flow implementations

func createFeatureWorktree(svc *Service, prefix string) error {
	// Fetch JIRA tickets
	issues, err := svc.Jira.GetJiraIssues(false)

	items := []FilterableItem{}
	if err == nil && len(issues) > 0 {
		// Add JIRA tickets as filterable items
		for _, issue := range issues {
			items = append(items, FilterableItem{
				Label: fmt.Sprintf("%s: %s", issue.Key, issue.Summary),
				Value: issue.Key,
			})
		}
	}

	// Variables to collect across wizard steps
	var worktreeName string
	var branchName string
	var baseBranch string

	// Outer loop: Allows going back to screen 1 (worktree name)
	for {
		// Step 1: Worktree name (using filterable select)
		title := "Worktree name"
		description := "Select a JIRA ticket or enter a custom worktree name"
		filterSelect := NewFilterableSelect(title, description, items)

		step1Wizard := NewWizard([]WizardStep{
			{customModel: filterSelect, isCustom: true},
		})

		completed, cancelled, err := step1Wizard.Run()
		if err != nil {
			return err
		}
		if cancelled {
			// User pressed Ctrl+C - cancel entirely
			return fmt.Errorf("cancelled")
		}
		if !completed {
			// User pressed ESC on worktree name - go back to type selection
			return ErrGoBack
		}

		// Extract worktree name
		filterModel := step1Wizard.Steps[0].customModel.(FilterableSelectModel)
		worktreeName = filterModel.GetSelected()

		// Validate worktree name
		if err := createWorktreeNameValidator(svc)(worktreeName); err != nil {
			// Show error and loop back to worktree name selection
			errorForm := huh.NewForm(
				huh.NewGroup(
					huh.NewNote().
						Title("Validation Error").
						Description(err.Error()),
				),
			)
			errorWizard := NewWizard([]WizardStep{{form: errorForm}})
			_, _, _ = errorWizard.Run()
			continue
		}

		// Determine if this is a JIRA key and set branch name default
		var selectedIssue *jira.JiraIssue
		for i, issue := range issues {
			if issue.Key == worktreeName {
				selectedIssue = &issues[i]
				break
			}
		}

		// Set branch name default based on selection
		if selectedIssue != nil {
			sanitizedSummary := sanitizeSummaryForBranch(selectedIssue.Summary)
			branchName = fmt.Sprintf("%s/%s-%s", prefix, selectedIssue.Key, sanitizedSummary)
		} else {
			branchName = prefix + "/"
		}

		// Middle loop: Allows going back to screen 2 (branch name)
		for {
			// Step 2: Branch name
			branchForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Branch name").
						Value(&branchName).
						Validate(createBranchNameValidator(svc)).
						Description("Edit if needed"),
				),
			)

			branchWizard := NewWizard([]WizardStep{{form: branchForm}})
			completed, cancelled, err = branchWizard.Run()
			if err != nil {
				return err
			}
			if cancelled {
				// User pressed Ctrl+C - cancel entirely
				return fmt.Errorf("cancelled")
			}
			if !completed {
				// User pressed ESC on branch name - go back to step 1
				break
			}

			// Try to create the worktree without creating branch (checkout existing)
			worktreesDir, err := svc.GetWorktreesPath()
			if err != nil {
				return fmt.Errorf("failed to get worktrees directory: %w", err)
			}

			wt, err := svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, false, "", false)
			if err == nil {
				// Success! Branch existed, worktree created
				if err := svc.CopyFilesToWorktree(worktreeName); err != nil {
					fmt.Printf("Warning: failed to copy files to worktree: %v\n", err)
				}

				fmt.Printf("\n✓ Worktree created successfully!\n")
				fmt.Printf("  Name:   %s\n", wt.Name)
				fmt.Printf("  Path:   %s\n", wt.Path)
				fmt.Printf("  Branch: %s\n", wt.Branch)
				fmt.Printf("  Commit: %s\n", wt.Commit)
				return nil
			}

			// Check if it's a "branch doesn't exist" error
			errMsg := err.Error()
			isBranchNotExist := strings.Contains(errMsg, "does not exist") || strings.Contains(errMsg, "invalid reference")
			if !isBranchNotExist {
				// Some other error - return it
				return err
			}

			// Branch doesn't exist - CONSOLIDATED WIZARD for screens 3-4
			baseBranch = "" // Reset base branch
			for {
				var createNewBranch bool

				// Get sorted and labeled branch items
				branchItems, err := createSortedBranchItems(svc)
				if err != nil {
					return fmt.Errorf("failed to get branches: %w", err)
				}

				// Create filterable select for base branch
				baseBranchTitle := "Base branch"
				baseBranchDesc := fmt.Sprintf("Branch '%s' does not exist. Select base branch to create from", branchName)
				baseBranchSelect := NewFilterableSelect(baseBranchTitle, baseBranchDesc, branchItems)

				confirmForm := huh.NewForm(
					huh.NewGroup(
						huh.NewConfirm().
							Title(fmt.Sprintf("Create branch '%s'?", branchName)).
							Description("Confirm creation of new branch from the specified base").
							Value(&createNewBranch),
					),
				)

				// Consolidated wizard combining base branch + confirmation
				wizard34 := NewWizard([]WizardStep{
					{customModel: baseBranchSelect, isCustom: true},
					{form: confirmForm},
				})

				completed, cancelled, err = wizard34.Run()
				if err != nil {
					return err
				}
				if cancelled {
					// User pressed Ctrl+C - cancel entirely
					return fmt.Errorf("cancelled")
				}
				if !completed {
					// User pressed ESC on first screen - go back to screen 2 (branch name)
					break
				}

				// Extract selected base branch from the filterable select
				baseBranchModel := wizard34.Steps[0].customModel.(FilterableSelectModel)
				baseBranch = baseBranchModel.GetSelected()

				if !createNewBranch {
					// User said No - exit entirely
					return fmt.Errorf("cancelled")
				}

				// Validate baseBranch was provided
				if baseBranch == "" {
					baseBranch = "main" // Default to main if not provided
				}

				// Create worktree with new branch
				wt, err = svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, true, baseBranch, false)
				if err != nil {
					return fmt.Errorf("failed to add worktree: %w", err)
				}

				// Copy files
				if err := svc.CopyFilesToWorktree(worktreeName); err != nil {
					fmt.Printf("Warning: failed to copy files to worktree: %v\n", err)
				}

				// Display success
				fmt.Printf("\n✓ Worktree created successfully!\n")
				fmt.Printf("  Name:   %s\n", wt.Name)
				fmt.Printf("  Path:   %s\n", wt.Path)
				fmt.Printf("  Branch: %s\n", wt.Branch)
				fmt.Printf("  Commit: %s\n", wt.Commit)
				return nil
			}
		}
	}
}

func collectBranchNameWithDefault(prefix string, issue *jira.JiraIssue) (string, error) {
	// Generate branch name: {prefix}/{KEY}-{sanitized-summary}
	// E.g., "feature/ABC-123-fix-the-bug-in-login"
	sanitizedSummary := sanitizeSummaryForBranch(issue.Summary)
	defaultBranch := fmt.Sprintf("%s/%s-%s", prefix, issue.Key, sanitizedSummary)

	branchInput := defaultBranch

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Branch name").
				Value(&branchInput).
				Validate(validateBranchName).
				Description("Edit if needed"),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return branchInput, nil
}

func collectBranchNameCustom(prefix string) (string, error) {
	// Start with just the prefix, let user complete it
	branchInput := prefix + "/"

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Branch name").
				Value(&branchInput).
				Validate(validateBranchName).
				Placeholder(fmt.Sprintf("%s/my-branch", prefix)),
		),
	)

	if err := form.Run(); err != nil {
		return "", err
	}

	return branchInput, nil
}

func sanitizeSummaryForBranch(summary string) string {
	// Convert summary to lowercase
	s := strings.ToLower(summary)

	// Replace spaces and special chars with hyphens
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")

	// Remove leading/trailing hyphens
	s = strings.Trim(s, "-")

	// Limit length to reasonable size
	if len(s) > 50 {
		s = s[:50]
	}

	// Remove trailing hyphen if we truncated mid-word
	s = strings.TrimRight(s, "-")

	return s
}

func executeWorktreeCreation(svc *Service, worktreeName, branchName string, createBranch bool, baseBranch string) error {
	// Get worktrees directory
	worktreesDir, err := svc.GetWorktreesPath()
	if err != nil {
		return fmt.Errorf("failed to get worktrees directory: %w", err)
	}

	// Get dry-run flag value (if set)
	dryRun := false // TODO: Get this from cobra command context if needed

	// Create the worktree
	wt, err := svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, createBranch, baseBranch, dryRun)
	if err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	// Copy files to worktree
	if err := svc.CopyFilesToWorktree(worktreeName); err != nil {
		fmt.Printf("Warning: failed to copy files to worktree: %v\n", err)
	}

	// Display success message
	fmt.Printf("\n✓ Worktree created successfully!\n")
	fmt.Printf("  Name:   %s\n", wt.Name)
	fmt.Printf("  Path:   %s\n", wt.Path)
	fmt.Printf("  Branch: %s\n", wt.Branch)
	fmt.Printf("  Commit: %s\n", wt.Commit)

	return nil
}

func createHotfixWorktree(svc *Service) error {
	// Fetch JIRA tickets
	issues, err := svc.Jira.GetJiraIssues(false)

	items := []FilterableItem{}
	if err == nil && len(issues) > 0 {
		// Add JIRA tickets as filterable items
		for _, issue := range issues {
			items = append(items, FilterableItem{
				Label: fmt.Sprintf("%s: %s", issue.Key, issue.Summary),
				Value: issue.Key,
			})
		}
	}

	// Step 1: Ask for worktree name (can select JIRA ticket or enter custom)
	title := "Worktree name"
	description := "Select a JIRA ticket or enter a custom worktree name"

	filterSelect := NewFilterableSelect(title, description, items)
	step1Wizard := NewWizard([]WizardStep{
		{customModel: filterSelect, isCustom: true},
	})

	completed, cancelled, err := step1Wizard.Run()
	if err != nil {
		return err
	}
	if cancelled {
		// User pressed Ctrl+C - cancel entirely
		return fmt.Errorf("cancelled")
	}
	if !completed {
		// User pressed ESC on worktree name - go back to type selection
		return ErrGoBack
	}

	// Extract worktree name
	filterModel := step1Wizard.Steps[0].customModel.(FilterableSelectModel)
	selectedName := filterModel.GetSelected()

	// Step 2: Determine if this is a JIRA key and get the corresponding issue
	var selectedIssue *jira.JiraIssue
	for i, issue := range issues {
		if issue.Key == selectedName {
			selectedIssue = &issues[i]
			break
		}
	}

	// Add HOTFIX_ prefix to avoid naming collisions
	worktreeName := "HOTFIX_" + selectedName

	// Validate that the prefixed worktree name doesn't already exist
	if err := createWorktreeNameValidator(svc)(worktreeName); err != nil {
		return fmt.Errorf("worktree validation failed: %w", err)
	}

	// Set branch name default based on selection
	var branchName string
	if selectedIssue != nil {
		sanitizedSummary := sanitizeSummaryForBranch(selectedIssue.Summary)
		branchName = fmt.Sprintf("hotfix/%s-%s", selectedIssue.Key, sanitizedSummary)
	} else {
		branchName = "hotfix/"
	}

	// CONSOLIDATED WIZARD: Steps 3-4 (Base branch + Branch name)
	var baseBranch string

	// Get base branch options using filterable select
	baseBranchSelect, err := createHotfixBaseBranchFilterableSelect(svc)
	if err != nil {
		return err
	}

	branchForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Branch name").
				Value(&branchName).
				Validate(createBranchNameValidator(svc)).
				Description("Edit if needed"),
		),
	)

	// Create consolidated wizard with base branch selection + branch name
	wizard34 := NewWizard([]WizardStep{
		{customModel: baseBranchSelect, isCustom: true},
		{form: branchForm},
	})

	completed, cancelled, err = wizard34.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return fmt.Errorf("cancelled")
	}
	if !completed {
		// User pressed ESC - go back to type selection
		return ErrGoBack
	}

	// Extract selected base branch from the filterable select
	baseBranchModel := wizard34.Steps[0].customModel.(FilterableSelectModel)
	baseBranch = baseBranchModel.GetSelected()

	// Create the worktree (hotfixes always create new branches)
	return executeWorktreeCreation(svc, worktreeName, branchName, true, baseBranch)
}

// createHotfixBaseBranchFilterableSelect creates a filterable select for hotfix base branch
func createHotfixBaseBranchFilterableSelect(svc *Service) (FilterableSelectModel, error) {
	// Get sorted and labeled branch items (with config/worktree associations)
	items, err := createSortedBranchItems(svc)
	if err != nil {
		return FilterableSelectModel{}, err
	}

	title := "Base branch for hotfix"
	description := "Select the branch to base the hotfix on (typically production or release)"

	return NewFilterableSelect(title, description, items), nil
}

func createMergebackWorktree(svc *Service) error {
	// Variables to collect
	var sourceBranch string
	var targetBranch string
	var worktreeName string
	var confirm bool

	// WIZARD 1: Steps 1-2 (Source branch + Target branch)
	sourceBranchForm, err := createSourceBranchSelect(svc, &sourceBranch)
	if err != nil {
		return err
	}

	// Create wizard with just source branch to get the selection first
	wizard1 := NewWizard([]WizardStep{{form: sourceBranchForm}})
	completed, cancelled, err := wizard1.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return fmt.Errorf("cancelled")
	}
	if !completed {
		// User pressed ESC - go back to type selection
		return ErrGoBack
	}

	// Now we know sourceBranch, create target branch form
	targetBranchForm, err := createTargetBranchSelect(svc, sourceBranch, &targetBranch)
	if err != nil {
		return err
	}

	wizard2 := NewWizard([]WizardStep{{form: targetBranchForm}})
	completed, cancelled, err = wizard2.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return fmt.Errorf("cancelled")
	}
	if !completed {
		// User pressed ESC - would need to go back to source branch
		// For simplicity, go back to type selection
		return ErrGoBack
	}

	// WIZARD 3: Worktree name
	// Auto-suggest: "MERGE_<source>-to-<target>"
	worktreeName = fmt.Sprintf("MERGE_%s-to-%s",
		sanitizeBranchName(sourceBranch),
		sanitizeBranchName(targetBranch))

	worktreeNameForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Worktree name").
				Value(&worktreeName).
				Validate(createWorktreeNameValidator(svc)).
				Description("Edit if needed"),
		),
	)

	wizard3 := NewWizard([]WizardStep{{form: worktreeNameForm}})
	completed, cancelled, err = wizard3.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return fmt.Errorf("cancelled")
	}
	if !completed {
		// User pressed ESC - go back to type selection
		return ErrGoBack
	}

	// WIZARD 4: Branch name
	var branchName string
	// Auto-suggest branch name with format: merge/<source>-to-<target>
	branchName = fmt.Sprintf("merge/%s-to-%s",
		sanitizeBranchName(sourceBranch),
		sanitizeBranchName(targetBranch))

	branchNameForm := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Branch name").
				Value(&branchName).
				Validate(createBranchNameValidator(svc)).
				Description("Edit if needed"),
		),
	)

	wizard4 := NewWizard([]WizardStep{{form: branchNameForm}})
	completed, cancelled, err = wizard4.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return fmt.Errorf("cancelled")
	}
	if !completed {
		// User pressed ESC - go back to type selection
		return ErrGoBack
	}

	// WIZARD 5: Final confirmation
	confirmForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Create mergeback worktree?").
				Description(fmt.Sprintf(
					"Source: %s → Target: %s\nWorktree: %s\nBranch: %s",
					sourceBranch, targetBranch, worktreeName, branchName,
				)).
				Value(&confirm),
		),
	)

	wizard5 := NewWizard([]WizardStep{{form: confirmForm}})
	completed, cancelled, err = wizard5.Run()
	if err != nil {
		return err
	}
	if cancelled {
		return fmt.Errorf("cancelled")
	}
	if !completed {
		// User pressed ESC - go back to type selection
		return ErrGoBack
	}

	if !confirm {
		return fmt.Errorf("mergeback cancelled")
	}

	// Create worktree with new branch based on target branch
	worktreesDir, err := svc.GetWorktreesPath()
	if err != nil {
		return fmt.Errorf("failed to get worktrees directory: %w", err)
	}

	wt, err := svc.Git.AddWorktree(worktreesDir, worktreeName, branchName, true, targetBranch, false)
	if err != nil {
		return fmt.Errorf("failed to add worktree: %w", err)
	}

	// Initiate merge with commit message
	fmt.Printf("\n✓ Worktree created successfully!\n")
	fmt.Printf("  Name:   %s\n", wt.Name)
	fmt.Printf("  Path:   %s\n", wt.Path)
	fmt.Printf("  Branch: %s\n", wt.Branch)
	fmt.Printf("\nInitiating merge from %s...\n", sourceBranch)

	// Generate commit message based on config
	commitMessage := generateMergeCommitMessage(svc, sourceBranch, targetBranch)

	err = svc.Git.MergeBranchWithCommit(wt.Path, sourceBranch, commitMessage, false)
	if err != nil {
		fmt.Printf("\n⚠ Worktree created, but merge failed: %v\n", err)
		fmt.Printf("You can manually run the merge in the worktree:\n")
		fmt.Printf("  cd %s\n", wt.Path)
		fmt.Printf("  git merge %s\n", sourceBranch)
	} else {
		fmt.Printf("✓ Merge completed successfully!\n")
		fmt.Printf("  Commit message: %s\n", commitMessage)
		fmt.Printf("\nWorktree ready at:\n")
		fmt.Printf("  cd %s\n", wt.Path)
	}

	return nil
}

func createSourceBranchSelect(svc *Service, sourceBranch *string) (*huh.Form, error) {
	// Get sorted and labeled branch items
	branchItems, err := createSortedBranchItems(svc)
	if err != nil {
		return nil, err
	}

	// Convert to huh options
	options := make([]huh.Option[string], len(branchItems))
	for i, item := range branchItems {
		options[i] = huh.NewOption(item.Label, item.Value)
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select source branch (merge FROM)").
				Description("The branch containing changes to merge").
				Options(options...).
				Value(sourceBranch).
				Height(10),
		),
	)

	return form, nil
}

func createTargetBranchSelect(svc *Service, sourceBranch string, targetBranch *string) (*huh.Form, error) {
	// Get sorted and labeled branch items
	branchItems, err := createSortedBranchItems(svc)
	if err != nil {
		return nil, err
	}

	// Check if source worktree has merge_into configured
	suggestedTarget := ""
	config := svc.GetConfig()
	for _, wtConfig := range config.Worktrees {
		if wtConfig.Branch == sourceBranch && wtConfig.MergeInto != "" {
			suggestedTarget = wtConfig.MergeInto
			break
		}
	}

	// Filter out source branch and build options
	options := []huh.Option[string]{}

	// If we found a suggestion, put it first
	if suggestedTarget != "" {
		// Find the label for the suggested target
		var suggestedLabel string
		for _, item := range branchItems {
			if item.Value == suggestedTarget {
				suggestedLabel = item.Label
				break
			}
		}
		if suggestedLabel == "" {
			suggestedLabel = suggestedTarget
		}
		options = append(options, huh.NewOption(
			fmt.Sprintf("%s (suggested from config)", suggestedLabel),
			suggestedTarget,
		))
	}

	// Add other branches (excluding source and already-added suggestion)
	for _, item := range branchItems {
		if item.Value != sourceBranch && item.Value != suggestedTarget {
			options = append(options, huh.NewOption(item.Label, item.Value))
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select target branch (merge INTO)").
				Description("The branch that will receive the changes").
				Options(options...).
				Value(targetBranch).
				Height(10),
		),
	)

	return form, nil
}

func selectMergebackWorktreeName(sourceBranch, targetBranch string, worktreeName *string) error {
	// Auto-suggest: "MERGE_<source>-to-<target>"
	suggested := fmt.Sprintf("MERGE_%s-to-%s",
		sanitizeBranchName(sourceBranch),
		sanitizeBranchName(targetBranch))

	wtName := suggested

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Worktree name").
				Value(&wtName).
				Validate(validateWorktreeName).
				Description("Edit if needed"),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	*worktreeName = wtName
	return nil
}
