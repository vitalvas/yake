package github

import "os"

func GetReleasePleaseWorkflow(branch string, goreleaser bool) Workflow {
	jobs := OrderedJobs{
		{
			Name: "release-please",
			Job: WorkflowJob{
				Name:   "Creating release",
				RunsOn: "ubuntu-latest",
				Outputs: map[string]string{
					"release_created": "${{ steps.release.outputs.release_created }}",
					"tag_name":        "${{ steps.release.outputs.tag_name }}",
				},
				Steps: []WorkflowStep{
					{
						Uses: "googleapis/release-please-action@v4",
						ID:   "release",
						With: map[string]string{
							"token":         "${{ secrets.GITHUB_TOKEN }}",
							"config-file":   ".github/release-please-config.json",
							"manifest-file": ".github/release-please-manifest.json",
						},
					},
				},
			},
		},
	}

	if goreleaser {
		jobs = append(jobs, JobEntry{
			Name: "goreleaser",
			Job: WorkflowJob{
				Name:   "Build and release packages",
				RunsOn: "ubuntu-latest",
				Needs:  []string{"release-please"},
				Steps:  goreleaserSteps(),
			},
		})
	}

	return Workflow{
		Name: "release-please",
		On: WorkflowOn{
			Push: &WorkflowTrigger{
				Branches: []string{branch},
			},
		},
		Permissions: WorkflowPermissions{
			Contents:     "write",
			Issues:       "write",
			PullRequests: "write",
		},
		Jobs: jobs,
	}
}

func goreleaserSteps() []WorkflowStep {
	steps := []WorkflowStep{
		{
			Name: "Checkout code",
			Uses: "actions/checkout@v6",
			With: map[string]string{
				"fetch-depth": "0",
			},
		},
	}

	if _, err := os.Stat("go.mod"); err == nil {
		steps = append(steps, WorkflowStep{
			Name: "Set up Go",
			Uses: "actions/setup-go@v6",
			With: map[string]string{
				"go-version-file": "go.mod",
			},
		})
	}

	steps = append(steps,
		WorkflowStep{
			Name: "Test GoReleaser",
			If:   "${{ needs.release-please.outputs.release_created != 'true' }}",
			Uses: "goreleaser/goreleaser-action@v6",
			With: map[string]string{
				"distribution": "goreleaser",
				"version":      "~> v2",
				"args":         "release --clean --snapshot",
			},
		},
		WorkflowStep{
			Name: "Run GoReleaser",
			If:   "${{ needs.release-please.outputs.release_created }}",
			Uses: "goreleaser/goreleaser-action@v6",
			With: map[string]string{
				"distribution": "goreleaser",
				"version":      "~> v2",
				"args":         "release --clean",
			},
			Env: map[string]string{
				"GITHUB_TOKEN": "${{ secrets.GITHUB_TOKEN }}",
				"TAG":          "${{ needs.release-please.outputs.tag_name }}",
			},
		},
	)

	return steps
}

type ReleasePleaseConfig struct {
	ReleaseType string                          `json:"release-type"`
	Prerelease  bool                            `json:"prerelease"`
	Packages    map[string]ReleasePleasePackage `json:"packages"`
}

type ReleasePleasePackage struct{}

func GetReleasePleaseConfig() ReleasePleaseConfig {
	return ReleasePleaseConfig{
		ReleaseType: "simple",
		Prerelease:  false,
		Packages: map[string]ReleasePleasePackage{
			".": {},
		},
	}
}

func GetReleasePleaseManifest() map[string]string {
	return map[string]string{
		".": "0.0.1",
	}
}
