package core

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func createTestsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tests",
		Short: "Run Go tests with coverage, race detection, and linting",
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := os.Stat("go.mod"); err == nil {
				if err := runGoTests(); err != nil {
					return err
				}
			}

			if err := runGoreleaserCheck(); err != nil {
				return err
			}

			log.Println("All tests completed successfully")

			return nil
		},
	}

	return cmd
}

type command struct {
	name string
	args []string
}

func runGoTests() error {
	commands := []command{
		{name: "go", args: []string{"fmt", "./..."}},
		{name: "go", args: []string{"vet", "./..."}},
		{name: "go", args: []string{"mod", "tidy", "-v"}},
		{name: "go", args: []string{"clean", "-testcache"}},
		{name: "go", args: []string{"test", "-cover", "./..."}},
		{name: "go", args: []string{"test", "-race", "./..."}},
	}

	if _, err := os.Stat(".golangci.yml"); err == nil {
		commands = append(commands, command{name: "golangci-lint", args: []string{"run"}})
	}

	for _, cmdInfo := range commands {
		if err := runCommand(cmdInfo.name, cmdInfo.args...); err != nil {
			return err
		}
	}

	return nil
}

func runCommand(name string, args ...string) error {
	log.Printf("Running: %v", append([]string{name}, args...))

	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run %s: %w", append([]string{name}, args...), err)
	}

	return nil
}

func runGoreleaserCheck() error {
	if _, err := os.Stat(".goreleaser.yml"); err != nil {
		return nil
	}

	if _, err := exec.LookPath("goreleaser"); err != nil {
		return nil
	}

	return runCommand("goreleaser", "check")
}
