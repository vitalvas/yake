package github

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func newSingleJob(name string, job WorkflowJob) OrderedJobs {
	return OrderedJobs{{Name: name, Job: job}}
}

func TestWorkflowMarshal(t *testing.T) {
	t.Run("produces unquoted on key", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: newSingleJob("build", WorkflowJob{
				Name:   "Build",
				RunsOn: "ubuntu-latest",
				Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
			}),
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
			Jobs: newSingleJob("build", WorkflowJob{
				Name:   "Build",
				RunsOn: "ubuntu-latest",
				Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
			}),
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
			Jobs: newSingleJob("build", WorkflowJob{
				Name:   "Build",
				RunsOn: "ubuntu-latest",
				Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
			}),
		}

		data, err := w.Marshal()
		require.NoError(t, err)
		assert.NotContains(t, string(data), "permissions")
	})

	t.Run("omits nil triggers", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: newSingleJob("build", WorkflowJob{
				Name:   "Build",
				RunsOn: "ubuntu-latest",
				Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
			}),
		}

		data, err := w.Marshal()
		require.NoError(t, err)
		assert.NotContains(t, string(data), "pull_request")
	})

	t.Run("includes step fields when set", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: newSingleJob("test", WorkflowJob{
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
			}),
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
			Jobs: newSingleJob("build", WorkflowJob{
				Name:   "Build",
				RunsOn: "ubuntu-latest",
				Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
			}),
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
			Jobs: newSingleJob("build", WorkflowJob{
				RunsOn: "ubuntu-latest",
				Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
			}),
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
			Jobs: newSingleJob("test", WorkflowJob{
				RunsOn: "ubuntu-latest",
				Steps: []WorkflowStep{
					{
						Name: "Upload",
						Uses: "codecov/codecov-action@v5",
						Env:  map[string]string{"TOKEN": "secret"},
						With: map[string]string{"files": "coverage.txt"},
					},
				},
			}),
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
			Jobs: newSingleJob("build", WorkflowJob{
				RunsOn: "ubuntu-latest",
				Steps:  []WorkflowStep{{Uses: "actions/checkout@v4"}},
			}),
		}

		data, err := w.Marshal()
		require.NoError(t, err)
		assert.NotContains(t, string(data), "name: \"\"")
	})

	t.Run("preserves job order", func(t *testing.T) {
		w := Workflow{
			Name: "test",
			On:   WorkflowOn{Push: &WorkflowTrigger{Branches: []string{"main"}}},
			Jobs: OrderedJobs{
				{Name: "first", Job: WorkflowJob{RunsOn: "ubuntu-latest", Steps: []WorkflowStep{{Uses: "actions/checkout@v4"}}}},
				{Name: "second", Job: WorkflowJob{RunsOn: "ubuntu-latest", Steps: []WorkflowStep{{Uses: "actions/checkout@v4"}}}},
			},
		}

		data, err := w.Marshal()
		require.NoError(t, err)

		content := string(data)
		firstIdx := bytes.Index([]byte(content), []byte("first:"))
		secondIdx := bytes.Index([]byte(content), []byte("second:"))
		assert.Less(t, firstIdx, secondIdx)
	})
}
