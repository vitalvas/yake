package core

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/vitalvas/yake/internal/policy"
)

func createPolicyCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "policy",
		Short: "Policy-related commands",
	}

	for _, subCmd := range policySubcommands {
		cmd.AddCommand(subCmd)
	}

	return cmd
}

var policySubcommands = []*cobra.Command{
	{
		Use:   "run",
		Short: "Run policy checks for the project",
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := os.Stat("go.mod"); err == nil {
				if err := policy.RunGolangChecks(); err != nil {
					return err
				}
			}

			log.Println("All policy checks passed")

			return nil
		},
	},
}
