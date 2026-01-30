package tui

import "github.com/charmbracelet/lipgloss"

// truncate shortens text to maxLen, adding "..." if truncated
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

var (
	// Colors
	colorPrimary   = lipgloss.Color("#7C3AED")
	colorSecondary = lipgloss.Color("#06B6D4")
	colorSuccess   = lipgloss.Color("#10B981")
	colorError     = lipgloss.Color("#EF4444")
	colorMuted     = lipgloss.Color("#6B7280")
	colorWhite     = lipgloss.Color("#F9FAFB")
	colorDark      = lipgloss.Color("#1F2937")

	// Logo style
	styleLogo = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	// Subtitle
	styleSubtitle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Box
	styleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)

	// Status bar
	styleStatusBar = lipgloss.NewStyle().
			Foreground(colorMuted)

	// Centered container
	styleCenter = lipgloss.NewStyle().
			Align(lipgloss.Center)
)
