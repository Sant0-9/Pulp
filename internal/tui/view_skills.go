package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderSkills() string {
	var b strings.Builder

	// Header
	title := styleLogo.Render("Available Skills")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Description
	desc := styleSubtitle.Render("Skills provide specialized instructions for document processing")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, desc))
	b.WriteString("\n\n")

	// List skills
	if a.state.skillIndex == nil || a.state.skillIndex.Count() == 0 {
		noSkills := styleBox.Copy().
			Width(min(70, a.width-4)).
			Foreground(colorMuted).
			Render("No skills installed.\n\nCreate skills in: ~/.config/pulp/skills/\n\nEach skill is a folder with SKILL.md")
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, noSkills))
	} else {
		var skillList strings.Builder
		for _, meta := range a.state.skillIndex.GetAll() {
			skillList.WriteString(fmt.Sprintf("/%s\n", meta.Name))
			if meta.Description != "" {
				// Truncate long descriptions
				desc := meta.Description
				if len(desc) > 60 {
					desc = desc[:57] + "..."
				}
				skillList.WriteString(fmt.Sprintf("  %s\n", desc))
			}
			skillList.WriteString("\n")
		}

		skillBox := styleBox.Copy().
			Width(min(70, a.width-4)).
			BorderForeground(colorPrimary).
			Render(strings.TrimSpace(skillList.String()))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, skillBox))
	}
	b.WriteString("\n\n")

	// Usage hint
	usage := styleSubtitle.Render("Use /skill-name to invoke a skill, or let pulp auto-match")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, usage))
	b.WriteString("\n\n")

	// Status bar
	statusBar := styleStatusBar.Render("[Esc] Back")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, statusBar))

	return a.centerVertically(b.String())
}
