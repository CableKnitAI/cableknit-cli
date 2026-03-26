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

var connectorsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available connectors",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed

		if jsonOut || !ui.IsTTY() {
			return connectorsListPlain(jsonOut)
		}

		p := tea.NewProgram(newConnectorsListModel())
		m, err := p.Run()
		if err != nil {
			return err
		}
		cm := m.(connectorsListModel)
		if cm.err != nil {
			return cm.err
		}
		return nil
	},
}

func init() {
	connectorsCmd.AddCommand(connectorsListCmd)
}

func connectorsListPlain(jsonOut bool) error {
	client := api.NewClient()
	var resp api.ConnectorsResponse
	if err := client.JSON("GET", "/api/v1/cli/connectors", nil, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	for _, c := range resp.Connectors {
		fmt.Printf("%-18s %-15s %-15s %-18s %s\n",
			c.Name, c.Slug, c.Category, c.AuthType, c.Status)
	}
	return nil
}

// TUI

type connectorsListState int

const (
	connectorsListLoading connectorsListState = iota
	connectorsListReady
)

type connectorsListModel struct {
	state      connectorsListState
	connectors []api.Connector
	cursor     int
	err        error
	embedded   bool
}

type connectorsListResultMsg struct {
	connectors []api.Connector
	err        error
}

func newConnectorsListModel() connectorsListModel {
	return connectorsListModel{}
}

func (m connectorsListModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m connectorsListModel) Init() tea.Cmd {
	return m.fetchConnectors()
}

func (m connectorsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.connectors)-1 {
				m.cursor++
			}
		}

	case connectorsListResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.connectors = msg.connectors
		if len(m.connectors) == 0 {
			m.state = connectorsListReady
			return m, m.done()
		}
		m.state = connectorsListReady
		return m, nil
	}

	return m, nil
}

func (m connectorsListModel) View() tea.View {
	switch m.state {
	case connectorsListLoading:
		return tea.NewView("\n  Loading connectors...\n\n")

	case connectorsListReady:
		if len(m.connectors) == 0 {
			return tea.NewView("\n  No connectors found.\n\n")
		}

		headerStyle := lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
		selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Blue)
		normalStyle := lipgloss.NewStyle()

		var sb strings.Builder
		sb.WriteString("\n")

		// Header
		sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s  %s\n",
			headerStyle.Render(pad("NAME", 20)),
			headerStyle.Render(pad("SLUG", 16)),
			headerStyle.Render(pad("CATEGORY", 20)),
			headerStyle.Render(pad("AUTH", 14)),
			headerStyle.Render(pad("STATUS", 10)),
		))
		sb.WriteString("\n")

		// Rows
		for i, c := range m.connectors {
			row := fmt.Sprintf("  %s  %s  %s  %s  %s",
				pad(c.Name, 20),
				pad(c.Slug, 16),
				pad(c.Category, 20),
				pad(c.AuthType, 14),
				pad(c.Status, 10),
			)
			if i == m.cursor {
				sb.WriteString(selectedStyle.Render(row) + "\n")
			} else {
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

func (m connectorsListModel) fetchConnectors() tea.Cmd {
	return func() tea.Msg {
		client := api.NewClient()
		var resp api.ConnectorsResponse
		if err := client.JSON("GET", "/api/v1/cli/connectors", nil, &resp); err != nil {
			return connectorsListResultMsg{err: err}
		}
		return connectorsListResultMsg{connectors: resp.Connectors}
	}
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}
