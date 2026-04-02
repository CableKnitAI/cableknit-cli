package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"
	"github.com/cableknitai/cableknit-cli/internal/bundle"
	"github.com/cableknitai/cableknit-cli/internal/ui"
	"github.com/spf13/cobra"
)

var packageCmd = &cobra.Command{
	Use:   "package [path]",
	Short: "Package a plugin directory into a .sweater file",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := ""
		if len(args) > 0 {
			path = args[0]
		}

		if !ui.IsTTY() {
			return packagePlain(path)
		}

		p := tea.NewProgram(newPackageModel(path))
		m, err := p.Run()
		if err != nil {
			return err
		}
		pm := m.(packageModel)
		if pm.err != nil {
			return pm.err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(packageCmd)
}

func packagePlain(path string) error {
	resolved, isDir, err := bundle.Find(path)
	if err != nil {
		return err
	}
	if !isDir {
		return fmt.Errorf("path is already a file, not a directory: %s", resolved)
	}

	// Validate plugin.json is parseable
	manifest := filepath.Join(resolved, "plugin.json")
	data, err := os.ReadFile(manifest)
	if err != nil {
		return fmt.Errorf("cannot read plugin.json: %w", err)
	}
	var js map[string]interface{}
	if err := json.Unmarshal(data, &js); err != nil {
		return fmt.Errorf("plugin.json is not valid JSON: %w", err)
	}

	buf, err := bundle.Zip(resolved)
	if err != nil {
		return err
	}

	outName := filepath.Base(resolved) + ".sweater"
	if err := os.WriteFile(outName, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outName, err)
	}

	fmt.Printf("%s Packaged %s (%d bytes)\n", ui.SymbolCheck, outName, buf.Len())
	return nil
}

// TUI model

type packageState int

const (
	packageStateLoading packageState = iota
	packageStateDone
)

type packageModel struct {
	state    packageState
	spinner  spinner.Model
	path     string
	outName  string
	size     int
	err      error
	embedded bool
}

func (m packageModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

type packageResultMsg struct {
	outName string
	size    int
	err     error
}

func newPackageModel(path string) packageModel {
	return packageModel{
		spinner: ui.NewSpinner(spinner.MiniDot),
		path:    path,
	}
}

func (m packageModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.doPackage())
}

func (m packageModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}

	case packageResultMsg:
		m.outName = msg.outName
		m.size = msg.size
		m.err = msg.err
		m.state = packageStateDone
		return m, m.done()
	}

	if m.state == packageStateLoading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m packageModel) View() tea.View {
	var s string

	switch m.state {
	case packageStateLoading:
		s = "\n  " + m.spinner.View() + " Packaging bundle...\n\n"

	case packageStateDone:
		if m.err != nil {
			s = "\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n"
		} else {
			bold := lipgloss.NewStyle().Bold(true)
			s = fmt.Sprintf("\n%s %s %s (%d bytes)\n\n",
				ui.SuccessStyle.Render(ui.SymbolCheck),
				"Packaged",
				bold.Render(m.outName),
				m.size,
			)
		}
	}

	return tea.NewView(s)
}

func (m packageModel) doPackage() tea.Cmd {
	return func() tea.Msg {
		resolved, isDir, err := bundle.Find(m.path)
		if err != nil {
			return packageResultMsg{err: err}
		}
		if !isDir {
			return packageResultMsg{err: fmt.Errorf("path is already a file, not a directory: %s", resolved)}
		}

		// Validate plugin.json is parseable
		manifest := filepath.Join(resolved, "plugin.json")
		data, err := os.ReadFile(manifest)
		if err != nil {
			return packageResultMsg{err: fmt.Errorf("cannot read plugin.json: %w", err)}
		}
		var js map[string]interface{}
		if err := json.Unmarshal(data, &js); err != nil {
			return packageResultMsg{err: fmt.Errorf("plugin.json is not valid JSON: %w", err)}
		}

		buf, err := bundle.Zip(resolved)
		if err != nil {
			return packageResultMsg{err: err}
		}

		outName := filepath.Base(resolved) + ".sweater"
		if err := os.WriteFile(outName, buf.Bytes(), 0644); err != nil {
			return packageResultMsg{err: fmt.Errorf("failed to write %s: %w", outName, err)}
		}

		return packageResultMsg{outName: outName, size: buf.Len()}
	}
}
