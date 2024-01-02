package core

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"
	"github.com/vitalvas/yake/internal/github"
	"github.com/vitalvas/yake/internal/linter"
	"github.com/vitalvas/yake/internal/tools"
	"gopkg.in/yaml.v3"
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
	payload := linter.GetGolangCI()

	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	return tools.WriteStringToFile(".golangci.yml", string(data))
}

func codeGithubDependabotGolang() error {
	payload := github.GetGithub("go")

	data, err := yaml.Marshal(payload)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(".github", 0755); err != nil {
		return err
	}

	return tools.WriteStringToFile(".github/dependabot.yml", string(data))
}
