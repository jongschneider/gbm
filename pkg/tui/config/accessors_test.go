package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCoerceValue_String(t *testing.T) {
	tests := []struct {
		value       any
		assert      func(t *testing.T, got any)
		assertError func(t *testing.T, err error)
		name        string
	}{
		{
			name:  "string passes through",
			value: "main",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, "main", got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "empty string passes through",
			value: "",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Empty(t, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "int converted via Sprint",
			value: 42,
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, "42", got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CoerceValue(String, tc.value)
			tc.assertError(t, err)
			tc.assert(t, got)
		})
	}
}

func TestCoerceValue_SensitiveString(t *testing.T) {
	got, err := CoerceValue(SensitiveString, "secret")
	require.NoError(t, err)
	assert.Equal(t, "secret", got)
}

func TestCoerceValue_Int(t *testing.T) {
	tests := []struct {
		value       any
		assert      func(t *testing.T, got any)
		assertError func(t *testing.T, err error)
		name        string
	}{
		{
			name:  "int passes through",
			value: 42,
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, 42, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "int64 converted to int",
			value: int64(99),
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, 99, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "string parsed to int",
			value: "42",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, 42, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "string with whitespace trimmed",
			value: " 7 ",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, 7, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "non-numeric string fails",
			value: "abc",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, 0, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "cannot convert")
			},
		},
		{
			name:  "bool fails",
			value: true,
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, 0, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "cannot convert")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CoerceValue(Int, tc.value)
			tc.assertError(t, err)
			tc.assert(t, got)
		})
	}
}

func TestCoerceValue_Bool(t *testing.T) {
	tests := []struct {
		value       any
		assert      func(t *testing.T, got any)
		assertError func(t *testing.T, err error)
		name        string
	}{
		{
			name:  "bool passes through",
			value: true,
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, true, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "string true parsed",
			value: "true",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, true, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "string false parsed",
			value: "false",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, false, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "string 1 parsed as true",
			value: "1",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, true, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "invalid string fails",
			value: "maybe",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, false, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "cannot convert")
			},
		},
		{
			name:  "int fails",
			value: 1,
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, false, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "cannot convert")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CoerceValue(Bool, tc.value)
			tc.assertError(t, err)
			tc.assert(t, got)
		})
	}
}

func TestCoerceValue_StringList(t *testing.T) {
	tests := []struct {
		value       any
		assert      func(t *testing.T, got any)
		assertError func(t *testing.T, err error)
		name        string
	}{
		{
			name:  "slice passes through",
			value: []string{"a", "b"},
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, []string{"a", "b"}, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "comma-separated string split",
			value: "foo, bar, baz",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, []string{"foo", "bar", "baz"}, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "empty string produces empty slice",
			value: "",
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Equal(t, []string{}, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "int fails",
			value: 42,
			assert: func(t *testing.T, got any) {
				t.Helper()
				assert.Nil(t, got)
			},
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "cannot convert")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CoerceValue(StringList, tc.value)
			tc.assertError(t, err)
			tc.assert(t, got)
		})
	}
}

func TestCoerceValue_ObjectList(t *testing.T) {
	input := []map[string]any{{"key": "val"}}
	got, err := CoerceValue(ObjectList, input)
	require.NoError(t, err)
	assert.Equal(t, input, got)
}

func TestCoerceValue_unsupported_type(t *testing.T) {
	_, err := CoerceValue(FieldType(99), "x")
	require.Error(t, err)
	assert.ErrorContains(t, err, "unsupported field type")
}
