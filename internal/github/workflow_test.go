package github

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWorkflowMarshal(t *testing.T) {
	t.Run("produces unquoted on key", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: map[string]WorkflowJob{
				"build": {
					Name:   "Build",
					RunsOn: "ubuntu-latest",
					Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "\non:\n")
		assert.NotContains(t, content, `"on":`)
	})

	t.Run("marshals workflow to valid yaml", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On: WorkflowOn{
				Push: &WorkflowTrigger{Branches: []string{"main"}},
			},
			Permissions: WorkflowPermissions{Contents: "read"},
			Jobs: map[string]WorkflowJob{
				"build": {
					Name:   "Build",
					RunsOn: "ubuntu-latest",
					Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "name: test")
		assert.Contains(t, content, "- main")
		assert.Contains(t, content, "contents: read")
		assert.Contains(t, content, "uses: actions/checkout@v4")
	})

	t.Run("omits empty permissions", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: map[string]WorkflowJob{
				"build": {
					Name:   "Build",
					RunsOn: "ubuntu-latest",
					Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)
		assert.NotContains(t, string(data), "permissions")
	})

	t.Run("omits nil triggers", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: map[string]WorkflowJob{
				"build": {
					Name:   "Build",
					RunsOn: "ubuntu-latest",
					Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)
		assert.NotContains(t, string(data), "pull_request")
	})

	t.Run("includes step fields when set", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: map[string]WorkflowJob{
				"test": {
					Name:   "Test",
					RunsOn: "ubuntu-latest",
					Steps: []WorkflowStep{
						{
							Name: "Run tests",
							ID:   "test",
							Run:  "go test ./...",
							With: map[string]string{"key": "value"},
						},
					},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "name: Run tests")
		assert.Contains(t, content, "id: test")
		assert.Contains(t, content, "run: go test ./...")
		assert.Contains(t, content, "key: value")
	})

	t.Run("omits empty step fields", func(t *testing.T) {
		step := WorkflowStep{Uses: "actions/checkout@v4"}

		data, err := yaml.Marshal(step)
		require.NoError(t, err)

		content := string(data)
		assert.NotContains(t, content, "name:")
		assert.NotContains(t, content, "id:")
		assert.NotContains(t, content, "run:")
		assert.NotContains(t, content, "with:")
	})

	t.Run("renders workflow_dispatch as empty mapping", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On: WorkflowOn{
				WorkflowDispatch: &struct{}{},
				Push:             &WorkflowTrigger{Branches: []string{"main"}},
			},
			Jobs: map[string]WorkflowJob{
				"build": {
					Name:   "Build",
					RunsOn: "ubuntu-latest",
					Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)
		assert.Contains(t, string(data), "workflow_dispatch: {}")
	})

	t.Run("includes trigger types and paths", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On: WorkflowOn{
				PullRequest: &WorkflowTrigger{
					Types: []string{"opened", "synchronize"},
					Paths: []string{"**.go", "go.mod"},
				},
			},
			Jobs: map[string]WorkflowJob{
				"build": {
					RunsOn: "ubuntu-latest",
					Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "- opened")
		assert.Contains(t, content, "- synchronize")
		assert.Contains(t, content, "go.mod")
	})

	t.Run("includes step env", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: map[string]WorkflowJob{
				"test": {
					RunsOn: "ubuntu-latest",
					Steps: []WorkflowStep{
						{
							Name: "Upload",
							Uses: "codecov/codecov-action@v5",
							Env:  map[string]string{"TOKEN": "secret"},
							With: map[string]string{"files": "coverage.txt"},
						},
					},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "TOKEN: secret")
	})

	t.Run("omits job name when empty", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: map[string]WorkflowJob{
				"build": {
					RunsOn: "ubuntu-latest",
					Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
				},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)
		assert.NotContains(t, string(data), "name: \"\"")
	})

	t.Run("roundtrips through yaml", func(t *testing.T) {
		original := Workflow{
			Name: "ci",
			On: WorkflowOn{
				Push:        &WorkflowTrigger{Branches: []string{"main"}},
				PullRequest: &WorkflowTrigger{Branches: []string{"main"}},
			},
			Permissions: WorkflowPermissions{
				Contents:     "read",
				PullRequests: "write",
			},
			Jobs: map[string]WorkflowJob{
				"build": {
					Name:   "Build",
					RunsOn: "ubuntu-latest",
					Steps: []WorkflowStep{
						{Uses: "actions/checkout@v4"},
						{Name: "Test", Run: "go test ./..."},
					},
				},
			},
		}

		data, err := original.Marshal()
		require.NoError(t, err)

		var decoded Workflow
		err = yaml.Unmarshal(data, &decoded)
		require.NoError(t, err)

		assert.Equal(t, original.Name, decoded.Name)
		assert.Equal(t, original.On.Push.Branches, decoded.On.Push.Branches)
		assert.Equal(t, original.On.PullRequest.Branches, decoded.On.PullRequest.Branches)
		assert.Equal(t, original.Permissions, decoded.Permissions)

		job := decoded.Jobs["build"]
		assert.Equal(t, "Build", job.Name)
		assert.Equal(t, "ubuntu-latest", job.RunsOn)
		require.Len(t, job.Steps, 2)
	})
}
