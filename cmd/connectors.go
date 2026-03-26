package cmd

import (
	"github.com/spf13/cobra"
)

var connectorsCmd = &cobra.Command{
	Use:     "connectors",
	Aliases: []string{"connector"},
	Short:   "Browse available connectors",
}

func init() {
	rootCmd.AddCommand(connectorsCmd)
}
