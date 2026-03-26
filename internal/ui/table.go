package ui

import (
	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
)

func NewTable(cols []table.Column, rows []table.Row) table.Model {
	s := table.DefaultStyles()
	s.Header = lipgloss.NewStyle().
		Foreground(Blue).
		Bold(true).
		Padding(0, 1)
	s.Cell = lipgloss.NewStyle().
		Padding(0, 1)
	s.Selected = lipgloss.NewStyle().
		Foreground(White).
		Background(Blue).
		Padding(0, 1)

	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithStyles(s),
		table.WithHeight(len(rows)+1),
	)

	return t
}
