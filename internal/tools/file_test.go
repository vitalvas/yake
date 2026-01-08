package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteStringToFile(t *testing.T) {
	t.Run("writes content to file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")

		err := WriteStringToFile(filePath, "hello world")
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, "hello world", string(content))
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")

		err := WriteStringToFile(filePath, "first")
		require.NoError(t, err)

		err = WriteStringToFile(filePath, "second")
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Equal(t, "second", string(content))
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		err := WriteStringToFile("/nonexistent/dir/test.txt", "content")
		assert.Error(t, err)
	})

	t.Run("writes empty string", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "empty.txt")

		err := WriteStringToFile(filePath, "")
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Empty(t, string(content))
	})
}
