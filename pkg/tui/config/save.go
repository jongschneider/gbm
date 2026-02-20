package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"
)

// SaveResultMsg carries the outcome of a save operation back to the model.
// On success, Err is nil and Path contains the file that was written.
// On failure, Err describes what went wrong.
type SaveResultMsg struct {
	Err  error
	Path string
}

// SaveFlow orchestrates the full save sequence: validate, check external
// changes, backup, update YAML nodes, write to disk, and mark clean.
type SaveFlow struct {
	modTime  time.Time
	accessor ConfigAccessor
	root     *yaml.Node
	dirty    *DirtyTracker
	filePath string
	isNew    bool
}

// NewSaveFlow creates a SaveFlow with the given parameters.
func NewSaveFlow(
	filePath string,
	modTime time.Time,
	root *yaml.Node,
	dirty *DirtyTracker,
	accessor ConfigAccessor,
	isNew bool,
) *SaveFlow {
	return &SaveFlow{
		filePath: filePath,
		modTime:  modTime,
		root:     root,
		dirty:    dirty,
		accessor: accessor,
		isNew:    isNew,
	}
}

// Validate runs save-level validation and returns any errors found.
// If the returned slice is non-empty, the save should be aborted and
// the errors shown in the error overlay.
func (sf *SaveFlow) Validate() []ValidationError {
	return ValidateSave(sf.accessor)
}

// NeedsOverwriteConfirmation checks whether the config file has been
// modified externally since it was loaded. Returns true if the file's
// mod time has changed, meaning the user should confirm before overwriting.
// For new files (not yet on disk), this always returns false.
func (sf *SaveFlow) NeedsOverwriteConfirmation() (bool, error) {
	if sf.isNew {
		return false, nil
	}

	changed, err := CheckExternalChange(sf.filePath, sf.modTime)
	if err != nil {
		// If the file was deleted, treat as needing confirmation.
		if errors.Is(err, os.ErrNotExist) {
			return true, nil
		}
		return false, err
	}

	return changed, nil
}

// Execute performs the actual save: backup, update nodes, write to disk.
// It assumes validation and overwrite confirmation have already passed.
// On success it marks the DirtyTracker clean and returns the updated mod time.
func (sf *SaveFlow) Execute() (time.Time, error) {
	// Ensure parent directory exists (for new file creation).
	dir := filepath.Dir(sf.filePath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return time.Time{}, fmt.Errorf("create config directory: %w", err)
	}

	// Backup existing file (skip for new files that don't exist yet).
	if !sf.isNew {
		if err := BackupConfigFile(sf.filePath); err != nil {
			return time.Time{}, fmt.Errorf("backup: %w", err)
		}
	}

	// Update YAML nodes for all dirty keys.
	for _, key := range sf.dirty.DirtyKeys() {
		val := sf.accessor.GetValue(key)
		if val == nil {
			continue
		}
		if err := UpdateNodeValue(sf.root, key, val); err != nil {
			return time.Time{}, fmt.Errorf("update YAML node %q: %w", key, err)
		}
	}

	// Write YAML to disk.
	if err := SaveConfigFile(sf.filePath, sf.root); err != nil {
		return time.Time{}, err
	}

	// Mark dirty tracker clean.
	sf.dirty.MarkClean()

	// Read back the new mod time.
	info, err := os.Stat(sf.filePath)
	if err != nil {
		// File was just written successfully; stat failure is unexpected
		// but non-fatal. Return zero time -- the next save will detect
		// the mismatch and prompt for overwrite.
		return time.Time{}, nil //nolint:nilerr // intentional: save succeeded, stat is best-effort
	}

	return info.ModTime(), nil
}

// executeSaveCmd returns a tea.Cmd that runs the save flow and sends
// a SaveResultMsg back to the model.
func executeSaveCmd(sf *SaveFlow) tea.Cmd {
	return func() tea.Msg {
		_, err := sf.Execute()
		if err != nil {
			return SaveResultMsg{Err: err, Path: sf.filePath}
		}
		return SaveResultMsg{Path: sf.filePath}
	}
}
