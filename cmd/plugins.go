package cmd

import (
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:     "plugins",
	Aliases: []string{"plugin"},
	Short:   "Manage plugins",
}

func init() {
	rootCmd.AddCommand(pluginsCmd)
}
