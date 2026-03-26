package cmd

import (
	_ "embed"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/ui"
)

//go:embed assets/yarn-white.png
var logoPNG []byte

const splashTitle = "C A B L E K N I T  A I"

type screenDoneMsg struct{}

type splashModel struct {
	width       int
	height      int
	ready       bool
	logo        string
	quitting    bool
	embedded    bool
	revealIndex int
}

type splashTickMsg struct{}
type splashTypeTickMsg struct{}

func newSplashModel() splashModel {
	return splashModel{}
}

func (m splashModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m splashModel) Init() tea.Cmd {
	// Start typewriter tick; dismiss timer starts after typing completes
	return tea.Tick(90*time.Millisecond, func(t time.Time) tea.Msg {
		return splashTypeTickMsg{}
	})
}

func (m splashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		m.quitting = true
		return m, m.done()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Render logo at ~30 cols wide (60 half-blocks high ≈ 30 terminal rows)
		logoWidth := 30
		if m.width < 50 {
			logoWidth = 20
		}
		m.logo = ui.RenderImage(logoPNG, logoWidth)
		m.ready = true
		return m, nil

	case splashTypeTickMsg:
		if m.revealIndex < len(splashTitle) {
			m.revealIndex++
			if m.revealIndex >= len(splashTitle) {
				// Title done — hold 2s then dismiss
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return splashTickMsg{}
				})
			}
			return m, tea.Tick(90*time.Millisecond, func(t time.Time) tea.Msg {
				return splashTypeTickMsg{}
			})
		}
		return m, nil

	case splashTickMsg:
		m.quitting = true
		return m, m.done()
	}

	return m, nil
}

func (m splashModel) View() tea.View {
	if !m.ready {
		return tea.NewView("")
	}

	titleDone := m.revealIndex >= len(splashTitle)

	// Title text — only reveal up to revealIndex
	revealed := splashTitle[:m.revealIndex]
	cursor := ""
	if !titleDone {
		cursor = "█"
	}
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Blue).
		Render(revealed + cursor)

	// Subtitle + version appear only after title completes
	var subtitle, version string
	if titleDone {
		subtitle = lipgloss.NewStyle().
			Foreground(ui.Dim).
			Render("AI Automation Development Kit")
		version = lipgloss.NewStyle().
			Foreground(ui.White).
			Render("v 1.0")
	}

	// Combine logo + title
	content := m.logo + "\n\n" + title
	if titleDone {
		content += "\n" + subtitle + "\n" + version
	}

	// Calculate content dimensions
	contentLines := strings.Split(content, "\n")
	contentHeight := len(contentLines)
	contentWidth := 0
	for _, line := range contentLines {
		w := lipgloss.Width(line)
		if w > contentWidth {
			contentWidth = w
		}
	}

	// Center vertically
	topPad := (m.height - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	// Center horizontally — pad each line
	var centered []string
	for _, line := range contentLines {
		lineWidth := lipgloss.Width(line)
		leftPad := (m.width - lineWidth) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		centered = append(centered, strings.Repeat(" ", leftPad)+line)
	}

	// Build full screen
	var sb strings.Builder
	sb.WriteString(strings.Repeat("\n", topPad))
	sb.WriteString(strings.Join(centered, "\n"))

	// Fill remaining space
	bottomPad := m.height - topPad - contentHeight
	if bottomPad > 0 {
		sb.WriteString(strings.Repeat("\n", bottomPad))
	}

	v := tea.NewView(sb.String())
	if !m.embedded {
		v.AltScreen = true
	}
	return v
}

func runSplash() {
	if !ui.IsTTY() {
		return
	}
	p := tea.NewProgram(newSplashModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
	}
}
