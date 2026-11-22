package tools

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureGitignoreEntries(t *testing.T) {
	t.Run("creates file when missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, EnsureGitignoreEntries("/_dist_build"))

		content, err := os.ReadFile(gitignoreFile)
		require.NoError(t, err)
		assert.Equal(t, "/_dist_build\n", string(content))
	})

	t.Run("appends missing entries", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		require.NoError(t, os.WriteFile(gitignoreFile, []byte("/existing\n"), 0644))

		require.NoError(t, EnsureGitignoreEntries("/_dist_build", "/target"))

		content, err := os.ReadFile(gitignoreFile)
		require.NoError(t, err)
		assert.Equal(t, "/existing\n/_dist_build\n/target\n", string(content))
	})

	t.Run("does not duplicate existing entries", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		require.NoError(t, os.WriteFile(gitignoreFile, []byte("/_dist_build\n/target\n"), 0644))

		require.NoError(t, EnsureGitignoreEntries("/_dist_build", "/target"))

		content, err := os.ReadFile(gitignoreFile)
		require.NoError(t, err)
		assert.Equal(t, "/_dist_build\n/target\n", string(content))
	})

	t.Run("adds trailing newline before appending", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		require.NoError(t, os.WriteFile(gitignoreFile, []byte("/existing"), 0644))

		require.NoError(t, EnsureGitignoreEntries("/_dist_build"))

		content, err := os.ReadFile(gitignoreFile)
		require.NoError(t, err)
		assert.Equal(t, "/existing\n/_dist_build\n", string(content))
	})

	t.Run("appends only the missing subset", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		require.NoError(t, os.WriteFile(gitignoreFile, []byte("/target\n"), 0644))

		require.NoError(t, EnsureGitignoreEntries("/target", "/.zig-cache"))

		content, err := os.ReadFile(gitignoreFile)
		require.NoError(t, err)
		assert.Equal(t, "/target\n/.zig-cache\n", string(content))
	})

	t.Run("ignores surrounding whitespace when matching", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		require.NoError(t, os.WriteFile(gitignoreFile, []byte("  /target  \n"), 0644))

		require.NoError(t, EnsureGitignoreEntries("/target"))

		content, err := os.ReadFile(gitignoreFile)
		require.NoError(t, err)
		assert.Equal(t, "  /target  \n", string(content))
	})
}
