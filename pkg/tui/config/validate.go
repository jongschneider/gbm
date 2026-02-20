package config

import (
	"fmt"
	"strings"
)

// ValidationError represents a single field validation failure.
// It carries enough context for the error overlay to jump to the offending
// field: the tab it belongs to, the field index within that tab's section,
// and a human-readable message.
type ValidationError struct {
	FieldKey   string
	FieldLabel string
	Message    string
	Tab        SectionTab
	FieldIndex int
}

// String returns a human-readable representation of the validation error.
func (ve ValidationError) String() string {
	return fmt.Sprintf("%s: %s", ve.FieldLabel, ve.Message)
}

// tabFieldMap maps each tab to its field slice for save-level validation.
// This centralises the mapping so that validators and the error overlay
// agree on which fields belong to which tab.
var tabFieldMap = map[SectionTab][]FieldMeta{
	TabGeneral:  generalFields,
	TabJira:     jiraFields,
	TabFileCopy: fileCopyAutoFields,
}

// ValidateSave runs save-level validation across all tabs.
// It checks required fields, per-field Validate functions, and template
// variable correctness for worktrees_dir. The accessor is used to read
// the current value of each field.
//
// Returns nil when all checks pass. Returns a non-empty slice of
// ValidationError values on failure, sorted by tab order then field order.
func ValidateSave(accessor ConfigAccessor) []ValidationError {
	var errs []ValidationError

	for _, tab := range []SectionTab{TabGeneral, TabJira, TabFileCopy} {
		fields, ok := tabFieldMap[tab]
		if !ok {
			continue
		}
		for i, f := range fields {
			val := accessor.GetValue(f.Key)
			fieldErrs := validateField(f, val)
			for _, msg := range fieldErrs {
				errs = append(errs, ValidationError{
					Tab:        tab,
					FieldKey:   f.Key,
					FieldLabel: f.Label,
					Message:    msg,
					FieldIndex: i,
				})
			}
		}
	}

	// Template variable validation for worktrees_dir.
	if wtDir := accessor.GetValue("worktrees_dir"); wtDir != nil {
		if s, ok := wtDir.(string); ok && s != "" {
			if err := ValidateTemplateVars(s); err != nil {
				errs = append(errs, ValidationError{
					Tab:        TabGeneral,
					FieldKey:   "worktrees_dir",
					FieldLabel: "Worktrees Directory",
					Message:    err.Error(),
					FieldIndex: fieldIndexByKey(generalFields, "worktrees_dir"),
				})
			}
		}
	}

	return errs
}

// validateField runs all applicable checks for a single field and returns
// a list of error messages (empty when the field is valid).
// When the value is nil the validator is skipped, since nil means the field
// was never set (acceptable for optional fields). Required fields surface
// their own errors via ValidateRequired which receives the actual empty
// string value, not nil.
func validateField(f FieldMeta, val any) []string {
	if val == nil || f.Validate == nil {
		return nil
	}

	var msgs []string
	if err := f.Validate(val); err != nil {
		msgs = append(msgs, err.Error())
	}

	return msgs
}

// fieldIndexByKey returns the index of the field with the given key in the
// fields slice, or -1 if not found.
func fieldIndexByKey(fields []FieldMeta, key string) int {
	for i, f := range fields {
		if f.Key == key {
			return i
		}
	}
	return -1
}

// ValidateTemplateVars validates that a path only uses allowed template
// variables ({gitroot}, {branch}, {issue}). This is a TUI-friendly
// wrapper around the same logic in cmd/service/config.go.
func ValidateTemplateVars(path string) error {
	allowed := map[string]bool{
		"{gitroot}": true,
		"{branch}":  true,
		"{issue}":   true,
	}

	for i := 0; i < len(path); i++ {
		if path[i] != '{' {
			continue
		}

		j := i + 1
		for j < len(path) && path[j] != '}' {
			j++
		}
		if j >= len(path) {
			return fmt.Errorf("unclosed template variable in path: %s", path)
		}

		variable := path[i : j+1]
		if !allowed[variable] {
			return fmt.Errorf(
				"invalid template variable '%s' (allowed: {gitroot}, {branch}, {issue})",
				variable,
			)
		}

		i = j
	}

	return nil
}

// ErrorsByTab groups a list of validation errors by tab. The returned map
// only contains tabs that have at least one error.
func ErrorsByTab(errs []ValidationError) map[SectionTab][]ValidationError {
	m := make(map[SectionTab][]ValidationError)
	for _, e := range errs {
		m[e.Tab] = append(m[e.Tab], e)
	}
	return m
}

// TabsWithErrors returns the set of tabs that have at least one error.
func TabsWithErrors(errs []ValidationError) [tabCount]bool {
	var badges [tabCount]bool
	for _, e := range errs {
		if int(e.Tab) >= 0 && int(e.Tab) < tabCount {
			badges[e.Tab] = true
		}
	}
	return badges
}

// FormatErrorSummary returns a multi-line summary of all validation errors
// suitable for display in the error overlay.
func FormatErrorSummary(errs []ValidationError) string {
	if len(errs) == 0 {
		return ""
	}
	var b strings.Builder
	for i, e := range errs {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(e.String())
	}
	return b.String()
}
