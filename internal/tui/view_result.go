package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderResult() string {
	var b strings.Builder

	// Document info (small)
	if a.state.document != nil {
		docInfo := styleSubtitle.Render(a.state.document.Metadata.Title)
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, docInfo))
		b.WriteString("\n")
	}

	// Show what was asked (user message)
	if a.state.currentIntent != nil {
		asked := styleSubtitle.Render(fmt.Sprintf("> %s", a.state.currentIntent.RawPrompt))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, asked))
		b.WriteString("\n\n")
	}

	// Result box
	result := a.state.result
	if result == "" && a.state.streaming {
		result = "..."
	}

	// Calculate max height for result (account for input box when not streaming)
	maxResultHeight := a.height - 14
	if a.state.streaming {
		maxResultHeight = a.height - 10
	}
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

	// Input for follow-up (only show when not streaming)
	if !a.state.streaming {
		a.state.input.Placeholder = "Follow-up or revision..."
		inputBox := styleBox.Copy().
			Width(min(70, a.width-4)).
			BorderForeground(colorMuted).
			Render(a.state.input.View())
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
		b.WriteString("\n\n")
	}

	// Status bar
	var status string
	if a.state.streaming {
		status = styleStatusBar.Render("Streaming... [Esc] Cancel")
	} else {
		status = styleStatusBar.Render("[Enter] Submit  [c] Copy  [s] Save  [n] New document  [Esc] Quit")
	}
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	return a.centerVertically(b.String())
}
