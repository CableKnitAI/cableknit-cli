package cmd

import (
	"fmt"

	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
)

var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show current user",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		client := api.NewAPIClient()
		var user api.User
		if err := client.JSON("GET", "/api/v1/cli/me", nil, &user); err != nil {
			return err
		}

		bold := lipgloss.NewStyle().Bold(true)
		fmt.Printf("%s  %s\n", bold.Render("Name:"), user.Name)
		fmt.Printf("%s  %s\n", bold.Render("Email:"), user.Email)

		switch user.PublisherStatus {
		case "pending":
			fmt.Printf("%s  %s\n", bold.Render("Status:"),
				ui.WarningStyle.Render(ui.SymbolWarning+" Publisher approval pending"))
		default:
			fmt.Printf("%s  %s\n", bold.Render("Status:"),
				ui.SuccessStyle.Render(ui.SymbolCheck+" "+user.PublisherStatus))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(whoamiCmd)
}
