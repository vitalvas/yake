package goreleaser

import (
	"bytes"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version   int       `yaml:"version"`
	Before    Before    `yaml:"before"`
	Builds    []Build   `yaml:"builds"`
	Checksum  Checksum  `yaml:"checksum"`
	Snapshot  Snapshot  `yaml:"snapshot"`
	Changelog Changelog `yaml:"changelog"`
	Release   Release   `yaml:"release"`
}

type Before struct {
	Hooks []string `yaml:"hooks"`
}

type Build struct {
	ID      string   `yaml:"id"`
	Binary  string   `yaml:"binary"`
	Env     []string `yaml:"env"`
	Goos    []string `yaml:"goos"`
	Goarch  []string `yaml:"goarch"`
	Ldflags []string `yaml:"ldflags"`
}

type Checksum struct {
	NameTemplate string `yaml:"name_template"`
	Algorithm    string `yaml:"algorithm"`
}

type Snapshot struct {
	VersionTemplate string `yaml:"version_template"`
}

type Changelog struct {
	Disable bool `yaml:"disable"`
}

type Release struct {
	GitHub     ReleaseGitHub `yaml:"github"`
	Draft      bool          `yaml:"draft"`
	Prerelease string        `yaml:"prerelease"`
	Mode       string        `yaml:"mode"`
}

type ReleaseGitHub struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
}

func GetConfig(owner, repo string) Config {
	return Config{
		Version: 2,
		Before: Before{
			Hooks: []string{"go mod tidy"},
		},
		Builds: []Build{
			{
				ID:     repo,
				Binary: repo,
				Env:    []string{"CGO_ENABLED=0"},
				Goos:   []string{"linux"},
				Goarch: []string{"amd64", "arm64"},
				Ldflags: []string{
					"-s -w",
					"-X main.version={{.Version}}",
					"-X main.commit={{.Commit}}",
					"-X main.date={{.Date}}",
				},
			},
		},
		Checksum: Checksum{
			NameTemplate: "checksums.txt",
			Algorithm:    "sha256",
		},
		Snapshot: Snapshot{
			VersionTemplate: "{{ .Version }}-{{ .CommitTimestamp }}-{{ .ShortCommit }}",
		},
		Changelog: Changelog{
			Disable: true,
		},
		Release: Release{
			GitHub: ReleaseGitHub{
				Owner: owner,
				Name:  repo,
			},
			Draft:      false,
			Prerelease: "auto",
			Mode:       "keep-existing",
		},
	}
}

func (c Config) Marshal() ([]byte, error) {
	var buf bytes.Buffer

	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)

	if err := enc.Encode(c); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
