package cmd

import (
	"encoding/json"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/bundle"
	"github.com/jessewaites/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var validateCmd = &cobra.Command{
	Use:   "validate [path]",
	Short: "Validate a plugin bundle",
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
			return validatePlain(path, jsonOut)
		}

		p := tea.NewProgram(newValidateModel(path))
		m, err := p.Run()
		if err != nil {
			return err
		}
		vm := m.(validateModel)
		if vm.err != nil {
			return vm.err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func validatePlain(path string, jsonOut bool) error {
	r, _, name, err := bundle.Open(path)
	if err != nil {
		return err
	}

	client := api.NewAPIClient()
	var resp api.ValidateResponse
	if err := client.Multipart("/api/v1/cli/plugins/validate", "bundle", name, r, &resp); err != nil {
		return err
	}

	if jsonOut {
		b, _ := json.MarshalIndent(resp, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if resp.Valid {
		printValidateSummary(resp)
	} else {
		for _, e := range resp.Errors {
			fmt.Printf("%s %s: %s\n", ui.SymbolCross, e.File, e.Message)
		}
		for _, w := range resp.Warnings {
			fmt.Printf("%s %s: %s\n", ui.SymbolWarning, w.File, w.Message)
		}
	}
	return nil
}

// TUI model

type validateState int

const (
	validateStateLoading validateState = iota
	validateStateDone
)

type validateModel struct {
	state    validateState
	spinner  spinner.Model
	path     string
	resp     api.ValidateResponse
	err      error
	embedded bool
}

func (m validateModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

type validateResultMsg struct {
	resp api.ValidateResponse
	err  error
}

func newValidateModel(path string) validateModel {
	return validateModel{
		spinner: ui.NewSpinner(spinner.MiniDot),
		path:    path,
	}
}

func (m validateModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.doValidate())
}

func (m validateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case validateResultMsg:
		m.resp = msg.resp
		m.err = msg.err
		m.state = validateStateDone
		return m, m.done()
	}

	if m.state == validateStateLoading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m validateModel) View() tea.View {
	var s string

	switch m.state {
	case validateStateLoading:
		s = "\n  " + m.spinner.View() + " Validating bundle...\n\n"

	case validateStateDone:
		if m.err != nil {
			s = "\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n"
		} else if m.resp.Valid {
			s = "\n" + renderValidateSummary(m.resp) + "\n"
		} else {
			s = "\n" + ui.RenderValidationErrors(m.resp.Errors)
			s += ui.RenderWarnings(m.resp.Warnings)
		}
	}

	return tea.NewView(s)
}

func (m validateModel) doValidate() tea.Cmd {
	return func() tea.Msg {
		r, _, name, err := bundle.Open(m.path)
		if err != nil {
			return validateResultMsg{err: err}
		}

		client := api.NewAPIClient()
		var resp api.ValidateResponse
		if err := client.Multipart("/api/v1/cli/plugins/validate", "bundle", name, r, &resp); err != nil {
			return validateResultMsg{err: err}
		}

		return validateResultMsg{resp: resp}
	}
}

func printValidateSummary(r api.ValidateResponse) {
	bold := lipgloss.NewStyle().Bold(true)
	fmt.Printf("%s %s\n", ui.SuccessStyle.Render(ui.SymbolCheck), bold.Render("Valid"))
	fmt.Printf("  Plugin:      %s\n", r.PluginName)
	fmt.Printf("  Slug:        %s\n", r.Slug)
	fmt.Printf("  Version:     %s\n", r.Version)
	fmt.Printf("  Skills:      %d\n", r.Skills)
	fmt.Printf("  Automations: %d\n", r.Automations)
	fmt.Printf("  Blueprints:  %d\n", r.Blueprints)
	fmt.Printf("  Tools:       %d\n", r.Tools)
	fmt.Printf("  Docs:        %d\n", r.Docs)
}

func renderValidateSummary(r api.ValidateResponse) string {
	bold := lipgloss.NewStyle().Bold(true)
	content := fmt.Sprintf(
		"%s %s\n  Plugin:      %s\n  Slug:        %s\n  Version:     %s\n  Skills:      %d\n  Automations: %d\n  Blueprints:  %d\n  Tools:       %d\n  Docs:        %d",
		ui.SuccessStyle.Render(ui.SymbolCheck), bold.Render("Valid"),
		r.PluginName, r.Slug, r.Version, r.Skills, r.Automations, r.Blueprints, r.Tools, r.Docs,
	)
	return content
}
