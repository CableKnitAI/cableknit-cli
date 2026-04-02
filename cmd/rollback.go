package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cableknitai/cableknit-cli/internal/api"
	"github.com/cableknitai/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var rollbackCmd = &cobra.Command{
	Use:   "rollback <plugin-slug>",
	Short: "Roll back a plugin to a previous version",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		slug := args[0]
		version, _ := cmd.Flags().GetString("version")
		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed

		if version != "" {
			return rollbackExecute(slug, version, jsonOut)
		}

		if !ui.IsTTY() {
			return fmt.Errorf("--version is required in non-interactive mode")
		}

		p := tea.NewProgram(newRollbackModel(slug, jsonOut))
		m, err := p.Run()
		if err != nil {
			return err
		}
		rm := m.(rollbackModel)
		if rm.err != nil {
			return rm.err
		}
		return nil
	},
}

func init() {
	rollbackCmd.Flags().String("version", "", "version to roll back to")
	rootCmd.AddCommand(rollbackCmd)
}

func rollbackExecute(slug, version string, jsonOut bool) error {
	client := api.NewAPIClient()
	path := fmt.Sprintf("/api/v1/cli/plugins/%s/rollback", slug)

	var resp api.RollbackResponse
	if err := client.JSON("POST", path, map[string]string{"version": version}, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if resp.Success {
		fmt.Println(ui.SymbolCheck + " " + resp.Message)
	} else {
		fmt.Println(ui.SymbolCross + " Rollback failed")
	}
	return nil
}

// TUI — version picker + confirmation

type rollbackState int

const (
	rollbackLoading rollbackState = iota
	rollbackPicking
	rollbackConfirming
	rollbackExecuting
	rollbackDone
)

type rollbackModel struct {
	state    rollbackState
	versions []api.PluginVersion
	cursor   int
	err      error
	slug     string
	jsonOut  bool
	result   *api.RollbackResponse
	embedded bool
}

type rollbackVersionsMsg struct {
	versions []api.PluginVersion
	err      error
}

type rollbackResultMsg struct {
	resp api.RollbackResponse
	err  error
}

func newRollbackModel(slug string, jsonOut bool) rollbackModel {
	return rollbackModel{slug: slug, jsonOut: jsonOut}
}

func (m rollbackModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m rollbackModel) Init() tea.Cmd {
	return m.fetchVersions()
}

func (m rollbackModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			if m.state == rollbackConfirming {
				m.state = rollbackPicking
				return m, nil
			}
			return m, m.done()
		case "up", "k":
			if m.state == rollbackPicking && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.state == rollbackPicking && m.cursor < len(m.versions)-1 {
				m.cursor++
			}
		case "enter":
			if m.state == rollbackPicking {
				m.state = rollbackConfirming
				return m, nil
			}
		case "y", "Y":
			if m.state == rollbackConfirming {
				m.state = rollbackExecuting
				return m, m.executeRollback()
			}
		case "n", "N":
			if m.state == rollbackConfirming {
				m.state = rollbackPicking
				return m, nil
			}
		}

	case rollbackVersionsMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.versions = msg.versions
		if len(m.versions) == 0 {
			return m, m.done()
		}
		m.state = rollbackPicking
		return m, nil

	case rollbackResultMsg:
		if msg.err != nil {
			m.err = msg.err
			m.state = rollbackDone
			return m, nil
		}
		m.result = &msg.resp
		m.state = rollbackDone
		return m, nil
	}

	return m, nil
}

func (m rollbackModel) View() tea.View {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)

	switch m.state {
	case rollbackLoading:
		return tea.NewView("\n  Loading versions...\n\n")

	case rollbackPicking:
		if len(m.versions) == 0 {
			return tea.NewView("\n  No versions available for rollback.\n\n")
		}

		headerStyle := lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
		selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Blue)
		normalStyle := lipgloss.NewStyle()

		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString("  " + titleStyle.Render("Rollback — "+m.slug) + "\n\n")

		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			headerStyle.Render(pad("VERSION", 12)),
			headerStyle.Render(pad("PUSHED AT", 20)),
			headerStyle.Render(pad("NOTES", 40)),
		))
		sb.WriteString("\n")

		for i, v := range m.versions {
			notes := v.Notes
			if notes == "" {
				notes = "-"
			}
			pushed := v.PushedAt.Format("2006-01-02 15:04:05")

			row := fmt.Sprintf("  %s  %s  %s",
				pad(v.Version, 12),
				pad(pushed, 20),
				pad(notes, 40),
			)
			if i == m.cursor {
				sb.WriteString(selectedStyle.Render(row) + "\n")
			} else {
				sb.WriteString(normalStyle.Render(row) + "\n")
			}
		}

		sb.WriteString("\n")
		hint := "  ↑↓ navigate  enter select  q/esc back"
		sb.WriteString(ui.DimStyle.Render(hint) + "\n")
		return tea.NewView(sb.String())

	case rollbackConfirming:
		selected := m.versions[m.cursor]
		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString("  " + titleStyle.Render("Rollback — "+m.slug) + "\n\n")
		sb.WriteString(fmt.Sprintf("  Roll back to %s?\n\n",
			lipgloss.NewStyle().Bold(true).Render("v"+selected.Version)))
		sb.WriteString("  " + ui.DimStyle.Render("y confirm  n/esc cancel") + "\n")
		return tea.NewView(sb.String())

	case rollbackExecuting:
		return tea.NewView("\n  Rolling back...\n\n")

	case rollbackDone:
		var sb strings.Builder
		sb.WriteString("\n")
		if m.err != nil {
			sb.WriteString(ui.ErrorStyle.Render("  "+ui.SymbolCross+" "+m.err.Error()) + "\n")
		} else if m.result != nil && m.result.Success {
			sb.WriteString(ui.SuccessStyle.Render("  "+ui.SymbolCheck+" "+m.result.Message) + "\n")
		} else {
			sb.WriteString(ui.ErrorStyle.Render("  "+ui.SymbolCross+" Rollback failed") + "\n")
		}
		sb.WriteString("\n")
		if m.embedded {
			sb.WriteString(ui.DimStyle.Render("  q/esc back") + "\n")
		} else {
			sb.WriteString(ui.DimStyle.Render("  q to quit") + "\n")
		}
		return tea.NewView(sb.String())
	}

	return tea.NewView("")
}

func (m rollbackModel) fetchVersions() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		path := fmt.Sprintf("/api/v1/cli/plugins/%s/versions", m.slug)
		var resp api.VersionsResponse
		if err := client.JSON("GET", path, nil, &resp); err != nil {
			return rollbackVersionsMsg{err: err}
		}
		return rollbackVersionsMsg{versions: resp.Versions}
	}
}

func (m rollbackModel) executeRollback() tea.Cmd {
	selected := m.versions[m.cursor]
	return func() tea.Msg {
		client := api.NewAPIClient()
		path := fmt.Sprintf("/api/v1/cli/plugins/%s/rollback", m.slug)
		var resp api.RollbackResponse
		if err := client.JSON("POST", path, map[string]string{"version": selected.Version}, &resp); err != nil {
			return rollbackResultMsg{err: err}
		}
		return rollbackResultMsg{resp: resp}
	}
}
