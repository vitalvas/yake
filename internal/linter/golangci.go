package linter

import (
	"slices"
)

type GolangCI struct {
	Version    string             `yaml:"version,omitempty"`
	Linters    GolangCILinter     `yaml:"linters,omitempty"`
	Formatters GolangCIFormatters `yaml:"formatters,omitempty"`
}

type GolangCILinter struct {
	Default    string                 `yaml:"default,omitempty"`
	Enable     []string               `yaml:"enable,omitempty"`
	Settings   GolangCILinterSettings `yaml:"settings,omitempty"`
	Exclusions GolangCIExclusions     `yaml:"exclusions,omitempty"`
}

type GolangCILinterSettings struct {
	GoSec GolangCILinterSettingsGosec `yaml:"gosec,omitempty"`
}

type GolangCILinterSettingsGosec struct {
	Excludes []string `yaml:"excludes,omitempty"`
}

type GolangCIFormatters struct {
	Exclusions GolangCIExclusions `yaml:"exclusions,omitempty"`
}

type GolangCIExclusions struct {
	Generated string   `yaml:"generated,omitempty"`
	Presets   []string `yaml:"presets,omitempty"`
	Paths     []string `yaml:"paths,omitempty"`
}

func GetGolangCI() GolangCI {
	data := GolangCI{
		Version: "2",
		Linters: GolangCILinter{
			Default: "none",
			Enable: []string{
				"copyloopvar",
				"dogsled",
				"dupl",
				"gocritic",
				"gocyclo",
				"govet",
				"ineffassign",
				"misspell",
				"nakedret",
				"prealloc",
				"revive",
				"staticcheck",
				"unconvert",
				"unused",
			},
			Settings: GolangCILinterSettings{
				GoSec: GolangCILinterSettingsGosec{
					Excludes: []string{
						"G402",
					},
				},
			},
			Exclusions: GolangCIExclusions{
				Generated: "lax",
				Presets: []string{
					"comments",
					"common-false-positives",
					"legacy",
					"std-error-handling",
				},
				Paths: []string{
					"third_party$",
					"vendor$",
					"builtin$",
					"examples$",
				},
			},
		},
		Formatters: GolangCIFormatters{
			Exclusions: GolangCIExclusions{
				Generated: "lax",
				Paths: []string{
					"third_party$",
					"vendor$",
					"builtin$",
					"examples$",
				},
			},
		},
	}

	slices.Sort(data.Linters.Enable)
	slices.Sort(data.Linters.Exclusions.Presets)
	slices.Sort(data.Linters.Exclusions.Paths)
	slices.Sort(data.Formatters.Exclusions.Presets)
	slices.Sort(data.Formatters.Exclusions.Paths)
	slices.Sort(data.Linters.Settings.GoSec.Excludes)

	return data
}
