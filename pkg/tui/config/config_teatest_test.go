package config

import (
	"bytes"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Set ASCII color profile for consistent test output across environments.
	lipgloss.SetColorProfile(termenv.Ascii)
}

// waitFor polls until condition returns true or timeout is reached.
func waitFor(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Fatal("waitFor: condition not met within timeout")
}

// newTestConfigModel creates a ConfigModel suitable for teatest with proper dimensions.
func newTestConfigModel() *ConfigModel {
	m := NewConfigModel()
	m.width = 80
	m.height = 24
	return m
}

// --- ConfigModel teatest integration tests ---.

func TestTeatest_ConfigModel_InitialRender(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render showing the tab bar with General active.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// The initial view should contain all four tab labels.
	view := m.View()
	assert.Contains(t, view, "General")
	assert.Contains(t, view, "JIRA")
	assert.Contains(t, view, "File Copy")
	assert.Contains(t, view, "Worktrees")

	// General should be the active tab (bracketed).
	assert.Contains(t, view, "[General]")
	assert.Equal(t, TabGeneral, m.ActiveTab())

	// State should be browsing.
	assert.Equal(t, StateBrowsing, m.State())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_TabSwitching(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("[General]"))
	}, teatest.WithDuration(time.Second))

	// Press tab to switch to JIRA.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabJira }, time.Second)
	assert.Equal(t, TabJira, m.ActiveTab())

	// Press tab to switch to File Copy.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabFileCopy }, time.Second)
	assert.Equal(t, TabFileCopy, m.ActiveTab())

	// Press tab to switch to Worktrees.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabWorktrees }, time.Second)
	assert.Equal(t, TabWorktrees, m.ActiveTab())

	// Press tab to wrap back to General.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabGeneral }, time.Second)
	assert.Equal(t, TabGeneral, m.ActiveTab())

	// Press shift-tab to go back to Worktrees.
	tm.Send(tea.KeyMsg{Type: tea.KeyShiftTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabWorktrees }, time.Second)
	assert.Equal(t, TabWorktrees, m.ActiveTab())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_HelpOverlayToggle(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	assert.Equal(t, StateBrowsing, m.State())

	// Press ? to open help overlay.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	waitFor(t, func() bool { return m.State() == StateHelp }, time.Second)
	assert.Equal(t, StateHelp, m.State())

	// Press esc to close help overlay.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)
	assert.Equal(t, StateBrowsing, m.State())

	// Open again with ? and close with ? (toggle).
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	waitFor(t, func() bool { return m.State() == StateHelp }, time.Second)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)
	assert.Equal(t, StateBrowsing, m.State())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_HelpOverlayScrolling(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Open help overlay.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	waitFor(t, func() bool { return m.State() == StateHelp }, time.Second)

	initialScroll := m.helpOverlay.Scroll()

	// Press j to scroll down.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	waitFor(t, func() bool { return m.helpOverlay.Scroll() > initialScroll }, time.Second)
	assert.Greater(t, m.helpOverlay.Scroll(), initialScroll)

	// Press k to scroll up.
	scrollAfterDown := m.helpOverlay.Scroll()
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	waitFor(t, func() bool { return m.helpOverlay.Scroll() < scrollAfterDown }, time.Second)
	assert.Less(t, m.helpOverlay.Scroll(), scrollAfterDown)

	// Tab switching should NOT work while in help state.
	currentTab := m.ActiveTab()
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	// Brief wait -- tab should not change.
	time.Sleep(5 * time.Millisecond)
	assert.Equal(t, currentTab, m.ActiveTab(), "tab should not change while help is open")

	// Close help.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_QuitWithQ(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Press q to quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_QuitWithCtrlC(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Press ctrl+c to quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_WindowResize(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Resize to too small -- should show "Terminal too small".
	tm.Send(tea.WindowSizeMsg{Width: 40, Height: 10})
	waitFor(t, func() bool { return m.Width() == 40 && m.Height() == 10 }, time.Second)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Terminal too small"))
	}, teatest.WithDuration(time.Second))

	// Resize back to normal.
	tm.Send(tea.WindowSizeMsg{Width: 80, Height: 24})
	waitFor(t, func() bool { return m.Width() == 80 && m.Height() == 24 }, time.Second)

	// Should show normal content again.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_StatusBarContent(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Status bar should show keybinding hints (tab, help).
	view := m.View()
	assert.Contains(t, view, "tab")
	assert.Contains(t, view, "help")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_EditingBlocksTabs(t *testing.T) {
	m := newTestConfigModel()
	m.state = StateEditing

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for any render.
	time.Sleep(10 * time.Millisecond)

	// Press tab -- should not switch tabs in editing state.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	time.Sleep(5 * time.Millisecond)
	assert.Equal(t, TabGeneral, m.ActiveTab(), "tab should not switch during editing")

	// Press esc to cancel editing and return to browsing.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)
	assert.Equal(t, StateBrowsing, m.State())

	// Now tab should work.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabJira }, time.Second)
	assert.Equal(t, TabJira, m.ActiveTab())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_ErrorOverlayWorkflow(t *testing.T) {
	m := newTestConfigModel()
	m.ShowErrorOverlay(sampleErrors())

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for error overlay render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Validation Errors"))
	}, teatest.WithDuration(time.Second))

	assert.Equal(t, StateErrors, m.State())

	// Navigate down in the error list.
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return m.errorOverlay.Cursor() == 1 }, time.Second)
	assert.Equal(t, 1, m.errorOverlay.Cursor())

	// Navigate to the JIRA error.
	tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	waitFor(t, func() bool { return m.errorOverlay.Cursor() == 2 }, time.Second)

	// Press enter to jump to the JIRA tab.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)
	assert.Equal(t, StateBrowsing, m.State())
	assert.Equal(t, TabJira, m.ActiveTab())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_ErrorOverlayEscCloses(t *testing.T) {
	m := newTestConfigModel()
	m.ShowErrorOverlay(sampleErrors())

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for error overlay render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Validation Errors"))
	}, teatest.WithDuration(time.Second))

	assert.Equal(t, StateErrors, m.State())

	// Press esc to close.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)
	assert.Equal(t, StateBrowsing, m.State())

	// Tab should still be General (esc doesn't jump).
	assert.Equal(t, TabGeneral, m.ActiveTab())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_FlashMessageClearsAutomatically(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Set a flash message by sending the internal message directly.
	// We simulate what SetFlash does by setting the message and sending flashClearMsg.
	m.flashMessage = "saved config"
	assert.Equal(t, "saved config", m.flashMessage)

	// Send the clear message to simulate timer expiry.
	tm.Send(flashClearMsg{})
	waitFor(t, func() bool { return m.flashMessage == "" }, time.Second)
	assert.Empty(t, m.flashMessage)

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_TabContentUpdatesOnSwitch(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for General tab content.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General section content"))
	}, teatest.WithDuration(time.Second))

	// Switch to JIRA tab.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabJira }, time.Second)

	// Wait for JIRA content to appear.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("JIRA section content"))
	}, teatest.WithDuration(time.Second))

	// Switch to File Copy tab.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabFileCopy }, time.Second)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("File Copy section content"))
	}, teatest.WithDuration(time.Second))

	// Switch to Worktrees tab.
	tm.Send(tea.KeyMsg{Type: tea.KeyTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabWorktrees }, time.Second)

	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("Worktrees section content"))
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_FullTabCycleWithShiftTab(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Shift-tab from General wraps to Worktrees.
	tm.Send(tea.KeyMsg{Type: tea.KeyShiftTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabWorktrees }, time.Second)

	// Shift-tab from Worktrees to File Copy.
	tm.Send(tea.KeyMsg{Type: tea.KeyShiftTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabFileCopy }, time.Second)

	// Shift-tab from File Copy to JIRA.
	tm.Send(tea.KeyMsg{Type: tea.KeyShiftTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabJira }, time.Second)

	// Shift-tab from JIRA to General.
	tm.Send(tea.KeyMsg{Type: tea.KeyShiftTab})
	waitFor(t, func() bool { return m.ActiveTab() == TabGeneral }, time.Second)

	assert.Equal(t, TabGeneral, m.ActiveTab())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_ErrorBadgesVisibleAfterErrors(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for initial render.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("General"))
	}, teatest.WithDuration(time.Second))

	// Show error overlay.
	m.ShowErrorOverlay(sampleErrors())
	assert.Equal(t, StateErrors, m.State())

	// Close the overlay.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)

	// Tab badges should still be visible in the tab bar.
	view := m.View()
	assert.Contains(t, view, "General (!)")
	assert.Contains(t, view, "JIRA (!)")

	// Clear validation errors.
	m.ClearValidationErrors()
	view = m.View()
	assert.NotContains(t, view, "General (!)")
	assert.NotContains(t, view, "JIRA (!)")

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_EditingEscCancels(t *testing.T) {
	m := newTestConfigModel()
	m.state = StateEditing

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for any output.
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, StateEditing, m.State())

	// Press esc to cancel.
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)
	assert.Equal(t, StateBrowsing, m.State())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_EditingEnterConfirms(t *testing.T) {
	m := newTestConfigModel()
	m.state = StateEditing

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for any output.
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, StateEditing, m.State())

	// Press enter to confirm.
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)
	assert.Equal(t, StateBrowsing, m.State())

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_EditingCtrlCCancels(t *testing.T) {
	m := newTestConfigModel()
	m.state = StateEditing

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for any output.
	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, StateEditing, m.State())

	// Press ctrl+c during editing -- should cancel edit, not quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	waitFor(t, func() bool { return m.State() == StateBrowsing }, time.Second)
	assert.Equal(t, StateBrowsing, m.State())

	// The program should still be running (ctrl+c in editing doesn't quit).
	// Press q to actually quit.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}

func TestTeatest_ConfigModel_ViewSeparators(t *testing.T) {
	m := newTestConfigModel()

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))
	t.Cleanup(func() {
		//nolint:errcheck // Best-effort cleanup in test
		tm.Quit()
	})

	// Wait for render with separator character.
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return bytes.Contains(bts, []byte("\u2500"))
	}, teatest.WithDuration(time.Second))

	// Quit
	tm.Send(tea.KeyMsg{Type: tea.KeyCtrlC})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
