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

var errorsCmd = &cobra.Command{
	Use:   "errors <plugin-slug>",
	Short: "View plugin errors",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		slug := args[0]
		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed
		since, _ := cmd.Flags().GetString("since")
		limit, _ := cmd.Flags().GetInt("limit")

		if jsonOut || !ui.IsTTY() {
			return errorsPlain(slug, since, limit, jsonOut)
		}

		p := tea.NewProgram(newErrorsListModel(slug, since, limit))
		m, err := p.Run()
		if err != nil {
			return err
		}
		em := m.(errorsListModel)
		if em.err != nil {
			return em.err
		}
		return nil
	},
}

func init() {
	errorsCmd.Flags().String("since", "", "show errors since (e.g. 2024-01-01T00:00:00Z)")
	errorsCmd.Flags().Int("limit", 25, "max results (max 100)")
	rootCmd.AddCommand(errorsCmd)
}

func errorsPlain(slug, since string, limit int, jsonOut bool) error {
	client := api.NewAPIClient()
	path := buildErrorsPath(slug, since, limit)

	var resp api.ErrorsResponse
	if err := client.JSON("GET", path, nil, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	for _, l := range resp.Errors {
		detail := formatLogDetail(l)
		fmt.Printf("%-20s %-22s %s\n",
			l.OccurredAt.Format("2006-01-02 15:04:05"),
			l.EventType, detail)
	}
	return nil
}

func buildErrorsPath(slug, since string, limit int) string {
	path := fmt.Sprintf("/api/v1/cli/plugins/%s/errors", slug)
	sep := "?"
	if since != "" {
		path += sep + "since=" + since
		sep = "&"
	}
	if limit > 0 {
		path += sep + "limit=" + fmt.Sprintf("%d", limit)
	}
	return path
}

// TUI

type errorsListState int

const (
	errorsListLoading errorsListState = iota
	errorsListReady
)

type errorsListModel struct {
	state    errorsListState
	errors   []api.LogEntry
	cursor   int
	err      error
	slug     string
	since    string
	limit    int
	embedded bool
}

type errorsListResultMsg struct {
	errors []api.LogEntry
	err    error
}

func newErrorsListModel(slug, since string, limit int) errorsListModel {
	return errorsListModel{slug: slug, since: since, limit: limit}
}

func (m errorsListModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m errorsListModel) Init() tea.Cmd {
	return m.fetchErrors()
}

func (m errorsListModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
			if m.cursor < len(m.errors)-1 {
				m.cursor++
			}
		}

	case errorsListResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.errors = msg.errors
		m.state = errorsListReady
		if len(m.errors) == 0 {
			return m, m.done()
		}
		return m, nil
	}

	return m, nil
}

func (m errorsListModel) View() tea.View {
	switch m.state {
	case errorsListLoading:
		return tea.NewView("\n  Loading errors...\n\n")

	case errorsListReady:
		if len(m.errors) == 0 {
			return tea.NewView("\n  No errors found.\n\n")
		}

		headerStyle := lipgloss.NewStyle().Foreground(ui.Red).Bold(true)
		selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Red)
		normalStyle := lipgloss.NewStyle()

		var sb strings.Builder
		sb.WriteString("\n")

		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.Red)
		sb.WriteString("  " + titleStyle.Render("Errors — "+m.slug) + "\n\n")

		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			headerStyle.Render(pad("TIME", 20)),
			headerStyle.Render(pad("TYPE", 22)),
			headerStyle.Render(pad("DETAIL", 50)),
		))
		sb.WriteString("\n")

		for i, l := range m.errors {
			detail := formatLogDetail(l)
			created := l.OccurredAt.Format("2006-01-02 15:04:05")

			row := fmt.Sprintf("  %s  %s  %s",
				pad(created, 20),
				pad(l.EventType, 22),
				pad(detail, 50),
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

func (m errorsListModel) fetchErrors() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		path := buildErrorsPath(m.slug, m.since, m.limit)
		var resp api.ErrorsResponse
		if err := client.JSON("GET", path, nil, &resp); err != nil {
			return errorsListResultMsg{err: err}
		}
		return errorsListResultMsg{errors: resp.Errors}
	}
}
