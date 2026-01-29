package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	ColorPrimary   = lipgloss.Color("#7C3AED")
	ColorSecondary = lipgloss.Color("#06B6D4")
	ColorSuccess   = lipgloss.Color("#10B981")
	ColorError     = lipgloss.Color("#EF4444")
	ColorMuted     = lipgloss.Color("#6B7280")
	ColorWhite     = lipgloss.Color("#F9FAFB")
	ColorDark      = lipgloss.Color("#1F2937")

	// Logo style
	Logo = lipgloss.NewStyle().
		Foreground(ColorPrimary).
		Bold(true)

	// Subtitle
	Subtitle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Box
	Box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorMuted).
		Padding(0, 1)

	// Status bar
	StatusBar = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Centered container
	Center = lipgloss.NewStyle().
		Align(lipgloss.Center)
)
