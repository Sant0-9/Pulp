package tui

import (
	"fmt"

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
		status = lipgloss.NewStyle().
			Foreground(colorError).
			Render(fmt.Sprintf("Provider error: %s", a.state.providerError))
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
		inputSection = styleBox.Copy().
			Width(60).
			BorderForeground(colorSecondary).
			Render(a.state.input.View())
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
