package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowState_SetField(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		value       any
		expect      func(t *testing.T, ws *WorkflowState)
		description string
	}{
		{
			name:  "set string field",
			key:   "merge_strategy",
			value: "squash",
			expect: func(t *testing.T, ws *WorkflowState) {
				assert.Equal(t, "squash", ws.GetField("merge_strategy"))
			},
			description: "should store and retrieve string custom field",
		},
		{
			name:  "set int field",
			key:   "review_count",
			value: 3,
			expect: func(t *testing.T, ws *WorkflowState) {
				val := ws.GetField("review_count")
				assert.Equal(t, 3, val)
			},
			description: "should store and retrieve int custom field",
		},
		{
			name:  "set bool field",
			key:   "auto_assign",
			value: true,
			expect: func(t *testing.T, ws *WorkflowState) {
				assert.Equal(t, true, ws.GetField("auto_assign"))
			},
			description: "should store and retrieve bool custom field",
		},
		{
			name:  "set struct field",
			key:   "metadata",
			value: struct{ name string }{name: "test"},
			expect: func(t *testing.T, ws *WorkflowState) {
				val := ws.GetField("metadata")
				assert.NotNil(t, val)
			},
			description: "should store and retrieve struct custom field",
		},
		{
			name:  "overwrite existing field",
			key:   "version",
			value: "1.0",
			expect: func(t *testing.T, ws *WorkflowState) {
				ws.SetField("version", "2.0")
				assert.Equal(t, "2.0", ws.GetField("version"))
			},
			description: "should allow overwriting existing custom fields",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ws := &WorkflowState{}
			ws.SetField(tc.key, tc.value)
			tc.expect(t, ws)
		})
	}
}

func TestWorkflowState_GetField(t *testing.T) {
	testCases := []struct {
		setup       func() *WorkflowState
		expect      func(t *testing.T, val any)
		name        string
		key         string
		description string
		expectFound bool
	}{
		{
			name: "get existing field",
			setup: func() *WorkflowState {
				ws := &WorkflowState{}
				ws.SetField("color", "blue")
				return ws
			},
			key:         "color",
			expectFound: true,
			expect: func(t *testing.T, val any) {
				assert.Equal(t, "blue", val)
			},
			description: "should retrieve existing field",
		},
		{
			name: "get non-existing field",
			setup: func() *WorkflowState {
				return &WorkflowState{}
			},
			key:         "missing",
			expectFound: false,
			expect: func(t *testing.T, val any) {
				assert.Nil(t, val)
			},
			description: "should return nil for missing field",
		},
		{
			name: "get from uninitialized CustomFields",
			setup: func() *WorkflowState {
				ws := &WorkflowState{} // CustomFields is nil
				return ws
			},
			key:         "any_key",
			expectFound: false,
			expect: func(t *testing.T, val any) {
				assert.Nil(t, val)
			},
			description: "should return nil when CustomFields is nil",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ws := tc.setup()
			val := ws.GetField(tc.key)
			if tc.expectFound {
				assert.NotNil(t, val, tc.description)
			} else {
				assert.Nil(t, val, tc.description)
			}
			tc.expect(t, val)
		})
	}
}

func TestWorkflowState_CustomFields_Initialization(t *testing.T) {
	t.Run("CustomFields initialized on first SetField", func(t *testing.T) {
		ws := &WorkflowState{}
		assert.Nil(t, ws.CustomFields, "should start with nil CustomFields")

		ws.SetField("test", "value")
		assert.NotNil(t, ws.CustomFields, "should initialize CustomFields on SetField")
		assert.Equal(t, "value", ws.CustomFields["test"])
	})
}

func TestWorkflowState_MultipleCustomFields(t *testing.T) {
	t.Run("store and retrieve multiple custom fields", func(t *testing.T) {
		ws := &WorkflowState{}

		ws.SetField("field1", "value1")
		ws.SetField("field2", 42)
		ws.SetField("field3", true)

		assert.Equal(t, "value1", ws.GetField("field1"))
		assert.Equal(t, 42, ws.GetField("field2"))
		assert.Equal(t, true, ws.GetField("field3"))
	})
}

func TestWorkflowState_StandardFieldsPreserved(t *testing.T) {
	t.Run("custom fields do not interfere with standard fields", func(t *testing.T) {
		ws := &WorkflowState{
			WorkflowType: "feature",
			WorktreeName: "my-feature",
		}

		ws.SetField("custom_field", "custom_value")

		// Standard fields should be unchanged
		assert.Equal(t, "feature", ws.WorkflowType)
		assert.Equal(t, "my-feature", ws.WorktreeName)
		// Custom field should be accessible
		assert.Equal(t, "custom_value", ws.GetField("custom_field"))
	})
}

func TestWorkflowState_CustomFieldsTypeAssertion(t *testing.T) {
	t.Run("type assertion on retrieved custom fields", func(t *testing.T) {
		ws := &WorkflowState{}
		ws.SetField("count", 100)
		ws.SetField("enabled", true)
		ws.SetField("label", "test")

		count, ok := ws.GetField("count").(int)
		assert.True(t, ok)
		assert.Equal(t, 100, count)

		enabled, ok := ws.GetField("enabled").(bool)
		assert.True(t, ok)
		assert.True(t, enabled)

		label, ok := ws.GetField("label").(string)
		assert.True(t, ok)
		assert.Equal(t, "test", label)
	})
}

func TestWorkflowState_CustomFieldsWithNilValue(t *testing.T) {
	t.Run("store nil as custom field value", func(t *testing.T) {
		ws := &WorkflowState{}
		ws.SetField("nullable", nil)

		// Field exists but has nil value
		assert.Nil(t, ws.GetField("nullable"))
		// CustomFields map should contain the key
		assert.NotNil(t, ws.CustomFields)
		_, exists := ws.CustomFields["nullable"]
		assert.True(t, exists, "nil value should still be stored in map")
	})
}

func TestWorkflowState_UpdateCustomField(t *testing.T) {
	t.Run("update custom field multiple times", func(t *testing.T) {
		ws := &WorkflowState{}

		ws.SetField("version", "1.0")
		assert.Equal(t, "1.0", ws.GetField("version"))

		ws.SetField("version", "1.1")
		assert.Equal(t, "1.1", ws.GetField("version"))

		ws.SetField("version", "2.0")
		assert.Equal(t, "2.0", ws.GetField("version"))
	})
}

func TestNewContext_WorkflowStateCustomFields(t *testing.T) {
	t.Run("new context has clean workflow state", func(t *testing.T) {
		ctx := NewContext()

		assert.NotNil(t, ctx.State)
		assert.Empty(t, ctx.State.WorkflowType)
		assert.Nil(t, ctx.State.CustomFields, "should start with nil CustomFields")

		// Can immediately use SetField/GetField
		ctx.State.SetField("key", "value")
		assert.Equal(t, "value", ctx.State.GetField("key"))
	})
}
