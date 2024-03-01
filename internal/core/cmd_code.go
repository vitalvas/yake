package core

import (
	"fmt"
	"log"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/vitalvas/yake/internal/github"
	"github.com/vitalvas/yake/internal/linter"
	"github.com/vitalvas/yake/internal/tools"
)

var codeSubcommands = []*cli.Command{
	{
		Name: "defaults",
		Action: func(_ *cli.Context) error {
			var lang github.Lang

			if _, err := os.Stat("go.mod"); err == nil {
				lang = github.Golang

				if _, err := os.Stat(".golangci.yml"); err != nil {
					if err := codeLinterNewGolang(); err != nil {
						return err
					}
				}
			}

			if _, err := os.Stat(".github/dependabot.yml"); err != nil {
				if err := codeGithubDependabot(lang); err != nil {
					return err
				}
			}

			return nil
		},
	},
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

				if err := codeGithubDependabot("go"); err != nil {
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
	payload := linter.GetGolangCI()

	log.Println("Creating .golangci.yml")

	return tools.WriteYamlFile(".golangci.yml", payload)
}

func codeGithubDependabot(lang github.Lang) error {
	payload := github.GetGithub(lang)

	if err := os.MkdirAll(".github", 0755); err != nil {
		return err
	}

	log.Println("Creating .github/dependabot.yml")

	return tools.WriteYamlFile(".github/dependabot.yml", payload)
}
