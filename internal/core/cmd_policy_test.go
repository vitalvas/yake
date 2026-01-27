package core

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreatePolicyCommand(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := createPolicyCommand()

		require.NotNil(t, cmd)
		assert.Equal(t, "policy", cmd.Use)
		assert.Equal(t, "Policy-related commands", cmd.Short)
	})

	t.Run("has run subcommand", func(t *testing.T) {
		cmd := createPolicyCommand()

		var runCmd *cobra.Command

		for _, subCmd := range cmd.Commands() {
			if subCmd.Use == "run" {
				runCmd = subCmd

				break
			}
		}

		require.NotNil(t, runCmd, "run subcommand should exist")
		assert.Equal(t, "Run policy checks for the project", runCmd.Short)
	})
}

func TestPolicyRunCommand(t *testing.T) {
	t.Run("returns nil when no go.mod exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cmd := createPolicyCommand()

		runCmd, _, err := cmd.Find([]string{"run"})
		require.NoError(t, err)

		err = runCmd.RunE(runCmd, nil)
		assert.NoError(t, err)
	})
}
