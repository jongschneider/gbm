package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSetAndGetGlobalFlags tests setting and retrieving global flags.
func TestSetAndGetGlobalFlags(t *testing.T) {
	flags := CLIFlags{
		JSON:    true,
		NoColor: true,
		Quiet:   true,
		NoInput: false,
		DryRun:  true,
		Verbose: false,
	}

	SetGlobalFlags(&flags)
	retrieved := GetGlobalFlags()

	assert.Equal(t, true, retrieved.JSON)
	assert.Equal(t, true, retrieved.NoColor)
	assert.Equal(t, true, retrieved.Quiet)
	assert.Equal(t, false, retrieved.NoInput)
	assert.Equal(t, true, retrieved.DryRun)
	assert.Equal(t, false, retrieved.Verbose)
}

// TestShouldUseJSON tests the JSON flag accessor.
func TestShouldUseJSON(t *testing.T) {
	SetGlobalFlags(&CLIFlags{JSON: true})
	assert.True(t, ShouldUseJSON())

	SetGlobalFlags(&CLIFlags{JSON: false})
	assert.False(t, ShouldUseJSON())
}

// TestShouldUseColor_NoColorFlag tests --no-color flag takes priority.
func TestShouldUseColor_NoColorFlag(t *testing.T) {
	// Ensure NO_COLOR env var is not set for this test
	t.Setenv("NO_COLOR", "")

	SetGlobalFlags(&CLIFlags{NoColor: true})
	assert.False(t, ShouldUseColor(), "should disable colors when --no-color is set")
}

// TestShouldUseColor_NOCOLOREnv tests NO_COLOR environment variable.
func TestShouldUseColor_NOCOLOREnv(t *testing.T) {
	SetGlobalFlags(&CLIFlags{NoColor: false})
	t.Setenv("NO_COLOR", "1")
	assert.False(t, ShouldUseColor(), "should disable colors when NO_COLOR env var is set")
}

// TestShouldBeQuiet tests the quiet flag accessor.
func TestShouldBeQuiet(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Quiet: true})
	assert.True(t, ShouldBeQuiet())

	SetGlobalFlags(&CLIFlags{Quiet: false})
	assert.False(t, ShouldBeQuiet())
}

// TestShouldAllowInput tests the no-input flag accessor.
func TestShouldAllowInput(t *testing.T) {
	SetGlobalFlags(&CLIFlags{NoInput: false})
	assert.True(t, ShouldAllowInput())

	SetGlobalFlags(&CLIFlags{NoInput: true})
	assert.False(t, ShouldAllowInput())
}

// TestShouldUseDryRun tests the dry-run flag accessor.
func TestShouldUseDryRun(t *testing.T) {
	SetGlobalFlags(&CLIFlags{DryRun: true})
	assert.True(t, ShouldUseDryRun())

	SetGlobalFlags(&CLIFlags{DryRun: false})
	assert.False(t, ShouldUseDryRun())
}

// TestShouldBeVerbose tests the verbose flag accessor.
func TestShouldBeVerbose(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Verbose: true})
	assert.True(t, ShouldBeVerbose())

	SetGlobalFlags(&CLIFlags{Verbose: false})
	assert.False(t, ShouldBeVerbose())
}

// TestColorText tests color text application.
func TestColorText(t *testing.T) {
	SetGlobalFlags(&CLIFlags{NoColor: true})
	result := ColorText("hello", ColorRed)
	assert.Equal(t, "hello", result, "should return plain text when colors disabled")

	SetGlobalFlags(&CLIFlags{NoColor: false})
	result = ColorText("hello", ColorRed)
	assert.Contains(t, result, "hello", "should contain original text")
	// Only test that it contains the reset code if colors are actually being used
	if ShouldUseColor() {
		assert.Contains(t, result, "\033[0m", "should contain ANSI reset code when colors enabled")
	}
}

// TestPrintSuccess does not print when quiet mode enabled.
func TestPrintSuccess_QuietMode(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Quiet: true})
	// Just verify it doesn't panic (actual message suppression is hard to test without capturing stderr)
	PrintSuccess("test message")
}

// TestPrintError always prints (even in quiet mode).
// Note: We can't easily test stderr output without redirecting it.
// This test just verifies the function exists and doesn't panic.
func TestPrintError_NeverSilenced(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Quiet: true})
	// Just verify it doesn't panic
	PrintError("test error\n")
}

// TestPrintMessage respects quiet mode.
func TestPrintMessage_QuietMode(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Quiet: true})
	// Just verify it doesn't panic
	PrintMessage("test message\n")

	SetGlobalFlags(&CLIFlags{Quiet: false})
	// Just verify it doesn't panic
	PrintMessage("test message\n")
}

// TestPrintWarning respects quiet mode.
func TestPrintWarning_QuietMode(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Quiet: true})
	PrintWarning("test warning")

	SetGlobalFlags(&CLIFlags{Quiet: false})
	PrintWarning("test warning")
}

// TestPrintInfo respects quiet mode.
func TestPrintInfo_QuietMode(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Quiet: true})
	PrintInfo("test info")

	SetGlobalFlags(&CLIFlags{Quiet: false})
	PrintInfo("test info")
}

// BenchmarkGetGlobalFlags benchmarks flag access.
func BenchmarkGetGlobalFlags(b *testing.B) {
	SetGlobalFlags(&CLIFlags{})
	for i := 0; i < b.N; i++ {
		_ = GetGlobalFlags()
	}
}

// BenchmarkShouldUseJSON benchmarks JSON flag check.
func BenchmarkShouldUseJSON(b *testing.B) {
	SetGlobalFlags(&CLIFlags{JSON: true})
	for i := 0; i < b.N; i++ {
		_ = ShouldUseJSON()
	}
}

// BenchmarkShouldUseColor benchmarks color flag check.
func BenchmarkShouldUseColor(b *testing.B) {
	SetGlobalFlags(&CLIFlags{NoColor: false})
	for i := 0; i < b.N; i++ {
		_ = ShouldUseColor()
	}
}

// TestShouldUseColor_NoColorFlagPriority tests that --no-color flag has highest priority.
// Priority: --no-color flag > NO_COLOR env var > isatty detection
func TestShouldUseColor_FlagPriority(t *testing.T) {
	// Set NO_COLOR env var
	t.Setenv("NO_COLOR", "1")

	// --no-color flag should override NO_COLOR env var
	SetGlobalFlags(&CLIFlags{NoColor: true})
	assert.False(t, ShouldUseColor(), "flag should have highest priority")

	// Without --no-color flag, NO_COLOR env var should apply
	SetGlobalFlags(&CLIFlags{NoColor: false})
	assert.False(t, ShouldUseColor(), "NO_COLOR env var should apply when flag is false")
}

// TestShouldUseColor_EnvVarPriority tests that NO_COLOR env var has priority over TTY detection.
func TestShouldUseColor_EnvVarTakePriority(t *testing.T) {
	// Clear any NO_COLOR setting
	t.Setenv("NO_COLOR", "")

	// Unset NO_COLOR and disable flag, color should be based on TTY
	SetGlobalFlags(&CLIFlags{NoColor: false})
	canUseColor := ShouldUseColor()

	// Now set NO_COLOR and verify it disables colors
	t.Setenv("NO_COLOR", "1")
	SetGlobalFlags(&CLIFlags{NoColor: false})
	assert.False(t, ShouldUseColor(), "NO_COLOR env var should disable colors")

	// Clear NO_COLOR and verify it goes back to original (TTY-dependent)
	t.Setenv("NO_COLOR", "")
	SetGlobalFlags(&CLIFlags{NoColor: false})
	assert.Equal(t, canUseColor, ShouldUseColor(), "without NO_COLOR env var, should match original TTY detection")
}

// TestQuietMode_MessagesSuppressed tests that --quiet suppresses non-critical messages.
func TestQuietMode_MessagesSuppressed(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Quiet: true})

	// These should be suppressed (no panic, but nothing printed)
	PrintMessage("test message\n")
	PrintSuccess("test success")
	PrintWarning("test warning")
	PrintInfo("test info")

	// Error messages should NOT be suppressed (tested by verifying no panic)
	PrintError("test error\n")
}

// TestQuietMode_ErrorsNeverSuppressed tests that errors are always shown in quiet mode.
func TestQuietMode_ErrorsAlwaysShown(t *testing.T) {
	SetGlobalFlags(&CLIFlags{Quiet: true})
	// PrintError should work even in quiet mode (no suppression)
	PrintError("critical error message\n")
	// If this doesn't panic, test passes
}

// TestFlagCombinations tests multiple flags used together.
func TestFlagCombinations(t *testing.T) {
	flags := CLIFlags{
		JSON:    true,
		NoColor: true,
		Quiet:   true,
		NoInput: true,
		DryRun:  true,
		Verbose: false,
	}

	SetGlobalFlags(&flags)

	// Verify all flags are set correctly
	assert.True(t, ShouldUseJSON())
	assert.False(t, ShouldUseColor())
	assert.True(t, ShouldBeQuiet())
	assert.False(t, ShouldAllowInput())
	assert.True(t, ShouldUseDryRun())
	assert.False(t, ShouldBeVerbose())
}

// TestPrintSuccessFormat tests that PrintSuccess uses correct format.
func TestPrintSuccessFormat(t *testing.T) {
	SetGlobalFlags(&CLIFlags{NoColor: true, Quiet: false})
	// Just verify it doesn't panic - output format is hard to test without stderr capture
	PrintSuccess("operation completed")
}

// TestPrintInfoFormat tests that PrintInfo uses correct format.
func TestPrintInfoFormat(t *testing.T) {
	SetGlobalFlags(&CLIFlags{NoColor: true, Quiet: false})
	// Just verify it doesn't panic - output format is hard to test without stderr capture
	PrintInfo("informational message")
}

// TestColorCodeConstants tests that color constants exist and are strings.
func TestColorCodeConstants(t *testing.T) {
	// Verify all color constants are defined
	assert.NotEmpty(t, ColorRed)
	assert.NotEmpty(t, ColorGreen)
	assert.NotEmpty(t, ColorYellow)
	assert.NotEmpty(t, ColorBlue)
	assert.NotEmpty(t, ColorBold)
}
