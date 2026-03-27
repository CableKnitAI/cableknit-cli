package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/viewport"
	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/config"
	"github.com/jessewaites/cableknit-cli/internal/ui"
)

// --- App screen enum ---

type appScreen int

const (
	screenSplash appScreen = iota
	screenMenu
	screenLogin
	screenWhoami
	screenPluginsList
	screenRunsList
	screenRunsTailInput
	screenRunsTail
	screenValidate
	screenPush
	screenReadme
	screenSamplePlugin
	screenConnectorsList
	screenGenerate
	screenPackage
	screenToolsList
	screenMCPSetup
	screenLogs
	screenMetrics
	screenErrors
	screenVersions
)

// --- Messages ---

type manifestLoadedMsg struct {
	manifest *api.Manifest
	err      error
}

type menuSelectedMsg struct {
	item menuItem
}

type statsResultMsg struct {
	stats api.StatsResponse
	err   error
}

type whoamiResultMsg struct {
	user api.User
	err  error
}

// --- Menu items ---

type menuItem int

const (
	menuReadme menuItem = iota
	menuSamplePlugin
	menuGenerate
	menuPackage
	menuLogin
	menuDevApply
	menuWhoami
	menuPluginsList
	menuRunsList
	menuRunsTail
	menuConnectorsList
	menuToolsList
	menuValidate
	menuPush
	menuMCPSetup
	menuLogs
	menuMetrics
	menuErrors
	menuVersions
	menuLogout
	menuExit
)

func menuLabel(item menuItem) string {
	switch item {
	case menuReadme:
		return "Read Me"
	case menuSamplePlugin:
		return "View Sample Plugin"
	case menuGenerate:
		return "Generate Plugin Scaffold"
	case menuPackage:
		return "Package as .sweater"
	case menuLogin:
		return "Login"
	case menuDevApply:
		return "Apply to Developer Program"
	case menuWhoami:
		return "Whoami"
	case menuPluginsList:
		return "Plugins — List"
	case menuRunsList:
		return "Runs — List"
	case menuRunsTail:
		return "Runs — Tail"
	case menuConnectorsList:
		return "Connectors — List"
	case menuToolsList:
		return "Tools — List"
	case menuValidate:
		return "Validate"
	case menuPush:
		return "Push"
	case menuMCPSetup:
		return "MCP Setup"
	case menuLogs:
		return "Logs"
	case menuMetrics:
		return "Metrics"
	case menuErrors:
		return "Errors"
	case menuVersions:
		return "Versions"
	case menuLogout:
		return "Logout"
	case menuExit:
		return "Exit"
	default:
		return ""
	}
}

func needsAuth(item menuItem) bool {
	switch item {
	case menuWhoami, menuPluginsList, menuRunsList, menuRunsTail, menuConnectorsList, menuGenerate, menuPackage, menuValidate, menuPush, menuLogs, menuMetrics, menuErrors, menuVersions, menuLogout:
		return true
	default:
		return false
	}
}

// --- Menu groups ---

type menuGroup struct {
	title string
	items []menuItem
}

var menuGroups = []menuGroup{
	{title: "Build", items: []menuItem{menuReadme, menuSamplePlugin, menuToolsList, menuConnectorsList, menuGenerate, menuPackage, menuValidate, menuPush, menuMCPSetup}},
	{title: "Monitor", items: []menuItem{menuRunsList, menuRunsTail, menuLogs, menuMetrics, menuErrors, menuVersions}},
	{title: "Account", items: []menuItem{menuLogin, menuDevApply, menuWhoami, menuPluginsList, menuLogout, menuExit}},
}

// cursorToGroupPos converts a flat cursor index to (groupIndex, rowInGroup).
func cursorToGroupPos(cursor int) (int, int) {
	offset := 0
	for gi, g := range menuGroups {
		if cursor < offset+len(g.items) {
			return gi, cursor - offset
		}
		offset += len(g.items)
	}
	return len(menuGroups) - 1, 0
}

// groupPosToFlat converts (groupIndex, rowInGroup) to a flat cursor index.
func groupPosToFlat(groupIdx, row int) int {
	offset := 0
	for i := 0; i < groupIdx; i++ {
		offset += len(menuGroups[i].items)
	}
	return offset + row
}

func flatMenuItems() []menuItem {
	var items []menuItem
	for _, g := range menuGroups {
		items = append(items, g.items...)
	}
	return items
}

// visibleRowInGroup returns the visual row index (skipping hidden items) for a flat row in a group.
func visibleRowInGroup(groupIdx, rowInGroup int, loggedIn bool) int {
	vis := 0
	for i := 0; i < rowInGroup; i++ {
		if !isHidden(menuGroups[groupIdx].items[i], loggedIn) {
			vis++
		}
	}
	return vis
}

// flatIndexFromVisibleRow returns the flat cursor index for the nth visible row in a group.
// If visRow exceeds visible items, clamps to the last visible item.
func flatIndexFromVisibleRow(groupIdx, visRow int, loggedIn bool) int {
	vis := 0
	lastVisible := 0
	for i, item := range menuGroups[groupIdx].items {
		if isHidden(item, loggedIn) {
			continue
		}
		if vis == visRow {
			return groupPosToFlat(groupIdx, i)
		}
		lastVisible = i
		vis++
	}
	return groupPosToFlat(groupIdx, lastVisible)
}

// skipHidden advances the cursor in the given direction (+1 or -1) past hidden items, wrapping around.
func skipHidden(cursor, dir int, items []menuItem, loggedIn bool) int {
	n := len(items)
	for range n {
		if !isHidden(items[cursor], loggedIn) {
			return cursor
		}
		cursor = (cursor + dir + n) % n
	}
	return cursor
}

// --- Menu model ---

type menuModel struct {
	cursor    int
	width     int
	height    int
	flash     string
	stats     *api.StatsResponse
	statsErr  error
}

func newMenuModel() menuModel {
	return menuModel{}
}

func (m menuModel) Init() tea.Cmd {
	return m.fetchStats()
}

func (m menuModel) fetchStats() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		var stats api.StatsResponse
		if err := client.JSON("GET", "/api/v1/stats", nil, &stats); err != nil {
			return statsResultMsg{err: err}
		}
		return statsResultMsg{stats: stats}
	}
}

func (m menuModel) Update(msg tea.Msg) (menuModel, tea.Cmd) {
	items := flatMenuItems()
	switch msg := msg.(type) {
	case statsResultMsg:
		if msg.err == nil {
			m.stats = &msg.stats
		} else {
			m.statsErr = msg.err
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		m.flash = ""
		loggedIn := config.Token() != "" || api.DemoLoggedIn
		switch msg.String() {
		case "up", "k":
			m.cursor--
			if m.cursor < 0 {
				m.cursor = len(items) - 1
			}
			m.cursor = skipHidden(m.cursor, -1, items, loggedIn)
		case "down", "j":
			m.cursor++
			if m.cursor >= len(items) {
				m.cursor = 0
			}
			m.cursor = skipHidden(m.cursor, 1, items, loggedIn)
		case "left", "h":
			groupIdx, rowInGroup := cursorToGroupPos(m.cursor)
			if groupIdx > 0 {
				visRow := visibleRowInGroup(groupIdx, rowInGroup, loggedIn)
				groupIdx--
				m.cursor = flatIndexFromVisibleRow(groupIdx, visRow, loggedIn)
			}
		case "right", "l":
			groupIdx, rowInGroup := cursorToGroupPos(m.cursor)
			if groupIdx < len(menuGroups)-1 {
				visRow := visibleRowInGroup(groupIdx, rowInGroup, loggedIn)
				groupIdx++
				m.cursor = flatIndexFromVisibleRow(groupIdx, visRow, loggedIn)
			}
		case "enter":
			selected := items[m.cursor]
			loggedIn := config.Token() != "" || api.DemoLoggedIn
			if needsAuth(selected) && !loggedIn && selected != menuLogout {
				m.flash = "Log in first"
				return m, nil
			}
			return m, func() tea.Msg { return menuSelectedMsg{item: selected} }
		}
	}
	return m, nil
}

func (m menuModel) View() string {
	loggedIn := config.Token() != "" || api.DemoLoggedIn

	cardGap := 3
	totalCards := len(menuGroups)
	minCardWidth := 28 // enough for longest label + borders

	// Decide layout: horizontal if cards fit at minCardWidth, else stack
	// Default to horizontal (width 0 means we haven't got WindowSizeMsg yet)
	horizontalWidth := minCardWidth*totalCards + cardGap*(totalCards-1)
	horizontal := m.width == 0 || m.width >= horizontalWidth+4

	width := m.width
	if width == 0 {
		width = 120 // sensible default before first WindowSizeMsg
	}

	var cardWidth, totalWidth int
	if horizontal {
		cardWidth = (width - 6 - cardGap*(totalCards-1)) / totalCards
		totalWidth = cardWidth*totalCards + cardGap*(totalCards-1)
	} else {
		cardWidth = min(width-4, 50)
		totalWidth = cardWidth
	}
	if cardWidth < minCardWidth {
		cardWidth = minCardWidth
	}

	// Title + status
	title := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue).Render("C A B L E K N I T  A I")
	var status string
	if loggedIn {
		status = ui.SuccessStyle.Render(ui.SymbolDot + " logged in")
	} else {
		status = ui.DimStyle.Render(ui.SymbolDot + " not logged in")
	}
	titleW := lipgloss.Width(title)
	statusW := lipgloss.Width(status)
	titleGap := totalWidth - titleW - statusW
	if titleGap < 2 {
		titleGap = 2
	}
	titleLine := title + strings.Repeat(" ", titleGap) + status

	// Render cards
	flatIdx := 0
	var cardBlock string

	if horizontal {
		// Horizontal layout — join cards side by side
		var allCardRows [][]string
		maxRows := 0
		for _, group := range menuGroups {
			rows := renderCardRows(group, cardWidth, m.cursor, flatIdx, loggedIn)
			allCardRows = append(allCardRows, rows)
			if len(rows) > maxRows {
				maxRows = len(rows)
			}
			flatIdx += len(group.items)
		}

		gapStr := strings.Repeat(" ", cardGap)
		emptyCard := strings.Repeat(" ", cardWidth)
		var cb strings.Builder
		for row := 0; row < maxRows; row++ {
			if row > 0 {
				cb.WriteByte('\n')
			}
			for ci, rows := range allCardRows {
				if ci > 0 {
					cb.WriteString(gapStr)
				}
				if row < len(rows) {
					cb.WriteString(rows[row])
				} else {
					cb.WriteString(emptyCard)
				}
			}
		}
		cardBlock = cb.String()
	} else {
		// Stacked layout
		var cb strings.Builder
		for gi, group := range menuGroups {
			if gi > 0 {
				cb.WriteByte('\n')
			}
			rows := renderCardRows(group, cardWidth, m.cursor, flatIdx, loggedIn)
			cb.WriteString(strings.Join(rows, "\n"))
			flatIdx += len(group.items)
		}
		cardBlock = cb.String()
	}

	// Flash message
	var flashLine string
	if m.flash != "" {
		flashLine = ui.ErrorStyle.Render(ui.SymbolCross+" "+m.flash) + "\n"
	}

	// Stats line
	var statsLine string
	if m.stats != nil && loggedIn {
		dollars := fmt.Sprintf("$%d.%02d", m.stats.PayoutYTDCents/100, m.stats.PayoutYTDCents%100)
		payoutStr := ui.SuccessStyle.Render(fmt.Sprintf("Payouts Year To Date USD: %s", dollars))
		statsLine = ui.BlueStyle.Render(fmt.Sprintf(
			"Apps Published: %d    Apps Subscribed: %d    ",
			m.stats.TotalPublishedApps, m.stats.SubscribedAppsCount,
		)) + payoutStr + "\n\n"
	}

	hint := ui.DimStyle.Render("↑↓ navigate  ←→ switch card  enter select  q quit")

	content := titleLine + "\n\n" + cardBlock + "\n\n" + statsLine + flashLine + hint

	// Top-align with small padding
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	topPad := 2

	leftPad := (width - totalWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	padStr := strings.Repeat(" ", leftPad)

	var centered []string
	for _, line := range lines {
		centered = append(centered, padStr+line)
	}

	var out strings.Builder
	out.WriteString(strings.Repeat("\n", topPad))
	out.WriteString(strings.Join(centered, "\n"))

	bottomPad := m.height - topPad - contentHeight
	if bottomPad > 0 {
		out.WriteString(strings.Repeat("\n", bottomPad))
	}

	return out.String()
}

// renderCardRows returns each line of a card as a separate string, all exactly cardWidth wide.
func isHidden(item menuItem, loggedIn bool) bool {
	return (item == menuDevApply && loggedIn) ||
		(item == menuLogin && loggedIn) ||
		(item == menuLogout && !loggedIn)
}

func renderCardRows(group menuGroup, cardWidth, cursor, cursorOffset int, loggedIn bool) []string {
	borderStyle := lipgloss.NewStyle().Foreground(ui.Dim)
	titleStyle := lipgloss.NewStyle().Foreground(ui.Blue).Bold(true)
	selectedStyle := lipgloss.NewStyle().Foreground(ui.White).Background(ui.Blue)
	dimItemStyle := lipgloss.NewStyle().Foreground(ui.Dim)

	innerWidth := cardWidth - 2

	// Top border: ╭─ Title ────...─╮
	titleStr := " " + group.title + " "
	titleRendered := titleStyle.Render(titleStr)
	titleVisW := lipgloss.Width(titleRendered)
	dashesAfter := innerWidth - 1 - titleVisW
	if dashesAfter < 1 {
		dashesAfter = 1
	}
	topBorder := borderStyle.Render("╭─") + titleRendered + borderStyle.Render(strings.Repeat("─", dashesAfter)+"╮")

	// Bottom border
	bottomBorder := borderStyle.Render("╰" + strings.Repeat("─", innerWidth) + "╯")

	var rows []string
	rows = append(rows, topBorder)

	// Find max visible items across all groups for uniform height
	maxItems := 0
	for _, g := range menuGroups {
		visible := 0
		for _, it := range g.items {
			if !isHidden(it, loggedIn) {
				visible++
			}
		}
		if visible > maxItems {
			maxItems = visible
		}
	}

	visibleCount := 0
	for i, item := range group.items {
		if isHidden(item, loggedIn) {
			continue
		}
		visibleCount++

		globalIdx := cursorOffset + i
		isSelected := globalIdx == cursor
		isDimmed := needsAuth(item) && !loggedIn && item != menuLogout

		label := menuLabel(item)
		contentWidth := innerWidth - 2
		labelW := lipgloss.Width(label)
		if labelW < contentWidth {
			label += strings.Repeat(" ", contentWidth-labelW)
		}

		var rendered string
		switch {
		case isSelected:
			rendered = selectedStyle.Render(" " + label + " ")
		case isDimmed:
			rendered = dimItemStyle.Render(" " + label + " ")
		default:
			rendered = " " + label + " "
		}

		rows = append(rows, borderStyle.Render("│")+rendered+borderStyle.Render("│"))
	}

	// Pad with empty rows to match tallest card
	emptyRow := borderStyle.Render("│") + strings.Repeat(" ", innerWidth) + borderStyle.Render("│")
	for i := visibleCount; i < maxItems; i++ {
		rows = append(rows, emptyRow)
	}

	rows = append(rows, bottomBorder)
	return rows
}

// --- Whoami TUI model ---

type whoamiState int

const (
	whoamiLoading whoamiState = iota
	whoamiDone
)

type whoamiModel struct {
	state   whoamiState
	spinner spinner.Model
	user    api.User
	err     error
}

func newWhoamiModel() whoamiModel {
	return whoamiModel{
		spinner: ui.NewSpinner(spinner.Dot),
	}
}

func (m whoamiModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetch())
}

func (m whoamiModel) Update(msg tea.Msg) (whoamiModel, tea.Cmd) {
	switch msg := msg.(type) {
	case whoamiResultMsg:
		m.state = whoamiDone
		m.user = msg.user
		m.err = msg.err
		return m, nil

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.state == whoamiDone && (msg.String() == "q" || msg.String() == "esc" || msg.String() == "enter") {
			return m, func() tea.Msg { return screenDoneMsg{} }
		}
	}

	if m.state == whoamiLoading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m whoamiModel) View() string {
	switch m.state {
	case whoamiLoading:
		return "\n  " + m.spinner.View() + " Fetching user info...\n\n"
	case whoamiDone:
		if m.err != nil {
			return "\n" + ui.ErrorStyle.Render(ui.SymbolCross+" "+m.err.Error()) + "\n\n"
		}
		bold := lipgloss.NewStyle().Bold(true)
		status := ui.SuccessStyle.Render(ui.SymbolCheck + " " + m.user.PublisherStatus)
		if m.user.PublisherStatus == "pending" {
			status = ui.WarningStyle.Render(ui.SymbolWarning + " Publisher approval pending")
		}
		content := fmt.Sprintf(
			"%s  %s\n%s  %s\n%s  %s",
			bold.Render("Name:"), m.user.Name,
			bold.Render("Email:"), m.user.Email,
			bold.Render("Status:"), status,
		)
		return "\n" + ui.SuccessBox.Render(content) + "\n\n" + ui.DimStyle.Render("  q/esc back") + "\n"
	}
	return ""
}

func (m whoamiModel) fetch() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		var user api.User
		if err := client.JSON("GET", "/api/v1/cli/me", nil, &user); err != nil {
			return whoamiResultMsg{err: err}
		}
		return whoamiResultMsg{user: user}
	}
}

// --- Readme model ---

type docViewModel struct {
	title    string
	content  string
	viewport viewport.Model
	ready    bool
	width    int
	height   int
}

func newDocViewModel(title, content string, width, height int) docViewModel {
	m := docViewModel{title: title, content: content, width: width, height: height}
	if width > 0 && height > 0 {
		m.initViewport()
	}
	return m
}

func (m *docViewModel) initViewport() {
	headerHeight := 3
	footerHeight := 2
	vpHeight := m.height - headerHeight - footerHeight
	if vpHeight < 1 {
		vpHeight = 1
	}
	m.viewport = viewport.New(
		viewport.WithWidth(m.width),
		viewport.WithHeight(vpHeight),
	)
	m.viewport.SetContent(m.styledContent())
	m.ready = true
}

func (m docViewModel) styledContent() string {
	bold := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)
	code := lipgloss.NewStyle().Foreground(ui.Red)
	dim := ui.DimStyle
	pad := "    "

	subtitle := lipgloss.NewStyle().Bold(true).Foreground(ui.White)

	var sb strings.Builder
	for _, line := range strings.Split(m.content, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "":
			sb.WriteString("\n")
		case strings.HasPrefix(line, "> "):
			sb.WriteString(pad + subtitle.Render(trimmed[2:]) + "\n")
		case strings.HasPrefix(line, "  "):
			sb.WriteString(pad + code.Render(line) + "\n")
		case len(trimmed) > 0 && trimmed == strings.ToUpper(trimmed) && !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "}") && !strings.HasPrefix(trimmed, "#") && !strings.HasPrefix(trimmed, "[") && !strings.HasPrefix(trimmed, "|") && !strings.HasPrefix(trimmed, "─") && !strings.HasPrefix(trimmed, "┌") && !strings.HasPrefix(trimmed, "├") && !strings.HasPrefix(trimmed, "└"):
			// ALL-CAPS lines are section headers
			sb.WriteString(pad + bold.Render(trimmed) + "\n")
		case len(trimmed) > 2 && trimmed[0] >= 'A' && trimmed[0] <= 'Z' && !strings.Contains(trimmed, "=>") && !strings.Contains(trimmed, "{") && len(trimmed) < 60 && !strings.HasSuffix(trimmed, ".") && !strings.HasSuffix(trimmed, ":") && !strings.HasSuffix(trimmed, ","):
			// Title-case short lines = subsection headers
			sb.WriteString(pad + bold.Render(trimmed) + "\n")
		default:
			sb.WriteString(pad + dim.Render(trimmed) + "\n")
		}
	}
	return sb.String()
}

func (m docViewModel) Init() tea.Cmd { return nil }

func (m docViewModel) Update(msg tea.Msg) (docViewModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.initViewport()
		} else {
			headerHeight := 3
			footerHeight := 2
			vpHeight := m.height - headerHeight - footerHeight
			if vpHeight < 1 {
				vpHeight = 1
			}
			m.viewport.SetWidth(m.width)
			m.viewport.SetHeight(vpHeight)
		}

	case tea.KeyPressMsg:
		switch msg.String() {
		case "q", "esc":
			return m, func() tea.Msg { return screenDoneMsg{} }
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	if m.ready {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m docViewModel) View() string {
	if !m.ready {
		return ""
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue).
		Render(m.title)
	hint := ui.DimStyle.Render("↑↓ scroll  q/esc back")

	scrollPct := int(m.viewport.ScrollPercent() * 100)
	pct := ui.DimStyle.Render(fmt.Sprintf(" %d%%", scrollPct))

	return "    " + title + "\n\n" + m.viewport.View() + "\n" + "    " + hint + pct
}

// --- Run ID input model ---

type runInputModel struct {
	runID   string
	width   int
	height  int
}

func newRunInputModel() runInputModel {
	return runInputModel{}
}

func (m runInputModel) Init() tea.Cmd { return nil }

func (m runInputModel) Update(msg tea.Msg) (runInputModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m, func() tea.Msg { return screenDoneMsg{} }
		case "enter":
			if strings.TrimSpace(m.runID) != "" {
				return m, nil // handled by appModel
			}
		case "backspace":
			if len(m.runID) > 0 {
				m.runID = m.runID[:len(m.runID)-1]
			}
		default:
			// Only accept printable single chars
			k := msg.String()
			if len(k) == 1 && k[0] >= 32 && k[0] <= 126 {
				m.runID += k
			}
		}
	}
	return m, nil
}

func (m runInputModel) View() string {
	title := lipgloss.NewStyle().Bold(true).Foreground(ui.Blue).Render("Runs — Tail")
	prompt := lipgloss.NewStyle().Bold(true).Render("Run ID: ")
	cursor := lipgloss.NewStyle().Foreground(ui.Blue).Render("█")
	hint := ui.DimStyle.Render("enter to stream  esc to cancel")

	content := title + "\n\n  " + prompt + m.runID + cursor + "\n\n  " + hint
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)

	topPad := (m.height - contentHeight) / 2
	if topPad < 0 {
		topPad = 0
	}

	var centered []string
	for _, line := range lines {
		lineW := lipgloss.Width(line)
		leftPad := (m.width - lineW) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		centered = append(centered, strings.Repeat(" ", leftPad)+line)
	}

	var sb strings.Builder
	sb.WriteString(strings.Repeat("\n", topPad))
	sb.WriteString(strings.Join(centered, "\n"))
	return sb.String()
}

// --- App model (parent) ---

type appModel struct {
	screen      appScreen
	width       int
	height      int
	manifest    *api.Manifest
	splash      splashModel
	menu        menuModel
	readme        docViewModel
	samplePlugin  docViewModel
	mcpSetup      docViewModel
	login         loginModel
	whoami      whoamiModel
	pluginsList pluginsListModel
	runsList        runsListModel
	connectorsList  connectorsListModel
	toolsList       toolsListModel
	runInput        runInputModel
	tail        tailModel
	validate    validateModel
	push        pushModel
	generate    generateModel
	pkg         packageModel
	logsList    logsListModel
	metricsView metricsModel
	errorsList    errorsListModel
	versionsList  versionsListModel
	altSplash     altSplashModel
	useAlt      bool
}

func newAppModel() appModel {
	splash := newSplashModel()
	splash.embedded = true
	m := appModel{
		screen: screenSplash,
		splash: splash,
		menu:   newMenuModel(),
		useAlt: useAltSplash,
	}
	if m.useAlt {
		alt := newAltSplashModel()
		alt.embedded = true
		m.altSplash = alt
	}
	return m
}

func (m appModel) Init() tea.Cmd {
	if m.useAlt {
		return tea.Batch(m.altSplash.Init(), m.fetchManifest())
	}
	return tea.Batch(m.splash.Init(), m.fetchManifest())
}

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to all active models
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		if m.screen == screenMenu && (msg.String() == "q" || msg.String() == "Q") {
			return m, tea.Quit
		}
	case manifestLoadedMsg:
		if msg.err == nil && msg.manifest != nil {
			m.manifest = msg.manifest
			colorsJSON := msg.manifest.JSONContent("styles", "brand_colors")
			symbolsJSON := msg.manifest.JSONContent("styles", "prefix_symbols")
			ui.ApplyManifestStyles(colorsJSON, symbolsJSON)
		}
		return m, nil
	case statsResultMsg:
		m.menu, _ = m.menu.Update(msg)
		return m, nil
	case screenDoneMsg:
		return m.returnToMenu()
	case menuSelectedMsg:
		return m.handleMenuSelection(msg.item)
	}

	// Route to active child
	switch m.screen {
	case screenSplash:
		if m.useAlt {
			var cmd tea.Cmd
			var model tea.Model
			model, cmd = m.altSplash.Update(msg)
			m.altSplash = model.(altSplashModel)
			return m, cmd
		}
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.splash.Update(msg)
		m.splash = model.(splashModel)
		return m, cmd

	case screenMenu:
		var cmd tea.Cmd
		m.menu, cmd = m.menu.Update(msg)
		return m, cmd

	case screenReadme:
		var cmd tea.Cmd
		m.readme, cmd = m.readme.Update(msg)
		return m, cmd

	case screenSamplePlugin:
		var cmd tea.Cmd
		m.samplePlugin, cmd = m.samplePlugin.Update(msg)
		return m, cmd

	case screenMCPSetup:
		var cmd tea.Cmd
		m.mcpSetup, cmd = m.mcpSetup.Update(msg)
		return m, cmd

	case screenLogin:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.login.Update(msg)
		m.login = model.(loginModel)
		return m, cmd

	case screenWhoami:
		var cmd tea.Cmd
		m.whoami, cmd = m.whoami.Update(msg)
		return m, cmd

	case screenPluginsList:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.pluginsList.Update(msg)
		m.pluginsList = model.(pluginsListModel)
		return m, cmd

	case screenRunsList:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.runsList.Update(msg)
		m.runsList = model.(runsListModel)
		return m, cmd

	case screenConnectorsList:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.connectorsList.Update(msg)
		m.connectorsList = model.(connectorsListModel)
		return m, cmd

	case screenToolsList:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.toolsList.Update(msg)
		m.toolsList = model.(toolsListModel)
		return m, cmd

	case screenRunsTailInput:
		var cmd tea.Cmd
		m.runInput, cmd = m.runInput.Update(msg)
		// Check if enter was pressed with a run ID
		if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "enter" && strings.TrimSpace(m.runInput.runID) != "" {
			return m.startTail(strings.TrimSpace(m.runInput.runID))
		}
		return m, cmd

	case screenRunsTail:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.tail.Update(msg)
		m.tail = model.(tailModel)
		return m, cmd

	case screenValidate:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.validate.Update(msg)
		m.validate = model.(validateModel)
		return m, cmd

	case screenPush:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.push.Update(msg)
		m.push = model.(pushModel)
		return m, cmd

	case screenGenerate:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.generate.Update(msg)
		m.generate = model.(generateModel)
		return m, cmd

	case screenPackage:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.pkg.Update(msg)
		m.pkg = model.(packageModel)
		return m, cmd

	case screenLogs:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.logsList.Update(msg)
		m.logsList = model.(logsListModel)
		return m, cmd

	case screenMetrics:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.metricsView.Update(msg)
		m.metricsView = model.(metricsModel)
		return m, cmd

	case screenErrors:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.errorsList.Update(msg)
		m.errorsList = model.(errorsListModel)
		return m, cmd

	case screenVersions:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.versionsList.Update(msg)
		m.versionsList = model.(versionsListModel)
		return m, cmd
	}

	return m, nil
}

func (m appModel) View() tea.View {
	var s string

	switch m.screen {
	case screenSplash:
		if m.useAlt {
			v := m.altSplash.View()
			v.AltScreen = true
			return v
		}
		v := m.splash.View()
		v.AltScreen = true
		return v
	case screenMenu:
		s = m.menu.View()
	case screenReadme:
		s = m.readme.View()
	case screenSamplePlugin:
		s = m.samplePlugin.View()
	case screenMCPSetup:
		s = m.mcpSetup.View()
	case screenLogin:
		s = m.login.View().Content
	case screenWhoami:
		s = m.whoami.View()
	case screenPluginsList:
		s = m.pluginsList.View().Content
	case screenRunsList:
		s = m.runsList.View().Content
	case screenConnectorsList:
		s = m.connectorsList.View().Content
	case screenToolsList:
		s = m.toolsList.View().Content
	case screenRunsTailInput:
		s = m.runInput.View()
	case screenRunsTail:
		s = m.tail.View().Content
	case screenValidate:
		s = m.validate.View().Content
	case screenPush:
		s = m.push.View().Content
	case screenGenerate:
		s = m.generate.View().Content
	case screenPackage:
		s = m.pkg.View().Content
	case screenLogs:
		s = m.logsList.View().Content
	case screenMetrics:
		s = m.metricsView.View().Content
	case screenErrors:
		s = m.errorsList.View().Content
	case screenVersions:
		s = m.versionsList.View().Content
	}

	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

func (m appModel) firstPluginSlug() string {
	if m.pluginsList.plugins != nil && len(m.pluginsList.plugins) > 0 {
		return m.pluginsList.plugins[0].Slug
	}
	return ""
}

func (m appModel) returnToMenu() (tea.Model, tea.Cmd) {
	m.screen = screenMenu
	m.menu.flash = ""
	return m, m.menu.fetchStats()
}

func (m appModel) fetchManifest() tea.Cmd {
	return func() tea.Msg {
		client := api.NewAPIClient()
		manifest, err := api.FetchManifest(client)
		return manifestLoadedMsg{manifest: manifest, err: err}
	}
}

func (m appModel) handleMenuSelection(item menuItem) (tea.Model, tea.Cmd) {
	switch item {
	case menuReadme:
		content := ""
		if m.manifest != nil {
			content = m.manifest.Doc("readme_text")
		}
		if content == "" {
			content = "Unable to load content. Check your connection."
		}
		m.readme = newDocViewModel("Read Me — CableKnit CLI", content, m.width, m.height)
		m.screen = screenReadme
		return m, nil

	case menuSamplePlugin:
		content := ""
		if m.manifest != nil {
			content = m.manifest.Doc("sample_plugin_text")
		}
		if content == "" {
			content = "Unable to load content. Check your connection."
		}
		m.samplePlugin = newDocViewModel("View Sample Plugin — Invoice Sorter", content, m.width, m.height)
		m.screen = screenSamplePlugin
		return m, nil

	case menuGenerate:
		m.generate = newGenerateModel()
		m.generate.embedded = true
		m.screen = screenGenerate
		return m, m.generate.Init()

	case menuPackage:
		m.pkg = newPackageModel("")
		m.pkg.embedded = true
		m.screen = screenPackage
		return m, m.pkg.Init()

	case menuLogin:
		m.login = newLoginModel()
		m.login.embedded = true
		m.screen = screenLogin
		return m, m.login.Init()

	case menuWhoami:
		m.whoami = newWhoamiModel()
		m.screen = screenWhoami
		return m, m.whoami.Init()

	case menuPluginsList:
		m.pluginsList = newPluginsListModel()
		m.pluginsList.embedded = true
		m.screen = screenPluginsList
		return m, m.pluginsList.Init()

	case menuRunsList:
		m.runsList = newRunsListModel("", 0)
		m.runsList.embedded = true
		m.screen = screenRunsList
		return m, m.runsList.Init()

	case menuConnectorsList:
		m.connectorsList = newConnectorsListModel()
		m.connectorsList.embedded = true
		m.screen = screenConnectorsList
		return m, m.connectorsList.Init()

	case menuToolsList:
		m.toolsList = newToolsListModel()
		m.toolsList.embedded = true
		m.screen = screenToolsList
		return m, m.toolsList.Init()

	case menuRunsTail:
		m.runInput = newRunInputModel()
		m.runInput.width = m.width
		m.runInput.height = m.height
		m.screen = screenRunsTailInput
		return m, nil

	case menuValidate:
		m.validate = newValidateModel("")
		m.validate.embedded = true
		m.screen = screenValidate
		return m, m.validate.Init()

	case menuPush:
		m.push = newPushModel("")
		m.push.embedded = true
		m.screen = screenPush
		return m, m.push.Init()

	case menuMCPSetup:
		m.mcpSetup = newDocViewModel("MCP Setup — AI Editor Integration", mcpSetupContent(), m.width, m.height)
		m.screen = screenMCPSetup
		return m, nil

	case menuLogs:
		// Use first plugin slug as default; could add plugin selector later
		slug := m.firstPluginSlug()
		if slug == "" {
			m.menu.flash = "No plugins found"
			return m, nil
		}
		m.logsList = newLogsListModel(slug, "", "", "", 50)
		m.logsList.embedded = true
		m.screen = screenLogs
		return m, m.logsList.Init()

	case menuMetrics:
		slug := m.firstPluginSlug()
		if slug == "" {
			m.menu.flash = "No plugins found"
			return m, nil
		}
		m.metricsView = newMetricsModel(slug)
		m.metricsView.embedded = true
		m.screen = screenMetrics
		return m, m.metricsView.Init()

	case menuErrors:
		slug := m.firstPluginSlug()
		if slug == "" {
			m.menu.flash = "No plugins found"
			return m, nil
		}
		m.errorsList = newErrorsListModel(slug, "", 25)
		m.errorsList.embedded = true
		m.screen = screenErrors
		return m, m.errorsList.Init()

	case menuVersions:
		slug := m.firstPluginSlug()
		if slug == "" {
			m.menu.flash = "No plugins found"
			return m, nil
		}
		m.versionsList = newVersionsListModel(slug)
		m.versionsList.embedded = true
		m.screen = screenVersions
		return m, m.versionsList.Init()

	case menuDevApply:
		url := config.APIURL() + "/developer/apply"
		switch runtime.GOOS {
		case "darwin":
			exec.Command("open", url).Start()
		case "linux":
			exec.Command("xdg-open", url).Start()
		default:
			exec.Command("open", url).Start()
		}
		m.menu.flash = "Opening browser..."
		m.screen = screenMenu
		return m, nil

	case menuLogout:
		// Instant logout
		if config.Token() != "" {
			client := api.NewAPIClient()
			_ = client.JSON("DELETE", "/api/v1/cli/sessions", nil, nil)
		}
		_ = config.ClearToken()
		api.DemoLoggedIn = false
		m.menu.flash = ""
		m.screen = screenMenu
		return m, nil

	case menuExit:
		return m, tea.Quit
	}

	return m, nil
}

func (m appModel) startTail(runID string) (tea.Model, tea.Cmd) {
	m.tail = newTailModel(runID)
	m.tail.embedded = true
	m.screen = screenRunsTail
	return m, m.tail.Init()
}

func runAppShell() {
	p := tea.NewProgram(newAppModel())
	if _, err := p.Run(); err != nil {
		fmt.Println(err)
	}
}
