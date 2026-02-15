package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuitConfirm(t *testing.T) {
	testCases := []struct {
		name        string
		keys        []tea.KeyMsg
		assert      func(t *testing.T, f Field)
		assertError func(t *testing.T, err error)
	}{
		{
			name: "y completes with true",
			keys: []tea.KeyMsg{
				{Type: tea.KeyRunes, Runes: []rune{'y'}},
			},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.True(t, f.IsComplete())
				assert.Equal(t, true, f.GetValue())
				assert.False(t, f.IsCancelled())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "Y completes with true",
			keys: []tea.KeyMsg{
				{Type: tea.KeyRunes, Runes: []rune{'Y'}},
			},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.True(t, f.IsComplete())
				assert.Equal(t, true, f.GetValue())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "n completes with false",
			keys: []tea.KeyMsg{
				{Type: tea.KeyRunes, Runes: []rune{'n'}},
			},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.True(t, f.IsComplete())
				assert.Equal(t, false, f.GetValue())
				assert.True(t, f.IsCancelled())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "N completes with false",
			keys: []tea.KeyMsg{
				{Type: tea.KeyRunes, Runes: []rune{'N'}},
			},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.True(t, f.IsComplete())
				assert.Equal(t, false, f.GetValue())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "enter with default selection (Yes) completes with true",
			keys: []tea.KeyMsg{
				{Type: tea.KeyEnter},
			},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.True(t, f.IsComplete())
				assert.Equal(t, true, f.GetValue())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "right then enter selects No",
			keys: []tea.KeyMsg{
				{Type: tea.KeyRight},
				{Type: tea.KeyEnter},
			},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.True(t, f.IsComplete())
				assert.Equal(t, false, f.GetValue())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "left after right selects Yes again",
			keys: []tea.KeyMsg{
				{Type: tea.KeyRight},
				{Type: tea.KeyLeft},
				{Type: tea.KeyEnter},
			},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.True(t, f.IsComplete())
				assert.Equal(t, true, f.GetValue())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "tab toggles selection",
			keys: []tea.KeyMsg{
				{Type: tea.KeyTab},
				{Type: tea.KeyEnter},
			},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.True(t, f.IsComplete())
				assert.Equal(t, false, f.GetValue())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "not complete before input",
			keys: []tea.KeyMsg{},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				assert.False(t, f.IsComplete())
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "view contains question text",
			keys: []tea.KeyMsg{},
			assert: func(t *testing.T, f Field) {
				t.Helper()
				view := f.View()
				assert.Contains(t, view, "Discard unsaved changes?")
				assert.Contains(t, view, "Yes")
				assert.Contains(t, view, "No")
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			f := newQuitConfirm(DefaultTheme())
			require.NotNil(t, f)

			f.Focus()

			for _, key := range tc.keys {
				var err error
				f, _ = f.Update(key)
				tc.assertError(t, err)
			}

			tc.assert(t, f)
		})
	}
}

func TestQuitConfirm_Accessors(t *testing.T) {
	f := newQuitConfirm(DefaultTheme())

	assert.Equal(t, "quit_confirm", f.GetKey())
	assert.False(t, f.Skip())
	assert.Nil(t, f.Error())
}

func TestQuitConfirm_IgnoresInputWhenBlurred(t *testing.T) {
	f := newQuitConfirm(DefaultTheme())
	// Don't focus - field is blurred

	f, _ = f.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	assert.False(t, f.IsComplete(), "blurred field should not process input")
}
