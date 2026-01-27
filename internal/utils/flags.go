package utils

import (
	"github.com/spf13/cobra"
)

// GetStringFlagOrConfig returns the flag value if explicitly set, otherwise returns the config value.
// This implements the flag override pattern: explicit flags > config file > defaults.
//
// Example:
//
//	baseBranch := GetStringFlagOrConfig(cmd, "base", config.DefaultBranch)
//	// If --base was set: uses flag value
//	// If --base not set: uses config.DefaultBranch
//
// This allows users to override config values without modifying the config file.
func GetStringFlagOrConfig(cmd *cobra.Command, flagName, configValue string) string {
	if cmd.Flags().Changed(flagName) {
		val, _ := cmd.Flags().GetString(flagName) //nolint:errcheck // Flag exists since Changed() returned true
		return val
	}
	return configValue
}

// GetBoolFlagOrConfig returns the flag value if explicitly set, otherwise returns the config value.
// This implements the flag override pattern for boolean flags.
//
// Example:
//
//	dryRun := GetBoolFlagOrConfig(cmd, "dry-run", config.DryRun)
//	// If --dry-run was set: uses flag value
//	// If --dry-run not set: uses config.DryRun
func GetBoolFlagOrConfig(cmd *cobra.Command, flagName string, configValue bool) bool {
	if cmd.Flags().Changed(flagName) {
		val, _ := cmd.Flags().GetBool(flagName) //nolint:errcheck // Flag exists since Changed() returned true
		return val
	}
	return configValue
}

// GetIntFlagOrConfig returns the flag value if explicitly set, otherwise returns the config value.
// This implements the flag override pattern for integer flags.
//
// Example:
//
//	timeout := GetIntFlagOrConfig(cmd, "timeout", config.Timeout)
//	// If --timeout was set: uses flag value
//	// If --timeout not set: uses config.Timeout
func GetIntFlagOrConfig(cmd *cobra.Command, flagName string, configValue int) int {
	if cmd.Flags().Changed(flagName) {
		val, _ := cmd.Flags().GetInt(flagName) //nolint:errcheck // Flag exists since Changed() returned true
		return val
	}
	return configValue
}
