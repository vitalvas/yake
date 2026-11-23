package github

import (
	"fmt"
	"os"
	"strings"
)

func GetReleasePleaseWorkflow(branch, dist string, goreleaser bool) Workflow {
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
						Uses: "googleapis/release-please-action@v5",
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
				Steps:  goreleaserSteps(dist),
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
			IDToken:      "write",
		},
		Jobs: jobs,
	}
}

func goreleaserSteps(dist string) []WorkflowStep {
	steps := []WorkflowStep{
		{
			Name: "Checkout code",
			Uses: "actions/checkout@v7",
			With: map[string]string{
				"fetch-depth": "0",
			},
		},
	}

	if _, err := os.Stat("go.mod"); err == nil {
		steps = append(steps, WorkflowStep{
			Name: "Set up Go",
			Uses: "actions/setup-go@v7",
			With: map[string]string{
				"go-version-file": "go.mod",
			},
		})
	}

	if _, err := os.Stat("Cargo.toml"); err == nil {
		steps = append(steps,
			WorkflowStep{
				Name: "Setup Zig",
				Uses: "mlugg/setup-zig@v2",
			},
			WorkflowStep{
				Name: "Cache Rust artifacts",
				Uses: "actions/cache@v5",
				With: map[string]string{
					"path":         "~/.cargo/registry\n~/.cargo/git\n~/.cache/zig\ntarget\n",
					"key":          "rust-artifacts-${{ runner.os }}-${{ hashFiles('Cargo.lock', '.goreleaser.yml', '.github/workflows/release-please.yml') }}",
					"restore-keys": "rust-artifacts-${{ runner.os }}-",
				},
			},
		)
	}

	steps = append(steps,
		WorkflowStep{
			Name: "Test GoReleaser",
			If:   "${{ needs.release-please.outputs.release_created != 'true' }}",
			Uses: "goreleaser/goreleaser-action@v7",
			With: map[string]string{
				"distribution": "goreleaser",
				"version":      "~> v2",
				"args":         "release --clean --snapshot",
			},
		},
		WorkflowStep{
			Name: "Run GoReleaser",
			If:   "${{ needs.release-please.outputs.release_created == 'true' }}",
			Uses: "goreleaser/goreleaser-action@v7",
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
		WorkflowStep{
			Name: "Check artifact uploader secret",
			ID:   "check-uploader",
			Run:  `if [ -n "$UPLOAD_URL" ]; then echo "available=true" >> "$GITHUB_OUTPUT"; fi`,
			Env: map[string]string{
				"UPLOAD_URL": "${{ secrets.ARTIFACT_UPLOADER_URL }}",
			},
		},
		WorkflowStep{
			Name: "Upload artifacts",
			If:   "${{ steps.check-uploader.outputs.available == 'true' }}",
			Env: map[string]string{
				"UPLOAD_URL": "${{ secrets.ARTIFACT_UPLOADER_URL }}",
				"PREFIX":     "${{ needs.release-please.outputs.release_created == 'true' && 'release' || 'snapshot' }}",
			},
			Run: artifactUploadScript(dist),
		},
	)

	return steps
}

func artifactUploadScript(dist string) string {
	dist = strings.TrimRight(dist, "/")
	if dist == "" {
		dist = "dist"
	}

	return fmt.Sprintf(`TOKEN=$(curl -sSf  -H "Authorization: Bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" "$ACTIONS_ID_TOKEN_REQUEST_URL" | jq -r '.value')
for f in $(jq -r '.[] | select(.type=="Archive" or .type=="Linux Package") | .path' %s/artifacts.json); do
  curl -sSf -w "\n" -X PUT -H "Authorization: Bearer $TOKEN" --data-binary "@$f" "${UPLOAD_URL%%/}/$PREFIX/$(basename "$f")"
done
`, dist)
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
