package linter

type GolangCI struct {
	Linters struct {
		Enable     []string `yaml:"enable,omitempty"`
		Fast       bool     `yaml:"fast"`
		DisableAll bool     `yaml:"disable-all,omitempty"`
	} `yaml:"linters"`
	LintersSettings struct {
		GoSec struct {
			Excludes []string `yaml:"excludes"`
		} `yaml:"gosec"`
	} `yaml:"linters-settings"`
}

func GetGolangCI() GolangCI {
	data := GolangCI{}
	data.Linters.Enable = []string{
		"dogsled",
		"dupl",
		"exportloopref",
		"gas",
		"gocritic",
		"gocyclo",
		"gosimple",
		"govet",
		"ineffassign",
		"megacheck",
		"megacheck",
		"misspell",
		"nakedret",
		"prealloc",
		"revive",
		"staticcheck",
		"stylecheck",
		"typecheck",
		"unconvert",
		"unused",
	}
	data.Linters.Fast = false
	data.Linters.DisableAll = true

	data.LintersSettings.GoSec.Excludes = []string{
		"G402",
	}

	return data
}
