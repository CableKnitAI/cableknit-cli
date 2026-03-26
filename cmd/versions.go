package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var versionsCmd = &cobra.Command{
	Use:   "versions <plugin-slug>",
	Short: "View version history for a plugin",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		slug := args[0]
		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed

		if jsonOut || !ui.IsTTY() {
			return versionsPlain(slug, jsonOut)
		}

		p := tea.NewProgram(newVersionsListModel(slug))
		m, err := p.Run()
		if err != nil {
			return err
		}
		vm := m.(versionsListModel)
		if vm.err != nil {
			return vm.err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionsCmd)
}

func versionsPlain(slug string, jsonOut bool) error {
	client := api.NewAPIClient()
	path := fmt.Sprintf("/api/v1/cli/plugins/%s/versions", slug)

	var resp api.VersionsResponse
	if err := client.JSON("GET", path, nil, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	for _, v := range resp.Versions {
		notes := v.Notes
		if notes == "" {
			notes = "-"
		}
		fmt.Printf("%-12s  %-20s  %s\n",
			v.Version,
			v.PushedAt.Format("2006-01-02 15:04:05"),
			notes)
	}
	return nil
}

// TUI

type versionsListState int

const (
	versionsListLoading versionsListState = iota
	versionsListReady
)

type versionsListModel struct {
	state    versionsListState
	versions []api.PluginVersion
	cursor   int
	err      error
	slug     string
	embedded bool
}

type versionsListResultMsg struct {
	versions []api.PluginVersion
	err      error
}

func newVersionsListModel(slug string) versionsListModel {
	return versionsListModel{slug: slug}
}

func (m versionsListModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m versionsListModel) Init() tea.Cmd {
	return m.fetchVersions()
}

func (m versionsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, m.done()
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.versions)-1 {
				m.cursor++
			}
		}

	case versionsListResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.versions = msg.versions
		m.state = versionsListReady
		if len(m.versions) == 0 {
			return m, m.done()
		}
		return m, nil
	}

	return m, nil
}

func (m versionsListModel) View() tea.View {
	switch m.state {
	case versionsListLoading:
		return tea.NewView("\n  Loading versions...\n\n")

	case versionsListReady:
		if len(m.versions) == 0 {
			return tea.NewView("\n  No versions found.\n\n")
		}

		headerStyle := lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
		selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Blue)
		normalStyle := lipgloss.NewStyle()

		var sb strings.Builder
		sb.WriteString("\n")

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)
		sb.WriteString("  " + titleStyle.Render("Version History — "+m.slug) + "\n\n")

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

			if i == m.cursor {
				row := fmt.Sprintf("  %s  %s  %s",
					pad(v.Version, 12),
					pad(pushed, 20),
					pad(notes, 40),
				)
				sb.WriteString(selectedStyle.Render(row) + "\n")
			} else {
				row := fmt.Sprintf("  %s  %s  %s",
					pad(v.Version, 12),
					pad(pushed, 20),
					pad(notes, 40),
				)
				sb.WriteString(normalStyle.Render(row) + "\n")
			}
		}

		sb.WriteString("\n")
		if m.embedded {
			sb.WriteString(ui.DimStyle.Render("  ↑↓ navigate  q/esc back") + "\n")
		} else {
			sb.WriteString(ui.DimStyle.Render("  ↑↓ navigate  q to quit") + "\n")
		}

		return tea.NewView(sb.String())
	}

	if m.err != nil {
		return tea.NewView("\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n")
	}

	return tea.NewView("")
}

func (m versionsListModel) fetchVersions() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		path := fmt.Sprintf("/api/v1/cli/plugins/%s/versions", m.slug)
		var resp api.VersionsResponse
		if err := client.JSON("GET", path, nil, &resp); err != nil {
			return versionsListResultMsg{err: err}
		}
		return versionsListResultMsg{versions: resp.Versions}
	}
}
