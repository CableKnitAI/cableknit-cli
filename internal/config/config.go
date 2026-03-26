package config

import (
	"os"
	"path/filepath"

	"github.com/spf13/viper"
)

func Dir() string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".cableknit")
	_ = os.MkdirAll(dir, 0o700)
	return dir
}

func Token() string {
	return viper.GetString("token")
}

func SetToken(token string) error {
	viper.Set("token", token)
	return writeConfig()
}

func ClearToken() error {
	viper.Set("token", "")
	return writeConfig()
}

func APIURL() string {
	u := viper.GetString("api_url")
	if u == "" {
		return "https://api.cableknit.ai"
	}
	return u
}

func writeConfig() error {
	cfgFile := filepath.Join(Dir(), "config.yaml")
	return viper.WriteConfigAs(cfgFile)
}
