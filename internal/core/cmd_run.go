package core

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"

	"go.vitalvas.com/yake/internal/config"
	"go.vitalvas.com/yake/internal/policy"
)

// ensureProjectRoot verifies the current directory is the project root by
// requiring a .git entry. Without it, checks that walk from the current
// directory find nothing and report success from a subdirectory, hiding real
// violations.
func ensureProjectRoot() error {
	if _, err := os.Stat(".git"); err != nil {
		return fmt.Errorf("must run from the project root (no .git found in the current directory)")
	}

	return nil
}

func createRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run tests and policy checks",
		RunE: func(_ *cobra.Command, _ []string) error {
			if err := ensureProjectRoot(); err != nil {
				return err
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			goPackages := hasGoPackages()

			if goPackages {
				if err := runGoTests(cfg.Tests); err != nil {
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

			if goPackages {
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
