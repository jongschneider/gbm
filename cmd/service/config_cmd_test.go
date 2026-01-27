package service

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigCommand(t *testing.T) {
	testCases := []struct {
		name   string
		expect func(t *testing.T, svc *Service, cmd any)
	}{
		{
			name: "creates command with correct use and short description",
			expect: func(t *testing.T, svc *Service, cmd any) {
				cobraCmd := cmd.(*cobra.Command)
				assert.Equal(t, "config", cobraCmd.Use)
				assert.Equal(t, "Manage GBM configuration interactively", cobraCmd.Short)
			},
		},
		{
			name: "command has RunE handler",
			expect: func(t *testing.T, svc *Service, cmd any) {
				cobraCmd := cmd.(*cobra.Command)
				assert.NotNil(t, cobraCmd.RunE)
			},
		},
	}

	svc := NewService()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := newConfigCommand(svc)
			tc.expect(t, svc, cmd)
		})
	}
}
