package cmd

import (
	"github.com/spf13/cobra"
)

var runsCmd = &cobra.Command{
	Use:     "runs",
	Aliases: []string{"run"},
	Short:   "Manage automation runs",
}

func init() {
	rootCmd.AddCommand(runsCmd)
}
