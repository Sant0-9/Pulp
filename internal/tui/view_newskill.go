package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderNewSkill() string {
	var b strings.Builder

	// Header
	title := styleLogo.Render("Create New Skill")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Description
	desc := styleSubtitle.Render("Describe what the skill should do")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, desc))
	b.WriteString("\n\n")

	// Show generating status or input
	if a.state.generatingSkill {
		generating := styleBox.Copy().
			Width(min(70, a.width-4)).
			BorderForeground(colorSecondary).
			Render("Generating skill...")
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, generating))
	} else if a.state.newSkillError != nil {
		errorBox := styleBox.Copy().
			Width(min(70, a.width-4)).
			BorderForeground(colorError).
			Render("Error: " + a.state.newSkillError.Error())
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, errorBox))
		b.WriteString("\n\n")

		// Input box
		inputBox := styleBox.Copy().
			Width(min(70, a.width-4)).
			BorderForeground(colorMuted).
			Render(a.state.input.View())
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
	} else {
		// Input box
		inputBox := styleBox.Copy().
			Width(min(70, a.width-4)).
			BorderForeground(colorPrimary).
			Render(a.state.input.View())
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
	}
	b.WriteString("\n\n")

	// Examples
	if !a.state.generatingSkill {
		examples := styleSubtitle.Render("Examples: \"extract action items\" or \"summarize for executives\"")
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, examples))
		b.WriteString("\n\n")
	}

	// Status bar
	var status string
	if a.state.generatingSkill {
		status = styleStatusBar.Render("Generating...")
	} else {
		status = styleStatusBar.Render("[Enter] Create  [Esc] Cancel")
	}
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	return a.centerVertically(b.String())
}
