package ui

import (
	"encoding/json"
	"os"

	"charm.land/lipgloss/v2"
	"golang.org/x/term"
)

// Brand colors — defaults, overridable via manifest
var (
	Blue   = lipgloss.Color("#5B9BD5")
	Green  = lipgloss.Color("#6BCB77")
	Yellow = lipgloss.Color("#FFD93D")
	Red    = lipgloss.Color("#FF6B6B")
	Dim    = lipgloss.Color("#666666")
	White  = lipgloss.Color("#FFFFFF")
)

// Prefix symbols — defaults, overridable via manifest
var (
	SymbolCheck   = "✓"
	SymbolCross   = "✗"
	SymbolWarning = "⚠"
	SymbolDot     = "●"
	SymbolArrow   = "→"
)

// ApplyManifestStyles updates colors and symbols from manifest JSON data.
func ApplyManifestStyles(colorsJSON, symbolsJSON []byte) {
	applyColors(colorsJSON)
	applySymbols(symbolsJSON)
	rebuildStyles()
}

func applyColors(data []byte) {
	if data == nil {
		return
	}
	var colors map[string]string
	if json.Unmarshal(data, &colors) != nil {
		return
	}
	if v, ok := colors["blue"]; ok {
		Blue = lipgloss.Color(v)
	}
	if v, ok := colors["green"]; ok {
		Green = lipgloss.Color(v)
	}
	if v, ok := colors["yellow"]; ok {
		Yellow = lipgloss.Color(v)
	}
	if v, ok := colors["red"]; ok {
		Red = lipgloss.Color(v)
	}
	if v, ok := colors["dim"]; ok {
		Dim = lipgloss.Color(v)
	}
	if v, ok := colors["white"]; ok {
		White = lipgloss.Color(v)
	}
}

func applySymbols(data []byte) {
	if data == nil {
		return
	}
	var symbols map[string]string
	if json.Unmarshal(data, &symbols) != nil {
		return
	}
	if v, ok := symbols["check"]; ok {
		SymbolCheck = v
	}
	if v, ok := symbols["cross"]; ok {
		SymbolCross = v
	}
	if v, ok := symbols["warning"]; ok {
		SymbolWarning = v
	}
	if v, ok := symbols["dot"]; ok {
		SymbolDot = v
	}
	if v, ok := symbols["arrow"]; ok {
		SymbolArrow = v
	}
}

func rebuildStyles() {
	SuccessStyle = lipgloss.NewStyle().Foreground(Green)
	ErrorStyle = lipgloss.NewStyle().Foreground(Red)
	WarningStyle = lipgloss.NewStyle().Foreground(Yellow)
	DimStyle = lipgloss.NewStyle().Foreground(Dim)
	BoldStyle = lipgloss.NewStyle().Bold(true)
	BlueStyle = lipgloss.NewStyle().Foreground(Blue)

	SuccessBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Blue).
		Padding(1, 2)

	ErrorBox = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Red).
		Padding(1, 2)

	TableHeaderStyle = lipgloss.NewStyle().
		Foreground(Blue).
		Bold(true)
}

// Common styles
var (
	SuccessStyle = lipgloss.NewStyle().Foreground(Green)
	ErrorStyle   = lipgloss.NewStyle().Foreground(Red)
	WarningStyle = lipgloss.NewStyle().Foreground(Yellow)
	DimStyle     = lipgloss.NewStyle().Foreground(Dim)
	BoldStyle    = lipgloss.NewStyle().Bold(true)
	BlueStyle    = lipgloss.NewStyle().Foreground(Blue)

	// Bordered boxes
	SuccessBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Blue).
			Padding(1, 2)

	ErrorBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Red).
			Padding(1, 2)

	// Table header
	TableHeaderStyle = lipgloss.NewStyle().
				Foreground(Blue).
				Bold(true)
)

func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
