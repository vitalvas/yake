package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func stringPtr(v string) *string { return &v }

func TestLoad(t *testing.T) {
	t.Run("returns defaults when file missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cfg, err := Load()

		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Nil(t, cfg.Policy.EntryPoints)
		assert.Nil(t, cfg.Policy.Coverage)
		assert.Empty(t, cfg.Tests.Tags)
	})

	t.Run("parses valid config", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		content := `policy:
  entry_points:
    enable: false
  coverage:
    min_coverage: 70.0
    max_uncovered_func_lines: 30
  func_signature:
    max_params: 6
    max_results: 4
  test_duration:
    max_duration: "15s"
  package_naming:
    pattern: "^[a-z]{2,16}$"
  string_concat:
    enable: false
tests:
  tags:
    - integration
    - e2e
`
		require.NoError(t, os.WriteFile(File, []byte(content), 0644))

		cfg, err := Load()

		require.NoError(t, err)
		assert.False(t, *cfg.Policy.EntryPoints.Enabled)
		assert.Equal(t, 70.0, *cfg.Policy.Coverage.MinCoverage)
		assert.Equal(t, 30, *cfg.Policy.Coverage.MaxUncoveredFuncLines)
		assert.Equal(t, 6, *cfg.Policy.FuncSignature.MaxParams)
		assert.Equal(t, 4, *cfg.Policy.FuncSignature.MaxResults)
		assert.Equal(t, "15s", *cfg.Policy.TestDuration.MaxDuration)
		assert.Equal(t, "^[a-z]{2,16}$", *cfg.Policy.PackageNaming.Pattern)
		assert.False(t, *cfg.Policy.StringConcat.Enabled)
		assert.Equal(t, []string{"integration", "e2e"}, cfg.Tests.Tags)
	})

	t.Run("returns error on invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile(File, []byte(":::invalid"), 0644))

		_, err := Load()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("returns error on invalid duration", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		content := `policy:
  test_duration:
    max_duration: "not-a-duration"
`
		require.NoError(t, os.WriteFile(File, []byte(content), 0644))

		_, err := Load()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test_duration.max_duration")
	})

	t.Run("parses coverage exclude packages", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		content := `policy:
  coverage:
    exclude_packages:
      - internal/cmd
      - internal/database
`
		require.NoError(t, os.WriteFile(File, []byte(content), 0644))

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, []string{"internal/cmd", "internal/database"}, cfg.Policy.Coverage.ExcludePackages)
	})

	t.Run("parses coverage package overrides", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		content := `policy:
  coverage:
    min_coverage: 80.0
    package_overrides:
      internal/cmd: 40.0
      internal/database: 50.0
`
		require.NoError(t, os.WriteFile(File, []byte(content), 0644))

		cfg, err := Load()

		require.NoError(t, err)
		assert.Equal(t, 80.0, *cfg.Policy.Coverage.MinCoverage)
		assert.Equal(t, map[string]float64{"internal/cmd": 40.0, "internal/database": 50.0}, cfg.Policy.Coverage.PackageOverrides)
	})

	t.Run("returns error on invalid regex pattern", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		content := `policy:
  package_naming:
    pattern: "[invalid"
`
		require.NoError(t, os.WriteFile(File, []byte(content), 0644))

		_, err := Load()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "package_naming.pattern")
	})
}

func Test_Config_validate(t *testing.T) {
	t.Run("empty config is valid", func(t *testing.T) {
		cfg := &Config{}
		assert.NoError(t, cfg.validate())
	})

	t.Run("valid duration passes", func(t *testing.T) {
		cfg := &Config{
			Policy: PolicyConfig{
				TestDuration: &TestDurationPolicy{MaxDuration: stringPtr("5s")},
			},
		}
		assert.NoError(t, cfg.validate())
	})

	t.Run("invalid duration fails", func(t *testing.T) {
		cfg := &Config{
			Policy: PolicyConfig{
				TestDuration: &TestDurationPolicy{MaxDuration: stringPtr("abc")},
			},
		}
		err := cfg.validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test_duration.max_duration")
	})

	t.Run("valid pattern passes", func(t *testing.T) {
		cfg := &Config{
			Policy: PolicyConfig{
				PackageNaming: &PackageNamingPolicy{Pattern: stringPtr("^[a-z]+$")},
			},
		}
		assert.NoError(t, cfg.validate())
	})

	t.Run("invalid pattern fails", func(t *testing.T) {
		cfg := &Config{
			Policy: PolicyConfig{
				PackageNaming: &PackageNamingPolicy{Pattern: stringPtr("[invalid")},
			},
		}
		err := cfg.validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "package_naming.pattern")
	})
}
