package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReleasePleaseWorkflow(t *testing.T) {
	t.Run("returns workflow with correct name", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", false)
		assert.Equal(t, "release-please", result.Name)
	})

	t.Run("uses provided branch", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", false)
		require.NotNil(t, result.On.Push)
		require.Len(t, result.On.Push.Branches, 1)
		assert.Equal(t, "main", result.On.Push.Branches[0])
	})

	t.Run("uses master branch when specified", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("master", false)
		require.NotNil(t, result.On.Push)
		assert.Equal(t, "master", result.On.Push.Branches[0])
	})

	t.Run("sets correct permissions", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", false)
		assert.Equal(t, "write", result.Permissions.Contents)
		assert.Equal(t, "write", result.Permissions.Issues)
		assert.Equal(t, "write", result.Permissions.PullRequests)
	})

	t.Run("contains release-please job", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", false)
		job, ok := result.Jobs["release-please"]
		require.True(t, ok)
		assert.Equal(t, "Creating release", job.Name)
		assert.Equal(t, "ubuntu-latest", job.RunsOn)
	})

	t.Run("exposes job outputs", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", false)
		job := result.Jobs["release-please"]
		assert.Equal(t, "${{ steps.release.outputs.release_created }}", job.Outputs["release_created"])
		assert.Equal(t, "${{ steps.release.outputs.tag_name }}", job.Outputs["tag_name"])
	})

	t.Run("configures release-please action step", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", false)
		job := result.Jobs["release-please"]
		require.Len(t, job.Steps, 1)

		step := job.Steps[0]
		assert.Equal(t, "googleapis/release-please-action@v4", step.Uses)
		assert.Equal(t, "release", step.ID)
		assert.Equal(t, "${{ secrets.GITHUB_TOKEN }}", step.With["token"])
		assert.Equal(t, ".github/release-please-config.json", step.With["config-file"])
		assert.Equal(t, ".github/release-please-manifest.json", step.With["manifest-file"])
	})

	t.Run("does not include goreleaser job by default", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", false)
		_, ok := result.Jobs["goreleaser"]
		assert.False(t, ok)
		assert.Len(t, result.Jobs, 1)
	})

	t.Run("includes goreleaser job when enabled", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", true)
		job, ok := result.Jobs["goreleaser"]
		require.True(t, ok)
		assert.Equal(t, "Build and release packages", job.Name)
		assert.Equal(t, "ubuntu-latest", job.RunsOn)
		assert.Equal(t, []string{"release-please"}, job.Needs)
		assert.Equal(t, "${{ needs.release-please.outputs.release_created }}", job.If)
	})

	t.Run("goreleaser job has correct steps", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", true)
		job := result.Jobs["goreleaser"]
		require.Len(t, job.Steps, 3)

		assert.Equal(t, "Checkout code", job.Steps[0].Name)
		assert.Equal(t, "actions/checkout@v6", job.Steps[0].Uses)
		assert.Equal(t, "0", job.Steps[0].With["fetch-depth"])

		assert.Equal(t, "Set up Go", job.Steps[1].Name)
		assert.Equal(t, "actions/setup-go@v6", job.Steps[1].Uses)
		assert.Equal(t, "go.mod", job.Steps[1].With["go-version-file"])

		assert.Equal(t, "Run GoReleaser", job.Steps[2].Name)
		assert.Equal(t, "goreleaser/goreleaser-action@v6", job.Steps[2].Uses)
		assert.Equal(t, "goreleaser", job.Steps[2].With["distribution"])
		assert.Equal(t, "~> v2", job.Steps[2].With["version"])
		assert.Equal(t, "release --clean", job.Steps[2].With["args"])
		assert.Equal(t, "${{ secrets.GITHUB_TOKEN }}", job.Steps[2].Env["GITHUB_TOKEN"])
		assert.Equal(t, "${{ needs.release-please.outputs.tag_name }}", job.Steps[2].Env["TAG"])
	})
}

func TestGetReleasePleaseConfig(t *testing.T) {
	t.Run("returns config with correct release type", func(t *testing.T) {
		result := GetReleasePleaseConfig()
		assert.Equal(t, "simple", result.ReleaseType)
	})

	t.Run("prerelease is disabled", func(t *testing.T) {
		result := GetReleasePleaseConfig()
		assert.False(t, result.Prerelease)
	})

	t.Run("contains root package", func(t *testing.T) {
		result := GetReleasePleaseConfig()
		_, ok := result.Packages["."]
		assert.True(t, ok)
	})
}

func TestGetReleasePleaseManifest(t *testing.T) {
	t.Run("returns manifest with initial version", func(t *testing.T) {
		result := GetReleasePleaseManifest()
		assert.Equal(t, "0.0.1", result["."])
	})

	t.Run("contains only root entry", func(t *testing.T) {
		result := GetReleasePleaseManifest()
		assert.Len(t, result, 1)
	})
}
