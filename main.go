package main

import (
	"github.com/jessewaites/cableknit-cli/cmd"
)

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd.Execute(version, commit)
}
