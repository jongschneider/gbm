package service

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/term"
)

// CLIFlags represents the set of global CLI flags available to all commands.
type CLIFlags struct {
	JSON    bool // Output in JSON format
	NoColor bool // Disable colored output
	Quiet   bool // Suppress non-essential messages
	NoInput bool // Disable interactive prompts
	DryRun  bool // Preview operations without executing
	Verbose bool // Enable verbose output (existing, kept for backward compatibility)
}

var (
	globalFlags CLIFlags
	flagsMutex  sync.RWMutex
)

// SetGlobalFlags sets the global flags from command-line parsing.
// This should be called in PersistentPreRun to make flags available to all commands.
func SetGlobalFlags(flags *CLIFlags) {
	flagsMutex.Lock()
	defer flagsMutex.Unlock()
	globalFlags = *flags
}

// GetGlobalFlags returns the current global flags.
func GetGlobalFlags() CLIFlags {
	flagsMutex.RLock()
	defer flagsMutex.RUnlock()
	return globalFlags
}

// ShouldUseJSON returns true if JSON output is requested.
func ShouldUseJSON() bool {
	return GetGlobalFlags().JSON
}

// ShouldUseColor returns true if colored output should be used.
// Respects NO_COLOR environment variable, --no-color flag, and TTY detection.
// Priority: --no-color flag > NO_COLOR env var > isatty detection.
func ShouldUseColor() bool {
	flags := GetGlobalFlags()

	// Explicit --no-color flag takes highest priority
	if flags.NoColor {
		return false
	}

	// Check NO_COLOR environment variable (any value means disable colors)
	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		return false
	}

	// Check if stdout is a TTY (terminal)
	// If not a TTY (e.g., piped or in CI/CD), disable colors
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// ShouldBeQuiet returns true if quiet mode is enabled.
func ShouldBeQuiet() bool {
	return GetGlobalFlags().Quiet
}

// ShouldAllowInput returns true if interactive input should be allowed.
// Returns false if --no-input flag is set.
func ShouldAllowInput() bool {
	return !GetGlobalFlags().NoInput
}

// ShouldUseDryRun returns true if dry-run mode is enabled.
func ShouldUseDryRun() bool {
	return GetGlobalFlags().DryRun
}

// ShouldBeVerbose returns true if verbose output is enabled.
func ShouldBeVerbose() bool {
	return GetGlobalFlags().Verbose
}

// PrintMessage prints a status/info message to stderr (unless quiet mode).
// Messages are subject to --quiet flag.
func PrintMessage(format string, args ...any) {
	if ShouldBeQuiet() {
		return
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

// PrintSuccess prints a success message with checkmark to stderr (unless quiet mode).
func PrintSuccess(message string) {
	if ShouldBeQuiet() {
		return
	}
	// Use ✓ if colors enabled, fallback to text
	symbol := "✓"
	if !ShouldUseColor() {
		symbol = "[✓]"
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", symbol, message)
}

// PrintWarning prints a warning message to stderr (unless quiet mode).
func PrintWarning(message string) {
	if ShouldBeQuiet() {
		return
	}
	// Use ⚠ if colors enabled, fallback to text
	symbol := "⚠"
	if !ShouldUseColor() {
		symbol := "[!]"
		fmt.Fprintf(os.Stderr, "%s %s\n", symbol, message)
		return
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", symbol, message)
}

// PrintError prints an error message to stderr.
// Errors are ALWAYS shown, regardless of --quiet flag.
func PrintError(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "Error: "+format, args...)
}

// PrintInfo prints an info message with 'ℹ' symbol to stderr (unless quiet mode).
func PrintInfo(message string) {
	if ShouldBeQuiet() {
		return
	}
	// Use ℹ if colors enabled, fallback to text
	symbol := "ℹ"
	if !ShouldUseColor() {
		symbol = "[i]"
	}
	fmt.Fprintf(os.Stderr, "%s %s\n", symbol, message)
}

// ColorText applies ANSI color codes to text if colors are enabled.
// Returns plain text if colors are disabled.
func ColorText(text, colorCode string) string {
	if !ShouldUseColor() {
		return text
	}
	// ANSI reset code
	reset := "\033[0m"
	return colorCode + text + reset
}

// ANSI Color Codes (use with ColorText).
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorBlue   = "\033[34m"
	ColorBold   = "\033[1m"
)
