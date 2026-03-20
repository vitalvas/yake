package core

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateGitCommand(t *testing.T) {
	t.Run("returns valid command", func(t *testing.T) {
		cmd := createGitCommand()

		require.NotNil(t, cmd)
		assert.Equal(t, "git", cmd.Use)
		assert.Equal(t, "Git-related commands", cmd.Short)
	})

	t.Run("has hook subcommand", func(t *testing.T) {
		cmd := createGitCommand()

		var hookCmd *cobra.Command

		for _, subCmd := range cmd.Commands() {
			if subCmd.Use == "hook" {
				hookCmd = subCmd

				break
			}
		}

		require.NotNil(t, hookCmd, "hook subcommand should exist")
		assert.Equal(t, "Git hook handlers", hookCmd.Short)
	})

	t.Run("hook has commit-msg and pre-commit subcommands", func(t *testing.T) {
		cmd := createGitCommand()

		hookCmd, _, err := cmd.Find([]string{"hook"})
		require.NoError(t, err)

		subCmds := make(map[string]bool)
		for _, sub := range hookCmd.Commands() {
			subCmds[sub.Name()] = true
		}

		assert.True(t, subCmds["commit-msg"], "commit-msg subcommand should exist")
		assert.True(t, subCmds["pre-commit"], "pre-commit subcommand should exist")
	})
}

func TestGitHookCommitMsgCommand(t *testing.T) {
	t.Run("validates commit message", func(t *testing.T) {
		tmpDir := t.TempDir()
		msgPath := filepath.Join(tmpDir, "COMMIT_EDITMSG")
		require.NoError(t, os.WriteFile(msgPath, []byte("feat: valid message"), 0644))

		cmd := createGitCommand()
		commitMsgCmd, _, err := cmd.Find([]string{"hook", "commit-msg"})
		require.NoError(t, err)

		err = commitMsgCmd.RunE(commitMsgCmd, []string{msgPath})
		assert.NoError(t, err)
	})

	t.Run("rejects denied type", func(t *testing.T) {
		tmpDir := t.TempDir()
		msgPath := filepath.Join(tmpDir, "COMMIT_EDITMSG")
		require.NoError(t, os.WriteFile(msgPath, []byte("style: format code"), 0644))

		cmd := createGitCommand()
		commitMsgCmd, _, err := cmd.Find([]string{"hook", "commit-msg"})
		require.NoError(t, err)

		err = commitMsgCmd.RunE(commitMsgCmd, []string{msgPath})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "commit type 'style' is not allowed")
	})
}

func TestGitHookPreCommitCommand(t *testing.T) {
	t.Run("runs successfully", func(t *testing.T) {
		cmd := createGitCommand()
		preCommitCmd, _, err := cmd.Find([]string{"hook", "pre-commit"})
		require.NoError(t, err)

		err = preCommitCmd.RunE(preCommitCmd, nil)
		assert.NoError(t, err)
	})
}
