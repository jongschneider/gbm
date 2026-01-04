package service

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestJSONOutput_OutputJSON tests basic JSON output functionality
func TestJSONOutput_OutputJSON(t *testing.T) {
	// Arrange: Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdout := os.Stdout
	os.Stdout = w

	// Create test data
	testData := map[string]interface{}{
		"name": "test-worktree",
		"path": "/repo/worktrees/test",
	}

	// Act: Output JSON
	outputErr := OutputJSON(testData)
	_ = w.Close()

	// Restore stdout
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Assert: No error in outputting
	require.NoError(t, outputErr)

	// Assert: Output is valid JSON
	var result JSONOutput
	err = json.Unmarshal(buf.Bytes()[:len(buf.String())-1], &result) // -1 to remove trailing newline
	require.NoError(t, err, "output should be valid JSON")

	// Assert: Structure is correct
	assert.True(t, result.Success)
	assert.NotNil(t, result.Data)
	assert.Empty(t, result.Error)
}

// TestJSONOutput_OutputJSONError tests error JSON output
func TestJSONOutput_OutputJSONError(t *testing.T) {
	// Arrange: Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdout := os.Stdout
	os.Stdout = w

	// Act: Output error JSON
	errMessage := "worktree not found"
	outputErr := OutputJSONError(errMessage)
	_ = w.Close()

	// Restore stdout
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Assert: No error in outputting
	require.NoError(t, outputErr)

	// Assert: Output is valid JSON
	var result JSONOutput
	err = json.Unmarshal(buf.Bytes()[:len(buf.String())-1], &result)
	require.NoError(t, err)

	// Assert: Structure is correct
	assert.False(t, result.Success)
	assert.Equal(t, errMessage, result.Error)
	assert.Nil(t, result.Data)
}

// TestJSONOutput_OutputJSONWithMessage tests JSON output with message
func TestJSONOutput_OutputJSONWithMessage(t *testing.T) {
	// Arrange: Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdout := os.Stdout
	os.Stdout = w

	testData := map[string]string{"name": "test"}
	message := "operation successful"

	// Act
	outputErr := OutputJSONWithMessage(testData, message)
	_ = w.Close()

	// Restore stdout
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Assert
	require.NoError(t, outputErr)

	var result JSONOutput
	err = json.Unmarshal(buf.Bytes()[:len(buf.String())-1], &result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.Equal(t, message, result.Message)
}

// TestJSONOutput_OutputJSONArray tests JSON array output
func TestJSONOutput_OutputJSONArray(t *testing.T) {
	// Arrange: Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdout := os.Stdout
	os.Stdout = w

	testArray := []map[string]string{
		{"name": "main", "branch": "main"},
		{"name": "feature", "branch": "feature/x"},
	}

	// Act
	outputErr := OutputJSONArray(testArray)
	_ = w.Close()

	// Restore stdout
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Assert
	require.NoError(t, outputErr)

	var result JSONOutput
	err = json.Unmarshal(buf.Bytes()[:len(buf.String())-1], &result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.NotNil(t, result.Data)
}

// TestJSONOutput_OutputRawJSON tests raw JSON output (no wrapper)
func TestJSONOutput_OutputRawJSON(t *testing.T) {
	// Arrange: Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdout := os.Stdout
	os.Stdout = w

	testData := map[string]string{"name": "test", "status": "active"}

	// Act
	outputErr := OutputRawJSON(testData)
	_ = w.Close()

	// Restore stdout
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Assert: No error
	require.NoError(t, outputErr)

	// Assert: Output is valid JSON (should not have JSONOutput wrapper)
	var result map[string]string
	err = json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result)
	require.NoError(t, err)

	assert.Equal(t, "test", result["name"])
	assert.Equal(t, "active", result["status"])
}

// TestJSONOutput_GetOutputFormat tests format detection
func TestJSONOutput_GetOutputFormat(t *testing.T) {
	// Setup
	SetGlobalFlags(&CLIFlags{JSON: false})

	// Test text format (default)
	assert.Equal(t, "text", GetOutputFormat())

	// Test JSON format
	SetGlobalFlags(&CLIFlags{JSON: true})
	assert.Equal(t, "json", GetOutputFormat())

	// Cleanup
	SetGlobalFlags(&CLIFlags{})
}

// TestJSONOutput_HandleOutput tests output routing based on flags
func TestJSONOutput_HandleOutput(t *testing.T) {
	testData := map[string]string{"name": "test"}

	t.Run("JSON output when flag set", func(t *testing.T) {
		// Arrange: Capture stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		oldStdout := os.Stdout
		os.Stdout = w

		SetGlobalFlags(&CLIFlags{JSON: true})

		// Act
		err = HandleOutput(testData)
		_ = w.Close()

		// Restore stdout
		os.Stdout = oldStdout

		// Assert
		require.NoError(t, err)

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)

		// Verify JSON output
		var result JSONOutput
		err = json.Unmarshal(buf.Bytes()[:len(buf.String())-1], &result)
		require.NoError(t, err)
		assert.True(t, result.Success)
	})

	t.Run("No output when text format", func(t *testing.T) {
		SetGlobalFlags(&CLIFlags{JSON: false})

		// Act
		err := HandleOutput(testData)

		// Assert: No error and no output
		require.NoError(t, err)
	})

	// Cleanup
	SetGlobalFlags(&CLIFlags{})
}

// TestJSONOutput_HandleError tests error routing based on flags
func TestJSONOutput_HandleError(t *testing.T) {
	errMsg := "test error"

	t.Run("JSON error when flag set", func(t *testing.T) {
		// Arrange: Capture stdout
		r, w, err := os.Pipe()
		require.NoError(t, err)
		oldStdout := os.Stdout
		os.Stdout = w

		SetGlobalFlags(&CLIFlags{JSON: true})

		// Act
		err = HandleError(errMsg)
		_ = w.Close()

		// Restore stdout
		os.Stdout = oldStdout

		// Assert
		require.NoError(t, err)

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)

		var result JSONOutput
		err = json.Unmarshal(buf.Bytes()[:len(buf.String())-1], &result)
		require.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, errMsg, result.Error)
	})

	t.Run("Text error when flag not set", func(t *testing.T) {
		// Capture stderr for error output
		r, w, err := os.Pipe()
		require.NoError(t, err)
		oldStderr := os.Stderr
		os.Stderr = w

		SetGlobalFlags(&CLIFlags{JSON: false})

		// Act
		err = HandleError(errMsg)
		_ = w.Close()

		// Restore stderr
		os.Stderr = oldStderr

		// Assert
		require.NoError(t, err)

		var buf bytes.Buffer
		_, err = io.Copy(&buf, r)
		require.NoError(t, err)

		// Should print to stderr in text format
		assert.Contains(t, buf.String(), "Error:")
		assert.Contains(t, buf.String(), errMsg)
	})

	// Cleanup
	SetGlobalFlags(&CLIFlags{})
}

// TestJSONOutput_ComplexDataStructure tests JSON output with complex nested data
func TestJSONOutput_ComplexDataStructure(t *testing.T) {
	// Arrange: Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdout := os.Stdout
	os.Stdout = w

	complexData := map[string]interface{}{
		"worktree": "test",
		"branch":   "feature/test",
		"status": map[string]interface{}{
			"ahead":  2,
			"behind": 1,
		},
		"files": []string{"file1.go", "file2.go"},
	}

	// Act
	outputErr := OutputJSON(complexData)
	_ = w.Close()

	// Restore stdout
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Assert
	require.NoError(t, outputErr)

	var result JSONOutput
	err = json.Unmarshal(buf.Bytes()[:len(buf.String())-1], &result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.NotNil(t, result.Data)

	// Verify nested structure is preserved
	dataMap := result.Data.(map[string]interface{})
	assert.Equal(t, "test", dataMap["worktree"])
	assert.Equal(t, "feature/test", dataMap["branch"])
}

// TestJSONOutput_EmptyData tests JSON output with empty/nil data
func TestJSONOutput_EmptyData(t *testing.T) {
	// Arrange: Capture stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdout := os.Stdout
	os.Stdout = w

	// Act: Output empty map
	outputErr := OutputJSON(map[string]interface{}{})
	_ = w.Close()

	// Restore stdout
	os.Stdout = oldStdout

	// Read output
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	require.NoError(t, err)

	// Assert
	require.NoError(t, outputErr)

	var result JSONOutput
	err = json.Unmarshal(buf.Bytes()[:len(buf.String())-1], &result)
	require.NoError(t, err)

	assert.True(t, result.Success)
	assert.NotNil(t, result.Data)
}

// TestJSONOutput_ValidJSON tests that all JSON output is valid and parseable
func TestJSONOutput_ValidJSON(t *testing.T) {
	testCases := []struct {
		name   string
		testFn func() (string, error)
		hasFn  string
	}{
		{
			name: "OutputJSON",
			testFn: func() (string, error) {
				r, w, _ := os.Pipe()
				oldStdout := os.Stdout
				os.Stdout = w

				_ = OutputJSON(map[string]string{"test": "data"})
				_ = w.Close()
				os.Stdout = oldStdout

				var buf bytes.Buffer
				_, _ = io.Copy(&buf, r)
				return buf.String(), nil
			},
			hasFn: "OutputJSON",
		},
		{
			name: "OutputJSONError",
			testFn: func() (string, error) {
				r, w, _ := os.Pipe()
				oldStdout := os.Stdout
				os.Stdout = w

				_ = OutputJSONError("test error")
				_ = w.Close()
				os.Stdout = oldStdout

				var buf bytes.Buffer
				_, _ = io.Copy(&buf, r)
				return buf.String(), nil
			},
			hasFn: "OutputJSONError",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := tc.testFn()
			require.NoError(t, err)

			// Trim newline and test JSON validity
			output = output[:len(output)-1]
			var result map[string]interface{}
			err = json.Unmarshal([]byte(output), &result)
			assert.NoError(t, err, "output should be valid JSON for %s", tc.hasFn)
		})
	}
}
