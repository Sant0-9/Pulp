package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sant0-9/pulp/internal/config"
)

func (a *App) renderSettings() string {
	switch a.state.settingsMode {
	case "provider":
		return a.renderSettingsProvider()
	case "model":
		return a.renderSettingsModel()
	case "apikey":
		return a.renderSettingsAPIKey()
	default:
		return a.renderSettingsMain()
	}
}

func (a *App) renderSettingsMain() string {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Settings")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Current config
	provider := config.GetProvider(a.state.config.Provider)
	providerName := a.state.config.Provider
	if provider != nil {
		providerName = provider.Name
	}

	// Mask API key
	maskedKey := "Not set"
	if a.state.config.APIKey != "" {
		if len(a.state.config.APIKey) > 8 {
			maskedKey = a.state.config.APIKey[:4] + "****" + a.state.config.APIKey[len(a.state.config.APIKey)-4:]
		} else {
			maskedKey = "****"
		}
	}

	configLines := []string{
		fmt.Sprintf("  Provider: %s", providerName),
		fmt.Sprintf("  Model:    %s", a.state.config.Model),
		fmt.Sprintf("  API Key:  %s", maskedKey),
	}

	if a.state.config.Local != nil && a.state.config.Local.Enabled {
		configLines = append(configLines, "")
		configLines = append(configLines, "  Local Model:")
		configLines = append(configLines, fmt.Sprintf("    Provider: %s", a.state.config.Local.Provider))
		configLines = append(configLines, fmt.Sprintf("    Model:    %s", a.state.config.Local.Model))
	}

	configBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(configLines, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, configBox))
	b.WriteString("\n\n")

	// Actions
	actions := []string{
		"  [p] Change provider",
		"  [m] Change model",
		"  [k] Update API key",
		"  [r] Reset setup",
	}
	actionsBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(actions, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, actionsBox))
	b.WriteString("\n\n")

	// Instructions
	instructions := styleStatusBar.Render("[Esc] Back")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}

func (a *App) renderSettingsProvider() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Select Provider")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	var lines []string
	for i, p := range config.Providers {
		cursor := "  "
		if i == a.state.settingsSelected {
			cursor = "> "
		}
		line := fmt.Sprintf("%s%s", cursor, p.Name)
		if i == a.state.settingsSelected {
			line = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(line)
		}
		lines = append(lines, line)
	}

	listBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(lines, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, listBox))
	b.WriteString("\n\n")

	instructions := styleStatusBar.Render("[Up/Down] Navigate  [Enter] Select  [Esc] Cancel")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}

func (a *App) renderSettingsModel() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Select Model")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	provider := config.GetProvider(a.state.config.Provider)
	if provider == nil {
		desc := styleSubtitle.Render("No provider selected")
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, desc))
		return a.centerVertically(b.String())
	}

	providerDesc := styleSubtitle.Render(fmt.Sprintf("Provider: %s", provider.Name))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, providerDesc))
	b.WriteString("\n\n")

	var lines []string
	for i, model := range provider.Models {
		cursor := "  "
		if i == a.state.settingsSelected {
			cursor = "> "
		}
		// Mark current model
		current := ""
		if model == a.state.config.Model {
			current = " (current)"
		}
		line := fmt.Sprintf("%s%s%s", cursor, model, current)
		if i == a.state.settingsSelected {
			line = lipgloss.NewStyle().Foreground(colorPrimary).Bold(true).Render(line)
		}
		lines = append(lines, line)
	}

	listBox := styleBox.Copy().
		Width(50).
		Render(strings.Join(lines, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, listBox))
	b.WriteString("\n\n")

	instructions := styleStatusBar.Render("[Up/Down] Navigate  [Enter] Select  [Esc] Cancel")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}

func (a *App) renderSettingsAPIKey() string {
	var b strings.Builder

	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Update API Key")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	desc := styleSubtitle.Render("Enter your new API key")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, desc))
	b.WriteString("\n\n")

	inputBox := styleBox.Copy().
		Width(50).
		BorderForeground(colorPrimary).
		Render(a.state.apiKeyInput.View())
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
	b.WriteString("\n\n")

	instructions := styleStatusBar.Render("[Enter] Save  [Esc] Cancel")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions))

	return a.centerVertically(b.String())
}
