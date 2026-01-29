package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderHelp() string {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Help")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Commands
	commands := []string{
		"  /help, /h      Show this help",
		"  /settings, /s  Open settings",
		"  /quit, /q      Quit pulp",
		"",
		"  Or drop a file path to process a document",
	}

	commandsBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(commands, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, commandsBox))
	b.WriteString("\n\n")

	// Keyboard shortcuts
	shortcuts := []string{
		"  Esc            Go back / Quit",
		"  Enter          Submit input",
		"  s              Quick settings (from welcome)",
	}

	shortcutsTitle := styleSubtitle.Render("Keyboard Shortcuts")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, shortcutsTitle))
	b.WriteString("\n\n")

	shortcutsBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(shortcuts, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, shortcutsBox))
	b.WriteString("\n\n")

	// Instructions
	instructions := styleStatusBar.Render("[Esc] Back")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}
