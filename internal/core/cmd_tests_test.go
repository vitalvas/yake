package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTestsCommand(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := createTestsCommand()

		require.NotNil(t, cmd)
		assert.Equal(t, "tests", cmd.Use)
		assert.Equal(t, "Run Go tests with coverage, race detection, and linting", cmd.Short)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("command is executable", func(t *testing.T) {
		cmd := createTestsCommand()

		assert.True(t, cmd.Runnable())
	})

	t.Run("returns nil when no go.mod exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cmd := createTestsCommand()
		err := cmd.RunE(cmd, nil)
		assert.NoError(t, err)
	})
}
