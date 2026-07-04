package goreleaser

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Version   int       `yaml:"version"`
	Dist      string    `yaml:"dist"`
	Before    Before    `yaml:"before"`
	Builds    []Build   `yaml:"builds"`
	UPX       []UPX     `yaml:"upx"`
	NFPMs     []NFPM    `yaml:"nfpms,omitempty"`
	Checksum  Checksum  `yaml:"checksum"`
	Snapshot  Snapshot  `yaml:"snapshot"`
	Changelog Changelog `yaml:"changelog"`
	Release   Release   `yaml:"release"`
}

type ConfigOptions struct {
	DebianPackage bool
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
	Flags   []string `yaml:"flags"`
	Ldflags []string `yaml:"ldflags"`
}

type Checksum struct {
	NameTemplate string `yaml:"name_template"`
	Algorithm    string `yaml:"algorithm"`
}

type UPX struct {
	Enabled  bool     `yaml:"enabled"`
	Goos     []string `yaml:"goos"`
	Goarch   []string `yaml:"goarch"`
	Compress string   `yaml:"compress"`
	Lzma     bool     `yaml:"lzma"`
}

type NFPM struct {
	ID          string   `yaml:"id"`
	PackageName string   `yaml:"package_name"`
	Vendor      string   `yaml:"vendor"`
	Homepage    string   `yaml:"homepage"`
	Maintainer  string   `yaml:"maintainer"`
	Description string   `yaml:"description"`
	License     string   `yaml:"license"`
	Formats     []string `yaml:"formats"`
	Section     string   `yaml:"section"`
	Priority    string   `yaml:"priority"`
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
	return GetConfigWithOptions(owner, repo, ConfigOptions{})
}

func GetConfigWithOptions(owner, repo string, opts ConfigOptions) Config {
	cfg := Config{
		Version: 2,
		Dist:    "_dist_build/",
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
				Flags:  []string{"-trimpath"},
				Ldflags: []string{
					"-s -w",
					"-X main.version={{.Version}}",
					"-X main.commit={{.Commit}}",
					"-X main.date={{.Date}}",
				},
			},
		},
		UPX: []UPX{
			{
				Enabled:  true,
				Goos:     []string{"linux"},
				Goarch:   []string{"amd64", "arm64"},
				Compress: "best",
				Lzma:     true,
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

	if opts.DebianPackage {
		cfg.NFPMs = []NFPM{
			{
				ID:          repo,
				PackageName: repo,
				Vendor:      owner,
				Homepage:    fmt.Sprintf("https://github.com/%s/%s", owner, repo),
				Maintainer:  fmt.Sprintf("%s <%s@users.noreply.github.com>", owner, owner),
				Description: fmt.Sprintf("%s command line tool", repo),
				License:     "Unknown",
				Formats:     []string{"deb"},
				Section:     "utils",
				Priority:    "optional",
			},
		}
	}

	return cfg
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
