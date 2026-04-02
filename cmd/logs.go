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

var logsCmd = &cobra.Command{
	Use:   "logs <plugin-slug>",
	Short: "View plugin logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		slug := args[0]
		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed
		eventType, _ := cmd.Flags().GetString("type")
		severity, _ := cmd.Flags().GetString("severity")
		since, _ := cmd.Flags().GetString("since")
		limit, _ := cmd.Flags().GetInt("limit")

		if jsonOut || !ui.IsTTY() {
			return logsPlain(slug, eventType, severity, since, limit, jsonOut)
		}

		p := tea.NewProgram(newLogsListModel(slug, eventType, severity, since, limit))
		m, err := p.Run()
		if err != nil {
			return err
		}
		lm := m.(logsListModel)
		if lm.err != nil {
			return lm.err
		}
		return nil
	},
}

func init() {
	logsCmd.Flags().String("type", "", "filter by event type")
	logsCmd.Flags().String("severity", "", "filter by severity (info, warning, error)")
	logsCmd.Flags().String("since", "", "show logs since (e.g. 2024-01-01T00:00:00Z)")
	logsCmd.Flags().Int("limit", 50, "max results (max 200)")
	rootCmd.AddCommand(logsCmd)
}

func severityStyle(severity string) lipgloss.Style {
	switch severity {
	case "info":
		return lipgloss.NewStyle().Foreground(ui.Blue)
	case "warning":
		return lipgloss.NewStyle().Foreground(ui.Yellow)
	case "error":
		return lipgloss.NewStyle().Foreground(ui.Red)
	default:
		return lipgloss.NewStyle().Foreground(ui.White)
	}
}

func logsPlain(slug, eventType, severity, since string, limit int, jsonOut bool) error {
	client := api.NewAPIClient()
	path := buildLogsPath(slug, eventType, severity, since, limit)

	var resp api.LogsResponse
	if err := client.JSON("GET", path, nil, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	for _, l := range resp.Logs {
		detail := formatLogDetail(l)
		fmt.Printf("%-20s %-22s %-10s %s\n",
			l.OccurredAt.Format("2006-01-02 15:04:05"),
			l.EventType, l.Severity, detail)
	}
	return nil
}

func buildLogsPath(slug, eventType, severity, since string, limit int) string {
	path := fmt.Sprintf("/api/v1/cli/plugins/%s/logs", slug)
	sep := "?"
	if eventType != "" {
		path += sep + "type=" + eventType
		sep = "&"
	}
	if severity != "" {
		path += sep + "severity=" + severity
		sep = "&"
	}
	if since != "" {
		path += sep + "since=" + since
		sep = "&"
	}
	if limit > 0 {
		path += sep + "limit=" + fmt.Sprintf("%d", limit)
	}
	return path
}

func formatLogDetail(l api.LogEntry) string {
	parts := []string{}
	if v, ok := l.Metadata["tool_name"]; ok {
		parts = append(parts, fmt.Sprintf("tool=%v", v))
	}
	if v, ok := l.Metadata["automation_name"]; ok {
		parts = append(parts, fmt.Sprintf("automation=%v", v))
	}
	if v, ok := l.Metadata["error"]; ok {
		parts = append(parts, fmt.Sprintf("error=%v", v))
	}
	if v, ok := l.Metadata["version"]; ok {
		parts = append(parts, fmt.Sprintf("version=%v", v))
	}
	if v, ok := l.Metadata["duration_ms"]; ok {
		parts = append(parts, fmt.Sprintf("duration=%vms", v))
	}
	return strings.Join(parts, " ")
}

// TUI

type logsListState int

const (
	logsListLoading logsListState = iota
	logsListReady
)

type logsListModel struct {
	state     logsListState
	logs      []api.LogEntry
	cursor    int
	err       error
	slug      string
	eventType string
	severity  string
	since     string
	limit     int
	embedded  bool
}

type logsListResultMsg struct {
	logs []api.LogEntry
	err  error
}

func newLogsListModel(slug, eventType, severity, since string, limit int) logsListModel {
	return logsListModel{slug: slug, eventType: eventType, severity: severity, since: since, limit: limit}
}

func (m logsListModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m logsListModel) Init() tea.Cmd {
	return m.fetchLogs()
}

func (m logsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.logs)-1 {
				m.cursor++
			}
		}

	case logsListResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.logs = msg.logs
		m.state = logsListReady
		if len(m.logs) == 0 {
			return m, m.done()
		}
		return m, nil
	}

	return m, nil
}

func (m logsListModel) View() tea.View {
	switch m.state {
	case logsListLoading:
		return tea.NewView("\n  Loading logs...\n\n")

	case logsListReady:
		if len(m.logs) == 0 {
			return tea.NewView("\n  No logs found.\n\n")
		}

		headerStyle := lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
		selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Blue)
		normalStyle := lipgloss.NewStyle()

		var sb strings.Builder
		sb.WriteString("\n")

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)
		sb.WriteString("  " + titleStyle.Render("Plugin Logs — "+m.slug) + "\n\n")

		sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s\n",
			headerStyle.Render(pad("TIME", 20)),
			headerStyle.Render(pad("TYPE", 22)),
			headerStyle.Render(pad("SEVERITY", 10)),
			headerStyle.Render(pad("DETAIL", 40)),
		))
		sb.WriteString("\n")

		for i, l := range m.logs {
			styledSev := severityStyle(l.Severity).Render(pad(l.Severity, 10))
			detail := formatLogDetail(l)
			created := l.OccurredAt.Format("2006-01-02 15:04:05")

			if i == m.cursor {
				row := fmt.Sprintf("  %s  %s  %s  %s",
					pad(created, 20),
					pad(l.EventType, 22),
					pad(l.Severity, 10),
					pad(detail, 40),
				)
				sb.WriteString(selectedStyle.Render(row) + "\n")
			} else {
				row := fmt.Sprintf("  %s  %s  %s  %s",
					pad(created, 20),
					pad(l.EventType, 22),
					styledSev,
					pad(detail, 40),
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

func (m logsListModel) fetchLogs() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		path := buildLogsPath(m.slug, m.eventType, m.severity, m.since, m.limit)
		var resp api.LogsResponse
		if err := client.JSON("GET", path, nil, &resp); err != nil {
			return logsListResultMsg{err: err}
		}
		return logsListResultMsg{logs: resp.Logs}
	}
}
