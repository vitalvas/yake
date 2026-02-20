package github

func GetGolangWorkflow() Workflow {
	goPaths := []string{"**.go", "go.mod", "go.sum"}

	return Workflow{
		Name: "golang",
		On: WorkflowOn{
			WorkflowDispatch: &struct{}{},
			PullRequest: &WorkflowTrigger{
				Types: []string{"opened", "synchronize", "reopened"},
				Paths: goPaths,
			},
			Push: &WorkflowTrigger{
				Paths: goPaths,
			},
		},
		Jobs: map[string]WorkflowJob{
			"linter": {
				RunsOn: "ubuntu-latest",
				Steps: []WorkflowStep{
					{Uses: "actions/checkout@v4"},
					{
						Uses: "actions/setup-go@v5",
						With: map[string]string{"go-version-file": "go.mod"},
					},
					{
						Name: "golangci-lint",
						Uses: "golangci/golangci-lint-action@v8",
						With: map[string]string{"args": "--timeout=5m"},
					},
				},
			},
			"tests": {
				RunsOn: "ubuntu-latest",
				Steps: []WorkflowStep{
					{Uses: "actions/checkout@v4"},
					{
						Uses: "actions/setup-go@v5",
						With: map[string]string{"go-version-file": "go.mod"},
					},
					{
						Name: "Test",
						Run:  "go test -coverprofile=coverage.txt -covermode=atomic ./...",
					},
					{
						Name: "Test Race",
						Run:  "go test -race ./...",
					},
					{
						Name: "Publish coverage",
						Uses: "codecov/codecov-action@v5",
						Env:  map[string]string{"CODECOV_TOKEN": "${{ secrets.CODECOV_TOKEN }}"},
						With: map[string]string{"files": "./coverage.txt"},
					},
				},
			},
		},
	}
}
