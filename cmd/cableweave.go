package cmd

import (
	"math/rand/v2"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/ui"
)

const (
	rainTickRate = 70 * time.Millisecond
	rainChars = "01"
)

type rainTickMsg struct{}

type raindrop struct {
	col    int
	row    int     // current head position
	length int     // trail length
	speed  float64 // rows per tick (allows fractional speeds)
	acc    float64 // accumulated fractional movement
}

// clearZone defines a rectangular area where rain is suppressed.
type clearZone struct {
	top, bottom, left, right int
}

type matrixRain struct {
	width, height int
	drops         []raindrop
	grid          [][]rune // current display chars
	rng           *rand.Rand
	chars         []rune
	clear         *clearZone
}

func newMatrixRain() *matrixRain {
	return &matrixRain{
		rng:   rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		chars: []rune(rainChars),
	}
}

func (m *matrixRain) Init() tea.Cmd {
	return m.scheduleTick()
}

func (m *matrixRain) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.resize(msg.Width, msg.Height)
	case rainTickMsg:
		m.step()
		return m.scheduleTick()
	}
	return nil
}

// SetClearZone defines a rectangular area where rain won't render.
func (m *matrixRain) SetClearZone(top, bottom, left, right int) {
	m.clear = &clearZone{top: top, bottom: bottom, left: left, right: right}
}

func (m *matrixRain) inClearZone(row, col int) bool {
	if m.clear == nil {
		return false
	}
	return row >= m.clear.top && row <= m.clear.bottom &&
		col >= m.clear.left && col <= m.clear.right
}

// RenderRows returns each row as a separate string for splicing.
func (m *matrixRain) RenderRows() []string {
	if m.width <= 0 || m.height <= 0 {
		return nil
	}

	bright := m.buildBrightnessMap()
	headStyle := lipgloss.NewStyle().Foreground(ui.White).Bold(true)
	brightStyle := lipgloss.NewStyle().Foreground(ui.Green)
	dimStyle := lipgloss.NewStyle().Foreground(ui.Dim)

	rows := make([]string, m.height)
	for r := 0; r < m.height; r++ {
		var sb strings.Builder
		for c := 0; c < m.width; c++ {
			b := bright[r][c]
			if b == 0 {
				sb.WriteByte(' ')
				continue
			}
			ch := m.grid[r][c]
			if ch == 0 {
				sb.WriteByte(' ')
				continue
			}
			switch b {
			case 3:
				sb.WriteString(headStyle.Render(string(ch)))
			case 2:
				sb.WriteString(brightStyle.Render(string(ch)))
			default:
				sb.WriteString(dimStyle.Render(string(ch)))
			}
		}
		rows[r] = sb.String()
	}
	return rows
}

// RenderCols returns per-column rendered strings for a given row.
// Each element is either a styled char or a space.
func (m *matrixRain) RenderCols(row int, bright [][]int) []string {
	if row < 0 || row >= m.height {
		return nil
	}

	headStyle := lipgloss.NewStyle().Foreground(ui.White).Bold(true)
	brightStyle := lipgloss.NewStyle().Foreground(ui.Green)
	dimStyle := lipgloss.NewStyle().Foreground(ui.Dim)

	cols := make([]string, m.width)
	for c := 0; c < m.width; c++ {
		b := bright[row][c]
		if b == 0 || m.grid[row][c] == 0 {
			cols[c] = " "
			continue
		}
		ch := string(m.grid[row][c])
		switch b {
		case 3:
			cols[c] = headStyle.Render(ch)
		case 2:
			cols[c] = brightStyle.Render(ch)
		default:
			cols[c] = dimStyle.Render(ch)
		}
	}
	return cols
}

func (m *matrixRain) Render() string {
	rows := m.RenderRows()
	return strings.Join(rows, "\n")
}

func (m *matrixRain) buildBrightnessMap() [][]int {
	bright := make([][]int, m.height)
	for r := range bright {
		bright[r] = make([]int, m.width)
	}
	for i := range m.drops {
		d := &m.drops[i]
		for j := 0; j < d.length; j++ {
			r := d.row - j
			if r < 0 || r >= m.height || d.col < 0 || d.col >= m.width {
				continue
			}
			if j == 0 {
				bright[r][d.col] = 3
			} else if j < d.length/3 {
				if bright[r][d.col] < 2 {
					bright[r][d.col] = 2
				}
			} else {
				if bright[r][d.col] < 1 {
					bright[r][d.col] = 1
				}
			}
		}
	}
	return bright
}

// Private

func (m *matrixRain) resize(width, height int) {
	m.width = width
	m.height = height

	m.grid = make([][]rune, height)
	for r := range m.grid {
		m.grid[r] = make([]rune, width)
		for c := range m.grid[r] {
			m.grid[r][c] = m.randChar()
		}
	}

	// Spawn initial drops — roughly 1 per 2 columns, staggered
	m.drops = nil
	for col := 0; col < width; col++ {
		if m.rng.IntN(2) == 0 {
			m.drops = append(m.drops, m.spawnDrop(col, true))
		}
	}
}

func (m *matrixRain) step() {
	// Advance drops
	alive := m.drops[:0]
	for i := range m.drops {
		d := &m.drops[i]
		d.acc += d.speed
		for d.acc >= 1.0 {
			d.row++
			d.acc -= 1.0
		}
		// Drop is off screen (tail has fully exited)
		if d.row-d.length >= m.height {
			continue
		}
		alive = append(alive, *d)
	}
	m.drops = alive

	// Randomly spawn new drops
	for col := 0; col < m.width; col++ {
		if m.rng.IntN(20) == 0 {
			m.drops = append(m.drops, m.spawnDrop(col, false))
		}
	}

	// Randomly mutate some grid chars for the flickering effect
	mutations := max(m.width*m.height/80, 1)
	for range mutations {
		r := m.rng.IntN(m.height)
		c := m.rng.IntN(m.width)
		m.grid[r][c] = m.randChar()
	}
}

func (m *matrixRain) spawnDrop(col int, stagger bool) raindrop {
	startRow := 0
	if stagger {
		startRow = -m.rng.IntN(m.height)
	}
	return raindrop{
		col:    col,
		row:    startRow,
		length: 4 + m.rng.IntN(m.height/2),
		speed:  0.4 + m.rng.Float64()*0.8,
	}
}

func (m *matrixRain) randChar() rune {
	return m.chars[m.rng.IntN(len(m.chars))]
}

func (m *matrixRain) scheduleTick() tea.Cmd {
	return tea.Tick(rainTickRate, func(time.Time) tea.Msg {
		return rainTickMsg{}
	})
}
