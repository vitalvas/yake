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
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
