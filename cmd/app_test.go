package cmd

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
	"github.com/jessewaites/cableknit-cli/internal/config"
)

func resetAuthState() {
	api.DemoEnabled = false
	api.DemoLoggedIn = false
	_ = config.ClearToken()
}

// --- isHidden tests ---

func TestIsHidden_LoggedOut(t *testing.T) {
	tests := []struct {
		item   menuItem
		hidden bool
	}{
		{menuLogin, false},
		{menuDevApply, false},
		{menuLogout, true},
		{menuWhoami, false},
		{menuExit, false},
	}
	for _, tt := range tests {
		if got := isHidden(tt.item, false); got != tt.hidden {
			t.Errorf("isHidden(%s, loggedOut) = %v, want %v", menuLabel(tt.item), got, tt.hidden)
		}
	}
}

func TestIsHidden_LoggedIn(t *testing.T) {
	tests := []struct {
		item   menuItem
		hidden bool
	}{
		{menuLogin, true},
		{menuDevApply, true},
		{menuLogout, false},
		{menuWhoami, false},
		{menuExit, false},
	}
	for _, tt := range tests {
		if got := isHidden(tt.item, true); got != tt.hidden {
			t.Errorf("isHidden(%s, loggedIn) = %v, want %v", menuLabel(tt.item), got, tt.hidden)
		}
	}
}

// --- Navigation helper tests ---

func TestSkipHidden_SkipsLogoutWhenLoggedOut(t *testing.T) {
	items := flatMenuItems()
	// Find menuLogout index
	logoutIdx := -1
	for i, item := range items {
		if item == menuLogout {
			logoutIdx = i
			break
		}
	}
	if logoutIdx < 0 {
		t.Fatal("menuLogout not found in flat items")
	}

	result := skipHidden(logoutIdx, 1, items, false)
	if items[result] == menuLogout {
		t.Error("skipHidden should skip menuLogout when not logged in")
	}
}

func TestSkipHidden_SkipsLoginWhenLoggedIn(t *testing.T) {
	items := flatMenuItems()
	loginIdx := -1
	for i, item := range items {
		if item == menuLogin {
			loginIdx = i
			break
		}
	}
	if loginIdx < 0 {
		t.Fatal("menuLogin not found in flat items")
	}

	result := skipHidden(loginIdx, 1, items, true)
	if items[result] == menuLogin {
		t.Error("skipHidden should skip menuLogin when logged in")
	}
}

func TestVisibleRowInGroup_SkipsHiddenItems(t *testing.T) {
	// Account group is index 2, contains: Login, DevApply, Whoami, PluginsList, Logout, Exit
	// When logged out, Logout is hidden
	// Login is row 0 (visible 0), DevApply is row 1 (visible 1), Whoami is row 2 (visible 2)
	accountIdx := 2

	visRow := visibleRowInGroup(accountIdx, 2, false) // Whoami
	if visRow != 2 {
		t.Errorf("visibleRowInGroup for Whoami (logged out) = %d, want 2", visRow)
	}

	// When logged in, Login (row 0) and DevApply (row 1) are hidden
	// Whoami is row 2 but visible row 0
	visRow = visibleRowInGroup(accountIdx, 2, true)
	if visRow != 0 {
		t.Errorf("visibleRowInGroup for Whoami (logged in) = %d, want 0", visRow)
	}
}

func TestFlatIndexFromVisibleRow_ClampsToLastVisible(t *testing.T) {
	accountIdx := 2
	// When logged out, Account has 5 visible items (Login, DevApply, Whoami, PluginsList, Exit)
	// Requesting visible row 99 should clamp to Exit
	idx := flatIndexFromVisibleRow(accountIdx, 99, false)
	items := flatMenuItems()
	if items[idx] != menuExit {
		t.Errorf("flatIndexFromVisibleRow clamped to %s, want Exit", menuLabel(items[idx]))
	}
}

// --- Menu view tests ---

func TestMenuView_LoggedOut_ShowsDevApply(t *testing.T) {
	resetAuthState()
	m := newMenuModel()
	m.width = 120
	m.height = 40

	view := m.View()
	if !strings.Contains(view, "Apply to Developer Program") {
		t.Error("menu should show 'Apply to Developer Program' when logged out")
	}
	if strings.Contains(view, "Logout") {
		t.Error("menu should not show 'Logout' when logged out")
	}
}

func TestMenuView_LoggedIn_HidesDevApply(t *testing.T) {
	resetAuthState()
	api.DemoLoggedIn = true
	defer resetAuthState()

	m := newMenuModel()
	m.width = 120
	m.height = 40

	view := m.View()
	if strings.Contains(view, "Apply to Developer Program") {
		t.Error("menu should not show 'Apply to Developer Program' when logged in")
	}
	if !strings.Contains(view, "Logout") {
		t.Error("menu should show 'Logout' when logged in")
	}
}

func TestMenuView_LoggedIn_HidesLogin(t *testing.T) {
	resetAuthState()
	api.DemoLoggedIn = true
	defer resetAuthState()

	m := newMenuModel()
	m.width = 120
	m.height = 40

	view := m.View()
	if strings.Contains(view, "Login") && !strings.Contains(view, "logged in") {
		// "Login" as a menu item vs "logged in" status — check more carefully
		// Remove the status line and check
		lines := strings.Split(view, "\n")
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed == "Login" || strings.Contains(trimmed, "│") && strings.Contains(trimmed, "Login") && !strings.Contains(trimmed, "logged") {
				t.Error("menu should not show 'Login' menu item when logged in")
				break
			}
		}
	}
}

func TestMenuView_LoggedOut_HidesStats(t *testing.T) {
	resetAuthState()
	m := newMenuModel()
	m.width = 120
	m.height = 40
	m.stats = &api.StatsResponse{
		TotalPublishedApps:  12,
		SubscribedAppsCount: 3,
		PayoutYTDCents:      48500,
	}

	view := m.View()
	if strings.Contains(view, "Apps Published") {
		t.Error("menu should not show stats when logged out")
	}
}

func TestMenuView_LoggedIn_ShowsStats(t *testing.T) {
	resetAuthState()
	api.DemoLoggedIn = true
	defer resetAuthState()

	m := newMenuModel()
	m.width = 120
	m.height = 40
	m.stats = &api.StatsResponse{
		TotalPublishedApps:  12,
		SubscribedAppsCount: 3,
		PayoutYTDCents:      48500,
	}

	view := m.View()
	if !strings.Contains(view, "Apps Published") {
		t.Error("menu should show stats when logged in")
	}
}

// --- Arrow key navigation tests ---

func TestNavigation_DownSkipsHiddenLogout(t *testing.T) {
	resetAuthState()
	items := flatMenuItems()

	m := newMenuModel()
	m.width = 120
	m.height = 40

	// Position cursor on the item before Logout in the Account group
	// Find Logout and go one before it
	for i, item := range items {
		if item == menuLogout {
			m.cursor = i - 1
			break
		}
	}

	// Press down
	um, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})

	if items[um.cursor] == menuLogout {
		t.Error("down arrow should skip hidden Logout when logged out")
	}
}

func TestNavigation_RightArrowMapsVisibleRows(t *testing.T) {
	resetAuthState()
	items := flatMenuItems()

	m := newMenuModel()
	m.width = 120
	m.height = 40

	// Position on last item in Monitor group (Versions)
	monitorGroupIdx := 1
	lastMonitorRow := len(menuGroups[monitorGroupIdx].items) - 1
	m.cursor = groupPosToFlat(monitorGroupIdx, lastMonitorRow)

	// Press right — should land on a visible item in Account group
	um, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyRight})

	landed := items[um.cursor]
	if isHidden(landed, false) {
		t.Errorf("right arrow landed on hidden item: %s", menuLabel(landed))
	}

	// Should be in Account group
	gi, _ := cursorToGroupPos(um.cursor)
	if gi != 2 {
		t.Errorf("right arrow should move to Account group (2), got group %d", gi)
	}
}
