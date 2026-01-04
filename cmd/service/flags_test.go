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
