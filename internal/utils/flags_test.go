package utils

import (
	"fmt"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestGetStringFlagOrConfig(t *testing.T) {
	tests := []struct {
		name        string
		flagValue   string
		setFlag     bool
		configValue string
		expected    string
	}{
		{
			name:        "flag explicitly set",
			flagValue:   "flag-value",
			setFlag:     true,
			configValue: "config-value",
			expected:    "flag-value",
		},
		{
			name:        "flag not set - use config",
			setFlag:     false,
			configValue: "config-value",
			expected:    "config-value",
		},
		{
			name:        "flag set to empty string",
			flagValue:   "",
			setFlag:     true,
			configValue: "config-value",
			expected:    "",
		},
		{
			name:        "config is empty",
			flagValue:   "flag-value",
			setFlag:     true,
			configValue: "",
			expected:    "flag-value",
		},
		{
			name:        "both empty - flag not set",
			setFlag:     false,
			configValue: "",
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("test-flag", "", "test flag")

			if tt.setFlag {
				err := cmd.Flags().Set("test-flag", tt.flagValue)
				assert.NoError(t, err)
			}

			result := GetStringFlagOrConfig(cmd, "test-flag", tt.configValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetBoolFlagOrConfig(t *testing.T) {
	tests := []struct {
		name        string
		flagValue   bool
		setFlag     bool
		configValue bool
		expected    bool
	}{
		{
			name:        "flag explicitly set to true",
			flagValue:   true,
			setFlag:     true,
			configValue: false,
			expected:    true,
		},
		{
			name:        "flag explicitly set to false",
			flagValue:   false,
			setFlag:     true,
			configValue: true,
			expected:    false,
		},
		{
			name:        "flag not set - use config true",
			setFlag:     false,
			configValue: true,
			expected:    true,
		},
		{
			name:        "flag not set - use config false",
			setFlag:     false,
			configValue: false,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Bool("test-flag", false, "test flag")

			if tt.setFlag {
				val := "false"
				if tt.flagValue {
					val = "true"
				}
				err := cmd.Flags().Set("test-flag", val)
				assert.NoError(t, err)
			}

			result := GetBoolFlagOrConfig(cmd, "test-flag", tt.configValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetIntFlagOrConfig(t *testing.T) {
	tests := []struct {
		name        string
		flagValue   int
		setFlag     bool
		configValue int
		expected    int
	}{
		{
			name:        "flag explicitly set",
			flagValue:   42,
			setFlag:     true,
			configValue: 10,
			expected:    42,
		},
		{
			name:        "flag not set - use config",
			setFlag:     false,
			configValue: 10,
			expected:    10,
		},
		{
			name:        "flag set to zero",
			flagValue:   0,
			setFlag:     true,
			configValue: 10,
			expected:    0,
		},
		{
			name:        "config is zero",
			flagValue:   42,
			setFlag:     true,
			configValue: 0,
			expected:    42,
		},
		{
			name:        "both zero - flag not set",
			setFlag:     false,
			configValue: 0,
			expected:    0,
		},
		{
			name:        "negative values",
			flagValue:   -5,
			setFlag:     true,
			configValue: 10,
			expected:    -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().Int("test-flag", 0, "test flag")

			if tt.setFlag {
				err := cmd.Flags().Set("test-flag", fmt.Sprintf("%d", tt.flagValue))
				assert.NoError(t, err)
			}

			result := GetIntFlagOrConfig(cmd, "test-flag", tt.configValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}
