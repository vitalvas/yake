package linter

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGolangCI(t *testing.T) {
	t.Run("returns valid config structure", func(t *testing.T) {
		result := GetGolangCI()

		assert.Equal(t, "2", result.Version)
		assert.Equal(t, "none", result.Linters.Default)
	})

	t.Run("contains expected linters", func(t *testing.T) {
		result := GetGolangCI()

		expectedLinters := []string{
			"govet",
			"staticcheck",
			"ineffassign",
			"unused",
			"revive",
		}

		for _, linter := range expectedLinters {
			assert.Contains(t, result.Linters.Enable, linter)
		}
	})

	t.Run("linters are sorted", func(t *testing.T) {
		result := GetGolangCI()

		assert.True(t, slices.IsSorted(result.Linters.Enable))
	})

	t.Run("exclusion presets are sorted", func(t *testing.T) {
		result := GetGolangCI()

		assert.True(t, slices.IsSorted(result.Linters.Exclusions.Presets))
	})

	t.Run("exclusion paths are sorted", func(t *testing.T) {
		result := GetGolangCI()

		assert.True(t, slices.IsSorted(result.Linters.Exclusions.Paths))
		assert.True(t, slices.IsSorted(result.Formatters.Exclusions.Paths))
	})

	t.Run("has gosec exclusions", func(t *testing.T) {
		result := GetGolangCI()

		require.NotEmpty(t, result.Linters.Settings.GoSec.Excludes)
		assert.Contains(t, result.Linters.Settings.GoSec.Excludes, "G402")
	})

	t.Run("has exclusion presets", func(t *testing.T) {
		result := GetGolangCI()

		expectedPresets := []string{
			"comments",
			"common-false-positives",
			"legacy",
			"std-error-handling",
		}

		for _, preset := range expectedPresets {
			assert.Contains(t, result.Linters.Exclusions.Presets, preset)
		}
	})

	t.Run("has exclusion paths", func(t *testing.T) {
		result := GetGolangCI()

		expectedPaths := []string{
			"third_party$",
			"vendor$",
			"builtin$",
			"examples$",
		}

		for _, path := range expectedPaths {
			assert.Contains(t, result.Linters.Exclusions.Paths, path)
		}
	})

	t.Run("formatters have same exclusion paths as linters", func(t *testing.T) {
		result := GetGolangCI()

		assert.Equal(t, result.Linters.Exclusions.Paths, result.Formatters.Exclusions.Paths)
	})

	t.Run("generated exclusion is set to lax", func(t *testing.T) {
		result := GetGolangCI()

		assert.Equal(t, "lax", result.Linters.Exclusions.Generated)
		assert.Equal(t, "lax", result.Formatters.Exclusions.Generated)
	})
}
