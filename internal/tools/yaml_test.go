package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteYamlFile(t *testing.T) {
	t.Run("writes struct to yaml file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.yaml")

		data := struct {
			Name  string `yaml:"name"`
			Value int    `yaml:"value"`
		}{
			Name:  "test",
			Value: 42,
		}

		err := WriteYamlFile(filePath, data)
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Contains(t, string(content), "name: test")
		assert.Contains(t, string(content), "value: 42")
	})

	t.Run("writes map to yaml file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "map.yaml")

		data := map[string]any{
			"key1": "value1",
			"key2": 123,
		}

		err := WriteYamlFile(filePath, data)
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Contains(t, string(content), "key1: value1")
		assert.Contains(t, string(content), "key2: 123")
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		data := map[string]string{"key": "value"}
		err := WriteYamlFile("/nonexistent/dir/test.yaml", data)
		assert.Error(t, err)
	})

	t.Run("writes nested structure", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "nested.yaml")

		data := struct {
			Parent struct {
				Child string `yaml:"child"`
			} `yaml:"parent"`
		}{
			Parent: struct {
				Child string `yaml:"child"`
			}{
				Child: "nested_value",
			},
		}

		err := WriteYamlFile(filePath, data)
		require.NoError(t, err)

		content, readErr := os.ReadFile(filePath)
		require.NoError(t, readErr)
		assert.Contains(t, string(content), "parent:")
		assert.Contains(t, string(content), "child: nested_value")
	})
}
