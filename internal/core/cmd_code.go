package core

import (
	"fmt"
	"os"
	"strings"

	"github.com/urfave/cli/v2"
	"github.com/vitalvas/yake/internal/tools"
)

var codeSubcommands = []*cli.Command{
	{
		Name: "linter-new",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "lang",
				Required: true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			switch cCtx.String("lang") {
			case "go":
				if _, err := os.Stat(".golangci.yml"); err == nil {
					return fmt.Errorf("linter config file already exists")
				}

				if err := codeLinterNewGolang(); err != nil {
					return err
				}

			default:
				return fmt.Errorf("unsupported language: %s", cCtx.String("lang"))
			}

			return nil
		},
	},
	{
		Name: "github-dependabot",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "lang",
				Required: true,
			},
		},
		Action: func(cCtx *cli.Context) error {
			switch cCtx.String("lang") {
			case "go":
				if _, err := os.Stat(".github/dependabot.yml"); err == nil {
					return fmt.Errorf("dependabot config file already exists")
				}

				if err := codeGithubDependabotGolang(); err != nil {
					return err
				}

			default:
				return fmt.Errorf("unsupported language: %s", cCtx.String("lang"))
			}

			return nil
		},
	},
}

func codeLinterNewGolang() error {
	payload := `
linters:
  enable:
    - megacheck
    - revive
    - govet
    - unconvert
    - megacheck
    - gas
    - gocyclo
    - dupl
    - misspell
    - typecheck
    - ineffassign
    - stylecheck
    - exportloopref
    - gocritic
    - nakedret
    - gosimple
    - prealloc
    - staticcheck
    - unused
    - dogsled
  fast: false
  disable-all: true

linters-settings:
  gosec:
    excludes:
      - G402
`
	return tools.WriteStringToFile(".golangci.yml", strings.TrimSpace(payload)+"\n")
}

func codeGithubDependabotGolang() error {
	payload := `
version: 2
updates:
  - package-ecosystem: gomod
    directory: "/"
    schedule:
        interval: monthly
    reviewers:
        - "vitalvas"
    assignees:
        - "vitalvas"
`
	if err := os.MkdirAll(".github", 0755); err != nil {
		return err
	}

	return tools.WriteStringToFile(".github/dependabot.yml", strings.TrimSpace(payload)+"\n")
}
