package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderResult() string {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Result")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Show result info (placeholder for Phase 7)
	if a.state.pipelineResult != nil && a.state.pipelineResult.Aggregated != nil {
		agg := a.state.pipelineResult.Aggregated

		// Summary of what was extracted
		summary := styleSubtitle.Render("Extraction complete")
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, summary))
		b.WriteString("\n\n")

		// Stats
		var stats []string
		if len(agg.KeyPoints) > 0 {
			stats = append(stats, lipgloss.NewStyle().Foreground(colorSuccess).Render(
				string(rune(len(agg.KeyPoints)))+" key points"))
		}
		if len(agg.Facts) > 0 {
			stats = append(stats, lipgloss.NewStyle().Foreground(colorSuccess).Render(
				string(rune(len(agg.Facts)))+" facts"))
		}
		if len(agg.Entities) > 0 {
			stats = append(stats, lipgloss.NewStyle().Foreground(colorSuccess).Render(
				string(rune(len(agg.Entities)))+" entities"))
		}

		// Show aggregated content preview
		content := agg.FormatForWriter()
		if len(content) > 500 {
			content = content[:500] + "..."
		}

		contentBox := styleBox.Copy().
			Width(min(70, a.width-4)).
			Foreground(colorMuted).
			Render(content)
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, contentBox))
		b.WriteString("\n\n")
	}

	// Status bar
	statusBar := styleStatusBar.Render("[n] New document  [Esc] Quit")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, statusBar))

	return a.centerVertically(b.String())
}
