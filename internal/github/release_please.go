package github

func GetReleasePleaseWorkflow(branch string) Workflow {
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
