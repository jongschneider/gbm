package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldType_String(t *testing.T) {
	tests := []struct {
		name string
		want string
		ft   FieldType
	}{
		{name: "String", ft: String, want: "String"},
		{name: "SensitiveString", ft: SensitiveString, want: "SensitiveString"},
		{name: "Int", ft: Int, want: "Int"},
		{name: "Bool", ft: Bool, want: "Bool"},
		{name: "StringList", ft: StringList, want: "StringList"},
		{name: "ObjectList", ft: ObjectList, want: "ObjectList"},
		{name: "unknown type", ft: FieldType(99), want: "FieldType(99)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, tc.ft.String())
		})
	}
}

func TestFieldType_enum_values(t *testing.T) {
	// Verify the enum constants have the expected iota values.
	assert.Equal(t, String, FieldType(0))
	assert.Equal(t, SensitiveString, FieldType(1))
	assert.Equal(t, Int, FieldType(2))
	assert.Equal(t, Bool, FieldType(3))
	assert.Equal(t, StringList, FieldType(4))
	assert.Equal(t, ObjectList, FieldType(5))
}

func TestValidateRequired(t *testing.T) {
	tests := []struct {
		value       any
		assertError func(t *testing.T, err error)
		name        string
	}{
		{
			name:  "non-empty string passes",
			value: "main",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "empty string fails",
			value: "",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, "this field is required")
			},
		},
		{
			name:  "non-string fails",
			value: 42,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, "expected a string value")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateRequired(tc.value)
			tc.assertError(t, err)
		})
	}
}

func TestValidatePositiveInt(t *testing.T) {
	tests := []struct {
		value       any
		assertError func(t *testing.T, err error)
		name        string
	}{
		{
			name:  "positive int passes",
			value: 5,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "zero fails",
			value: 0,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, "must be a positive integer")
			},
		},
		{
			name:  "negative int fails",
			value: -3,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, "must be a positive integer")
			},
		},
		{
			name:  "positive int64 passes",
			value: int64(50),
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "zero int64 fails",
			value: int64(0),
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, "must be a positive integer")
			},
		},
		{
			name:  "non-integer fails",
			value: "hello",
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "expected an integer")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidatePositiveInt(tc.value)
			tc.assertError(t, err)
		})
	}
}

func TestValidateNonNegativeInt(t *testing.T) {
	tests := []struct {
		value       any
		assertError func(t *testing.T, err error)
		name        string
	}{
		{
			name:  "positive int passes",
			value: 5,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "zero passes",
			value: 0,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "negative int fails",
			value: -1,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, "must be zero or a positive integer")
			},
		},
		{
			name:  "positive int64 passes",
			value: int64(100),
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "zero int64 passes",
			value: int64(0),
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
		},
		{
			name:  "negative int64 fails",
			value: int64(-5),
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.EqualError(t, err, "must be zero or a positive integer")
			},
		},
		{
			name:  "non-integer fails",
			value: true,
			assertError: func(t *testing.T, err error) {
				t.Helper()
				assert.ErrorContains(t, err, "expected an integer")
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateNonNegativeInt(tc.value)
			tc.assertError(t, err)
		})
	}
}
