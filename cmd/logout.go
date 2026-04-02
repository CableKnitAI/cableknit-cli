package cmd

import (
	"fmt"

	"github.com/cableknitai/cableknit-cli/internal/api"
	"github.com/cableknitai/cableknit-cli/internal/config"
	"github.com/cableknitai/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
)

var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out of CableKnit",
	RunE: func(cmd *cobra.Command, args []string) error {
		if config.Token() != "" {
			client := api.NewAPIClient()
			// Best-effort server-side session deletion
			_ = client.JSON("DELETE", "/api/v1/cli/sessions", nil, nil)
		}

		if err := config.ClearToken(); err != nil {
			return fmt.Errorf("failed to clear token: %w", err)
		}

		fmt.Println(ui.SuccessStyle.Render(ui.SymbolCheck + " Logged out."))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
