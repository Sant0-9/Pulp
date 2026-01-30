package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderResult() string {
	var b strings.Builder

	// Document info (small)
	if a.state.document != nil {
		docInfo := styleSubtitle.Render(a.state.document.Metadata.Title)
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, docInfo))
		b.WriteString("\n\n")
	}

	// Result box
	result := a.state.result
	if result == "" && a.state.streaming {
		result = "..."
	}

	// Calculate max height for result
	maxResultHeight := a.height - 10
	if maxResultHeight < 5 {
		maxResultHeight = 5
	}
	resultLines := strings.Split(result, "\n")
	if len(resultLines) > maxResultHeight {
		// Show last N lines when streaming
		resultLines = resultLines[len(resultLines)-maxResultHeight:]
		result = strings.Join(resultLines, "\n")
	}

	resultStyle := styleBox.Copy().
		Width(min(70, a.width-4)).
		BorderForeground(colorPrimary)

	if a.state.streaming {
		resultStyle = resultStyle.BorderForeground(colorSecondary)
	}

	resultBox := resultStyle.Render(result)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, resultBox))
	b.WriteString("\n\n")

	// Status bar
	var status string
	if a.state.streaming {
		status = styleStatusBar.Render("Streaming... [Esc] Cancel")
	} else {
		status = styleStatusBar.Render("[c] Copy  [s] Save  [Enter] Follow-up  [n] New document  [Esc] Quit")
	}
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	return a.centerVertically(b.String())
}
