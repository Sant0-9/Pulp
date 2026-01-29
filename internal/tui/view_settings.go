package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sant0-9/pulp/internal/config"
)

func (a *App) renderSettings() string {
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
