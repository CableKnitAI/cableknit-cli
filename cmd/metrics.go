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

var metricsCmd = &cobra.Command{
	Use:   "metrics <plugin-slug>",
	Short: "View plugin metrics dashboard",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		slug := args[0]
		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed

		if jsonOut || !ui.IsTTY() {
			return metricsPlain(slug, jsonOut)
		}

		p := tea.NewProgram(newMetricsModel(slug))
		m, err := p.Run()
		if err != nil {
			return err
		}
		mm := m.(metricsModel)
		if mm.err != nil {
			return mm.err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(metricsCmd)
}

func metricsPlain(slug string, jsonOut bool) error {
	client := api.NewAPIClient()
	path := fmt.Sprintf("/api/v1/cli/plugins/%s/metrics", slug)

	var resp api.MetricsResponse
	if err := client.JSON("GET", path, nil, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Tool Calls (24h): %d\n", resp.ToolCalls24h)
	fmt.Printf("Tool Calls (7d):  %d\n", resp.ToolCalls7d)
	fmt.Printf("Tool Calls (30d): %d\n", resp.ToolCalls30d)
	fmt.Printf("Errors (24h):     %d\n", resp.Errors24h)
	fmt.Printf("Error Rate (7d):  %.2f%%\n", resp.ErrorRate7d)
	fmt.Printf("Automation Runs (7d):     %d\n", resp.AutomationRuns7d)
	fmt.Printf("Automation Failures (7d): %d\n", resp.AutomationFailures7d)
	fmt.Printf("Active Installs:  %d\n", resp.ActiveInstalls)
	fmt.Printf("Avg Sandbox (ms): %.1f\n", resp.AvgSandboxDurationMs)
	return nil
}

// TUI

type metricsState int

const (
	metricsLoading metricsState = iota
	metricsReady
)

type metricsModel struct {
	state    metricsState
	metrics  api.MetricsResponse
	err      error
	slug     string
	embedded bool
}

type metricsResultMsg struct {
	metrics api.MetricsResponse
	err     error
}

func newMetricsModel(slug string) metricsModel {
	return metricsModel{slug: slug}
}

func (m metricsModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m metricsModel) Init() tea.Cmd {
	return m.fetchMetrics()
}

func (m metricsModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "q", "esc":
			return m, m.done()
		}

	case metricsResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, m.done()
		}
		m.metrics = msg.metrics
		m.state = metricsReady
		return m, nil
	}

	return m, nil
}

func (m metricsModel) View() tea.View {
	switch m.state {
	case metricsLoading:
		return tea.NewView("\n  Loading metrics...\n\n")

	case metricsReady:
		titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)
		labelStyle := lipgloss.NewStyle().Bold(true)
		sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(ui.Yellow)
		valueStyle := lipgloss.NewStyle().Foreground(ui.White)
		errorVal := lipgloss.NewStyle().Foreground(ui.Red)

		var sb strings.Builder
		sb.WriteString("\n")
		sb.WriteString("  " + titleStyle.Render("Metrics — "+m.slug) + "\n\n")

		mr := m.metrics

		// Tool Calls
		sb.WriteString("  " + sectionStyle.Render("TOOL CALLS") + "\n")
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("24h:"), valueStyle.Render(fmt.Sprintf("%d", mr.ToolCalls24h))))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("7d: "), valueStyle.Render(fmt.Sprintf("%d", mr.ToolCalls7d))))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("30d:"), valueStyle.Render(fmt.Sprintf("%d", mr.ToolCalls30d))))
		sb.WriteString("\n")

		// Errors
		sb.WriteString("  " + sectionStyle.Render("ERRORS") + "\n")
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("24h:      "), errorVal.Render(fmt.Sprintf("%d", mr.Errors24h))))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Rate (7d):"), errorVal.Render(fmt.Sprintf("%.2f%%", mr.ErrorRate7d))))
		sb.WriteString("\n")

		// Automations
		sb.WriteString("  " + sectionStyle.Render("AUTOMATIONS") + "\n")
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Runs (7d):    "), valueStyle.Render(fmt.Sprintf("%d", mr.AutomationRuns7d))))
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Failures (7d):"), errorVal.Render(fmt.Sprintf("%d", mr.AutomationFailures7d))))
		sb.WriteString("\n")

		// Sandbox
		sb.WriteString("  " + sectionStyle.Render("SANDBOX") + "\n")
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Avg Duration:"), valueStyle.Render(fmt.Sprintf("%.1f ms", mr.AvgSandboxDurationMs))))
		sb.WriteString("\n")

		// Installs
		sb.WriteString("  " + sectionStyle.Render("INSTALLS") + "\n")
		sb.WriteString(fmt.Sprintf("  %s  %s\n", labelStyle.Render("Active:"), valueStyle.Render(fmt.Sprintf("%d", mr.ActiveInstalls))))
		sb.WriteString("\n")

		if m.embedded {
			sb.WriteString(ui.DimStyle.Render("  q/esc back") + "\n")
		} else {
			sb.WriteString(ui.DimStyle.Render("  q to quit") + "\n")
		}

		return tea.NewView(sb.String())
	}

	if m.err != nil {
		return tea.NewView("\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n")
	}

	return tea.NewView("")
}

func (m metricsModel) fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		path := fmt.Sprintf("/api/v1/cli/plugins/%s/metrics", m.slug)
		var resp api.MetricsResponse
		if err := client.JSON("GET", path, nil, &resp); err != nil {
			return metricsResultMsg{err: err}
		}
		return metricsResultMsg{metrics: resp}
	}
}
