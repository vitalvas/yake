package github

type Github struct {
	Version int       `yaml:"version"`
	Updates []Updates `yaml:"updates"`
}

type Updates struct {
	PackageEcosystem string                   `yaml:"package-ecosystem"`
	Directory        string                   `yaml:"directory,omitempty"`
	Schedule         UpdatesSchedule          `yaml:"schedule"`
	Reviewers        []string                 `yaml:"reviewers,omitempty"`
	Assignees        []string                 `yaml:"assignees,omitempty"`
	Groups           map[string]UpdatesGroups `yaml:"groups,omitempty"`
}

type UpdatesGroups struct {
	Patterns []string `yaml:"patterns,omitempty"`
}

type UpdatesSchedule struct {
	Interval string `yaml:"interval"`
}

func GetGithub(lang Lang) Github {
	data := Github{
		Version: 2,
		Updates: []Updates{
			{
				PackageEcosystem: "github-actions",
				Directory:        "/",
				Schedule: UpdatesSchedule{
					Interval: "monthly",
				},
				Reviewers: []string{
					"vitalvas",
				},
				Assignees: []string{
					"vitalvas",
				},
			},
		},
	}

	if lang == Golang {
		data.Updates = append(data.Updates, Updates{
			PackageEcosystem: "gomod",
			Directory:        "/",
			Schedule: UpdatesSchedule{
				Interval: "monthly",
			},
			Reviewers: []string{
				"vitalvas",
			},
			Assignees: []string{
				"vitalvas",
			},
			Groups: map[string]UpdatesGroups{
				"dependencies": {
					Patterns: []string{
						"*",
					},
				},
			},
		})
	}

	return data
}
