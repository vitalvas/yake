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

	t.Run("sets dist directory", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		assert.Equal(t, "_dist_build/", cfg.Dist)
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
		assert.Equal(t, []string{"-trimpath"}, build.Flags)
		assert.Len(t, build.Ldflags, 4)
	})

	t.Run("configures checksum", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		assert.Equal(t, "checksums.txt", cfg.Checksum.NameTemplate)
		assert.Equal(t, "sha256", cfg.Checksum.Algorithm)
	})

	t.Run("configures upx", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		require.Len(t, cfg.UPX, 1)

		upx := cfg.UPX[0]
		assert.True(t, upx.Enabled)
		assert.Equal(t, []string{"linux"}, upx.Goos)
		assert.Equal(t, []string{"amd64", "arm64"}, upx.Goarch)
	})

	t.Run("configures deb package", func(t *testing.T) {
		cfg := GetConfigWithOptions("owner", "repo", ConfigOptions{
			DebianPackage: true,
		})
		require.Len(t, cfg.NFPMs, 1)

		pkg := cfg.NFPMs[0]
		assert.Equal(t, "repo", pkg.ID)
		assert.Equal(t, "repo", pkg.PackageName)
		assert.Equal(t, "owner", pkg.Vendor)
		assert.Equal(t, "https://github.com/owner/repo", pkg.Homepage)
		assert.Equal(t, "owner <owner@users.noreply.github.com>", pkg.Maintainer)
		assert.Equal(t, "repo command line tool", pkg.Description)
		assert.Equal(t, "Unknown", pkg.License)
		assert.Equal(t, []string{"deb"}, pkg.Formats)
		assert.Equal(t, "utils", pkg.Section)
		assert.Equal(t, "optional", pkg.Priority)
	})

	t.Run("omits deb package by default", func(t *testing.T) {
		cfg := GetConfig("owner", "repo")
		assert.Empty(t, cfg.NFPMs)
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
		assert.Contains(t, content, "dist: _dist_build/")
		assert.Contains(t, content, "binary: myapp")
		assert.Contains(t, content, "id: myapp")
		assert.Contains(t, content, "owner: myowner")
		assert.Contains(t, content, "name: myapp")
		assert.Contains(t, content, "CGO_ENABLED=0")
		assert.Contains(t, content, "flags:")
		assert.Contains(t, content, "- -trimpath")
		assert.Contains(t, content, "upx:")
		assert.Contains(t, content, "enabled: true")
		assert.NotContains(t, content, "nfpms:")
		assert.Contains(t, content, "disable: true")
		assert.Contains(t, content, "prerelease: auto")
		assert.Contains(t, content, "mode: keep-existing")
	})

	t.Run("contains deb package when enabled", func(t *testing.T) {
		cfg := GetConfigWithOptions("myowner", "myapp", ConfigOptions{
			DebianPackage: true,
		})
		data, err := cfg.Marshal()
		require.NoError(t, err)

		content := string(data)
		assert.Contains(t, content, "nfpms:")
		assert.Contains(t, content, "package_name: myapp")
		assert.Contains(t, content, "vendor: myowner")
		assert.Contains(t, content, "homepage: https://github.com/myowner/myapp")
		assert.Contains(t, content, "maintainer: myowner <myowner@users.noreply.github.com>")
		assert.Contains(t, content, "description: myapp command line tool")
		assert.Contains(t, content, "license: Unknown")
		assert.Contains(t, content, "- deb")
		assert.Contains(t, content, "section: utils")
		assert.Contains(t, content, "priority: optional")
		assert.NotContains(t, content, "bindir:")
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
