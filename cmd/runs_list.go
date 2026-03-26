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

var runsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List automation runs",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed
		status, _ := cmd.Flags().GetString("status")
		limit, _ := cmd.Flags().GetInt("limit")

		if jsonOut || !ui.IsTTY() {
			return runsListPlain(status, limit, jsonOut)
		}

		p := tea.NewProgram(newRunsListModel(status, limit))
		m, err := p.Run()
		if err != nil {
			return err
		}
		rm := m.(runsListModel)
		if rm.err != nil {
			return rm.err
		}
		return nil
	},
}

func init() {
	runsListCmd.Flags().String("status", "", "filter by status")
	runsListCmd.Flags().Int("limit", 0, "max results")
	runsCmd.AddCommand(runsListCmd)
}

func runStatusStyle(status string) lipgloss.Style {
	switch status {
	case "running":
		return lipgloss.NewStyle().Foreground(ui.Blue)
	case "paused_for_decision":
		return lipgloss.NewStyle().Foreground(ui.Yellow)
	case "completed":
		return lipgloss.NewStyle().Foreground(ui.Green)
	case "failed":
		return lipgloss.NewStyle().Foreground(ui.Red)
	case "cancelled":
		return lipgloss.NewStyle().Foreground(ui.Dim)
	default:
		return lipgloss.NewStyle().Foreground(ui.White)
	}
}

func runsListPlain(status string, limit int, jsonOut bool) error {
	client := api.NewAPIClient()
	path := "/api/v1/cli/runs"
	sep := "?"
	if status != "" {
		path += sep + "status=" + status
		sep = "&"
	}
	if limit > 0 {
		path += sep + "limit=" + strconv.Itoa(limit)
	}

	var resp api.RunsResponse
	if err := client.JSON("GET", path, nil, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	for _, r := range resp.Runs {
		fmt.Printf("%-12s %-20s %-10s %-20s %s\n",
			r.ShortToken, r.Automation, r.Status, r.State,
			r.CreatedAt.Format("2006-01-02 15:04:05"))
	}
	return nil
}

// TUI

type runsListState int

const (
	runsListLoading runsListState = iota
	runsListReady
)

type runsListModel struct {
	state    runsListState
	runs     []api.Run
	cursor   int
	err      error
	status   string
	limit    int
	embedded bool
}

func (m runsListModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

type runsListResultMsg struct {
	runs []api.Run
	err  error
}

func newRunsListModel(status string, limit int) runsListModel {
	return runsListModel{status: status, limit: limit}
}

func (m runsListModel) Init() tea.Cmd {
	return m.fetchRuns()
}

func (m runsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.runs)-1 {
				m.cursor++
			}
		}

	case runsListResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.runs = msg.runs
		if len(m.runs) == 0 {
			m.state = runsListReady
			return m, m.done()
		}
		m.state = runsListReady
		return m, nil
	}

	return m, nil
}

func (m runsListModel) View() tea.View {
	switch m.state {
	case runsListLoading:
		return tea.NewView("\n  Loading runs...\n\n")

	case runsListReady:
		if len(m.runs) == 0 {
			return tea.NewView("\n  No runs found.\n\n")
		}

		headerStyle := lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
		selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Blue)
		normalStyle := lipgloss.NewStyle()

		var sb strings.Builder
		sb.WriteString("\n")

		sb.WriteString(fmt.Sprintf("  %s  %s  %s  %s  %s\n",
			headerStyle.Render(pad("TOKEN", 12)),
			headerStyle.Render(pad("AUTOMATION", 20)),
			headerStyle.Render(pad("STATUS", 22)),
			headerStyle.Render(pad("STATE", 22)),
			headerStyle.Render(pad("CREATED", 20)),
		))
		sb.WriteString("\n")

		for i, r := range m.runs {
			styledStatus := runStatusStyle(r.Status).Render(pad(r.Status, 22))
			created := r.CreatedAt.Format("2006-01-02 15:04:05")

			if i == m.cursor {
				// For selected row, render without status color (highlight takes over)
				row := fmt.Sprintf("  %s  %s  %s  %s  %s",
					pad(r.ShortToken, 12),
					pad(r.Automation, 20),
					pad(r.Status, 22),
					pad(r.State, 22),
					pad(created, 20),
				)
				sb.WriteString(selectedStyle.Render(row) + "\n")
			} else {
				// Unselected: color the status column
				row := fmt.Sprintf("  %s  %s  %s  %s  %s",
					pad(r.ShortToken, 12),
					pad(r.Automation, 20),
					styledStatus,
					pad(r.State, 22),
					pad(created, 20),
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

func (m runsListModel) fetchRuns() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		path := "/api/v1/cli/runs"
		sep := "?"
		if m.status != "" {
			path += sep + "status=" + m.status
			sep = "&"
		}
		if m.limit > 0 {
			path += sep + "limit=" + strconv.Itoa(m.limit)
		}

		var resp api.RunsResponse
		if err := client.JSON("GET", path, nil, &resp); err != nil {
			return runsListResultMsg{err: err}
		}
		return runsListResultMsg{runs: resp.Runs}
	}
}
