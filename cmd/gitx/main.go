package main

import (
	"os"

	"github.com/user/gitx/internal/cli"
)

// version is the release version, injected at build time by goreleaser:
//
//	ldflags: -X main.version={{.Version}}
//
// Local builds default to "dev".
var version = "dev"

func main() {
	if err := cli.Execute(version); err != nil {
		os.Exit(1)
	}
}
