package config

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHelpOverlay(t *testing.T) {
	theme := tui.DefaultTheme()
	h := NewHelpOverlay(theme)

	require.NotNil(t, h)
	assert.Equal(t, 0, h.Scroll())
	assert.NotEmpty(t, h.sections)
}

func TestHelpOverlay_HandleKey(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, h *HelpOverlay, closed bool)
		name   string
		key    tea.KeyMsg
	}{
		{
			name: "question mark closes overlay",
			key:  runeKey('?'),
			assert: func(t *testing.T, h *HelpOverlay, closed bool) {
				t.Helper()
				assert.True(t, closed)
			},
		},
		{
			name: "esc closes overlay",
			key:  escKey(),
			assert: func(t *testing.T, h *HelpOverlay, closed bool) {
				t.Helper()
				assert.True(t, closed)
			},
		},
		{
			name: "down scrolls down",
			key:  tea.KeyMsg{Type: tea.KeyDown},
			assert: func(t *testing.T, h *HelpOverlay, closed bool) {
				t.Helper()
				assert.False(t, closed)
				// Scroll may or may not change depending on content vs viewport,
				// but it should not close.
			},
		},
		{
			name: "j scrolls down",
			key:  runeKey('j'),
			assert: func(t *testing.T, h *HelpOverlay, closed bool) {
				t.Helper()
				assert.False(t, closed)
			},
		},
		{
			name: "up does not close",
			key:  tea.KeyMsg{Type: tea.KeyUp},
			assert: func(t *testing.T, h *HelpOverlay, closed bool) {
				t.Helper()
				assert.False(t, closed)
			},
		},
		{
			name: "k does not close",
			key:  runeKey('k'),
			assert: func(t *testing.T, h *HelpOverlay, closed bool) {
				t.Helper()
				assert.False(t, closed)
			},
		},
		{
			name: "other keys do not close",
			key:  runeKey('x'),
			assert: func(t *testing.T, h *HelpOverlay, closed bool) {
				t.Helper()
				assert.False(t, closed)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHelpOverlay(tui.DefaultTheme())
			closed := h.HandleKey(tc.key, 20)
			tc.assert(t, h, closed)
		})
	}
}

func TestHelpOverlay_Scrolling(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, h *HelpOverlay)
		setup  func(h *HelpOverlay)
		name   string
	}{
		{
			name:  "starts at scroll 0",
			setup: func(_ *HelpOverlay) {},
			assert: func(t *testing.T, h *HelpOverlay) {
				t.Helper()
				assert.Equal(t, 0, h.Scroll())
			},
		},
		{
			name: "scroll down increases offset",
			setup: func(h *HelpOverlay) {
				// Use a small viewport so scrolling is possible.
				h.HandleKey(tea.KeyMsg{Type: tea.KeyDown}, 5)
			},
			assert: func(t *testing.T, h *HelpOverlay) {
				t.Helper()
				assert.Equal(t, 1, h.Scroll())
			},
		},
		{
			name: "scroll up at top stays at 0",
			setup: func(h *HelpOverlay) {
				h.HandleKey(tea.KeyMsg{Type: tea.KeyUp}, 5)
			},
			assert: func(t *testing.T, h *HelpOverlay) {
				t.Helper()
				assert.Equal(t, 0, h.Scroll())
			},
		},
		{
			name: "scroll down then up returns to previous position",
			setup: func(h *HelpOverlay) {
				h.HandleKey(tea.KeyMsg{Type: tea.KeyDown}, 5)
				h.HandleKey(tea.KeyMsg{Type: tea.KeyDown}, 5)
				h.HandleKey(tea.KeyMsg{Type: tea.KeyUp}, 5)
			},
			assert: func(t *testing.T, h *HelpOverlay) {
				t.Helper()
				assert.Equal(t, 1, h.Scroll())
			},
		},
		{
			name: "scroll does not exceed content bounds",
			setup: func(h *HelpOverlay) {
				// Scroll way past the content.
				for range 200 {
					h.HandleKey(tea.KeyMsg{Type: tea.KeyDown}, 5)
				}
			},
			assert: func(t *testing.T, h *HelpOverlay) {
				t.Helper()
				total := h.totalLines()
				maxScroll := max(total-5, 0)
				assert.Equal(t, maxScroll, h.Scroll())
			},
		},
		{
			name: "reset scroll returns to top",
			setup: func(h *HelpOverlay) {
				h.HandleKey(tea.KeyMsg{Type: tea.KeyDown}, 5)
				h.HandleKey(tea.KeyMsg{Type: tea.KeyDown}, 5)
				h.ResetScroll()
			},
			assert: func(t *testing.T, h *HelpOverlay) {
				t.Helper()
				assert.Equal(t, 0, h.Scroll())
			},
		},
		{
			name: "large viewport does not scroll",
			setup: func(h *HelpOverlay) {
				// Viewport larger than content.
				h.HandleKey(tea.KeyMsg{Type: tea.KeyDown}, 500)
			},
			assert: func(t *testing.T, h *HelpOverlay) {
				t.Helper()
				assert.Equal(t, 0, h.Scroll())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHelpOverlay(tui.DefaultTheme())
			tc.setup(h)
			tc.assert(t, h)
		})
	}
}

func TestHelpOverlay_View(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, view string)
		name   string
		width  int
		height int
	}{
		{
			name:   "contains title",
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Keybinding Reference")
			},
		},
		{
			name:   "contains close hint",
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "? or esc to close")
			},
		},
		{
			name:   "contains primary keys section",
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Primary Keys")
			},
		},
		{
			name:   "contains vim shortcuts section",
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Vim Shortcuts")
			},
		},
		{
			name:   "contains editing section",
			width:  80,
			height: 40,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "Editing")
			},
		},
		{
			name:   "contains specific keybindings",
			width:  80,
			height: 60,
			assert: func(t *testing.T, view string) {
				t.Helper()
				assert.Contains(t, view, "tab / shift-tab")
				assert.Contains(t, view, "save & quit")
			},
		},
		{
			name:   "shows scroll indicator for small viewport",
			width:  80,
			height: 10,
			assert: func(t *testing.T, view string) {
				t.Helper()
				// With a small viewport the content will overflow,
				// so a scroll indicator should appear.
				assert.Contains(t, view, "[1/")
			},
		},
		{
			name:   "handles very small viewport gracefully",
			width:  30,
			height: 5,
			assert: func(t *testing.T, view string) {
				t.Helper()
				// Should not panic and should produce some output.
				assert.NotEmpty(t, view)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := NewHelpOverlay(tui.DefaultTheme())
			view := h.View(tc.width, tc.height)
			tc.assert(t, view)
		})
	}
}

func TestHelpOverlay_Sections(t *testing.T) {
	sections := buildHelpSections()

	testCases := []struct {
		assert func(t *testing.T, sections []helpSection)
		name   string
	}{
		{
			name: "has primary keys section",
			assert: func(t *testing.T, sections []helpSection) {
				t.Helper()
				require.NotEmpty(t, sections)
				assert.Equal(t, "Primary Keys", sections[0].title)
				assert.NotEmpty(t, sections[0].entries)
			},
		},
		{
			name: "has vim shortcuts section",
			assert: func(t *testing.T, sections []helpSection) {
				t.Helper()
				found := false
				for _, s := range sections {
					if s.title == "Vim Shortcuts" {
						found = true
						assert.NotEmpty(t, s.entries)
						break
					}
				}
				assert.True(t, found, "expected Vim Shortcuts section")
			},
		},
		{
			name: "primary keys contains all browsing keybindings from design doc",
			assert: func(t *testing.T, sections []helpSection) {
				t.Helper()
				primary := sections[0]
				keys := make(map[string]bool)
				for _, e := range primary.entries {
					keys[e.key] = true
				}
				assert.True(t, keys["tab / shift-tab"], "missing tab / shift-tab")
				assert.True(t, keys["up / down"], "missing up / down")
				assert.True(t, keys["e"], "missing e (edit)")
				assert.True(t, keys["enter"], "missing enter")
				assert.True(t, keys["s"], "missing s (save)")
				assert.True(t, keys["r"], "missing r (reset)")
				assert.True(t, keys["R"], "missing R (reset all)")
				assert.True(t, keys["/"], "missing / (search)")
				assert.True(t, keys["?"], "missing ? (help)")
				assert.True(t, keys["a"], "missing a (add)")
				assert.True(t, keys["d"], "missing d (delete)")
				assert.True(t, keys["q"], "missing q (quit)")
				assert.True(t, keys["ctrl-c"], "missing ctrl-c")
			},
		},
		{
			name: "vim shortcuts contains j/k, g/G, curly braces",
			assert: func(t *testing.T, sections []helpSection) {
				t.Helper()
				var vim *helpSection
				for i := range sections {
					if sections[i].title == "Vim Shortcuts" {
						vim = &sections[i]
						break
					}
				}
				require.NotNil(t, vim)
				keys := make(map[string]bool)
				for _, e := range vim.entries {
					keys[e.key] = true
				}
				assert.True(t, keys["j / k"], "missing j / k")
				assert.True(t, keys["g"], "missing g")
				assert.True(t, keys["G"], "missing G")
				assert.True(t, keys["{ / }"], "missing { / }")
			},
		},
		{
			name: "has overlay sections (list, editor, errors)",
			assert: func(t *testing.T, sections []helpSection) {
				t.Helper()
				titles := make(map[string]bool)
				for _, s := range sections {
					titles[s.title] = true
				}
				assert.True(t, titles["List Overlay"], "missing List Overlay section")
				assert.True(t, titles["Editor Overlay"], "missing Editor Overlay section")
				assert.True(t, titles["Errors Overlay"], "missing Errors Overlay section")
			},
		},
		{
			name: "all sections have non-empty entries",
			assert: func(t *testing.T, sections []helpSection) {
				t.Helper()
				for _, s := range sections {
					assert.NotEmpty(t, s.entries, "section %q has no entries", s.title)
					for _, e := range s.entries {
						assert.NotEmpty(t, e.key, "entry in %q has empty key", s.title)
						assert.NotEmpty(t, e.desc, "entry in %q has empty desc", s.title)
					}
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.assert(t, sections)
		})
	}
}

func TestConfigModel_HelpOverlayIntegration(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, m *ConfigModel)
		name   string
		keys   []tea.KeyMsg
	}{
		{
			name: "question mark opens help overlay",
			keys: []tea.KeyMsg{runeKey('?')},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateHelp, m.State())
			},
		},
		{
			name: "question mark toggles help overlay (open then close)",
			keys: []tea.KeyMsg{runeKey('?'), runeKey('?')},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "esc closes help overlay",
			keys: []tea.KeyMsg{runeKey('?'), escKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateBrowsing, m.State())
			},
		},
		{
			name: "help overlay view shows keybinding reference",
			keys: []tea.KeyMsg{runeKey('?')},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				view := m.View()
				assert.Contains(t, view, "Keybinding Reference")
				assert.Contains(t, view, "Primary Keys")
			},
		},
		{
			name: "help overlay resets scroll on open",
			keys: nil, // handled in setup
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				// Open help, scroll down, close, reopen -- should be at top.
				m.Update(runeKey('?'))
				m.Update(tea.KeyMsg{Type: tea.KeyDown})
				m.Update(tea.KeyMsg{Type: tea.KeyDown})
				assert.Positive(t, m.helpOverlay.Scroll())

				m.Update(escKey()) // close
				assert.Equal(t, StateBrowsing, m.State())

				m.Update(runeKey('?')) // reopen
				assert.Equal(t, StateHelp, m.State())
				assert.Equal(t, 0, m.helpOverlay.Scroll())
			},
		},
		{
			name: "tab does not switch tabs in help state",
			keys: []tea.KeyMsg{runeKey('?'), tabKey()},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateHelp, m.State())
				assert.Equal(t, TabGeneral, m.ActiveTab())
			},
		},
		{
			name: "q does not quit in help state",
			keys: []tea.KeyMsg{runeKey('?'), runeKey('q')},
			assert: func(t *testing.T, m *ConfigModel) {
				t.Helper()
				assert.Equal(t, StateHelp, m.State())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			m := NewConfigModel()
			m.width = 80
			m.height = 40

			var result tea.Model = m
			for _, k := range tc.keys {
				result, _ = result.Update(k)
			}
			updated := result.(*ConfigModel)
			tc.assert(t, updated)
		})
	}
}

func TestHelpOverlay_TotalLines(t *testing.T) {
	h := NewHelpOverlay(tui.DefaultTheme())
	total := h.totalLines()

	// Should be positive and reasonable.
	assert.Positive(t, total)

	// Count expected lines: each section has title + separator + entries,
	// plus blank lines between sections.
	sections := buildHelpSections()
	expected := 0
	for i, sec := range sections {
		expected += 2 // title + separator
		expected += len(sec.entries)
		if i < len(sections)-1 {
			expected++ // blank line between sections
		}
	}
	assert.Equal(t, expected, total)
}
