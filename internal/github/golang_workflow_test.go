package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetGolangWorkflow(t *testing.T) {
	t.Run("returns workflow with correct name", func(t *testing.T) {
		result := GetGolangWorkflow()
		assert.Equal(t, "golang", result.Name)
	})

	t.Run("enables workflow dispatch", func(t *testing.T) {
		result := GetGolangWorkflow()
		assert.NotNil(t, result.On.WorkflowDispatch)
	})

	t.Run("configures merge group trigger", func(t *testing.T) {
		result := GetGolangWorkflow()
		require.NotNil(t, result.On.MergeGroup)
		assert.Equal(t, []string{"checks_requested"}, result.On.MergeGroup.Types)
	})

	t.Run("configures pull request trigger", func(t *testing.T) {
		result := GetGolangWorkflow()
		require.NotNil(t, result.On.PullRequest)
		assert.Equal(t, []string{"opened", "synchronize", "reopened"}, result.On.PullRequest.Types)
		assert.Equal(t, []string{"**.go", "go.mod", "go.sum"}, result.On.PullRequest.Paths)
	})

	t.Run("configures push trigger with paths", func(t *testing.T) {
		result := GetGolangWorkflow()
		require.NotNil(t, result.On.Push)
		assert.Empty(t, result.On.Push.Branches)
		assert.Equal(t, []string{"**.go", "go.mod", "go.sum"}, result.On.Push.Paths)
	})

	t.Run("contains linter job", func(t *testing.T) {
		result := GetGolangWorkflow()
		job, ok := result.Jobs.Get("linter")
		require.True(t, ok)
		assert.Equal(t, "ubuntu-latest", job.RunsOn)
		require.Len(t, job.Steps, 3)
		assert.Equal(t, "actions/checkout@v6", job.Steps[0].Uses)
		assert.Equal(t, "actions/setup-go@v6", job.Steps[1].Uses)
		assert.Equal(t, "go.mod", job.Steps[1].With["go-version-file"])
		assert.Equal(t, "golangci-lint", job.Steps[2].Name)
		assert.Equal(t, "golangci/golangci-lint-action@v9", job.Steps[2].Uses)
		assert.Equal(t, "--timeout=5m", job.Steps[2].With["args"])
	})

	t.Run("contains tests job", func(t *testing.T) {
		result := GetGolangWorkflow()
		job, ok := result.Jobs.Get("tests")
		require.True(t, ok)
		assert.Equal(t, "ubuntu-latest", job.RunsOn)
		assert.Empty(t, job.Env)
		require.Len(t, job.Steps, 6)
		assert.Equal(t, "actions/checkout@v6", job.Steps[0].Uses)
		assert.Equal(t, "actions/setup-go@v6", job.Steps[1].Uses)
		assert.Equal(t, "Test", job.Steps[2].Name)
		assert.Contains(t, job.Steps[2].Run, "coverprofile")
		assert.Equal(t, "Test Race", job.Steps[3].Name)
		assert.Contains(t, job.Steps[3].Run, "-race")
		assert.Equal(t, "Check codecov token", job.Steps[4].Name)
		assert.Equal(t, "check-codecov", job.Steps[4].ID)
		assert.Equal(t, "${{ secrets.CODECOV_TOKEN }}", job.Steps[4].Env["CODECOV_TOKEN"])
		assert.Contains(t, job.Steps[4].Run, "GITHUB_OUTPUT")
		assert.Equal(t, "Publish coverage", job.Steps[5].Name)
		assert.Equal(t, "steps.check-codecov.outputs.available == 'true'", job.Steps[5].If)
		assert.Equal(t, "codecov/codecov-action@v6", job.Steps[5].Uses)
		assert.Equal(t, "${{ secrets.CODECOV_TOKEN }}", job.Steps[5].Env["CODECOV_TOKEN"])
		assert.Equal(t, "./coverage.txt", job.Steps[5].With["files"])
	})

	t.Run("configures concurrency", func(t *testing.T) {
		result := GetGolangWorkflow()
		require.NotNil(t, result.Concurrency)
		assert.Equal(t, `go-${{ github.event.number || github.ref }}`, result.Concurrency.Group)
		assert.Equal(t, `${{ github.event.action != 'merge_group' }}`, result.Concurrency.CancelInProgress)
	})

	t.Run("has no permissions set", func(t *testing.T) {
		result := GetGolangWorkflow()
		assert.Empty(t, result.Permissions.Contents)
		assert.Empty(t, result.Permissions.Issues)
		assert.Empty(t, result.Permissions.PullRequests)
	})

	t.Run("marshals to valid yaml", func(t *testing.T) {
		result := GetGolangWorkflow()
		data, err := result.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "name: golang")
		assert.Contains(t, content, "workflow_dispatch: {}")
		assert.Contains(t, content, "pull_request:")
		assert.Contains(t, content, "golangci-lint")
		assert.Contains(t, content, "codecov")
	})
}
