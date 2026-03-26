package cmd

import (
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:     "tools",
	Aliases: []string{"tool"},
	Short:   "Browse available platform tools",
}

func init() {
	rootCmd.AddCommand(toolsCmd)
}
