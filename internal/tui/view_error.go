package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderError() string {
	var b strings.Builder

	// Error icon and title
	title := lipgloss.NewStyle().
		Foreground(colorError).
		Bold(true).
		Render("Something went wrong")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Error message
	errMsg := "Unknown error"
	if a.state.processingError != nil {
		errMsg = a.state.processingError.Error()
	} else if a.state.providerError != nil {
		errMsg = a.state.providerError.Error()
	} else if a.state.docError != nil {
		errMsg = a.state.docError.Error()
	}

	errBox := styleBox.Copy().
		Width(min(60, a.width-4)).
		BorderForeground(colorError).
		Render(errMsg)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, errBox))
	b.WriteString("\n\n")

	// Suggestions based on error type
	var suggestions []string
	errLower := strings.ToLower(errMsg)

	if strings.Contains(errLower, "api key") || strings.Contains(errLower, "401") || strings.Contains(errLower, "unauthorized") {
		suggestions = append(suggestions, "Check your API key in ~/.config/pulp/config.yaml")
		suggestions = append(suggestions, "Or press [s] to open settings")
	} else if strings.Contains(errLower, "connection") || strings.Contains(errLower, "connect") || strings.Contains(errLower, "timeout") {
		suggestions = append(suggestions, "Check your internet connection")
		suggestions = append(suggestions, "Or try using Ollama for offline mode")
	} else if strings.Contains(errLower, "ollama") {
		suggestions = append(suggestions, "Make sure Ollama is running: ollama serve")
		suggestions = append(suggestions, "Or switch to a cloud provider in settings")
	} else if strings.Contains(errLower, "not found") || strings.Contains(errLower, "no such file") {
		suggestions = append(suggestions, "Check the file path is correct")
		suggestions = append(suggestions, "Make sure the file exists and is readable")
	} else if strings.Contains(errLower, "docling") || strings.Contains(errLower, "python") {
		suggestions = append(suggestions, "Make sure Python and Docling are installed:")
		suggestions = append(suggestions, "  pip install docling")
	} else if strings.Contains(errLower, "rate limit") || strings.Contains(errLower, "429") {
		suggestions = append(suggestions, "You've hit the API rate limit")
		suggestions = append(suggestions, "Wait a moment and try again")
	}

	if len(suggestions) > 0 {
		suggBox := styleBox.Copy().
			Width(min(60, a.width-4)).
			BorderForeground(colorMuted).
			Render("Suggestions:\n" + strings.Join(suggestions, "\n"))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, suggBox))
		b.WriteString("\n\n")
	}

	// Actions
	status := styleStatusBar.Render("[r] Retry  [s] Settings  [n] New  [Esc] Back")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	return a.centerVertically(b.String())
}
