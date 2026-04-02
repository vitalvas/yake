package policy

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func boolPtr(v bool) *bool          { return &v }
func intPtr(v int) *int             { return &v }
func float64Ptr(v float64) *float64 { return &v }
func stringPtr(v string) *string    { return &v }

func Test_loadConfig(t *testing.T) {
	t.Run("returns defaults when file missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		cfg, err := loadConfig()

		require.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Nil(t, cfg.Policy.EntryPoints)
		assert.Nil(t, cfg.Policy.Coverage)
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
`
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

		cfg, err := loadConfig()

		require.NoError(t, err)
		assert.False(t, cfg.Policy.EntryPoints.isEnabled())
		assert.Equal(t, 70.0, *cfg.Policy.Coverage.MinCoverage)
		assert.Equal(t, 30, *cfg.Policy.Coverage.MaxUncoveredFuncLines)
		assert.Equal(t, 6, *cfg.Policy.FuncSignature.MaxParams)
		assert.Equal(t, 4, *cfg.Policy.FuncSignature.MaxResults)
		assert.Equal(t, "15s", *cfg.Policy.TestDuration.MaxDuration)
		assert.Equal(t, "^[a-z]{2,16}$", *cfg.Policy.PackageNaming.Pattern)
		assert.False(t, cfg.Policy.StringConcat.isEnabled())
	})

	t.Run("returns error on invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, _ := os.Getwd()
		defer os.Chdir(originalDir)

		os.Chdir(tmpDir)

		require.NoError(t, os.WriteFile(configFile, []byte(":::invalid"), 0644))

		_, err := loadConfig()

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
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

		_, err := loadConfig()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test_duration.max_duration")
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
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0644))

		_, err := loadConfig()

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "package_naming.pattern")
	})
}

func Test_policyToggle_isEnabled(t *testing.T) {
	t.Run("nil toggle returns true", func(t *testing.T) {
		var p *policyToggle
		assert.True(t, p.isEnabled())
	})

	t.Run("nil enabled returns true", func(t *testing.T) {
		p := &policyToggle{}
		assert.True(t, p.isEnabled())
	})

	t.Run("enabled true returns true", func(t *testing.T) {
		p := &policyToggle{Enabled: boolPtr(true)}
		assert.True(t, p.isEnabled())
	})

	t.Run("enabled false returns false", func(t *testing.T) {
		p := &policyToggle{Enabled: boolPtr(false)}
		assert.False(t, p.isEnabled())
	})
}

func Test_EntryPointsPolicy(t *testing.T) {
	t.Run("nil returns defaults", func(t *testing.T) {
		var p *EntryPointsPolicy
		assert.True(t, p.isEnabled())
		assert.Equal(t, defaultMaxMainLines, p.getMaxMainLines())
	})

	t.Run("custom values", func(t *testing.T) {
		p := &EntryPointsPolicy{
			Enabled:      boolPtr(false),
			MaxMainLines: intPtr(50),
		}
		assert.False(t, p.isEnabled())
		assert.Equal(t, 50, p.getMaxMainLines())
	})
}

func Test_PackageNamingPolicy(t *testing.T) {
	t.Run("nil returns defaults", func(t *testing.T) {
		var p *PackageNamingPolicy
		assert.True(t, p.isEnabled())
		assert.Equal(t, defaultPackageNamingPattern, p.getPattern())
	})

	t.Run("custom pattern", func(t *testing.T) {
		p := &PackageNamingPolicy{Pattern: stringPtr("^[a-z]+$")}
		assert.Equal(t, "^[a-z]+$", p.getPattern())
	})
}

func Test_FuncSignaturePolicy(t *testing.T) {
	t.Run("nil returns defaults", func(t *testing.T) {
		var p *FuncSignaturePolicy
		assert.True(t, p.isEnabled())
		assert.Equal(t, defaultMaxFuncParams, p.getMaxParams())
		assert.Equal(t, defaultMaxFuncResults, p.getMaxResults())
	})

	t.Run("custom values", func(t *testing.T) {
		p := &FuncSignaturePolicy{
			MaxParams:  intPtr(8),
			MaxResults: intPtr(3),
		}
		assert.Equal(t, 8, p.getMaxParams())
		assert.Equal(t, 3, p.getMaxResults())
	})
}

func Test_TestDurationPolicy(t *testing.T) {
	t.Run("nil returns default", func(t *testing.T) {
		var p *TestDurationPolicy
		assert.True(t, p.isEnabled())
		assert.Equal(t, defaultMaxTestDuration, p.getMaxDuration())
	})

	t.Run("valid duration", func(t *testing.T) {
		p := &TestDurationPolicy{MaxDuration: stringPtr("30s")}
		assert.Equal(t, 30*time.Second, p.getMaxDuration())
	})

	t.Run("invalid duration returns default", func(t *testing.T) {
		p := &TestDurationPolicy{MaxDuration: stringPtr("bad")}
		assert.Equal(t, defaultMaxTestDuration, p.getMaxDuration())
	})
}

func Test_CoveragePolicy(t *testing.T) {
	t.Run("nil returns defaults", func(t *testing.T) {
		var p *CoveragePolicy
		assert.True(t, p.isEnabled())
		assert.Equal(t, defaultMinCoverage, p.getMinCoverage())
		assert.Equal(t, defaultMaxUncoveredFuncLines, p.getMaxUncoveredFuncLines())
	})

	t.Run("custom values", func(t *testing.T) {
		p := &CoveragePolicy{
			MinCoverage:           float64Ptr(60.0),
			MaxUncoveredFuncLines: intPtr(40),
		}
		assert.Equal(t, 60.0, p.getMinCoverage())
		assert.Equal(t, 40, p.getMaxUncoveredFuncLines())
	})
}

func Test_Config_validate(t *testing.T) {
	t.Run("empty config is valid", func(t *testing.T) {
		cfg := &Config{}
		assert.NoError(t, cfg.validate())
	})

	t.Run("valid duration passes", func(t *testing.T) {
		cfg := &Config{
			Policy: policyConfig{
				TestDuration: &TestDurationPolicy{MaxDuration: stringPtr("5s")},
			},
		}
		assert.NoError(t, cfg.validate())
	})

	t.Run("invalid duration fails", func(t *testing.T) {
		cfg := &Config{
			Policy: policyConfig{
				TestDuration: &TestDurationPolicy{MaxDuration: stringPtr("abc")},
			},
		}
		err := cfg.validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "test_duration.max_duration")
	})

	t.Run("valid pattern passes", func(t *testing.T) {
		cfg := &Config{
			Policy: policyConfig{
				PackageNaming: &PackageNamingPolicy{Pattern: stringPtr("^[a-z]+$")},
			},
		}
		assert.NoError(t, cfg.validate())
	})

	t.Run("invalid pattern fails", func(t *testing.T) {
		cfg := &Config{
			Policy: policyConfig{
				PackageNaming: &PackageNamingPolicy{Pattern: stringPtr("[invalid")},
			},
		}
		err := cfg.validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "package_naming.pattern")
	})
}
