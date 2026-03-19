package goreleaser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetConfig(t *testing.T) {
	t.Run("sets version to 2", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		assert.Equal(t, 2, cfg.Version)
	})

	t.Run("sets before hooks", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		assert.Equal(t, []string{"go mod tidy"}, cfg.Before.Hooks)
	})

	t.Run("configures build with repo name", func(t *testing.T) {
		cfg := GetConfig("myowner", "myapp")
		require.Len(t, cfg.Builds, 1)

		build := cfg.Builds[0]
		assert.Equal(t, "myapp", build.ID)
		assert.Equal(t, "myapp", build.Binary)
		assert.Equal(t, []string{"CGO_ENABLED=0"}, build.Env)
		assert.Equal(t, []string{"linux"}, build.Goos)
		assert.Equal(t, []string{"amd64", "arm64"}, build.Goarch)
		assert.Len(t, build.Ldflags, 4)
	})

	t.Run("configures checksum", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		assert.Equal(t, "checksums.txt", cfg.Checksum.NameTemplate)
		assert.Equal(t, "sha256", cfg.Checksum.Algorithm)
	})

	t.Run("disables changelog", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		assert.True(t, cfg.Changelog.Disable)
	})

	t.Run("configures release with owner and repo", func(t *testing.T) {
		cfg := GetConfig("myowner", "myapp")
		assert.Equal(t, "myowner", cfg.Release.GitHub.Owner)
		assert.Equal(t, "myapp", cfg.Release.GitHub.Name)
		assert.False(t, cfg.Release.Draft)
		assert.Equal(t, "auto", cfg.Release.Prerelease)
		assert.Equal(t, "keep-existing", cfg.Release.Mode)
	})
}

func TestConfigMarshal(t *testing.T) {
	t.Run("produces valid YAML", func(t *testing.T) {
		cfg := GetConfig("myowner", "myapp")
		data, err := cfg.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "version: 2")
		assert.Contains(t, content, "binary: myapp")
		assert.Contains(t, content, "id: myapp")
		assert.Contains(t, content, "owner: myowner")
		assert.Contains(t, content, "name: myapp")
		assert.Contains(t, content, "CGO_ENABLED=0")
		assert.Contains(t, content, "disable: true")
		assert.Contains(t, content, "prerelease: auto")
		assert.Contains(t, content, "mode: keep-existing")
	})

	t.Run("contains ldflags with Go template syntax", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		data, err := cfg.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "{{.Version}}")
		assert.Contains(t, content, "{{.Commit}}")
		assert.Contains(t, content, "{{.Date}}")
	})

	t.Run("contains snapshot template", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		data, err := cfg.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "{{ .Version }}-{{ .CommitTimestamp }}-{{ .ShortCommit }}")
	})
}
