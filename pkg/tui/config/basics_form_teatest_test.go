package config

import (
	"gbm/pkg/tui"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

func initTestProfile() {
	lipgloss.SetColorProfile(termenv.Ascii)
}

// basicsFormModel wraps BasicsForm for teatest, quitting on boundary messages.
type basicsFormModel struct {
	form *BasicsForm
}

func newBasicsFormModel(f *BasicsForm) *basicsFormModel {
	return &basicsFormModel{form: f}
}

func (m *basicsFormModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m *basicsFormModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tui.BackBoundaryMsg, tui.FormFlushCompleteMsg:
		return m, tea.Quit
	}

	model, cmd := m.form.Update(msg)
	if f, ok := model.(*BasicsForm); ok {
		m.form = f
	}
	return m, cmd
}

func (m *basicsFormModel) View() string {
	return m.form.View()
}

// TestBasicsForm_SaveFlow tests the save flow.
func TestBasicsForm_SaveFlow(t *testing.T) {
	t.Parallel()
	initTestProfile()

	saveCalled := false
	savedData := map[string]string{}

	config := BasicsFormConfig{
		DefaultBranch: "main",
		WorktreesDir:  "./worktrees",
		Theme:         tui.DefaultTheme(),
		OnSave: func(data map[string]string) error {
			saveCalled = true
			savedData = data

			return nil
		},
	}

	form := NewBasicsForm(config)
	model := newBasicsFormModel(form)

	tm := teatest.NewTestModel(t, model, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return len(bts) > 0
	}, teatest.WithDuration(time.Second))

	// Send 's' key to save
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Wait for quit
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))

	// Verify save was called
	assert.True(t, saveCalled, "OnSave callback should have been called")
	assert.NotEmpty(t, savedData, "SaveData should not be empty")
}
