// Command meridian-cli exposes Meridian's engines on the command line.
package main

import (
	"os"

	"github.com/hriday/meridian/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
