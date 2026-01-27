package fields

import (
	"gbm/pkg/tui"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestFilePicker_Create(t *testing.T) {
	testCases := []struct {
		expect func(t *testing.T, fp *FilePicker)
		name   string
	}{
		{
			name: "creates with default values",
			expect: func(t *testing.T, fp *FilePicker) {
				t.Helper()
				assert.NotNil(t, fp)
				assert.Equal(t, "test_key", fp.key)
				assert.Equal(t, "Test Title", fp.title)
				assert.Equal(t, "Test description", fp.description)
				assert.False(t, fp.focused)
				assert.False(t, fp.complete)
				assert.False(t, fp.cancelled)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fp := NewFilePicker("test_key", "Test Title", "Test description")
			tc.expect(t, fp)
		})
	}
}

func TestFilePicker_WithCurrentDir(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp = fp.WithCurrentDir("/tmp")

	assert.Equal(t, "/tmp", fp.currentDir)
}

func TestFilePicker_WithAllowedTypes(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp = fp.WithAllowedTypes([]string{".go", ".yaml"})

	assert.Equal(t, []string{".go", ".yaml"}, fp.allowedTypes)
}

func TestFilePicker_WithDirAllowed(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp = fp.WithDirAllowed(false)

	assert.False(t, fp.dirAllowed)
}

func TestFilePicker_WithMultiSelect(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp = fp.WithMultiSelect(true)

	assert.True(t, fp.multiSelect)
}

func TestFilePicker_Focus(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")

	assert.False(t, fp.focused)

	_ = fp.Focus()

	assert.True(t, fp.focused)
}

func TestFilePicker_Blur(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	_ = fp.Focus()

	assert.True(t, fp.focused)

	_ = fp.Blur()

	assert.False(t, fp.focused)
}

func TestFilePicker_IsComplete(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")

	assert.False(t, fp.IsComplete())
}

func TestFilePicker_IsCancelled(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")

	assert.False(t, fp.IsCancelled())
}

func TestFilePicker_GetKey(t *testing.T) {
	fp := NewFilePicker("test_key", "Title", "Desc")

	assert.Equal(t, "test_key", fp.GetKey())
}

func TestFilePicker_GetValue(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")

	value := fp.GetValue()
	files, ok := value.([]string)

	assert.True(t, ok)
	assert.Empty(t, files)
}

func TestFilePicker_GetSelectedFiles(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp.selectedFiles = []string{"/path/to/file1", "/path/to/file2"}

	files := fp.GetSelectedFiles()

	assert.Len(t, files, 2)
	assert.Contains(t, files, "/path/to/file1")
	assert.Contains(t, files, "/path/to/file2")
}

func TestFilePicker_ClearSelection(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp.selectedFiles = []string{"/path/to/file1", "/path/to/file2"}

	fp.ClearSelection()

	assert.Empty(t, fp.selectedFiles)
}

func TestFilePicker_RemoveSelection(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp.selectedFiles = []string{"/path/to/file1", "/path/to/file2", "/path/to/file3"}

	fp.RemoveSelection("/path/to/file2")

	assert.Len(t, fp.selectedFiles, 2)
	assert.Contains(t, fp.selectedFiles, "/path/to/file1")
	assert.Contains(t, fp.selectedFiles, "/path/to/file3")
	assert.NotContains(t, fp.selectedFiles, "/path/to/file2")
}

func TestFilePicker_isAlreadySelected(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp.selectedFiles = []string{"/path/to/file1"}

	assert.True(t, fp.isAlreadySelected("/path/to/file1"))
	assert.False(t, fp.isAlreadySelected("/path/to/file2"))
}

func TestFilePicker_View(t *testing.T) {
	fp := NewFilePicker("key", "Test Title", "Test description")
	fp = fp.WithTheme(tui.DefaultTheme()).(*FilePicker)
	_ = fp.Focus()

	view := fp.View()

	assert.Contains(t, view, "Test Title")
	assert.Contains(t, view, "Test description")
	assert.Contains(t, view, "Esc cancel")
}

func TestFilePicker_View_WithSelectedFiles(t *testing.T) {
	fp := NewFilePicker("key", "Test Title", "Test description")
	fp = fp.WithTheme(tui.DefaultTheme()).(*FilePicker)
	fp.selectedFiles = []string{"/path/to/file1.txt"}
	_ = fp.Focus()

	view := fp.View()

	assert.Contains(t, view, "Selected files:")
	assert.Contains(t, view, "/path/to/file1.txt")
}

func TestFilePicker_View_MultiSelectHelp(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	fp = fp.WithMultiSelect(true)
	fp = fp.WithTheme(tui.DefaultTheme()).(*FilePicker)
	_ = fp.Focus()

	view := fp.View()

	assert.Contains(t, view, "Space add file")
	assert.Contains(t, view, "Enter confirm")
}

func TestFilePicker_UpdateEscapeCancels(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	_ = fp.Focus()

	result, _ := fp.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := result.(*FilePicker)

	assert.True(t, updated.IsCancelled())
}

func TestFilePicker_Skip(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")

	assert.False(t, fp.Skip())
}

func TestFilePicker_Error(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")

	assert.NoError(t, fp.Error())
}

func TestFilePicker_WithWidth(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	result := fp.WithWidth(100)

	assert.IsType(t, &FilePicker{}, result)
}

func TestFilePicker_WithHeight(t *testing.T) {
	fp := NewFilePicker("key", "Title", "Desc")
	result := fp.WithHeight(30)

	updated := result.(*FilePicker)
	assert.Equal(t, 30, updated.height)
}

func TestFilePicker_WithOnSelect(t *testing.T) {
	var selectedPath string
	fp := NewFilePicker("key", "Title", "Desc")
	fp = fp.WithOnSelect(func(path string) {
		selectedPath = path
	})

	assert.NotNil(t, fp.onSelect)

	fp.onSelect("/test/path")
	assert.Equal(t, "/test/path", selectedPath)
}

// Ensure FilePicker implements Field interface.
var _ tui.Field = (*FilePicker)(nil)
