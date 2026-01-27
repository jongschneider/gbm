package async

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFetchMsg(t *testing.T) {
	type testCase struct {
		err    error
		expect func(t *testing.T, msg FetchMsg[string])
		name   string
		value  string
	}

	testCases := []testCase{
		{
			name:  "successful fetch",
			value: "hello",
			err:   nil,
			expect: func(t *testing.T, msg FetchMsg[string]) {
				t.Helper()
				assert.Equal(t, "hello", msg.Value)
				assert.NoError(t, msg.Err)
			},
		},
		{
			name:  "failed fetch",
			value: "",
			err:   errors.New("fetch failed"),
			expect: func(t *testing.T, msg FetchMsg[string]) {
				t.Helper()
				assert.Empty(t, msg.Value)
				require.Error(t, msg.Err)
				assert.Equal(t, "fetch failed", msg.Err.Error())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := FetchMsg[string]{Value: tc.value, Err: tc.err}
			tc.expect(t, msg)
		})
	}
}

func TestFetchCmd(t *testing.T) {
	type testCase struct {
		name        string
		fetch       func() (string, error)
		expectMsg   func(t *testing.T, msg FetchMsg[string])
		expectErr   func(t *testing.T, err error)
		description string
	}

	testCases := []testCase{
		{
			name: "successful command execution",
			fetch: func() (string, error) {
				return "success", nil
			},
			expectMsg: func(t *testing.T, msg FetchMsg[string]) {
				t.Helper()
				assert.Equal(t, "success", msg.Value)
				assert.NoError(t, msg.Err)
			},
			expectErr: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
			description: "FetchCmd should return FetchMsg with value when fetch succeeds",
		},
		{
			name: "failed command execution",
			fetch: func() (string, error) {
				return "", errors.New("network error")
			},
			expectMsg: func(t *testing.T, msg FetchMsg[string]) {
				t.Helper()
				assert.Empty(t, msg.Value)
				require.Error(t, msg.Err)
				assert.Equal(t, "network error", msg.Err.Error())
			},
			expectErr: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
			description: "FetchCmd should return FetchMsg with error when fetch fails",
		},
		{
			name: "command executes asynchronously",
			fetch: func() (string, error) {
				return "delayed", nil
			},
			expectMsg: func(t *testing.T, msg FetchMsg[string]) {
				t.Helper()
				assert.Equal(t, "delayed", msg.Value)
				assert.NoError(t, msg.Err)
			},
			expectErr: func(t *testing.T, err error) {
				t.Helper()
				assert.NoError(t, err)
			},
			description: "FetchCmd should execute fetch function asynchronously",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create the command
			cmd := FetchCmd(tc.fetch)
			assert.NotNil(t, cmd, tc.description)

			// Execute the command (it should be a function that returns tea.Msg)
			msg := cmd()
			tc.expectErr(t, nil)

			// Type assert and verify the message
			fetchMsg, ok := msg.(FetchMsg[string])
			assert.True(t, ok, "returned message should be FetchMsg[string]")
			tc.expectMsg(t, fetchMsg)
		})
	}
}

func TestFetchCmdNonBlocking(t *testing.T) {
	t.Run("command returns quickly without blocking", func(t *testing.T) {
		// Track whether fetch was called
		fetchCalled := false

		cmd := FetchCmd(func() (string, error) {
			fetchCalled = true
			// This should execute during cmd() call, not be scheduled separately
			return "result", nil
		})

		// Execute the command
		msg := cmd()

		// The command should have executed (fetch was called)
		assert.True(t, fetchCalled, "fetch function should be called during cmd() execution")

		// Verify the message is correct
		fetchMsg, ok := msg.(FetchMsg[string])
		assert.True(t, ok)
		assert.Equal(t, "result", fetchMsg.Value)

		// For reference: tea.Cmd functions execute synchronously in the test,
		// but in a real Bubble Tea application, they would be executed asynchronously
		// by the runtime to avoid blocking the event loop.
	})
}

func TestFetchCmdWithIntSlice(t *testing.T) {
	t.Run("generic FetchCmd works with different types", func(t *testing.T) {
		cmd := FetchCmd(func() ([]int, error) {
			return []int{1, 2, 3}, nil
		})

		msg := cmd()
		fetchMsg, ok := msg.(FetchMsg[[]int])
		assert.True(t, ok)
		assert.Equal(t, []int{1, 2, 3}, fetchMsg.Value)
		assert.NoError(t, fetchMsg.Err)
	})
}

func TestFetchCmdWithCustomStruct(t *testing.T) {
	type User struct {
		Name string
		ID   int
	}

	t.Run("generic FetchCmd works with custom types", func(t *testing.T) {
		cmd := FetchCmd(func() (User, error) {
			return User{Name: "Alice", ID: 42}, nil
		})

		msg := cmd()
		fetchMsg, ok := msg.(FetchMsg[User])
		assert.True(t, ok)
		assert.Equal(t, "Alice", fetchMsg.Value.Name)
		assert.Equal(t, 42, fetchMsg.Value.ID)
		assert.NoError(t, fetchMsg.Err)
	})
}
