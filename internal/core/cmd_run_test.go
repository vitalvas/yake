package core

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_ensureProjectRoot(t *testing.T) {
	t.Run("passes when .git is present", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.Mkdir(".git", 0755))

		assert.NoError(t, ensureProjectRoot())
	})

	t.Run("passes when .git is a file", func(t *testing.T) {
		// Worktrees and submodules use a .git file rather than a directory.
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile(".git", []byte("gitdir: ../.git/worktrees/x\n"), 0644))

		assert.NoError(t, ensureProjectRoot())
	})

	t.Run("errors when .git is absent", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		err := ensureProjectRoot()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "project root")
	})
}

func TestCreateRunCommand(t *testing.T) {
	assertCommandBehavior(t, createRunCommand, "run", "Run tests and policy checks")

	t.Run("returns error when go tests fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.Mkdir(".git", 0755))
		require.NoError(t, os.WriteFile("go.mod", []byte("module testproject\n\ngo 1.21\n"), 0644))
		require.NoError(t, os.WriteFile("main.go", []byte("package main\n\nfunc main() {}\n"), 0644))
		require.NoError(t, os.WriteFile("main_test.go", []byte("package main\n\nimport \"testing\"\n\nfunc TestFail(t *testing.T) { t.Fatal(\"boom\") }\n"), 0644))

		cmd := createRunCommand()
		err := cmd.RunE(cmd, nil)
		assert.Error(t, err)
	})
}
