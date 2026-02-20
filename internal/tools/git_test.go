package tools

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initGitRepo(t *testing.T, branch string) {
	t.Helper()

	cmds := [][]string{
		{"git", "init", "-b", branch},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
		{"git", "commit", "--allow-empty", "-m", "init"},
	}

	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		require.NoError(t, cmd.Run())
	}
}

func TestDetectDefaultBranch(t *testing.T) {
	t.Run("detects branch from origin HEAD", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "develop")

		cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/custom")
		require.NoError(t, cmd.Run())

		branch, err := DetectDefaultBranch()
		require.NoError(t, err)
		assert.Equal(t, "custom", branch)
	})

	t.Run("falls back to main branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "main")

		branch, err := DetectDefaultBranch()
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("falls back to master branch", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "master")

		branch, err := DetectDefaultBranch()
		require.NoError(t, err)
		assert.Equal(t, "master", branch)
	})

	t.Run("prefers main over master", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "main")

		cmd := exec.Command("git", "branch", "master")
		require.NoError(t, cmd.Run())

		branch, err := DetectDefaultBranch()
		require.NoError(t, err)
		assert.Equal(t, "main", branch)
	})

	t.Run("returns error when no default branch found", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "develop")

		_, err := DetectDefaultBranch()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "could not detect default branch")
	})
}
