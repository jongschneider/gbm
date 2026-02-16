package tui

import tea "github.com/charmbracelet/bubbletea"

// GetSidebar returns the sidebar component.
func (m *ConfigModel) GetSidebar() *Sidebar {
	return m.sidebar
}

// GetPaneFocus returns which pane currently has focus.
func (m *ConfigModel) GetPaneFocus() PaneFocus {
	return m.paneFocus
}

// GetCurrentForm returns the current form being displayed.
func (m *ConfigModel) GetCurrentForm() tea.Model {
	return m.currentForm
}

// GetTheme returns the current theme.
func (m *ConfigModel) GetTheme() *Theme {
	return m.theme
}

// GetState returns the current config state.
func (m *ConfigModel) GetState() *ConfigState {
	return m.state
}

// SetState updates the config state.
func (m *ConfigModel) SetState(state *ConfigState) {
	m.state = state
}

// GetFormCache returns the form cache map.
func (m *ConfigModel) GetFormCache() map[string]tea.Model {
	return m.formCache
}

// IsDirty returns whether the config has unsaved changes.
func (m *ConfigModel) IsDirty() bool {
	return m.state != nil && m.state.dirty
}

// ShowSaveConfirm returns whether the save confirmation dialog is visible.
func (m *ConfigModel) ShowSaveConfirm() bool {
	return m.showSaveConfirm
}

// GetSaveError returns the current save error message, if any.
func (m *ConfigModel) GetSaveError() string {
	return m.saveError
}

// GetSaveConfirmContext returns the context that triggered the save confirmation.
func (m *ConfigModel) GetSaveConfirmContext() SaveConfirmContext {
	return m.saveConfirmContext
}

// GetValidationOverlay returns the current validation overlay, if any.
func (m *ConfigModel) GetValidationOverlay() *validationOverlay {
	return m.validationOverlay
}
