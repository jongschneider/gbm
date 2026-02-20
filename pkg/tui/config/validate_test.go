package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAccessor implements ConfigAccessor for testing.
type stubAccessor struct {
	values map[string]any
}

func (s *stubAccessor) GetValue(key string) any { return s.values[key] }
func (s *stubAccessor) SetValue(key string, value any) error {
	s.values[key] = value
	return nil
}

func TestValidateTemplateVars(t *testing.T) {
	testCases := []struct {
		assertError func(t *testing.T, err error)
		name        string
		path        string
	}{
		{
			name: "plain path no templates",
			path: "worktrees",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "valid gitroot template",
			path: "../{gitroot}-wt",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "valid branch template",
			path: "~/dev/{branch}",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "valid issue template",
			path: "wt/{issue}",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "multiple valid templates",
			path: "{gitroot}/{branch}/{issue}",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name: "invalid template variable",
			path: "{invalid}",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "invalid template variable")
				assert.Contains(t, err.Error(), "{invalid}")
			},
		},
		{
			name: "unclosed template variable",
			path: "{gitroot",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				require.Error(t, err)
				assert.Contains(t, err.Error(), "unclosed template variable")
			},
		},
		{
			name: "empty path",
			path: "",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateTemplateVars(tc.path)
			tc.assertError(t, err)
		})
	}
}

func TestValidateSave(t *testing.T) {
	testCases := []struct {
		accessor ConfigAccessor
		assert   func(t *testing.T, errs []ValidationError)
		name     string
	}{
		{
			name: "all required fields present passes",
			accessor: &stubAccessor{values: map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "worktrees",
			}},
			assert: func(t *testing.T, errs []ValidationError) {
				t.Helper()
				assert.Empty(t, errs)
			},
		},
		{
			name: "missing required default_branch",
			accessor: &stubAccessor{values: map[string]any{
				"default_branch": "",
				"worktrees_dir":  "worktrees",
			}},
			assert: func(t *testing.T, errs []ValidationError) {
				t.Helper()
				require.NotEmpty(t, errs)
				found := false
				for _, e := range errs {
					if e.FieldKey == "default_branch" {
						found = true
						assert.Equal(t, TabGeneral, e.Tab)
						assert.Contains(t, e.Message, "required")
					}
				}
				assert.True(t, found, "expected error for default_branch")
			},
		},
		{
			name: "missing required worktrees_dir",
			accessor: &stubAccessor{values: map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "",
			}},
			assert: func(t *testing.T, errs []ValidationError) {
				t.Helper()
				require.NotEmpty(t, errs)
				found := false
				for _, e := range errs {
					if e.FieldKey == "worktrees_dir" {
						found = true
						assert.Equal(t, TabGeneral, e.Tab)
					}
				}
				assert.True(t, found, "expected error for worktrees_dir")
			},
		},
		{
			name: "invalid template vars in worktrees_dir",
			accessor: &stubAccessor{values: map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "{invalid_var}",
			}},
			assert: func(t *testing.T, errs []ValidationError) {
				t.Helper()
				require.NotEmpty(t, errs)
				found := false
				for _, e := range errs {
					if e.FieldKey == "worktrees_dir" &&
						e.Message != "this field is required" {
						found = true
						assert.Contains(t, e.Message, "invalid template variable")
					}
				}
				assert.True(t, found, "expected template variable error for worktrees_dir")
			},
		},
		{
			name: "negative int fails validation",
			accessor: &stubAccessor{values: map[string]any{
				"default_branch":               "main",
				"worktrees_dir":                "worktrees",
				"jira.attachments.max_size_mb": -5,
			}},
			assert: func(t *testing.T, errs []ValidationError) {
				t.Helper()
				found := false
				for _, e := range errs {
					if e.FieldKey == "jira.attachments.max_size_mb" {
						found = true
						assert.Equal(t, TabJira, e.Tab)
						assert.Contains(t, e.Message, "positive integer")
					}
				}
				assert.True(t, found, "expected error for negative max_size_mb")
			},
		},
		{
			name: "nil values for optional fields are fine",
			accessor: &stubAccessor{values: map[string]any{
				"default_branch": "main",
				"worktrees_dir":  "worktrees",
				// All jira fields are nil -- they have no Validate func requiring values.
			}},
			assert: func(t *testing.T, errs []ValidationError) {
				t.Helper()
				assert.Empty(t, errs)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			errs := ValidateSave(tc.accessor)
			tc.assert(t, errs)
		})
	}
}

func TestValidationError_String(t *testing.T) {
	ve := ValidationError{
		FieldLabel: "Default Branch",
		Message:    "this field is required",
	}
	assert.Equal(t, "Default Branch: this field is required", ve.String())
}

func TestErrorsByTab(t *testing.T) {
	errs := []ValidationError{
		{Tab: TabGeneral, FieldKey: "default_branch", FieldLabel: "Default Branch", Message: "required"},
		{Tab: TabGeneral, FieldKey: "worktrees_dir", FieldLabel: "Worktrees Dir", Message: "required"},
		{Tab: TabJira, FieldKey: "jira.host", FieldLabel: "Host", Message: "invalid"},
	}

	byTab := ErrorsByTab(errs)

	assert.Len(t, byTab[TabGeneral], 2)
	assert.Len(t, byTab[TabJira], 1)
	assert.Empty(t, byTab[TabFileCopy])
}

func TestTabsWithErrors(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, badges [tabCount]bool)
		name   string
		errs   []ValidationError
	}{
		{
			name: "no errors no badges",
			errs: nil,
			assert: func(t *testing.T, badges [tabCount]bool) {
				t.Helper()
				for i := range tabCount {
					assert.False(t, badges[i], "tab %d should have no badge", i)
				}
			},
		},
		{
			name: "errors on general and jira",
			errs: []ValidationError{
				{Tab: TabGeneral},
				{Tab: TabJira},
			},
			assert: func(t *testing.T, badges [tabCount]bool) {
				t.Helper()
				assert.True(t, badges[TabGeneral])
				assert.True(t, badges[TabJira])
				assert.False(t, badges[TabFileCopy])
				assert.False(t, badges[TabWorktrees])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			badges := TabsWithErrors(tc.errs)
			tc.assert(t, badges)
		})
	}
}

func TestFormatErrorSummary(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, summary string)
		name   string
		errs   []ValidationError
	}{
		{
			name: "empty errors returns empty string",
			errs: nil,
			assert: func(t *testing.T, summary string) {
				t.Helper()
				assert.Empty(t, summary)
			},
		},
		{
			name: "single error",
			errs: []ValidationError{
				{FieldLabel: "Host", Message: "required"},
			},
			assert: func(t *testing.T, summary string) {
				t.Helper()
				assert.Equal(t, "Host: required", summary)
			},
		},
		{
			name: "multiple errors separated by newlines",
			errs: []ValidationError{
				{FieldLabel: "Host", Message: "required"},
				{FieldLabel: "Port", Message: "must be positive"},
			},
			assert: func(t *testing.T, summary string) {
				t.Helper()
				assert.Contains(t, summary, "Host: required")
				assert.Contains(t, summary, "Port: must be positive")
				assert.Contains(t, summary, "\n")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			summary := FormatErrorSummary(tc.errs)
			tc.assert(t, summary)
		})
	}
}

func TestFieldIndexByKey(t *testing.T) {
	testCases := []struct {
		assert func(t *testing.T, idx int)
		name   string
		key    string
	}{
		{
			name: "finds existing key",
			key:  "default_branch",
			assert: func(t *testing.T, idx int) {
				t.Helper()
				assert.Equal(t, 0, idx)
			},
		},
		{
			name: "finds second key",
			key:  "worktrees_dir",
			assert: func(t *testing.T, idx int) {
				t.Helper()
				assert.Equal(t, 1, idx)
			},
		},
		{
			name: "returns -1 for missing key",
			key:  "nonexistent",
			assert: func(t *testing.T, idx int) {
				t.Helper()
				assert.Equal(t, -1, idx)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			idx := fieldIndexByKey(generalFields, tc.key)
			tc.assert(t, idx)
		})
	}
}
