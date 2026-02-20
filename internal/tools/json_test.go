package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteJSONFile(t *testing.T) {
	t.Run("writes struct to json file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.json")

		data := struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}{
			Name:  "test",
			Value: 42,
		}

		err := WriteJSONFile(filePath, data)
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Contains(t, string(content), `"name": "test"`)
		assert.Contains(t, string(content), `"value": 42`)
	})

	t.Run("writes map to json file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "map.json")

		data := map[string]any{
			"key1": "value1",
			"key2": 123,
		}

		err := WriteJSONFile(filePath, data)
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Contains(t, string(content), `"key1": "value1"`)
		assert.Contains(t, string(content), `"key2": 123`)
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		err := WriteJSONFile("/nonexistent/dir/test.json", data)
		assert.Error(t, err)
	})

	t.Run("ends with newline", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "newline.json")

		data := map[string]string{"key": "value"}

		err := WriteJSONFile(filePath, data)
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		require.True(t, len(content) > 0)
		assert.Equal(t, byte('\n'), content[len(content)-1])
	})

	t.Run("uses four-space indentation", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "indent.json")

		data := struct {
			Name string `json:"name"`
		}{Name: "test"}

		err := WriteJSONFile(filePath, data)
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Contains(t, string(content), "    \"name\"")
	})

	t.Run("returns error for unmarshalable data", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "bad.json")

		err := WriteJSONFile(filePath, make(chan int))
		assert.Error(t, err)
	})
}
