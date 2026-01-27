package service

import (
	"encoding/json"
	"fmt"
)

// JSONOutput represents a standardized JSON response structure used by all commands.
// This provides consistent JSON output across the entire CLI for easy scripting and parsing.
type JSONOutput struct {
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Success bool   `json:"success"`
}

// OutputJSON marshals data to JSON and outputs to stdout.
// This should be used when ShouldUseJSON() returns true.
// Errors from marshaling are printed to stderr and return an error.
func OutputJSON(data any) error {
	output := JSONOutput{
		Success: true,
		Data:    data,
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		PrintError("failed to marshal JSON: %v\n", err)
		return err
	}

	// Output to stdout (machine-readable data)
	fmt.Println(string(jsonBytes))
	return nil
}

// OutputJSONError outputs an error in JSON format to stdout.
// The error message is included in the Error field.
// This should be called instead of OutputJSON when an error occurs.
func OutputJSONError(errMessage string) error {
	output := JSONOutput{
		Success: false,
		Error:   errMessage,
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		PrintError("failed to marshal error JSON: %v\n", err)
		return err
	}

	// Output to stdout (machine-readable error response)
	fmt.Println(string(jsonBytes))
	return nil
}

// OutputJSONWithMessage outputs data with a success message in JSON format.
// Useful for operations that return both data and a status message.
func OutputJSONWithMessage(data any, message string) error {
	output := JSONOutput{
		Success: true,
		Data:    data,
		Message: message,
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		PrintError("failed to marshal JSON: %v\n", err)
		return err
	}

	fmt.Println(string(jsonBytes))
	return nil
}

// OutputJSONArray outputs an array of items in JSON format.
// Converts the array to a JSON array automatically.
func OutputJSONArray(items any) error {
	output := JSONOutput{
		Success: true,
		Data:    items,
	}

	jsonBytes, err := json.Marshal(output)
	if err != nil {
		PrintError("failed to marshal JSON: %v\n", err)
		return err
	}

	fmt.Println(string(jsonBytes))
	return nil
}

// OutputRawJSON outputs raw JSON directly without the JSONOutput wrapper.
// Use this for commands that want to output raw JSON arrays or objects.
// Useful for streaming or simple data structures.
func OutputRawJSON(data any) error {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		PrintError("failed to marshal JSON: %v\n", err)
		return err
	}

	// Output to stdout (raw JSON)
	fmt.Println(string(jsonBytes))
	return nil
}

// GetOutputFormat returns the format to use for output.
// Returns "json" if --json flag is set, otherwise returns "text".
// Useful for command implementations that support multiple output formats.
func GetOutputFormat() string {
	if ShouldUseJSON() {
		return "json"
	}
	return "text"
}

// HandleOutput is a helper function that automatically handles output based on flags.
// If JSON output is requested, it outputs JSON to stdout.
// If text output is requested, it returns the data for the command to handle.
// This provides a consistent pattern for commands that support both formats.
//
// Example usage:
//
//	result := doSomething()
//	if err := HandleOutput(result); err != nil {
//		return err
//	}
//	// Only reached if not using JSON output
//	fmt.Println(result.Path)
//	PrintSuccess("Created worktree")
func HandleOutput(data any) error {
	if ShouldUseJSON() {
		return OutputJSON(data)
	}
	return nil
}

// HandleError is a helper function that handles error output based on flags.
// If JSON output is requested, it outputs error JSON to stdout.
// If text output is requested, it prints error to stderr.
func HandleError(errMessage string) error {
	if ShouldUseJSON() {
		return OutputJSONError(errMessage)
	}
	PrintError("%s\n", errMessage)
	return nil
}
