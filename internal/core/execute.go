package core

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"
)

func Execute() {
	app := &cli.App{
		Name:  "yake",
		Usage: "Yet Another ToolKit",
		Commands: []*cli.Command{
			{
				Name:        "code",
				Subcommands: codeSubcommands,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
