package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your plugins",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed

		if jsonOut || !ui.IsTTY() {
			return pluginsListPlain(jsonOut)
		}

		p := tea.NewProgram(newPluginsListModel())
		m, err := p.Run()
		if err != nil {
			return err
		}
		pm := m.(pluginsListModel)
		if pm.err != nil {
			return pm.err
		}
		return nil
	},
}

func init() {
	pluginsCmd.AddCommand(pluginsListCmd)
}

func pluginsListPlain(jsonOut bool) error {
	client := api.NewAPIClient()
	var resp api.PluginsResponse
	if err := client.JSON("GET", "/api/v1/cli/plugins", nil, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	for _, p := range resp.Plugins {
		fmt.Printf("%-20s %-15s v%-8s %-10s %-10s %d\n",
			p.Name, p.Slug, p.Version, p.Status, p.Visibility, p.Installs)
	}
	return nil
}

// TUI

type pluginsListState int

const (
	pluginsListLoading pluginsListState = iota
	pluginsListReady
)

type pluginsListModel struct {
	state    pluginsListState
	plugins  []api.Plugin
	cursor   int
	err      error
	embedded bool
}

func (m pluginsListModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

type pluginsListResultMsg struct {
	plugins []api.Plugin
	err     error
}

func newPluginsListModel() pluginsListModel {
	return pluginsListModel{}
}

func (m pluginsListModel) Init() tea.Cmd {
	return m.fetchPlugins()
}

func (m pluginsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.plugins)-1 {
				m.cursor++
			}
		}

	case pluginsListResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.plugins = msg.plugins
		if len(m.plugins) == 0 {
			m.state = pluginsListReady
			return m, m.done()
		}
		m.state = pluginsListReady
		return m, nil
	}

	return m, nil
}

func (m pluginsListModel) View() tea.View {
	switch m.state {
	case pluginsListLoading:
		return tea.NewView("\n  Loading plugins...\n\n")

	case pluginsListReady:
		if len(m.plugins) == 0 {
			return tea.NewView("\n  No plugins found.\n\n")
		}

		headerStyle := lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
		selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Blue)
		normalStyle := lipgloss.NewStyle()

		var sb strings.Builder
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s  %s  %s\n",
			headerStyle.Render(pad("NAME", 20)),
			headerStyle.Render(pad("SLUG", 16)),
			headerStyle.Render(pad("VERSION", 10)),
			headerStyle.Render(pad("STATUS", 12)),
			headerStyle.Render(pad("VISIBILITY", 12)),
			headerStyle.Render(pad("INSTALLS", 10)),
		))
		sb.WriteString("\n")

		for i, p := range m.plugins {
			row := fmt.Sprintf("  %s  %s  %s  %s  %s  %s",
				pad(p.Name, 20),
				pad(p.Slug, 16),
				pad(p.Version, 10),
				pad(p.Status, 12),
				pad(p.Visibility, 12),
				pad(strconv.Itoa(p.Installs), 10),
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

func (m pluginsListModel) fetchPlugins() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		var resp api.PluginsResponse
		if err := client.JSON("GET", "/api/v1/cli/plugins", nil, &resp); err != nil {
			return pluginsListResultMsg{err: err}
		}
		return pluginsListResultMsg{plugins: resp.Plugins}
	}
}
