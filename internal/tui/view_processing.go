package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderProcessing() string {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Processing")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Document info
	if a.state.document != nil {
		docInfo := styleSubtitle.Render(truncate(a.state.document.Metadata.Title, 60))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, docInfo))
		b.WriteString("\n\n")
	}

	// Intent info
	if a.state.currentIntent != nil {
		intentInfo := styleSubtitle.Render("> " + truncate(a.state.currentIntent.RawPrompt, 55))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, intentInfo))
		b.WriteString("\n\n")
	}

	// Progress stages
	stages := []string{"Chunking", "Extracting", "Aggregating"}
	currentStage := 0
	if a.state.pipelineProgress != nil {
		currentStage = a.state.pipelineProgress.StageIndex
	}

	var stageLines []string
	for i, stage := range stages {
		var icon string
		var style lipgloss.Style

		if i < currentStage {
			// Completed
			icon = "[x]"
			style = lipgloss.NewStyle().Foreground(colorSuccess)
		} else if i == currentStage {
			// Current
			icon = "[>]"
			style = lipgloss.NewStyle().Foreground(colorSecondary).Bold(true)
		} else {
			// Pending
			icon = "[ ]"
			style = lipgloss.NewStyle().Foreground(colorMuted)
		}

		// Progress bar for extraction
		var progressBar string
		if i == currentStage && a.state.pipelineProgress != nil {
			p := a.state.pipelineProgress
			if p.TotalItems > 0 {
				pct := float64(p.ItemIndex) / float64(p.TotalItems)
				filled := int(pct * 30)
				empty := 30 - filled
				progressBar = "  " +
					lipgloss.NewStyle().Foreground(colorSecondary).Render(strings.Repeat("=", filled)) +
					lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("-", empty)) +
					fmt.Sprintf("  %d/%d", p.ItemIndex, p.TotalItems)
			}
		}

		line := style.Render(fmt.Sprintf("  %s  %-12s", icon, stage)) + progressBar
		stageLines = append(stageLines, line)
	}

	stagesBox := styleBox.Copy().
		Width(min(60, a.width-4)).
		Render(strings.Join(stageLines, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, stagesBox))
	b.WriteString("\n\n")

	// Message
	if a.state.pipelineProgress != nil && a.state.pipelineProgress.Message != "" {
		msg := styleSubtitle.Render(truncate(a.state.pipelineProgress.Message, 60))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, msg))
	}

	return a.centerVertically(b.String())
}
