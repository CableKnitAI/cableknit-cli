package main

import (
	"github.com/cableknitai/cableknit-cli/cmd"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd.Execute(version, commit)
}
