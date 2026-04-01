package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRunCommand(t *testing.T) {
	assertCommandBehavior(t, createRunCommand, "run", "Run tests and policy checks")

	t.Run("returns error when go tests fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.21\n"), 0644))
		require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() { invalid }\n"), 0644))

		cmd := createRunCommand()
		err := cmd.RunE(cmd, nil)
		assert.Error(t, err)
	})
}
