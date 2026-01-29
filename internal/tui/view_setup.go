package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sant0-9/pulp/internal/config"
)

func (a *App) renderSetup() string {
	switch a.state.setupStep {
	case 0:
		return a.renderProviderSelection()
	case 1:
		return a.renderAPIKeyEntry()
	default:
		return ""
	}
}

func (a *App) renderProviderSelection() string {
	var b strings.Builder

	// Header
	header := styleLogo.Render(logo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, header))
	b.WriteString("\n\n")

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorWhite).
		Bold(true).
		Render("Welcome! Choose your LLM provider:")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Provider list
	var providerLines []string
	for i, p := range config.Providers {
		var line string
		cursor := "  "
		if i == a.state.selectedProvider {
			cursor = "> "
			line = lipgloss.NewStyle().
				Foreground(colorSecondary).
				Bold(true).
				Render(fmt.Sprintf("%s[x] %-12s %s", cursor, p.Name, p.Description))
		} else {
			line = lipgloss.NewStyle().
				Foreground(colorMuted).
				Render(fmt.Sprintf("%s[ ] %-12s %s", cursor, p.Name, p.Description))
		}
		providerLines = append(providerLines, line)
	}

	providerBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(providerLines, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, providerBox))
	b.WriteString("\n\n")

	// Instructions
	instructions := styleStatusBar.Render("[j/k] Navigate  [Enter] Select")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}

func (a *App) renderAPIKeyEntry() string {
	var b strings.Builder

	provider := config.GetProvider(a.state.config.Provider)

	// Header
	header := styleLogo.Render(logo)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, header))
	b.WriteString("\n\n")

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorWhite).
		Bold(true).
		Render(fmt.Sprintf("Enter your %s API key:", provider.Name))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Signup link
	if provider.SignupURL != "" {
		link := styleSubtitle.Render(fmt.Sprintf("Get one at: %s", provider.SignupURL))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, link))
		b.WriteString("\n\n")
	}

	// Input
	inputBox := styleBox.Copy().
		Width(60).
		BorderForeground(colorSecondary).
		Render(a.state.apiKeyInput.View())
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
	b.WriteString("\n\n")

	// Instructions
	instructions := styleStatusBar.Render("[Enter] Continue  [Esc] Back")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}

func (a *App) centerVertically(content string) string {
	lines := strings.Count(content, "\n") + 1
	padding := (a.height - lines) / 2
	if padding < 0 {
		padding = 0
	}
	return strings.Repeat("\n", padding) + content
}
