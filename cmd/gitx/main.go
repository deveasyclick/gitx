package main

import (
	"os"

	"github.com/user/gitx/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(1)
	}
}
