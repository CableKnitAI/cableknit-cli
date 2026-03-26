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

var toolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available platform tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed

		if jsonOut || !ui.IsTTY() {
			return toolsListPlain(jsonOut)
		}

		p := tea.NewProgram(newToolsListModel())
		m, err := p.Run()
		if err != nil {
			return err
		}
		tm := m.(toolsListModel)
		if tm.err != nil {
			return tm.err
		}
		return nil
	},
}

func init() {
	toolsCmd.AddCommand(toolsListCmd)
}

func toolsListPlain(jsonOut bool) error {
	client := api.NewClient()
	var resp api.PlatformToolsResponse
	if err := client.JSON("GET", "/api/v1/cli/platform_tools", nil, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	for _, t := range resp.PlatformTools {
		fmt.Printf("%-24s %-22s %-12s %s\n",
			t.Name, t.Slug, t.Category, t.Description)
	}
	return nil
}

// TUI

type toolsListState int

const (
	toolsListLoading toolsListState = iota
	toolsListReady
	toolsListDetail
)

type toolsListModel struct {
	state    toolsListState
	tools    []api.PlatformTool
	cursor   int
	err      error
	embedded bool
}

type toolsListResultMsg struct {
	tools []api.PlatformTool
	err   error
}

func newToolsListModel() toolsListModel {
	return toolsListModel{}
}

func (m toolsListModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m toolsListModel) Init() tea.Cmd {
	return m.fetchTools()
}

func (m toolsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			if m.state == toolsListDetail {
				m.state = toolsListReady
				return m, nil
			}
			return m, m.done()
		case "up", "k":
			if m.state == toolsListReady && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.state == toolsListReady && m.cursor < len(m.tools)-1 {
				m.cursor++
			}
		case "enter":
			if m.state == toolsListReady && len(m.tools) > 0 {
				m.state = toolsListDetail
				return m, nil
			}
			if m.state == toolsListDetail {
				m.state = toolsListReady
				return m, nil
			}
		}

	case toolsListResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.tools = msg.tools
		if len(m.tools) == 0 {
			m.state = toolsListReady
			return m, m.done()
		}
		m.state = toolsListReady
		return m, nil
	}

	return m, nil
}

func (m toolsListModel) View() tea.View {
	switch m.state {
	case toolsListLoading:
		return tea.NewView("\n  Loading platform tools...\n\n")

	case toolsListDetail:
		t := m.tools[m.cursor]
		bold := lipgloss.NewStyle().Bold(true)
		labelStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)

		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString("  " + labelStyle.Render("Platform Tool") + "\n\n")
		sb.WriteString(fmt.Sprintf("  %s  %s\n", bold.Render("Name:"), t.Name))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", bold.Render("Slug:"), t.Slug))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", bold.Render("Category:"), t.Category))
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n", bold.Render("Description:")))
		sb.WriteString("  " + t.Description + "\n")
		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("  %s\n", bold.Render("Usage in plugin.json:")))
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(ui.Red).Render(
			fmt.Sprintf(`"platform_tools": ["%s"]`, t.Slug),
		) + "\n")
		sb.WriteString("\n")
		sb.WriteString(ui.DimStyle.Render("  enter/esc back to list") + "\n")

		return tea.NewView(sb.String())

	case toolsListReady:
		if len(m.tools) == 0 {
			return tea.NewView("\n  No platform tools found.\n\n")
		}

		headerStyle := lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
		selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Blue)
		normalStyle := lipgloss.NewStyle()

		var sb strings.Builder
		sb.WriteString("\n")

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)
		sb.WriteString("  " + titleStyle.Render("Platform Tools") + "\n\n")
		sb.WriteString(ui.DimStyle.Render("  Plugins can call shared tools provided by the platform at runtime via") + "\n")
		sb.WriteString(ui.DimStyle.Render("  LLM function calling. Declare the slugs you need in plugin.json:") + "\n")
		sb.WriteString(ui.DimStyle.Render(`  "platform_tools": ["lookup-employees", "fetch-briefing"]`) + "\n\n")
		sb.WriteString(ui.DimStyle.Render("  Platform tools are always available. Contextual tools activate based") + "\n")
		sb.WriteString(ui.DimStyle.Render("  on user state (e.g. attached files, active spreadsheet).") + "\n\n")

		// Header
		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			headerStyle.Render(pad("NAME", 24)),
			headerStyle.Render(pad("SLUG", 22)),
			headerStyle.Render(pad("CATEGORY", 12)),
		))
		sb.WriteString("\n")

		// Rows
		for i, t := range m.tools {
			row := fmt.Sprintf("  %s  %s  %s",
				pad(t.Name, 24),
				pad(t.Slug, 22),
				pad(t.Category, 12),
			)
			if i == m.cursor {
				sb.WriteString(selectedStyle.Render(row) + "\n")
			} else {
				sb.WriteString(normalStyle.Render(row) + "\n")
			}
		}

		sb.WriteString("\n")
		if m.embedded {
			sb.WriteString(ui.DimStyle.Render("  ↑↓ navigate  enter detail  q/esc back") + "\n")
		} else {
			sb.WriteString(ui.DimStyle.Render("  ↑↓ navigate  enter detail  q to quit") + "\n")
		}

		return tea.NewView(sb.String())
	}

	if m.err != nil {
		return tea.NewView("\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n")
	}

	return tea.NewView("")
}

func (m toolsListModel) fetchTools() tea.Cmd {
	return func() tea.Msg {
		client := api.NewClient()
		var resp api.PlatformToolsResponse
		if err := client.JSON("GET", "/api/v1/cli/platform_tools", nil, &resp); err != nil {
			return toolsListResultMsg{err: err}
		}
		return toolsListResultMsg{tools: resp.PlatformTools}
	}
}
