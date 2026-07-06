package config

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"gopkg.in/yaml.v3"
)

// File is the name of the shared yake configuration file.
const File = ".yake.yaml"

const (
	DefaultMaxMainLines          = 25
	DefaultMaxFuncParams         = 5
	DefaultMaxFuncResults        = 5
	DefaultMinCoverage           = 80.0
	DefaultMaxUncoveredFuncLines = 25
	DefaultMaxTestDuration       = 10 * time.Second
	DefaultPackageNamingPattern  = `^[0-9a-z]{3,32}$`
	DefaultMaxSingleLineFields   = 5
)

type Config struct {
	Policy PolicyConfig `yaml:"policy"`
	Tests  TestsConfig  `yaml:"tests"`
}

type TestsConfig struct {
	Tags []string `yaml:"tags"`
}

type PolicyConfig struct {
	EntryPoints            *EntryPointsPolicy      `yaml:"entry_points"`
	PackageNaming          *PackageNamingPolicy    `yaml:"package_naming"`
	ASCIIOnly              *PolicyToggle           `yaml:"ascii_only"`
	StringConcat           *PolicyToggle           `yaml:"string_concat"`
	StdlibWrappers         *PolicyToggle           `yaml:"stdlib_wrappers"`
	FuncSignature          *FuncSignaturePolicy    `yaml:"func_signature"`
	CompositeLiteral       *CompositeLiteralPolicy `yaml:"composite_literal"`
	Stuttering             *PolicyToggle           `yaml:"stuttering"`
	GetterNaming           *PolicyToggle           `yaml:"getter_naming"`
	PrivateExportedMethods *PolicyToggle           `yaml:"private_exported_methods"`
	NoInit                 *PolicyToggle           `yaml:"no_init"`
	TestFileNaming         *PolicyToggle           `yaml:"test_file_naming"`
	TestDuration           *TestDurationPolicy     `yaml:"test_duration"`
	Coverage               *CoveragePolicy         `yaml:"coverage"`
}

type PolicyToggle struct {
	Enabled *bool `yaml:"enable"`
}

type EntryPointsPolicy struct {
	Enabled      *bool `yaml:"enable"`
	MaxMainLines *int  `yaml:"max_main_lines"`
}

type PackageNamingPolicy struct {
	Enabled *bool   `yaml:"enable"`
	Pattern *string `yaml:"pattern"`
}

type CompositeLiteralPolicy struct {
	Enabled             *bool `yaml:"enable"`
	MaxSingleLineFields *int  `yaml:"max_single_line_fields"`
}

type FuncSignaturePolicy struct {
	Enabled    *bool `yaml:"enable"`
	MaxParams  *int  `yaml:"max_params"`
	MaxResults *int  `yaml:"max_results"`
}

type TestDurationPolicy struct {
	Enabled     *bool   `yaml:"enable"`
	MaxDuration *string `yaml:"max_duration"`
}

type CoveragePolicy struct {
	Enabled               *bool              `yaml:"enable"`
	MinCoverage           *float64           `yaml:"min_coverage"`
	MaxUncoveredFuncLines *int               `yaml:"max_uncovered_func_lines"`
	ExcludePackages       []string           `yaml:"exclude_packages"`
	PackageOverrides      map[string]float64 `yaml:"package_overrides"`
}

// Load reads File, unmarshals and validates it. When the file does not exist
// it returns an empty Config so callers rely on zero values and defaults.
func Load() (*Config, error) {
	data, err := os.ReadFile(File)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}

		return nil, fmt.Errorf("failed to read %s: %w", File, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", File, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid %s: %w", File, err)
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
