package policy

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

const configFile = ".yake.yaml"

const (
	defaultMaxMainLines          = 25
	defaultMaxFuncParams         = 5
	defaultMaxFuncResults        = 5
	defaultMinCoverage           = 80.0
	defaultMaxUncoveredFuncLines = 25
	defaultMaxTestDuration       = 10 * time.Second
	defaultPackageNamingPattern  = `^[0-9a-z]{3,32}$`
	defaultMaxSingleLineFields   = 5
)

type Config struct {
	Policy policyConfig `yaml:"policy"`
}

type policyConfig struct {
	EntryPoints            *EntryPointsPolicy      `yaml:"entry_points"`
	PackageNaming          *PackageNamingPolicy    `yaml:"package_naming"`
	StringConcat           *policyToggle           `yaml:"string_concat"`
	StdlibWrappers         *policyToggle           `yaml:"stdlib_wrappers"`
	FuncSignature          *FuncSignaturePolicy    `yaml:"func_signature"`
	CompositeLiteral       *CompositeLiteralPolicy `yaml:"composite_literal"`
	Stuttering             *policyToggle           `yaml:"stuttering"`
	GetterNaming           *policyToggle           `yaml:"getter_naming"`
	PrivateExportedMethods *policyToggle           `yaml:"private_exported_methods"`
	NoInit                 *policyToggle           `yaml:"no_init"`
	TestFileNaming         *policyToggle           `yaml:"test_file_naming"`
	TestDuration           *TestDurationPolicy     `yaml:"test_duration"`
	Coverage               *CoveragePolicy         `yaml:"coverage"`
}

type policyToggle struct {
	Enabled *bool `yaml:"enable"`
}

func (p *policyToggle) isEnabled() bool {
	if p == nil || p.Enabled == nil {
		return true
	}

	return *p.Enabled
}

type EntryPointsPolicy struct {
	Enabled      *bool `yaml:"enable"`
	MaxMainLines *int  `yaml:"max_main_lines"`
}

func (p *EntryPointsPolicy) isEnabled() bool {
	if p == nil || p.Enabled == nil {
		return true
	}

	return *p.Enabled
}

func (p *EntryPointsPolicy) getMaxMainLines() int {
	if p == nil || p.MaxMainLines == nil {
		return defaultMaxMainLines
	}

	return *p.MaxMainLines
}

type PackageNamingPolicy struct {
	Enabled *bool   `yaml:"enable"`
	Pattern *string `yaml:"pattern"`
}

func (p *PackageNamingPolicy) isEnabled() bool {
	if p == nil || p.Enabled == nil {
		return true
	}

	return *p.Enabled
}

func (p *PackageNamingPolicy) getPattern() string {
	if p == nil || p.Pattern == nil {
		return defaultPackageNamingPattern
	}

	return *p.Pattern
}

type CompositeLiteralPolicy struct {
	Enabled             *bool `yaml:"enable"`
	MaxSingleLineFields *int  `yaml:"max_single_line_fields"`
}

func (p *CompositeLiteralPolicy) isEnabled() bool {
	if p == nil || p.Enabled == nil {
		return true
	}

	return *p.Enabled
}

func (p *CompositeLiteralPolicy) getMaxSingleLineFields() int {
	if p == nil || p.MaxSingleLineFields == nil {
		return defaultMaxSingleLineFields
	}

	return *p.MaxSingleLineFields
}

type FuncSignaturePolicy struct {
	Enabled    *bool `yaml:"enable"`
	MaxParams  *int  `yaml:"max_params"`
	MaxResults *int  `yaml:"max_results"`
}

func (p *FuncSignaturePolicy) isEnabled() bool {
	if p == nil || p.Enabled == nil {
		return true
	}

	return *p.Enabled
}

func (p *FuncSignaturePolicy) getMaxParams() int {
	if p == nil || p.MaxParams == nil {
		return defaultMaxFuncParams
	}

	return *p.MaxParams
}

func (p *FuncSignaturePolicy) getMaxResults() int {
	if p == nil || p.MaxResults == nil {
		return defaultMaxFuncResults
	}

	return *p.MaxResults
}

type TestDurationPolicy struct {
	Enabled     *bool   `yaml:"enable"`
	MaxDuration *string `yaml:"max_duration"`
}

func (p *TestDurationPolicy) isEnabled() bool {
	if p == nil || p.Enabled == nil {
		return true
	}

	return *p.Enabled
}

func (p *TestDurationPolicy) getMaxDuration() time.Duration {
	if p == nil || p.MaxDuration == nil {
		return defaultMaxTestDuration
	}

	d, err := time.ParseDuration(*p.MaxDuration)
	if err != nil {
		return defaultMaxTestDuration
	}

	return d
}

type CoveragePolicy struct {
	Enabled               *bool              `yaml:"enable"`
	MinCoverage           *float64           `yaml:"min_coverage"`
	MaxUncoveredFuncLines *int               `yaml:"max_uncovered_func_lines"`
	ExcludePackages       []string           `yaml:"exclude_packages"`
	PackageOverrides      map[string]float64 `yaml:"package_overrides"`
}

func (p *CoveragePolicy) isEnabled() bool {
	if p == nil || p.Enabled == nil {
		return true
	}

	return *p.Enabled
}

func (p *CoveragePolicy) getMinCoverage() float64 {
	if p == nil || p.MinCoverage == nil {
		return defaultMinCoverage
	}

	return *p.MinCoverage
}

func (p *CoveragePolicy) getMaxUncoveredFuncLines() int {
	if p == nil || p.MaxUncoveredFuncLines == nil {
		return defaultMaxUncoveredFuncLines
	}

	return *p.MaxUncoveredFuncLines
}

func (p *CoveragePolicy) getExcludePackages() []string {
	if p == nil {
		return nil
	}

	return p.ExcludePackages
}

func (p *CoveragePolicy) getPackageOverrides() map[string]float64 {
	if p == nil {
		return nil
	}

	return p.PackageOverrides
}

func loadConfig() (*Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}

		return nil, fmt.Errorf("failed to read %s: %w", configFile, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configFile, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", configFile, err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Policy.TestDuration != nil && c.Policy.TestDuration.MaxDuration != nil {
		if _, err := time.ParseDuration(*c.Policy.TestDuration.MaxDuration); err != nil {
			return fmt.Errorf("test_duration.max_duration: %w", err)
		}
	}

	if c.Policy.PackageNaming != nil && c.Policy.PackageNaming.Pattern != nil {
		if _, err := regexp.Compile(*c.Policy.PackageNaming.Pattern); err != nil {
			return fmt.Errorf("package_naming.pattern: %w", err)
		}
	}

	return nil
}
