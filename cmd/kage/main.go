package main

import (
	"os"

	"github.com/6abe/kage/internal/cli"
	"github.com/6abe/kage/internal/host"
)

func main() {
	os.Exit(cli.Run(host.Live{}, os.Args[1:], os.Stdout, os.Stderr))
}
