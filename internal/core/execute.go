package core

import (
	"log"
	"os"

	"github.com/spf13/cobra"
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:           "yake",
		Short:         "Yet Another ToolKit",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	codeCmd := &cobra.Command{
		Use:   "code",
		Short: "Code-related commands",
	}

	// Add code subcommands
	for _, subCmd := range codeSubcommands {
		codeCmd.AddCommand(subCmd)
	}

	rootCmd.AddCommand(codeCmd)
	rootCmd.AddCommand(createTestsCommand())
	rootCmd.AddCommand(createPolicyCommand())

	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}
