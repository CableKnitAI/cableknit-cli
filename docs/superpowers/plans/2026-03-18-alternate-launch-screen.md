# Alternate Launch Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an alternate splash screen with a cable-weave braille animation background, ASCII art logo with diagonal shine effect, then transition to a dashboard-style menu.

**Architecture:** Three new components: `CableWeave` (braille animation inspired by knitting patterns), `Logo` (ASCII art "CABLEKNIT" with shine sweep), and updated splash model that composites them. The existing splash remains available — a flag or config toggles which one runs. The menu gets a visual refresh with card-style grouping but same functionality.

**Tech Stack:** bubbletea v2, lipgloss v2, math/rand/v2, unicode braille (U+2800-U+28FF)

---

### File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `cmd/cableweave.go` | Create | Braille-based cable/knit pattern animation component |
| `cmd/logo.go` | Create | ASCII art "CABLEKNIT" with diagonal shine animation |
| `cmd/splash_alt.go` | Create | Alternate splash compositing weave bg + logo + typewriter subtitle |
| `cmd/app.go` | Modify | Wire alt splash, add toggle logic |
| `cmd/root.go` | Modify | Add `--alt` flag or env var to select splash variant |

---

### Task 1: Cable Weave Animation Component

**Files:**
- Create: `cmd/cableweave.go`

The cable weave uses braille characters to render a slowly shifting knit-like pattern. Instead of stars flying at you, interlocking diamond/cable shapes scroll upward, evoking yarn being knitted. Uses sine waves to create the interlocking cable pattern.

- [ ] **Step 1: Create `cmd/cableweave.go` with core structure**

```go
package cmd

import (
	"math"
	"math/rand/v2"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/ui"
)

const (
	weaveTickRate = 80 * time.Millisecond
	weaveSpeed    = 0.4
)

type weaveTickMsg struct{}

type cableWeave struct {
	width, height int
	offset        float64
	rng           *rand.Rand
}

func newCableWeave() *cableWeave {
	return &cableWeave{
		rng: rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
	}
}

func (w *cableWeave) Init() tea.Cmd {
	return w.scheduleTick()
}

func (w *cableWeave) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		w.width = msg.Width
		w.height = msg.Height
	case weaveTickMsg:
		w.offset += weaveSpeed
		return w.scheduleTick()
	}
	return nil
}

// RenderRow returns the weave pattern for a single row.
// Renders interlocking sine-wave cables using braille dots.
func (w *cableWeave) RenderRow(row int) string {
	if w.width <= 0 || w.height <= 0 {
		return ""
	}

	subW := w.width * 2  // braille has 2 dot columns per char
	subH := w.height * 4 // braille has 4 dot rows per char

	// Each cell accumulates braille dot bits
	cells := make([]rune, w.width)

	// We have several "cables" — sine waves offset horizontally
	cableCount := max(w.width/12, 2)
	spacing := float64(subW) / float64(cableCount)

	subRow := row * 4 // top sub-pixel row for this cell row

	for dotRow := 0; dotRow < 4; dotRow++ {
		sy := float64(subRow+dotRow) + w.offset*4

		for cable := 0; cable < cableCount; cable++ {
			centerX := float64(cable)*spacing + spacing/2

			// Two intertwined sine waves per cable (the "twist")
			amp := spacing * 0.25
			freq := 2 * math.Pi / 40.0
			x1 := centerX + amp*math.Sin(sy*freq)
			x2 := centerX + amp*math.Sin(sy*freq+math.Pi)

			for _, sx := range []float64{x1, x2} {
				sxi := int(sx)
				if sxi < 0 || sxi >= subW {
					continue
				}
				col := sxi / 2
				dotCol := sxi % 2
				dotIndex := 3 - dotRow

				if col >= 0 && col < w.width {
					if cells[col] == 0 {
						cells[col] = 0x2800
					}
					if dotCol == 0 {
						cells[col] |= leftDots[dotIndex]
					} else {
						cells[col] |= rightDots[dotIndex]
					}
				}
			}
		}
	}

	dimStyle := lipgloss.NewStyle().Foreground(ui.Dim)

	var sb strings.Builder
	for _, ch := range cells {
		if ch == 0 {
			sb.WriteByte(' ')
		} else {
			sb.WriteString(dimStyle.Render(string(ch)))
		}
	}
	return sb.String()
}

func (w *cableWeave) scheduleTick() tea.Cmd {
	return tea.Tick(weaveTickRate, func(time.Time) tea.Msg {
		return weaveTickMsg{}
	})
}

// Braille dot lookup tables — same encoding as Unicode braille block.
// Left column dots (bit positions): row0=0x40, row1=0x04, row2=0x02, row3=0x01
// Right column dots: row0=0x80, row1=0x20, row2=0x10, row3=0x08
var (
	leftDots  = [4]rune{0x01, 0x02, 0x04, 0x40}
	rightDots = [4]rune{0x08, 0x10, 0x20, 0x80}
)
```

- [ ] **Step 2: Build and verify it compiles**

Run: `cd /Users/jessewaites/Code/cli-cableknit && go build ./...`
Expected: clean compile

- [ ] **Step 3: Commit**

```bash
git add cmd/cableweave.go
git commit -m "add cable weave braille animation component"
```

---

### Task 2: ASCII Logo with Shine Animation

**Files:**
- Create: `cmd/logo.go`

Adapts the once logo pattern: ASCII art stored as rune arrays, diagonal shine band sweeps across. Uses cableknit brand colors (Blue base, White shine).

- [ ] **Step 1: Create `cmd/logo.go`**

```go
package cmd

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/ui"
)

const (
	shineInterval = 8 * time.Second
	shineDelay    = 1500 * time.Millisecond
	shineTickRate = 50 * time.Millisecond
	shineStep     = 2
	shineBand     = 5
)

var logoArt = []string{
	` ██████╗ █████╗ ██████╗ ██╗     ███████╗██╗  ██╗███╗   ██╗██╗████████╗`,
	`██╔════╝██╔══██╗██╔══██╗██║     ██╔════╝██║ ██╔╝████╗  ██║██║╚══██╔══╝`,
	`██║     ███████║██████╔╝██║     █████╗  █████╔╝ ██╔██╗ ██║██║   ██║   `,
	`██║     ██╔══██║██╔══██╗██║     ██╔══╝  ██╔═██╗ ██║╚██╗██║██║   ██║   `,
	`╚██████╗██║  ██║██████╔╝███████╗███████╗██║  ██╗██║ ╚████║██║   ██║   `,
	` ╚═════╝╚═╝  ╚═╝╚═════╝ ╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═══╝╚═╝   ╚═╝`,
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
```

- [ ] **Step 2: Build and verify**

Run: `cd /Users/jessewaites/Code/cli-cableknit && go build ./...`
Expected: clean compile

- [ ] **Step 3: Commit**

```bash
git add cmd/logo.go
git commit -m "add ASCII logo with diagonal shine animation"
```

---

### Task 3: Alternate Splash Screen

**Files:**
- Create: `cmd/splash_alt.go`

Composites the cable weave background with the logo centered on top. Typewriter reveals "AI Automation Development Kit" subtitle after logo loads. Auto-dismisses after 3s or on keypress.

- [ ] **Step 1: Create `cmd/splash_alt.go`**

```go
package cmd

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/ui"
)

const altSplashSubtitle = "AI Automation Development Kit"

type altSplashModel struct {
	width       int
	height      int
	ready       bool
	quitting    bool
	embedded    bool
	weave       *cableWeave
	logo        *logo
	revealIndex int
	titleDone   bool
}

type altSplashDismissMsg struct{}
type altSubtitleTickMsg struct{}

func newAltSplashModel() altSplashModel {
	return altSplashModel{
		weave: newCableWeave(),
		logo:  newLogo(),
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
		m.weave.Init(),
		m.logo.Init(),
		// Start subtitle typewriter after short delay
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
		cmd := m.weave.Update(msg)
		return m, cmd

	case weaveTickMsg:
		cmd := m.weave.Update(msg)
		return m, cmd

	case logoShineStartMsg, logoShineStepMsg:
		cmd := m.logo.Update(msg)
		return m, cmd

	case altSubtitleTickMsg:
		if m.revealIndex < len(altSplashSubtitle) {
			m.revealIndex++
			if m.revealIndex >= len(altSplashSubtitle) {
				m.titleDone = true
				// Hold 2.5s then auto-dismiss
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

	// Build the logo + subtitle content
	logoStr := m.logo.View()
	logoLines := strings.Split(logoStr, "\n")
	logoH := len(logoLines)
	logoW := m.logo.Width()

	// Subtitle with typewriter
	revealed := altSplashSubtitle[:m.revealIndex]
	cursor := ""
	if !m.titleDone {
		cursor = "█"
	}
	subtitleLine := lipgloss.NewStyle().Foreground(ui.Dim).Render(revealed + cursor)

	// Version
	var versionLine string
	if m.titleDone {
		versionLine = lipgloss.NewStyle().Foreground(ui.White).Render("v 1.0")
	}

	// Content block = logo + gap + subtitle + version
	contentH := logoH + 2 // logo + blank + subtitle
	if m.titleDone {
		contentH++ // version line
	}

	// Vertical centering
	topPad := (m.height - contentH) / 2
	if topPad < 0 {
		topPad = 0
	}

	// Determine logo left offset for centering
	logoLeft := (m.width - logoW) / 2
	if logoLeft < 0 {
		logoLeft = 0
	}

	var sb strings.Builder

	for row := 0; row < m.height; row++ {
		contentRow := row - topPad

		if contentRow >= 0 && contentRow < logoH {
			// Logo row — render weave bg then overlay logo
			weaveLine := m.weave.RenderRow(row)
			logoLine := ""
			if contentRow < len(logoLines) {
				logoLine = logoLines[contentRow]
			}
			// Overlay: replace center portion with logo
			line := overlayCenter(weaveLine, logoLine, m.width, logoLeft)
			sb.WriteString(line)
		} else if contentRow == logoH {
			// Blank separator — just weave
			sb.WriteString(m.weave.RenderRow(row))
		} else if contentRow == logoH+1 {
			// Subtitle row
			subW := lipgloss.Width(subtitleLine)
			subLeft := (m.width - subW) / 2
			if subLeft < 0 {
				subLeft = 0
			}
			weaveLine := m.weave.RenderRow(row)
			sb.WriteString(overlayCenter(weaveLine, subtitleLine, m.width, subLeft))
		} else if contentRow == logoH+2 && m.titleDone {
			// Version row
			verW := lipgloss.Width(versionLine)
			verLeft := (m.width - verW) / 2
			if verLeft < 0 {
				verLeft = 0
			}
			weaveLine := m.weave.RenderRow(row)
			sb.WriteString(overlayCenter(weaveLine, versionLine, m.width, verLeft))
		} else {
			// Pure weave background
			sb.WriteString(m.weave.RenderRow(row))
		}

		if row < m.height-1 {
			sb.WriteByte('\n')
		}
	}

	v := tea.NewView(sb.String())
	if !m.embedded {
		v.AltScreen = true
	}
	return v
}

// overlayCenter clears the area behind fg and centers it at leftOffset.
// The bg param is unused — weave rows around the logo are pure weave,
// but the logo area gets a clean background so text stays readable.
func overlayCenter(_, fg string, width, leftOffset int) string {
	if fg == "" {
		return bg
	}
	fgW := lipgloss.Width(fg)
	// Build: bg left portion + fg + bg right portion
	// For simplicity, just pad and place fg (bg is dim, won't clash visually)
	pad := strings.Repeat(" ", leftOffset)
	rightPad := width - leftOffset - fgW
	if rightPad < 0 {
		rightPad = 0
	}
	return pad + fg + strings.Repeat(" ", rightPad)
}
```

- [ ] **Step 2: Build and verify**

Run: `cd /Users/jessewaites/Code/cli-cableknit && go build ./...`
Expected: clean compile

- [ ] **Step 3: Commit**

```bash
git add cmd/splash_alt.go
git commit -m "add alternate splash compositing weave bg + logo"
```

---

### Task 4: Wire Alt Splash into App

**Files:**
- Modify: `cmd/app.go:601-636` (appModel struct, newAppModel, Init)
- Modify: `cmd/app.go:666-672` (screenSplash Update routing)
- Modify: `cmd/app.go:773-777` (screenSplash View routing)
- Modify: `cmd/root.go` (add flag)

- [ ] **Step 1: Add `--alt` flag to root.go**

Add a package-level `var useAltSplash bool` and register it with cobra in the root command's init. Also check env var `CABLEKNIT_ALT_SPLASH`.

- [ ] **Step 2: Update appModel to hold altSplash**

Add `altSplash altSplashModel` field and `useAlt bool` field to `appModel`. In `newAppModel()`, check the flag and initialize accordingly.

- [ ] **Step 3: Route Update/View to altSplash when useAlt is true**

In `Update()` screenSplash case (~line 666):
```go
case screenSplash:
	if m.useAlt {
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.altSplash.Update(msg)
		m.altSplash = model.(altSplashModel)
		return m, cmd
	}
	// existing splash code...
```

In `View()` screenSplash case (~line 773):
```go
case screenSplash:
	if m.useAlt {
		v := m.altSplash.View()
		v.AltScreen = true
		return v
	}
	// existing splash code...
```

- [ ] **Step 4: Build and test manually**

Run: `cd /Users/jessewaites/Code/cli-cableknit && go build -o cableknit . && ./cableknit --alt`
Expected: see cable weave background with CABLEKNIT ASCII logo + shine + typewriter subtitle

Run without flag: `./cableknit`
Expected: original splash (unchanged behavior)

- [ ] **Step 5: Commit**

```bash
git add cmd/root.go cmd/app.go
git commit -m "wire alt splash behind --alt flag"
```

---

### Task 5: Test and Polish

- [ ] **Step 1: Run full build**

Run: `cd /Users/jessewaites/Code/cli-cableknit && go build ./...`
Expected: clean

- [ ] **Step 2: Run the alt splash, eyeball it**

Run: `./cableknit --alt`
Verify: weave animates, logo shines, subtitle types out, auto-dismisses to menu

- [ ] **Step 3: Tune parameters if needed**

Adjustable knobs:
- `weaveTickRate` (animation speed)
- `weaveSpeed` (scroll speed)
- `shineInterval` / `shineBand` (logo shine frequency/width)
- Cable count formula in `RenderRow`
- Subtitle typewriter speed (60ms per char)

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "polish alt launch screen"
```

---

## Unresolved Questions

- Want the `--alt` flag as default eventually, or keep original as default?
- Dashboard-style menu (mentioned in brainstorm) — do as separate follow-up PR?
