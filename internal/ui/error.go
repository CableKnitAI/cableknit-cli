package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/jessewaites/cableknit-cli/internal/api"
)

func RenderValidationErrors(errors []api.ValidationError) string {
	if len(errors) == 0 {
		return ""
	}

	var sb strings.Builder
	// Group errors by file
	byFile := make(map[string][]api.ValidationError)
	var fileOrder []string
	for _, e := range errors {
		if _, seen := byFile[e.File]; !seen {
			fileOrder = append(fileOrder, e.File)
		}
		byFile[e.File] = append(byFile[e.File], e)
	}

	for _, file := range fileOrder {
		errs := byFile[file]
		var lines []string
		for _, e := range errs {
			loc := ""
			if e.Line > 0 {
				loc = fmt.Sprintf(" (line %d", e.Line)
				if e.Column > 0 {
					loc += fmt.Sprintf(":%d", e.Column)
				}
				loc += ")"
			}
			lines = append(lines, fmt.Sprintf("%s %s%s", SymbolCross, e.Message, loc))
		}

		title := lipgloss.NewStyle().Bold(true).Foreground(Red).Render(file)
		box := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Red).
			Padding(0, 1).
			Render(strings.Join(lines, "\n"))

		sb.WriteString(title + "\n" + box + "\n\n")
	}

	return sb.String()
}

func RenderWarnings(warnings []api.ValidationError) string {
	if len(warnings) == 0 {
		return ""
	}

	var lines []string
	for _, w := range warnings {
		loc := ""
		if w.File != "" {
			loc = w.File
			if w.Line > 0 {
				loc += fmt.Sprintf(":%d", w.Line)
			}
			loc += " "
		}
		lines = append(lines, WarningStyle.Render(fmt.Sprintf("%s %s%s", SymbolWarning, loc, w.Message)))
	}
	return strings.Join(lines, "\n") + "\n"
}
