package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (a *App) renderDocument() string {
	if a.state.document == nil {
		return a.renderWelcome()
	}

	var b strings.Builder
	doc := a.state.document
	meta := doc.Metadata

	// Document info header
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(meta.Title)

	// Metadata line
	var metaParts []string
	if meta.PageCount != nil {
		metaParts = append(metaParts, fmt.Sprintf("%d pages", *meta.PageCount))
	}
	metaParts = append(metaParts, strings.ToUpper(meta.SourceFormat))
	metaParts = append(metaParts, meta.FileSizeHuman())
	metaParts = append(metaParts, fmt.Sprintf("~%d words", meta.WordCount))

	metaLine := styleSubtitle.Render(strings.Join(metaParts, "  |  "))

	// Document info box
	infoContent := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		metaLine,
	)
	infoBox := styleBox.Copy().
		Width(min(70, a.width-4)).
		BorderForeground(colorSuccess).
		Render(infoContent)

	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, infoBox))
	b.WriteString("\n\n")

	// Preview
	previewLabel := styleSubtitle.Render("Preview:")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, previewLabel))
	b.WriteString("\n")

	previewBox := styleBox.Copy().
		Width(min(70, a.width-4)).
		Foreground(colorMuted).
		Render(doc.Preview)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, previewBox))
	b.WriteString("\n\n")

	// Instruction prompt
	promptLabel := lipgloss.NewStyle().
		Foreground(colorWhite).
		Render("What do you want to do with this document?")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, promptLabel))
	b.WriteString("\n\n")

	// Input
	inputBox := styleBox.Copy().
		Width(min(70, a.width-4)).
		BorderForeground(colorSecondary).
		Render(a.state.input.View())
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
	b.WriteString("\n\n")

	// Show parsing status or parsed intent
	if a.state.parsingIntent {
		parsingLabel := styleSubtitle.Render("Parsing instruction...")
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, parsingLabel))
		b.WriteString("\n\n")
	} else if a.state.currentIntent != nil {
		i := a.state.currentIntent
		intentInfo := fmt.Sprintf(
			"Parsed: action=%s, tone=%s, audience=%s, format=%s",
			i.Action, i.Tone, i.Audience, i.Format,
		)
		intentBox := styleBox.Copy().
			Width(min(70, a.width-4)).
			Foreground(colorSecondary).
			Render(intentInfo)
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, intentBox))
		b.WriteString("\n\n")
	}

	// Status bar
	statusBar := styleStatusBar.Render("[Enter] Submit  [n] New document  [Esc] Quit")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, statusBar))

	return a.centerVertically(b.String())
}
