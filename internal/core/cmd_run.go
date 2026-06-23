package core

import (
	"log"
	"os"

	"github.com/spf13/cobra"

	"github.com/vitalvas/yake/internal/config"
	"github.com/vitalvas/yake/internal/policy"
)

func createRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run tests and policy checks",
		RunE: func(_ *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if _, err := os.Stat("go.mod"); err == nil {
				if err := runGoTests(cfg.Tests.Tags); err != nil {
					return err
				}
			}

			if _, err := os.Stat("Cargo.toml"); err == nil {
				if err := runRustTests(); err != nil {
					return err
				}
			}

			if err := runGoreleaserCheck(); err != nil {
				return err
			}

			if _, err := os.Stat("go.mod"); err == nil {
				if err := policy.RunGolangChecks(); err != nil {
					return err
				}
			}

			log.Println("All checks passed")

			return nil
		},
	}

	return cmd
}
