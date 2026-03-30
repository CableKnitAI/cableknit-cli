package cmd

import (
	"fmt"
	"os"

	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/config"
	"github.com/jessewaites/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var useAltSplash = true

var rootCmd = &cobra.Command{
	Use:   "cableknit",
	Short: "CableKnit CLI — manage plugins and runs",
	Long:  "CableKnit CLI lets you build, validate, push, and monitor CableKnit plugins.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		if !ui.IsTTY() {
			cmd.Help()
			return
		}
		runAppShell()
	},
}

func Execute(version, commit string) {
	buildVersion = version
	buildCommit = commit

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().String("api-url", "", "override API base URL")
	rootCmd.PersistentFlags().Bool("no-color", false, "disable color output")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
	rootCmd.PersistentFlags().Bool("json", false, "output as JSON")
	rootCmd.PersistentFlags().Bool("demo", false, "use mock data for demo")
	rootCmd.PersistentFlags().BoolVar(&useAltSplash, "alt", true, "use alternate launch screen")

	if os.Getenv("CABLEKNIT_ALT_SPLASH") == "1" {
		useAltSplash = true
	}

	_ = viper.BindPFlag("api_url", rootCmd.PersistentFlags().Lookup("api-url"))
	_ = viper.BindPFlag("no_color", rootCmd.PersistentFlags().Lookup("no-color"))
	_ = viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

func requireAuth() error {
	if api.DemoLoggedIn {
		return nil
	}
	if config.Token() == "" {
		return fmt.Errorf("not logged in. Run `cableknit login` first")
	}
	return nil
}

func initConfig() {
	demo, _ := rootCmd.PersistentFlags().GetBool("demo")
	api.DemoEnabled = demo

	cfgDir := config.Dir()
	cfgFile := cfgDir + "/config.yaml"

	viper.SetConfigFile(cfgFile)
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("CABLEKNIT")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		// Config file not found is fine — first run
	}

	if viper.GetString("api_url") == "" {
		viper.SetDefault("api_url", "https://cableknit.ai")
	}
}
