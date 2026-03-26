package cmd

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/config"
	"github.com/jessewaites/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
)

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to CableKnit",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !ui.IsTTY() {
			return fmt.Errorf("login requires an interactive terminal")
		}

		p := tea.NewProgram(newLoginModel())
		m, err := p.Run()
		if err != nil {
			return err
		}
		lm := m.(loginModel)
		if lm.err != nil {
			return lm.err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(loginCmd)
}

type loginState int

const (
	loginStateForm loginState = iota
	loginStateLoading
	loginStateDone
	loginStateError
)

type loginModel struct {
	state    loginState
	form     *huh.Form
	spinner  spinner.Model
	email    string
	password string
	user     api.User
	err      error
	embedded bool
}

func (m loginModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

type loginResultMsg struct {
	user api.User
	err  error
}

func newLoginModel() loginModel {
	m := loginModel{}

	m.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Email").
				Placeholder("you@example.com").
				Value(&m.email),
			huh.NewInput().
				Title("Password").
				EchoMode(huh.EchoModePassword).
				Value(&m.password),
		),
	).WithTheme(huh.ThemeFunc(huh.ThemeCharm))

	m.spinner = ui.NewSpinner(spinner.Dot)

	return m
}

func (m loginModel) Init() tea.Cmd {
	return m.form.Init()
}

func (m loginModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case loginResultMsg:
		if msg.err != nil {
			m.state = loginStateError
			m.err = msg.err
			return m, m.done()
		}
		m.state = loginStateDone
		m.user = msg.user
		return m, m.done()
	}

	switch m.state {
	case loginStateForm:
		form, cmd := m.form.Update(msg)
		m.form = form.(*huh.Form)

		if m.form.State == huh.StateCompleted {
			m.state = loginStateLoading
			return m, tea.Batch(m.spinner.Tick, m.doLogin())
		}
		if m.form.State == huh.StateAborted {
			return m, m.done()
		}
		return m, cmd

	case loginStateLoading:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m loginModel) View() tea.View {
	var s string

	switch m.state {
	case loginStateForm:
		s = "\n" + m.form.View() + "\n"

	case loginStateLoading:
		s = "\n  " + m.spinner.View() + " Logging in...\n\n"

	case loginStateDone:
		status := ui.SuccessStyle.Render(ui.SymbolCheck + " Active")
		if m.user.PublisherStatus == "pending" {
			status = ui.WarningStyle.Render(ui.SymbolWarning + " Pending")
		}

		content := fmt.Sprintf(
			"%s  %s\n%s  %s\n%s  %s",
			lipgloss.NewStyle().Bold(true).Render("Name:"), m.user.Name,
			lipgloss.NewStyle().Bold(true).Render("Email:"), m.user.Email,
			lipgloss.NewStyle().Bold(true).Render("Status:"), status,
		)

		s = "\n" + ui.SuccessBox.Render(content) + "\n\n"

	case loginStateError:
		s = "\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n"
	}

	return tea.NewView(s)
}

func (m loginModel) doLogin() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		var resp api.LoginResponse
		err := client.JSON("POST", "/api/v1/cli/sessions", api.LoginRequest{
			Email:    m.email,
			Password: m.password,
		}, &resp)
		if err != nil {
			return loginResultMsg{err: err}
		}

		if err := config.SetToken(resp.Token); err != nil {
			return loginResultMsg{err: fmt.Errorf("failed to save token: %w", err)}
		}

		return loginResultMsg{user: resp.User}
	}
}
