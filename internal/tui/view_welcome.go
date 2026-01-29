package tui

import "github.com/charmbracelet/lipgloss"

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

	// Instructions
	instructions := styleSubtitle.Render("\nDrop a file or type a path to get started")

	// Status bar
	statusBar := styleStatusBar.Render("[Esc] Quit  [?] Help")

	// Combine main content
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logoRendered,
		subtitle,
		instructions,
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
