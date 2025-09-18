package core

import (
	"log"

	"github.com/spf13/cobra"
)

func Execute() {
	rootCmd := &cobra.Command{
		Use:   "yake",
		Short: "Yet Another ToolKit",
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

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
