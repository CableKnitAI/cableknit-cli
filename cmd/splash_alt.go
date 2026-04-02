package cmd

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cableknitai/cableknit-cli/internal/ui"
)

const altSplashSubtitle = "AI Automation Development Kit"

type altSplashModel struct {
	width       int
	height      int
	ready       bool
	quitting    bool
	embedded    bool
	rain        *matrixRain
	logo        *logo
	revealIndex int
	titleDone   bool
}

type altSplashDismissMsg struct{}
type altSubtitleTickMsg struct{}

func newAltSplashModel() altSplashModel {
	return altSplashModel{
		rain: newMatrixRain(),
		logo: newLogo(),
	}
}

func (m altSplashModel) done() tea.Cmd {
	if m.embedded {
		return func() tea.Msg { return screenDoneMsg{} }
	}
	return tea.Quit
}

func (m altSplashModel) Init() tea.Cmd {
	return tea.Batch(
		m.rain.Init(),
		m.logo.Init(),
		tea.Tick(800*time.Millisecond, func(time.Time) tea.Msg {
			return altSubtitleTickMsg{}
		}),
	)
}

func (m altSplashModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		m.quitting = true
		return m, m.done()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		cmd := m.rain.Update(msg)
		return m, cmd

	case rainTickMsg:
		cmd := m.rain.Update(msg)
		return m, cmd

	case logoShineStartMsg, logoShineStepMsg:
		cmd := m.logo.Update(msg)
		return m, cmd

	case altSubtitleTickMsg:
		if m.revealIndex < len(altSplashSubtitle) {
			m.revealIndex++
			if m.revealIndex >= len(altSplashSubtitle) {
				m.titleDone = true
				return m, tea.Tick(2500*time.Millisecond, func(time.Time) tea.Msg {
					return altSplashDismissMsg{}
				})
			}
			return m, tea.Tick(60*time.Millisecond, func(time.Time) tea.Msg {
				return altSubtitleTickMsg{}
			})
		}
		return m, nil

	case altSplashDismissMsg:
		m.quitting = true
		return m, m.done()
	}

	return m, nil
}

func (m altSplashModel) View() tea.View {
	if !m.ready {
		return tea.NewView("")
	}

	// Logo dimensions
	logoLines := strings.Split(m.logo.View(), "\n")
	logoH := len(logoLines)
	logoW := m.logo.Width()

	// Subtitle with typewriter
	revealed := altSplashSubtitle[:m.revealIndex]
	cursor := ""
	if !m.titleDone {
		cursor = "\u2588"
	}
	subtitleLine := lipgloss.NewStyle().Foreground(ui.Blue).Render(revealed + cursor)

	var versionLine string
	if m.titleDone {
		versionLine = lipgloss.NewStyle().Foreground(ui.Blue).Render("v 1.0")
	}

	contentH := logoH + 2
	if m.titleDone {
		contentH++
	}

	topPad := (m.height - contentH) / 2
	if topPad < 0 {
		topPad = 0
	}

	logoLeft := (m.width - logoW) / 2
	if logoLeft < 0 {
		logoLeft = 0
	}

	// Build brightness map once for the whole frame
	bright := m.rain.buildBrightnessMap()
	rainRows := m.rain.RenderRows()

	// Build output row by row
	var sb strings.Builder
	for row := 0; row < m.height; row++ {
		if row > 0 {
			sb.WriteByte('\n')
		}

		contentRow := row - topPad

		if contentRow >= 0 && contentRow < logoH && contentRow < len(logoLines) {
			// Logo row: rain on sides + rain-through-gaps in logo
			rainCols := m.rain.RenderCols(row, bright)
			// Extract rain cols that fall within the logo's horizontal span
			var logoRainCols []string
			if logoLeft < len(rainCols) {
				end := logoLeft + logoW
				if end > len(rainCols) {
					end = len(rainCols)
				}
				logoRainCols = rainCols[logoLeft:end]
			}
			logoRendered := m.logo.RenderLineWithRain(contentRow, logoRainCols)
			sb.WriteString(spliceRow(rainRows, row, logoRendered, m.width, logoLeft, logoW))
		} else if contentRow == logoH+1 {
			subW := lipgloss.Width(subtitleLine)
			subLeft := (m.width - subW) / 2
			if subLeft < 0 {
				subLeft = 0
			}
			sb.WriteString(spliceRow(rainRows, row, subtitleLine, m.width, subLeft, subW))
		} else if contentRow == logoH+2 && m.titleDone {
			verW := lipgloss.Width(versionLine)
			verLeft := (m.width - verW) / 2
			if verLeft < 0 {
				verLeft = 0
			}
			sb.WriteString(spliceRow(rainRows, row, versionLine, m.width, verLeft, verW))
		} else if row < len(rainRows) {
			sb.WriteString(rainRows[row])
		}
	}

	v := tea.NewView(sb.String())
	if !m.embedded {
		v.AltScreen = true
	}
	return v
}

// spliceRow takes a rain row and splices fg content at fgLeft, preserving rain on both sides.
func spliceRow(rainRows []string, row int, fg string, width, fgLeft, fgW int) string {
	var left, right string

	if row < len(rainRows) {
		rainRow := rainRows[row]
		// Rain uses single-width chars (0/1), so we can get per-column rendered strings
		left = renderRainCols(rainRow, 0, fgLeft)
		right = renderRainCols(rainRow, fgLeft+fgW, width)
	} else {
		left = strings.Repeat(" ", fgLeft)
		right = strings.Repeat(" ", max(width-fgLeft-fgW, 0))
	}

	return left + fg + right
}

// renderRainCols extracts columns [from, to) from a rain-rendered line.
// Since rain chars are single-width with per-char ANSI wrapping, we parse
// by walking ANSI sequences.
func renderRainCols(line string, from, to int) string {
	if from >= to {
		return ""
	}

	var sb strings.Builder
	col := 0
	i := 0
	runes := []rune(line)

	for i < len(runes) && col < to {
		if runes[i] == '\x1b' {
			// Start of ANSI escape — collect the whole sequence
			seqStart := i
			i++
			for i < len(runes) {
				i++
				if i > 0 && ((runes[i-1] >= 'a' && runes[i-1] <= 'z') || (runes[i-1] >= 'A' && runes[i-1] <= 'Z')) {
					break
				}
			}
			// This escape belongs to the next visible char
			if col >= from && col < to {
				sb.WriteString(string(runes[seqStart:i]))
			}
		} else {
			// Visible character
			if col >= from && col < to {
				sb.WriteRune(runes[i])
			}
			col++
			i++
		}
	}

	// Pad if we didn't have enough columns
	for col < to && col >= from {
		sb.WriteByte(' ')
		col++
	}

	return sb.String()
}
