package main

import (
	"os"

	"github.com/jv/twenty-crm-cli/internal/cli"
)

func main() {
	app := cli.New(os.Stdin, os.Stdout, os.Stderr)
	os.Exit(app.Run(os.Args[1:]))
}
