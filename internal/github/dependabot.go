package github

type Github struct {
	Version int       `yaml:"version"`
	Updates []Updates `yaml:"updates"`
}

type Updates struct {
	PackageEcosystem string          `yaml:"package-ecosystem"`
	Directory        string          `yaml:"directory,omitempty"`
	Schedule         UpdatesSchedule `yaml:"schedule"`
	Reviewers        []string        `yaml:"reviewers,omitempty"`
	Assignees        []string        `yaml:"assignees,omitempty"`
}

type UpdatesSchedule struct {
	Interval string `yaml:"interval"`
}

func GetGithub(lang string) Github {
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

	if lang == "go" {
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
		})
	}

	return data
}
