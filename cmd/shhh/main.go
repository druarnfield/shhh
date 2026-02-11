package main

import (
	"os"

	"github.com/druarnfield/shhh/internal/cli"
)

var version = "dev"

func main() {
	if err := cli.Execute(version); err != nil {
		os.Exit(1)
	}
}
