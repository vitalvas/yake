package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"

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

const localValidatorsDir string = ".ci/validators"

// runLocalValidators executes every executable file directly inside
// localValidatorsDir in name order. Output is shown only on failure.
func runLocalValidators() error {
	entries, err := os.ReadDir(localValidatorsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("failed to read %s: %w", localValidatorsDir, err)
	}

	for _, entry := range entries {
		path := filepath.Join(localValidatorsDir, entry.Name())

		// os.Stat follows symlinks so linked validators are run too.
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("failed to stat %s: %w", path, err)
		}

		if !info.Mode().IsRegular() || info.Mode().Perm()&0o111 == 0 {
			continue
		}

		log.Printf("Running validator: %s", path)

		ctx, cancel := context.WithTimeout(context.Background(), taskTimeout)

		var stdout, stderr bytes.Buffer

		cmd := exec.CommandContext(ctx, path)
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err = cmd.Run()

		cancel()

		if err != nil {
			return fmt.Errorf("validator %s failed: %w\nstdout:\n%s\nstderr:\n%s", path, err, stdout.String(), stderr.String())
		}
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

			if err := runLocalValidators(); err != nil {
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

			if _, err := os.Stat("Cargo.toml"); err == nil {
				if err := policy.RunRustChecks(); err != nil {
					return err
				}
			}

			log.Println("All checks passed")

			return nil
		},
	}

	return cmd
}
