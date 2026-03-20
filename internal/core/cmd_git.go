package core

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/vitalvas/yake/internal/githook"
)

func createGitCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "git",
		Short: "Git-related commands",
	}

	hookCmd := &cobra.Command{
		Use:   "hook",
		Short: "Git hook handlers",
	}

	hookCmd.AddCommand(
		&cobra.Command{
			Use:   "commit-msg [file]",
			Short: "Validate commit message",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				return githook.RunCommitMsg(args[0])
			},
		},
		&cobra.Command{
			Use:   "pre-commit",
			Short: "Run pre-commit checks",
			Args:  cobra.NoArgs,
			RunE: func(_ *cobra.Command, _ []string) error {
				if err := githook.RunPreCommit(); err != nil {
					return fmt.Errorf("pre-commit: %w", err)
				}

				return nil
			},
		},
	)

	cmd.AddCommand(hookCmd)

	return cmd
}
