package github

func GetGolangWorkflow() Workflow {
	goPaths := []string{"**.go", "go.mod", "go.sum"}

	return Workflow{
		Name: "golang",
		On: WorkflowOn{
			WorkflowDispatch: &struct{}{},
			MergeGroup: &WorkflowTrigger{
				Types: []string{"checks_requested"},
			},
			PullRequest: &WorkflowTrigger{
				Types: []string{"opened", "synchronize", "reopened"},
				Paths: goPaths,
			},
			Push: &WorkflowTrigger{
				Paths: goPaths,
			},
		},
		Concurrency: &WorkflowConcurrency{
			Group:            `go-${{ github.event.number || github.ref }}`,
			CancelInProgress: `${{ github.event.action != 'merge_group' }}`,
		},
		Jobs: OrderedJobs{
			{
				Name: "linter",
				Job: WorkflowJob{
					RunsOn: "ubuntu-latest",
					Steps: []WorkflowStep{
						{Uses: "actions/checkout@v7"},
						{
							Uses: "actions/setup-go@v6",
							With: map[string]string{"go-version-file": "go.mod"},
						},
						{
							Name: "golangci-lint",
							Uses: "golangci/golangci-lint-action@v9",
							With: map[string]string{"args": "--timeout=5m --output.text.path=lint-report.txt"},
						},
						{
							Name: "golangci-lint output",
							If:   "always()",
							Run:  "cat lint-report.txt\ncat lint-report.txt >> \"$GITHUB_STEP_SUMMARY\"",
						},
					},
				},
			},
			{
				Name: "tests",
				Job: WorkflowJob{
					RunsOn: "ubuntu-latest",
					Steps: []WorkflowStep{
						{Uses: "actions/checkout@v7"},
						{
							Uses: "actions/setup-go@v6",
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
							Name: "Check codecov token",
							ID:   "check-codecov",
							Env:  map[string]string{"CODECOV_TOKEN": "${{ secrets.CODECOV_TOKEN }}"},
							Run:  `if [ -n "$CODECOV_TOKEN" ]; then echo "available=true" >> "$GITHUB_OUTPUT"; fi`,
						},
						{
							Name: "Publish coverage",
							If:   "steps.check-codecov.outputs.available == 'true'",
							Uses: "codecov/codecov-action@v7",
							Env:  map[string]string{"CODECOV_TOKEN": "${{ secrets.CODECOV_TOKEN }}"},
							With: map[string]string{"files": "./coverage.txt"},
						},
					},
				},
			},
		},
	}
}
