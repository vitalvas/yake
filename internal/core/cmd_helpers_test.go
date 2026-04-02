//yake:skip-test
package core

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertCommandBehavior(t *testing.T, createCmd func() *cobra.Command, expectedUse, expectedShort string) {
	t.Helper()

	t.Run("returns valid command", func(t *testing.T) {
		cmd := createCmd()

		require.NotNil(t, cmd)
		assert.Equal(t, expectedUse, cmd.Use)
		assert.Equal(t, expectedShort, cmd.Short)
		assert.NotNil(t, cmd.RunE)
	})

	t.Run("command is executable", func(t *testing.T) {
		cmd := createCmd()

		assert.True(t, cmd.Runnable())
	})

	t.Run("returns nil when no project files exist", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cmd := createCmd()
		err := cmd.RunE(cmd, nil)
		assert.NoError(t, err)
	})

	t.Run("runs successfully with go.mod", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.21\n"), 0644))
		require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"), 0644))

		cmd := createCmd()
		err := cmd.RunE(cmd, nil)
		assert.NoError(t, err)
	})
}
