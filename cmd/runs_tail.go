package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/cableknitai/cableknit-cli/internal/api"
	"github.com/cableknitai/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
)

var runsTailCmd = &cobra.Command{
	Use:   "tail <run-id>",
	Short: "Stream run logs",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		noTUI, _ := cmd.Flags().GetBool("no-tui")
		runID := args[0]

		if noTUI || !ui.IsTTY() {
			return tailPlain(runID)
		}

		p := tea.NewProgram(newTailModel(runID))
		_, err := p.Run()
		return err
	},
}

func init() {
	runsTailCmd.Flags().Bool("no-tui", false, "plain text output, no TUI")
	runsCmd.AddCommand(runsTailCmd)
}

func tailPlain(runID string) error {
	client := api.NewAPIClient()
	resp, err := client.SSE(fmt.Sprintf("/api/v1/cli/runs/%s/tail", runID))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimPrefix(line, "data:")
		data = strings.TrimSpace(data)

		var ev api.RunEvent
		if json.Unmarshal([]byte(data), &ev) == nil {
			ts := ev.Timestamp.Format("15:04:05")
			fmt.Printf("[%s] %s\n", ts, ev.Message)
		}
	}
	return scanner.Err()
}

// TUI

type tailPhase int

const (
	tailPhaseConnecting tailPhase = iota
	tailPhaseStreaming
	tailPhaseDone
)

type tailModel struct {
	phase      tailPhase
	runID      string
	spinner    spinner.Model
	viewport   viewport.Model
	lines      []string
	elapsed    time.Duration
	finalState string
	err        error
	width      int
	height     int
	ready      bool
	eventCh    chan tea.Msg
	embedded   bool
}

func (m tailModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

type tailEventMsg struct {
	event api.RunEvent
}

type tailErrorMsg struct {
	err error
}

type tailDoneMsg struct {
	lastEvent api.RunEvent
}

type tailTickMsg struct{}

func newTailModel(runID string) tailModel {
	return tailModel{
		runID:   runID,
		spinner: ui.NewSpinner(spinner.Globe),
		eventCh: make(chan tea.Msg, 64),
	}
}

func (m tailModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.startSSE(), m.waitForEvent(), m.tickElapsed())
}

func (m tailModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if msg.String() == "q" || msg.String() == "esc" {
			return m, m.done()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		headerHeight := 3
		footerHeight := 2
		vpHeight := m.height - headerHeight - footerHeight
		if vpHeight < 1 {
			vpHeight = 1
		}
		if !m.ready {
			m.viewport = viewport.New(
				viewport.WithWidth(m.width),
				viewport.WithHeight(vpHeight),
			)
			m.ready = true
		} else {
			m.viewport.SetWidth(m.width)
			m.viewport.SetHeight(vpHeight)
		}

	case tailEventMsg:
		m.phase = tailPhaseStreaming
		styled := m.styleLine(msg.event)
		m.lines = append(m.lines, styled)
		if m.ready {
			m.viewport.SetContent(strings.Join(m.lines, "\n"))
			m.viewport.GotoBottom()
		}
		return m, m.waitForEvent()

	case tailDoneMsg:
		m.phase = tailPhaseDone
		m.finalState = msg.lastEvent.Type
		styled := m.styleLine(msg.lastEvent)
		m.lines = append(m.lines, styled)
		if m.ready {
			m.viewport.SetContent(strings.Join(m.lines, "\n"))
			m.viewport.GotoBottom()
		}
		// Brief pause before exit
		done := m.done()
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return done()
		})

	case tailErrorMsg:
		m.err = msg.err
		return m, m.done()

	case tailTickMsg:
		if m.phase == tailPhaseStreaming {
			m.elapsed += time.Second
		}
		return m, m.tickElapsed()
	}

	if m.phase != tailPhaseDone {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds := []tea.Cmd{cmd}

		if m.ready {
			var vpCmd tea.Cmd
			m.viewport, vpCmd = m.viewport.Update(msg)
			cmds = append(cmds, vpCmd)
		}
		return m, tea.Batch(cmds...)
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m tailModel) View() tea.View {
	if m.err != nil {
		return tea.NewView(ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n")
	}

	var s string

	// Header
	header := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue).
		Render(fmt.Sprintf(" Run %s", m.runID))

	switch m.phase {
	case tailPhaseConnecting:
		s = header + "\n\n"
		s += "  " + m.spinner.View() + fmt.Sprintf(" Connecting to run %s...", m.runID)
		s += "\n"

	case tailPhaseStreaming:
		elapsed := formatDuration(m.elapsed)
		status := m.spinner.View() + " streaming " + ui.DimStyle.Render(elapsed)
		s = header + "  " + status + "\n\n"
		if m.ready {
			s += m.viewport.View()
		}
		s += "\n" + ui.DimStyle.Render(" ctrl+c to exit")

	case tailPhaseDone:
		summary := m.summaryLine()
		s = header + "  " + summary + "\n\n"
		if m.ready {
			s += m.viewport.View()
		}
		s += "\n"
	}

	v := tea.NewView(s)
	if !m.embedded {
		v.AltScreen = true
	}
	return v
}

func (m tailModel) styleLine(ev api.RunEvent) string {
	ts := ui.DimStyle.Render(ev.Timestamp.Format("15:04:05"))
	var msg string

	switch ev.Type {
	case "completed":
		msg = ui.SuccessStyle.Render(ui.SymbolCheck + " " + ev.Message)
	case "failed":
		msg = ui.ErrorStyle.Render(ui.SymbolCross + " " + ev.Message)
	case "paused_for_decision":
		msg = ui.WarningStyle.Render(ui.SymbolWarning + " " + ev.Message)
	default:
		msg = ev.Message
	}

	return fmt.Sprintf("  %s  %s", ts, msg)
}

func (m tailModel) summaryLine() string {
	switch m.finalState {
	case "completed":
		return ui.SuccessStyle.Render(ui.SymbolCheck + " completed")
	case "failed":
		return ui.ErrorStyle.Render(ui.SymbolCross + " failed")
	case "cancelled":
		return ui.DimStyle.Render("cancelled")
	default:
		return m.finalState
	}
}

func (m tailModel) startSSE() tea.Cmd {
	ch := m.eventCh
	runID := m.runID
	return func() tea.Msg {
		client := api.NewAPIClient()
		resp, err := client.SSE(fmt.Sprintf("/api/v1/cli/runs/%s/tail", runID))
		if err != nil {
			ch <- tailErrorMsg{err: err}
			return nil
		}

		go func() {
			defer resp.Body.Close()
			scanner := bufio.NewScanner(resp.Body)
			for scanner.Scan() {
				line := scanner.Text()
				if !strings.HasPrefix(line, "data:") {
					continue
				}
				data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))

				var ev api.RunEvent
				if json.Unmarshal([]byte(data), &ev) != nil {
					continue
				}

				switch ev.Type {
				case "completed", "failed", "cancelled":
					ch <- tailDoneMsg{lastEvent: ev}
					return
				default:
					ch <- tailEventMsg{event: ev}
				}
			}
			if err := scanner.Err(); err != nil {
				ch <- tailErrorMsg{err: err}
				return
			}
			ch <- tailDoneMsg{}
		}()

		return nil
	}
}

func (m tailModel) waitForEvent() tea.Cmd {
	ch := m.eventCh
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return tailDoneMsg{}
		}
		return msg
	}
}

func (m tailModel) tickElapsed() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tailTickMsg{}
	})
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	s := (d - m*time.Minute) / time.Second
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
