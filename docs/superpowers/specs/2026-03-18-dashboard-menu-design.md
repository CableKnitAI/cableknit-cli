# Dashboard-Style Menu Design

## Goal

Replace the flat list menu with a grouped, bordered card layout. Three cards (Build, Monitor, Account) with rounded borders and group titles. Cleaner visual hierarchy, same functionality.

## Current State

The menu in `cmd/app.go` (`menuModel`) is a flat list of 14 items + separators + Exit. Items are centered on screen with basic styling (selected = white-on-blue, dimmed = auth-required when not logged in). Navigation wraps top/bottom.

## Design

### Layout

```
       C A B L E K N I T  A I          ● logged in

  ╭─ Build ──────────────────────────────────╮
  │  Read Me                                 │
  │  View Sample Plugin                      │
  │  Generate Plugin Scaffold                │
  │  Package as .sweater                     │
  │  Validate                                │
  │  Push                                    │
  ╰──────────────────────────────────────────╯

  ╭─ Monitor ────────────────────────────────╮
  │  Plugins — List                          │
  │  Runs — List                             │
  │  Runs — Tail                             │
  │  Connectors — List                       │
  ╰──────────────────────────────────────────╯

  ╭─ Account ────────────────────────────────╮
  │  Login                                   │
  │  Whoami                                  │
  │  Logout                                  │
  ╰──────────────────────────────────────────╯

  ↑↓ navigate  enter select  q quit
```

### Groups

| Group | Items | Auth Required |
|-------|-------|---------------|
| Build | Read Me, View Sample Plugin, Generate, Package, Validate, Push | Validate, Push |
| Monitor | Plugins — List, Runs — List, Runs — Tail, Connectors — List | All |
| Account | Login, Whoami, Logout | Whoami, Logout |

**Note:** This intentionally regroups items from their current order. Validate/Push move into Build (they're part of the build→validate→push workflow). Labels use existing `menuLabel()` values unchanged.

### Card Rendering

- Rounded borders using Unicode box-drawing: `╭ ╮ ╰ ╯ │ ─`
- Border color: `ui.Dim`
- Group title embedded in top border: `╭─ Title ───...─╮`
- Group title color: `ui.Blue`, bold
- Card width: `min(45, termWidth - 4)` — responsive but capped
- 1 blank line between cards

### Item Rendering

- Each item rendered inside the box: `│  Item Name                    │`
- Selected item: white text on blue background (existing `selectedStyle`)
- Auth-required + not logged in: dimmed text (existing `dimItemStyle`)
- Normal: default foreground (existing `normalItemStyle`)
- Left padding: 2 spaces inside the border

### Navigation

- Cursor moves sequentially through all items across all cards (no card-level navigation)
- Up from first item in Monitor → last item in Build
- Down from last item in Build → first item in Monitor
- Wraps: up from first item overall → last item overall (and vice versa)
- `q`/`Q` quits (Exit removed from menu items)
- `enter` selects

### Title + Status Line

- Title: "C A B L E K N I T  A I" in blue bold (same as current)
- Status: `● logged in` (green) or `● not logged in` (dim), rendered on same line as title using manual padding to right-align within card width
- Flash messages appear below status line (same as current)

### Centering

- Entire card block centered vertically and horizontally on screen
- Total height = title(1) + blank(1) + card1(8) + blank(1) + card2(6) + blank(1) + card3(5) + blank(1) + hints(1) = ~25 lines
- If terminal too short, cards still render but may not fully center

## Architecture

### Changes to `cmd/app.go`

**Remove:** `menuSep0..menuSep4`, `isSeparator()`, `menuExit` item, separator rendering logic

**Restructure `menuItems`** into groups:

```go
type menuGroup struct {
    title string
    items []menuItem
}

var menuGroups = []menuGroup{
    {title: "Build", items: []menuItem{menuReadme, menuSamplePlugin, menuGenerate, menuPackage, menuValidate, menuPush}},
    {title: "Monitor", items: []menuItem{menuPluginsList, menuRunsList, menuRunsTail, menuConnectorsList}},
    {title: "Account", items: []menuItem{menuLogin, menuWhoami, menuLogout}},
}
```

**`menuModel` changes:**
- `cursor` still indexes into a flat list of all selectable items
- `View()` renders cards with borders instead of flat list
- Navigation skips card boundaries seamlessly
- Flash message rendering unchanged

**New helper functions:**
- `renderCard(group menuGroup, cardWidth int, cursor int, cursorOffset int, loggedIn bool) string` — renders a single bordered card. `cursorOffset` is the index of this group's first item in the flat list, so the function can check if `cursor` falls within this card.
- `flatItems() []menuItem` — returns all items across groups in order (for cursor indexing)

**Box drawing:** Hand-drawn with Unicode chars rather than lipgloss borders, because we need the group title embedded in the top border (`╭─ Title ─╮`) which lipgloss's `Border()` doesn't support natively.

**Iota cleanup:** Separator and exit constants removed. Remaining iota values keep their numbers (they're identifiers, ordering doesn't matter).

### No new files needed

All changes are within `cmd/app.go`. The menu is self-contained — no new packages or types needed outside the existing file.

## What's NOT Changing

- `appModel` routing, screen transitions, all other screens
- `menuSelectedMsg`, `menuItem` enum values (except removing separators and exit)
- Auth checking logic (`needsAuth()`)
- Keyboard shortcuts (up/down/j/k/enter/q)
- Flash message behavior
