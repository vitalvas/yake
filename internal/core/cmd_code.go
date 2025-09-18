package core

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/vitalvas/yake/internal/github"
	"github.com/vitalvas/yake/internal/linter"
	"github.com/vitalvas/yake/internal/tools"
)

func createLinterNewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "linter-new",
		Short: "Create a new linter configuration file",
		RunE: func(cmd *cobra.Command, _ []string) error {
			lang, _ := cmd.Flags().GetString("lang")
			switch lang {
			case "go":
				if _, err := os.Stat(".golangci.yml"); err == nil {
					return fmt.Errorf("linter config file already exists")
				}

				if err := codeLinterNewGolang(); err != nil {
					return err
				}

			default:
				return fmt.Errorf("unsupported language: %s", lang)
			}

			return nil
		},
	}
	cmd.Flags().StringP("lang", "l", "", "Programming language (required)")
	cmd.MarkFlagRequired("lang")
	return cmd
}

func createGithubDependabotCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "github-dependabot",
		Short: "Create GitHub Dependabot configuration",
		RunE: func(cmd *cobra.Command, _ []string) error {
			lang, _ := cmd.Flags().GetString("lang")
			switch lang {
			case "go":
				if _, err := os.Stat(".github/dependabot.yml"); err == nil {
					return fmt.Errorf("dependabot config file already exists")
				}

				if err := codeGithubDependabot("go"); err != nil {
					return err
				}

			default:
				return fmt.Errorf("unsupported language: %s", lang)
			}

			return nil
		},
	}
	cmd.Flags().StringP("lang", "l", "", "Programming language (required)")
	cmd.MarkFlagRequired("lang")
	return cmd
}

var codeSubcommands = []*cobra.Command{
	{
		Use:   "defaults",
		Short: "Apply default configurations for the project",
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := os.Stat("go.mod"); err == nil {

				if _, err := os.Stat(".golangci.yml"); err != nil {
					if err := codeLinterNewGolang(); err != nil {
						return err
					}
				}
			}
			return nil
		},
	},
	createLinterNewCommand(),
	createGithubDependabotCommand(),
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
