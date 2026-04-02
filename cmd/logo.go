package cmd

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/cableknitai/cableknit-cli/internal/ui"
)

const (
	shineInterval = 8 * time.Second
	shineDelay    = 1500 * time.Millisecond
	shineTickRate = 50 * time.Millisecond
	shineStep     = 2
	shineBand     = 5
)

var logoArt = []string{
	` ██████╗ █████╗ ██████╗ ██╗     ███████╗██╗  ██╗███╗   ██╗██╗████████╗   █████╗ ██╗`,
	`██╔════╝██╔══██╗██╔══██╗██║     ██╔════╝██║ ██╔╝████╗  ██║██║╚══██╔══╝  ██╔══██╗██║`,
	`██║     ███████║██████╔╝██║     █████╗  █████╔╝ ██╔██╗ ██║██║   ██║     ███████║██║`,
	`██║     ██╔══██║██╔══██╗██║     ██╔══╝  ██╔═██╗ ██║╚██╗██║██║   ██║     ██╔══██║██║`,
	`╚██████╗██║  ██║██████╔╝███████╗███████╗██║  ██╗██║ ╚████║██║   ██║     ██║  ██║██║`,
	` ╚═════╝╚═╝  ╚═╝╚═════╝ ╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝   ╚═╝     ╚═╝  ╚═╝╚═╝`,
}

type logoShineStartMsg struct{}
type logoShineStepMsg struct{}

type logo struct {
	lines    [][]rune
	shinePos int
	maxDiag  int
}

func newLogo() *logo {
	lines := make([][]rune, len(logoArt))
	maxWidth := 0
	for i, line := range logoArt {
		lines[i] = []rune(line)
		if len(lines[i]) > maxWidth {
			maxWidth = len(lines[i])
		}
	}
	return &logo{
		lines:    lines,
		shinePos: -1,
		maxDiag:  maxWidth + len(logoArt),
	}
}

func (l *logo) Init() tea.Cmd {
	return tea.Tick(shineDelay, func(time.Time) tea.Msg {
		return logoShineStartMsg{}
	})
}

func (l *logo) Update(msg tea.Msg) tea.Cmd {
	switch msg.(type) {
	case logoShineStartMsg:
		l.shinePos = 0
		return l.shineTick()
	case logoShineStepMsg:
		l.shinePos += shineStep
		if l.shinePos > l.maxDiag+shineBand {
			l.shinePos = -1
			return tea.Tick(shineInterval, func(time.Time) tea.Msg {
				return logoShineStartMsg{}
			})
		}
		return l.shineTick()
	}
	return nil
}

func (l *logo) View() string {
	baseStyle := lipgloss.NewStyle().Foreground(ui.Blue)
	shineStyle := lipgloss.NewStyle().Foreground(ui.White).Bold(true)

	var sb strings.Builder
	for i, line := range l.lines {
		if i > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(renderLogoLine(line, i, l.shinePos, baseStyle, shineStyle))
	}
	return sb.String()
}

func (l *logo) Width() int {
	maxW := 0
	for _, line := range l.lines {
		if len(line) > maxW {
			maxW = len(line)
		}
	}
	return maxW
}

func (l *logo) Height() int {
	return len(l.lines)
}

// RenderLineWithRain renders a logo line, showing rainCols through spaces in the logo.
// rainCols is a slice of per-column rendered rain strings (one per terminal column),
// starting at the logo's left offset in the terminal.
func (l *logo) RenderLineWithRain(lineIdx int, rainCols []string) string {
	if lineIdx < 0 || lineIdx >= len(l.lines) {
		return ""
	}

	line := l.lines[lineIdx]
	baseStyle := lipgloss.NewStyle().Foreground(ui.Blue)
	shineStyle := lipgloss.NewStyle().Foreground(ui.White).Bold(true)

	shineStart := -1
	shineEnd := -1
	if l.shinePos >= 0 {
		shineStart = l.shinePos - lineIdx
		shineEnd = shineStart + shineBand
	}

	var sb strings.Builder
	for i, r := range line {
		if r == ' ' {
			// Show rain through the gap
			if i < len(rainCols) {
				sb.WriteString(rainCols[i])
			} else {
				sb.WriteByte(' ')
			}
		} else {
			// Logo character — apply base or shine style
			if shineStart >= 0 && i >= shineStart && i < shineEnd {
				sb.WriteString(shineStyle.Render(string(r)))
			} else {
				sb.WriteString(baseStyle.Render(string(r)))
			}
		}
	}
	return sb.String()
}

func renderLogoLine(line []rune, row, shinePos int, baseStyle, shineStyle lipgloss.Style) string {
	if shinePos < 0 {
		return baseStyle.Render(string(line))
	}

	shineStart := shinePos - row
	shineEnd := shineStart + shineBand
	lineLen := len(line)

	if shineStart >= lineLen || shineEnd <= 0 {
		return baseStyle.Render(string(line))
	}

	shineStart = max(shineStart, 0)
	shineEnd = min(shineEnd, lineLen)

	var sb strings.Builder
	if shineStart > 0 {
		sb.WriteString(baseStyle.Render(string(line[:shineStart])))
	}
	sb.WriteString(shineStyle.Render(string(line[shineStart:shineEnd])))
	if shineEnd < lineLen {
		sb.WriteString(baseStyle.Render(string(line[shineEnd:])))
	}
	return sb.String()
}

func (l *logo) shineTick() tea.Cmd {
	return tea.Tick(shineTickRate, func(time.Time) tea.Msg {
		return logoShineStepMsg{}
	})
}
