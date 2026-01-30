package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const logo = `
 ██████╗ ██╗   ██╗██╗     ██████╗
 ██╔══██╗██║   ██║██║     ██╔══██╗
 ██████╔╝██║   ██║██║     ██████╔╝
 ██╔═══╝ ██║   ██║██║     ██╔═══╝
 ██║     ╚██████╔╝███████╗██║
 ╚═╝      ╚═════╝ ╚══════╝╚═╝
`

func (a *App) renderWelcome() string {
	// Logo
	logoRendered := styleLogo.Render(logo)

	// Subtitle
	subtitle := styleSubtitle.Render("Document Intelligence")

	// Provider status
	var status string
	if a.state.loadingDoc {
		status = styleSubtitle.Render("Loading document...")
	} else if a.state.docError != nil {
		status = lipgloss.NewStyle().
			Foreground(colorError).
			Render(fmt.Sprintf("Error: %s", a.state.docError))
	} else if a.state.providerError != nil {
		errorLine := lipgloss.NewStyle().
			Foreground(colorError).
			Render(fmt.Sprintf("Provider error: %s", a.state.providerError))
		hint := lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1).
			Render("Press [s] for settings to fix")
		status = lipgloss.JoinVertical(lipgloss.Center, errorLine, hint)
	} else if a.state.providerReady {
		providerName := a.state.config.Provider
		status = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Render(fmt.Sprintf("Connected to %s", providerName))
	} else {
		status = styleSubtitle.Render("Connecting...")
	}

	// Input (only show if ready)
	var inputSection string
	if a.state.providerReady {
		inputBox := styleBox.Copy().
			Width(60).
			BorderForeground(colorSecondary).
			Render(a.state.input.View())

		// Show command palette when active
		if a.state.cmdPaletteActive && len(a.state.cmdPaletteItems) > 0 {
			palette := a.renderCommandPalette()
			inputSection = lipgloss.JoinVertical(
				lipgloss.Center,
				inputBox,
				palette,
			)
		} else {
			inputSection = inputBox
		}
	}

	// Status bar
	statusBar := styleStatusBar.Render("[s] Settings  [?] Help  [Esc] Quit")

	// Combine main content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logoRendered,
		subtitle,
		"",
		status,
		"",
		inputSection,
	)

	// Center content on screen (leave room for status bar)
	mainArea := lipgloss.Place(
		a.width,
		a.height-2,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	// Status bar centered at bottom
	statusLine := lipgloss.PlaceHorizontal(a.width, lipgloss.Center, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, mainArea, statusLine)
}

func (a *App) renderCommandPalette() string {
	items := a.state.cmdPaletteItems
	selected := a.state.cmdPaletteSelected

	// Limit to 8 visible items
	maxVisible := 8
	if len(items) > maxVisible {
		items = items[:maxVisible]
	}

	// Styles
	cmdStyle := lipgloss.NewStyle().Foreground(colorPrimary).Bold(true)
	descStyle := lipgloss.NewStyle().Foreground(colorMuted)
	selectedBg := lipgloss.NewStyle().Background(lipgloss.Color("237"))

	var lines []string
	for i, item := range items {
		cmd := cmdStyle.Render(item.cmd)
		desc := descStyle.Render(item.desc)
		line := fmt.Sprintf("  %s  %s", cmd, desc)

		if i == selected {
			// Highlight selected row
			line = selectedBg.Render(line)
		}

		lines = append(lines, line)
	}

	// Add hint at bottom
	hint := lipgloss.NewStyle().
		Foreground(colorMuted).
		Italic(true).
		Render("  [Up/Down] Navigate  [Tab] Complete  [Enter] Select")
	lines = append(lines, "", hint)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorSecondary).
		Padding(0, 1).
		MarginTop(1).
		Render(strings.Join(lines, "\n"))
}
