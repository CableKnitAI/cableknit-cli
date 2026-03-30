package cmd

import (
	"encoding/json"
	"fmt"
	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/bundle"
	"github.com/jessewaites/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var pushCmd = &cobra.Command{
	Use:   "push [path]",
	Short: "Push a plugin bundle",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		path := ""
		if len(args) > 0 {
			path = args[0]
		}

		jsonOut := viper.GetBool("json") || cmd.Flag("json").Changed

		if jsonOut || !ui.IsTTY() {
			return pushPlain(path, jsonOut)
		}

		p := tea.NewProgram(newPushModel(path))
		m, err := p.Run()
		if err != nil {
			return err
		}
		pm := m.(pushModel)
		if pm.err != nil {
			return pm.err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushCmd)
}

func pushPlain(path string, jsonOut bool) error {
	r, _, name, err := bundle.Open(path)
	if err != nil {
		return err
	}

	client := api.NewAPIClient()
	var resp api.PushResponse
	if err := client.Multipart("/api/v1/cli/plugins/push", "bundle", name, r, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(resp.Errors) > 0 {
		for _, e := range resp.Errors {
			fmt.Printf("%s %s: %s\n", ui.SymbolCross, e.File, e.Message)
		}
		return nil
	}

	action := "Created"
	if resp.Updated {
		action = "Updated"
	}
	fmt.Printf("%s %s %s v%s\n", ui.SuccessStyle.Render(ui.SymbolCheck), action, resp.PluginName, resp.Version)
	return nil
}

// TUI

type pushPhase int

const (
	pushPhaseZipping pushPhase = iota
	pushPhaseUploading
	pushPhaseProcessing
	pushPhaseDone
)

type pushModel struct {
	phase    pushPhase
	spinner  spinner.Model
	progress progress.Model
	path     string
	resp     api.PushResponse
	err      error
	uploaded float64
	embedded bool
}

func (m pushModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

type pushResultMsg struct {
	resp api.PushResponse
	err  error
}

type pushProgressMsg struct {
	percent float64
}

type pushUploadStartMsg struct{}

func newPushModel(path string) pushModel {
	return pushModel{
		spinner:  ui.NewSpinner(spinner.Points),
		progress: progress.New(progress.WithColors(lipgloss.Color("#5B9BD5"), lipgloss.Color("#5B9BD5"))),
		path:     path,
	}
}

func (m pushModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.doPush())
}

func (m pushModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case pushUploadStartMsg:
		m.phase = pushPhaseUploading
		return m, nil

	case pushProgressMsg:
		m.uploaded = msg.percent
		return m, m.progress.SetPercent(msg.percent)

	case pushResultMsg:
		m.resp = msg.resp
		m.err = msg.err
		m.phase = pushPhaseDone
		return m, m.done()

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m pushModel) View() tea.View {
	var s string

	switch m.phase {
	case pushPhaseZipping:
		s = "\n  " + m.spinner.View() + " Zipping bundle...\n\n"

	case pushPhaseUploading:
		s = "\n  " + m.spinner.View() + " Uploading...\n"
		s += "  " + m.progress.ViewAs(m.uploaded) + "\n\n"

	case pushPhaseProcessing:
		s = "\n  " + m.spinner.View() + " Processing...\n\n"

	case pushPhaseDone:
		if m.err != nil {
			s = "\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n"
		} else if len(m.resp.Errors) > 0 {
			s = "\n" + ui.RenderValidationErrors(m.resp.Errors)
		} else {
			action := "Created"
			if m.resp.Updated {
				action = "Updated"
			}
			bold := lipgloss.NewStyle().Bold(true)
			s = fmt.Sprintf("\n%s %s %s v%s\n\n",
				ui.SuccessStyle.Render(ui.SymbolCheck),
				action,
				bold.Render(m.resp.PluginName),
				m.resp.Version,
			)
		}
	}

	return tea.NewView(s)
}

func (m pushModel) doPush() tea.Cmd {
	return func() tea.Msg {
		r, _, name, err := bundle.Open(m.path)
		if err != nil {
			return pushResultMsg{err: err}
		}

		client := api.NewAPIClient()
		var resp api.PushResponse
		if err := client.Multipart("/api/v1/cli/plugins/push", "bundle", name, r, &resp); err != nil {
			return pushResultMsg{err: err}
		}

		return pushResultMsg{resp: resp}
	}
}
