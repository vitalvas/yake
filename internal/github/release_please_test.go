package github

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetReleasePleaseWorkflow(t *testing.T) {
	t.Run("returns workflow with correct name", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", false)
		assert.Equal(t, "release-please", result.Name)
	})

	t.Run("uses provided branch", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", false)
		require.NotNil(t, result.On.Push)
		require.Len(t, result.On.Push.Branches, 1)
		assert.Equal(t, "main", result.On.Push.Branches[0])
	})

	t.Run("uses master branch when specified", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("master", "dist", false)
		require.NotNil(t, result.On.Push)
		assert.Equal(t, "master", result.On.Push.Branches[0])
	})

	t.Run("sets correct permissions", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", false)
		assert.Equal(t, "write", result.Permissions.Contents)
		assert.Equal(t, "write", result.Permissions.Issues)
		assert.Equal(t, "write", result.Permissions.PullRequests)
		assert.Equal(t, "write", result.Permissions.IDToken)
	})

	t.Run("contains release-please job", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", false)
		job, ok := result.Jobs.Get("release-please")
		require.True(t, ok)
		assert.Equal(t, "Creating release", job.Name)
		assert.Equal(t, "ubuntu-latest", job.RunsOn)
	})

	t.Run("exposes job outputs", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", false)
		job, _ := result.Jobs.Get("release-please")
		assert.Equal(t, "${{ steps.release.outputs.release_created }}", job.Outputs["release_created"])
		assert.Equal(t, "${{ steps.release.outputs.tag_name }}", job.Outputs["tag_name"])
	})

	t.Run("configures release-please action step", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", false)
		job, _ := result.Jobs.Get("release-please")
		require.Len(t, job.Steps, 1)

		step := job.Steps[0]
		assert.Equal(t, "googleapis/release-please-action@v5", step.Uses)
		assert.Equal(t, "release", step.ID)
		assert.Equal(t, "${{ secrets.GITHUB_TOKEN }}", step.With["token"])
		assert.Equal(t, ".github/release-please-config.json", step.With["config-file"])
		assert.Equal(t, ".github/release-please-manifest.json", step.With["manifest-file"])
	})

	t.Run("does not include goreleaser job by default", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", false)
		_, ok := result.Jobs.Get("goreleaser")
		assert.False(t, ok)
		assert.Len(t, result.Jobs, 1)
	})

	t.Run("includes goreleaser job when enabled", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", true)
		job, ok := result.Jobs.Get("goreleaser")
		require.True(t, ok)
		assert.Equal(t, "Build and release packages", job.Name)
		assert.Equal(t, "ubuntu-latest", job.RunsOn)
		assert.Equal(t, []string{"release-please"}, job.Needs)
		assert.Empty(t, job.If)
	})

	t.Run("goreleaser job includes Go setup when go.mod exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		require.NoError(t, os.WriteFile("go.mod", []byte("module test\n"), 0644))

		result := GetReleasePleaseWorkflow("main", "dist", true)
		job, _ := result.Jobs.Get("goreleaser")
		require.Len(t, job.Steps, 6)

		assert.Equal(t, "Checkout code", job.Steps[0].Name)
		assert.Equal(t, "Set up Go", job.Steps[1].Name)
		assert.Equal(t, "Test GoReleaser", job.Steps[2].Name)
		assert.Equal(t, "Run GoReleaser", job.Steps[3].Name)
		assert.Equal(t, "Check artifact uploader secret", job.Steps[4].Name)
		assert.Equal(t, "Upload artifacts", job.Steps[5].Name)
	})

	t.Run("goreleaser job includes Zig setup and Rust cache when Cargo.toml exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)
		require.NoError(t, os.WriteFile("Cargo.toml", []byte("[package]\nname = \"test\"\n"), 0644))

		result := GetReleasePleaseWorkflow("main", "dist", true)
		job, _ := result.Jobs.Get("goreleaser")
		require.Len(t, job.Steps, 7)

		assert.Equal(t, "Checkout code", job.Steps[0].Name)

		zigStep := job.Steps[1]
		assert.Equal(t, "Setup Zig", zigStep.Name)
		assert.Equal(t, "mlugg/setup-zig@v2", zigStep.Uses)

		cacheStep := job.Steps[2]
		assert.Equal(t, "Cache Rust artifacts", cacheStep.Name)
		assert.Equal(t, "actions/cache@v5", cacheStep.Uses)
		assert.Equal(t, "~/.cargo/registry\n~/.cargo/git\n~/.cache/zig\ntarget\n", cacheStep.With["path"])
		assert.Equal(t, "rust-artifacts-${{ runner.os }}-${{ hashFiles('Cargo.lock', '.goreleaser.yml', '.github/workflows/release-please.yml') }}", cacheStep.With["key"])
		assert.Equal(t, "rust-artifacts-${{ runner.os }}-", cacheStep.With["restore-keys"])

		assert.Equal(t, "Test GoReleaser", job.Steps[3].Name)
	})

	t.Run("goreleaser test step runs snapshot on non-release", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		result := GetReleasePleaseWorkflow("main", "dist", true)
		job, _ := result.Jobs.Get("goreleaser")

		testStep := job.Steps[1]
		assert.Equal(t, "Test GoReleaser", testStep.Name)
		assert.Equal(t, "${{ needs.release-please.outputs.release_created != 'true' }}", testStep.If)
		assert.Equal(t, "release --clean --snapshot", testStep.With["args"])
		assert.Empty(t, testStep.Env)
	})

	t.Run("goreleaser release step runs on release", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		result := GetReleasePleaseWorkflow("main", "dist", true)
		job, _ := result.Jobs.Get("goreleaser")

		releaseStep := job.Steps[2]
		assert.Equal(t, "Run GoReleaser", releaseStep.Name)
		assert.Equal(t, "${{ needs.release-please.outputs.release_created == 'true' }}", releaseStep.If)
		assert.Equal(t, "release --clean", releaseStep.With["args"])
		assert.Equal(t, "${{ secrets.GITHUB_TOKEN }}", releaseStep.Env["GITHUB_TOKEN"])
		assert.Equal(t, "${{ needs.release-please.outputs.tag_name }}", releaseStep.Env["TAG"])
	})

	t.Run("artifact uploader check step gates on secret", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		result := GetReleasePleaseWorkflow("main", "dist", true)
		job, _ := result.Jobs.Get("goreleaser")

		checkStep := job.Steps[3]
		assert.Equal(t, "Check artifact uploader secret", checkStep.Name)
		assert.Equal(t, "check-uploader", checkStep.ID)
		assert.Equal(t, "${{ secrets.ARTIFACT_UPLOADER_URL }}", checkStep.Env["UPLOAD_URL"])
		assert.Contains(t, checkStep.Run, `echo "available=true" >> "$GITHUB_OUTPUT"`)
	})

	t.Run("artifact upload step runs when uploader available", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		result := GetReleasePleaseWorkflow("main", "dist", true)
		job, _ := result.Jobs.Get("goreleaser")

		uploadStep := job.Steps[4]
		assert.Equal(t, "Upload artifacts", uploadStep.Name)
		assert.Equal(t, "${{ steps.check-uploader.outputs.available == 'true' }}", uploadStep.If)
		assert.Equal(t, "${{ secrets.ARTIFACT_UPLOADER_URL }}", uploadStep.Env["UPLOAD_URL"])
		assert.Equal(t, "${{ needs.release-please.outputs.release_created == 'true' && 'release' || 'snapshot' }}", uploadStep.Env["PREFIX"])
		assert.Contains(t, uploadStep.Run, "dist/artifacts.json")
		assert.Contains(t, uploadStep.Run, `"${UPLOAD_URL%/}/$PREFIX/$(basename "$f")"`)
	})

	t.Run("artifact upload step uses configured dist directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		result := GetReleasePleaseWorkflow("main", "_dist_build/", true)
		job, _ := result.Jobs.Get("goreleaser")

		uploadStep := job.Steps[4]
		assert.Contains(t, uploadStep.Run, "_dist_build/artifacts.json")
		assert.NotContains(t, uploadStep.Run, "_dist_build//artifacts.json")
	})

	t.Run("artifact upload step falls back to dist when empty", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		result := GetReleasePleaseWorkflow("main", "", true)
		job, _ := result.Jobs.Get("goreleaser")

		uploadStep := job.Steps[4]
		assert.Contains(t, uploadStep.Run, "dist/artifacts.json")
	})

	t.Run("goreleaser job omits Go setup without go.mod", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		result := GetReleasePleaseWorkflow("main", "dist", true)
		job, _ := result.Jobs.Get("goreleaser")
		require.Len(t, job.Steps, 5)

		assert.Equal(t, "Checkout code", job.Steps[0].Name)
		assert.Equal(t, "Test GoReleaser", job.Steps[1].Name)
		assert.Equal(t, "Run GoReleaser", job.Steps[2].Name)
		assert.Equal(t, "Check artifact uploader secret", job.Steps[3].Name)
		assert.Equal(t, "Upload artifacts", job.Steps[4].Name)
	})

	t.Run("release-please job comes before goreleaser", func(t *testing.T) {
		result := GetReleasePleaseWorkflow("main", "dist", true)
		require.Len(t, result.Jobs, 2)
		assert.Equal(t, "release-please", result.Jobs[0].Name)
		assert.Equal(t, "goreleaser", result.Jobs[1].Name)
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
