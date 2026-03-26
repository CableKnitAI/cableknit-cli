package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	buildVersion = "dev"
	buildCommit  = "none"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("cableknit %s (%s/%s)\n", buildVersion, runtime.GOOS, runtime.GOARCH)
		if buildCommit != "none" {
			fmt.Printf("commit: %s\n", buildCommit)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
	rootCmd.Version = buildVersion
	rootCmd.SetVersionTemplate("cableknit {{.Version}}\n")
}
