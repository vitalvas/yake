package github

func GetReleasePleaseWorkflow(branch string, goreleaser bool) Workflow {
	workflow := Workflow{
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
		Jobs: map[string]WorkflowJob{
			"release-please": {
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
		workflow.Jobs["goreleaser"] = WorkflowJob{
			Name:   "Build and release packages",
			RunsOn: "ubuntu-latest",
			Needs:  []string{"release-please"},
			If:     "${{ needs.release-please.outputs.release_created }}",
			Steps: []WorkflowStep{
				{
					Name: "Checkout code",
					Uses: "actions/checkout@v6",
					With: map[string]string{
						"fetch-depth": "0",
					},
				},
				{
					Name: "Set up Go",
					Uses: "actions/setup-go@v6",
					With: map[string]string{
						"go-version-file": "go.mod",
					},
				},
				{
					Name: "Run GoReleaser",
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
			},
		}
	}

	return workflow
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
