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
				return runGoTests()
			}

			return nil
		},
	}

	return cmd
}

func runGoTests() error {
	commands := []struct {
		name string
		args []string
	}{
		{name: "go", args: []string{"clean", "-testcache"}},
		{name: "go", args: []string{"test", "-cover", "./..."}},
		{name: "go", args: []string{"test", "-race", "./..."}},
		{name: "golangci-lint", args: []string{"run"}},
	}

	for _, cmdInfo := range commands {
		log.Printf("Running: %s %v", cmdInfo.name, cmdInfo.args)

		cmd := exec.Command(cmdInfo.name, cmdInfo.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to run %s %v: %w", cmdInfo.name, cmdInfo.args, err)
		}
	}

	log.Println("All tests completed successfully")

	return nil
}
