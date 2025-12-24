package service

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

// StepModel interface for custom step implementations
type StepModel interface {
	tea.Model
	IsComplete() bool
	IsCancelled() bool
}

// WizardStep represents a single step in the wizard
type WizardStep struct {
	form        *huh.Form         // For huh forms
	customModel StepModel         // For custom Bubble Tea models
	isCustom    bool              // Track which type this is
}

// WizardModel manages a multi-step wizard with back navigation
type WizardModel struct {
	steps       []WizardStep
	currentStep int
	cancelled   bool
	completed   bool
	width       int
	height      int
}

// NewWizard creates a new wizard with the given steps
func NewWizard(steps []WizardStep) WizardModel {
	return WizardModel{
		steps:       steps,
		currentStep: 0,
		width:       80,
		height:      24,
	}
}

func (m WizardModel) Init() tea.Cmd {
	if len(m.steps) > 0 {
		step := m.steps[m.currentStep]
		if step.isCustom {
			return step.customModel.Init()
		}
		return step.form.Init()
	}
	return nil
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.currentStep >= len(m.steps) {
		return m, nil
	}

	step := &m.steps[m.currentStep]

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("esc"))):
			// ESC goes back to previous step
			if m.currentStep > 0 {
				m.currentStep--
				prevStep := m.steps[m.currentStep]
				if prevStep.isCustom {
					return m, prevStep.customModel.Init()
				}
				return m, prevStep.form.Init()
			}
			// On first step, ESC quits without setting cancelled
			// This signals "go back to previous screen" (not "cancel entirely")
			return m, tea.Quit

		case key.Matches(msg, key.NewBinding(key.WithKeys("ctrl+c"))):
			// Ctrl+C sets cancelled flag to signal "cancel entirely"
			m.cancelled = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	// Update current step (either huh form or custom model)
	var cmd tea.Cmd
	if step.isCustom {
		// Update custom model
		model, c := step.customModel.Update(msg)
		if sm, ok := model.(StepModel); ok {
			step.customModel = sm
			cmd = c

			// Check if complete or cancelled
			if sm.IsCancelled() {
				m.cancelled = true
				return m, tea.Quit
			}
			if sm.IsComplete() {
				// Move to next step
				m.currentStep++
				if m.currentStep >= len(m.steps) {
					m.completed = true
					return m, tea.Quit
				}
				nextStep := m.steps[m.currentStep]
				if nextStep.isCustom {
					return m, nextStep.customModel.Init()
				}
				return m, nextStep.form.Init()
			}
		}
	} else {
		// Update huh form
		form, c := step.form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			step.form = f
			cmd = c

			// Check if complete
			if f.State == huh.StateCompleted {
				// Move to next step
				m.currentStep++
				if m.currentStep >= len(m.steps) {
					m.completed = true
					return m, tea.Quit
				}
				nextStep := m.steps[m.currentStep]
				if nextStep.isCustom {
					return m, nextStep.customModel.Init()
				}
				return m, nextStep.form.Init()
			}
		}
	}

	return m, cmd
}

func (m WizardModel) View() string {
	if m.currentStep < len(m.steps) {
		step := m.steps[m.currentStep]

		var view string
		if step.isCustom {
			view = step.customModel.View()
		} else {
			view = step.form.View()
		}

		// Add navigation hint at bottom
		hint := "\n\n"
		if m.currentStep > 0 {
			hint += "esc: back • "
		}
		hint += "ctrl+c: cancel"

		return view + hint
	}
	return ""
}

// Run executes the wizard and returns (finalModel, error)
// Returns nil error if user completed all steps
// Returns ErrCancelled if user pressed Ctrl+C
// Returns ErrGoBack if user pressed ESC on first step to go back
func (m WizardModel) Run() (*WizardModel, error) {
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, runErr := p.Run()
	if runErr != nil {
		return nil, runErr
	}

	if model, ok := finalModel.(WizardModel); ok {
		// Check completion state
		if model.cancelled {
			return nil, ErrCancelled
		}
		if !model.completed {
			// User pressed ESC on first step - go back
			return nil, ErrGoBack
		}
		// Success - return the wizard with user selections
		return &model, nil
	}

	return nil, fmt.Errorf("unexpected model type: %T", finalModel)
}

// GetStep returns the step at the given index
func (m *WizardModel) GetStep(index int) (WizardStep, error) {
	if index < 0 || index >= len(m.steps) {
		return WizardStep{}, fmt.Errorf("step index %d out of range (0-%d)", index, len(m.steps)-1)
	}
	return m.steps[index], nil
}
