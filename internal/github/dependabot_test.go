package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGithub(t *testing.T) {
	t.Run("returns base config with github-actions", func(t *testing.T) {
		result := GetGithub("")

		assert.Equal(t, 2, result.Version)
		require.Len(t, result.Updates, 1)
		assert.Equal(t, "github-actions", result.Updates[0].PackageEcosystem)
		assert.Equal(t, "/", result.Updates[0].Directory)
		assert.Equal(t, "monthly", result.Updates[0].Schedule.Interval)
	})

	t.Run("includes gomod for golang", func(t *testing.T) {
		result := GetGithub(Golang)

		assert.Equal(t, 2, result.Version)
		require.Len(t, result.Updates, 2)

		assert.Equal(t, "github-actions", result.Updates[0].PackageEcosystem)
		assert.Equal(t, "gomod", result.Updates[1].PackageEcosystem)
		assert.Equal(t, "/", result.Updates[1].Directory)
		assert.Equal(t, "monthly", result.Updates[1].Schedule.Interval)
		assert.Contains(t, result.Updates[1].Groups, "dependencies")
	})

	t.Run("sets reviewers and assignees", func(t *testing.T) {
		result := GetGithub(Golang)

		for _, update := range result.Updates {
			assert.Contains(t, update.Reviewers, "vitalvas")
			assert.Contains(t, update.Assignees, "vitalvas")
		}
	})
}

func TestLangConstant(t *testing.T) {
	t.Run("golang constant value", func(t *testing.T) {
		assert.Equal(t, Lang("go"), Golang)
	})
}
