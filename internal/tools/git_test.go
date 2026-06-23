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
		{"git", "config", "core.hooksPath", "/dev/null"},
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

	t.Run("falls back to current branch name on empty repo", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cmd := exec.Command("git", "init", "-b", "main")
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

func TestDetectGitHubRepo(t *testing.T) {
	t.Run("detects repo from HTTPS origin", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "main")

		cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/myowner/myrepo.git")
		require.NoError(t, cmd.Run())

		repo, err := DetectGitHubRepo()
		require.NoError(t, err)
		assert.Equal(t, "myowner", repo.Owner)
		assert.Equal(t, "myrepo", repo.Name)
	})

	t.Run("detects repo from SSH origin", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "main")

		cmd := exec.Command("git", "remote", "add", "origin", "git@github.com:myowner/myrepo.git")
		require.NoError(t, cmd.Run())

		repo, err := DetectGitHubRepo()
		require.NoError(t, err)
		assert.Equal(t, "myowner", repo.Owner)
		assert.Equal(t, "myrepo", repo.Name)
	})

	t.Run("handles HTTPS URL without .git suffix", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "main")

		cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/myowner/myrepo")
		require.NoError(t, cmd.Run())

		repo, err := DetectGitHubRepo()
		require.NoError(t, err)
		assert.Equal(t, "myowner", repo.Owner)
		assert.Equal(t, "myrepo", repo.Name)
	})

	t.Run("returns error without origin remote", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "main")

		_, err := DetectGitHubRepo()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no origin remote")
	})

	t.Run("returns error for non-GitHub remote", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		initGitRepo(t, "main")

		cmd := exec.Command("git", "remote", "add", "origin", "https://gitlab.com/owner/repo.git")
		require.NoError(t, cmd.Run())

		_, err := DetectGitHubRepo()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "could not parse GitHub repository")
	})
}

func Test_parseGitHubURL(t *testing.T) {
	t.Run("parses HTTPS URL", func(t *testing.T) {
		owner, name, ok := parseGitHubURL("https://github.com/foo/bar.git")
		assert.True(t, ok)
		assert.Equal(t, "foo", owner)
		assert.Equal(t, "bar", name)
	})

	t.Run("parses SSH URL", func(t *testing.T) {
		owner, name, ok := parseGitHubURL("git@github.com:foo/bar.git")
		assert.True(t, ok)
		assert.Equal(t, "foo", owner)
		assert.Equal(t, "bar", name)
	})

	t.Run("rejects non-GitHub URL", func(t *testing.T) {
		_, _, ok := parseGitHubURL("https://gitlab.com/foo/bar.git")
		assert.False(t, ok)
	})

	t.Run("rejects malformed path", func(t *testing.T) {
		_, _, ok := parseGitHubURL("https://github.com/foo")
		assert.False(t, ok)
	})
}
