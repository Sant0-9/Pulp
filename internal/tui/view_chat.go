package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Loading messages shown during connecting/thinking phase
var loadingMessages = []string{
	"Thinking...",
	"Processing...",
	"Contemplating...",
	"Pondering...",
	"Analyzing...",
	"Brewing thoughts...",
	"Gathering wisdom...",
	"Connecting neurons...",
}

// Spinner frames for animation
var spinnerFrames = []string{".", "o", "O", "o"}

func (a *App) renderChat() string {
	// Use full width with padding
	contentWidth := a.width - 4
	if contentWidth < 40 {
		contentWidth = 40
	}
	leftPad := 2

	// Fixed footer height: input line + status line = 2
	footerHeight := 2

	// Header is minimal: just 1 line for context info
	headerHeight := 1

	// Available height for messages
	availableHeight := a.height - headerHeight - footerHeight - 1 // -1 for spacing
	if availableHeight < 3 {
		availableHeight = 3
	}

	// === HEADER (minimal, like Claude Code) ===
	var headerParts []string
	modelName := a.getModelDisplayName()
	if modelName != "" {
		headerParts = append(headerParts, modelName)
	}
	if a.state.chatSkill != nil {
		headerParts = append(headerParts, fmt.Sprintf("[%s]", a.state.chatSkill.Name))
	}
	if a.state.contextLimit > 0 && a.state.contextUsed > 0 {
		pct := float64(a.state.contextUsed) / float64(a.state.contextLimit) * 100
		headerParts = append(headerParts, fmt.Sprintf("%.1fk/%.0fk ctx (%.0f%%)",
			float64(a.state.contextUsed)/1000,
			float64(a.state.contextLimit)/1000,
			pct))
	}

	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(strings.Join(headerParts, "  "))

	// === BUILD MESSAGE LINES ===
	var messageLines []string
	indent := strings.Repeat(" ", leftPad)

	// Show error if any
	if a.state.docError != nil {
		errLine := lipgloss.NewStyle().
			Foreground(colorError).
			Render("Error: " + a.state.docError.Error())
		messageLines = append(messageLines, indent+errLine, "")
	}

	for i, msg := range a.state.chatHistory {
		// Skip the last assistant message if streaming (shown separately)
		if a.state.chatStreaming && i == len(a.state.chatHistory)-1 && msg.role == "assistant" {
			continue
		}

		if msg.role == "user" {
			// User messages with ">" prefix
			content := wrapText(msg.content, contentWidth-4)
			lines := strings.Split(content, "\n")
			for j, line := range lines {
				prefix := "> "
				if j > 0 {
					prefix = "  "
				}
				styled := lipgloss.NewStyle().
					Foreground(colorSecondary).
					Bold(true).
					Render(prefix + line)
				messageLines = append(messageLines, indent+styled)
			}
		} else {
			// Assistant messages
			content := wrapText(msg.content, contentWidth-4)
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				styled := lipgloss.NewStyle().
					Foreground(colorWhite).
					Render("  " + line)
				messageLines = append(messageLines, indent+styled)
			}
		}
		messageLines = append(messageLines, "") // Blank line between messages
	}

	// Add streaming content
	if a.state.chatStreaming {
		if a.state.chatResult == "" {
			// Show animated loading message
			spinner := spinnerFrames[a.state.spinnerFrame%len(spinnerFrames)]
			elapsed := time.Since(a.state.streamStart).Seconds()
			msgIdx := int(elapsed*2) % len(loadingMessages)
			loadingText := lipgloss.NewStyle().
				Foreground(colorPrimary).
				Render(fmt.Sprintf("  %s %s", spinner, loadingMessages[msgIdx]))
			messageLines = append(messageLines, indent+loadingText)
		} else {
			// Show streaming response
			content := wrapText(a.state.chatResult, contentWidth-4)
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				styled := lipgloss.NewStyle().
					Foreground(colorWhite).
					Render("  " + line)
				messageLines = append(messageLines, indent+styled)
			}
			// Show cursor at end during streaming
			cursor := lipgloss.NewStyle().
				Foreground(colorPrimary).
				Render("_")
			if len(messageLines) > 0 {
				messageLines[len(messageLines)-1] += cursor
			}
		}
	}

	// === APPLY SCROLL ===
	totalLines := len(messageLines)

	// Clamp scroll offset
	maxScroll := totalLines - availableHeight
	if maxScroll < 0 {
		maxScroll = 0
	}
	if a.state.chatScrollOffset > maxScroll {
		a.state.chatScrollOffset = maxScroll
	}
	if a.state.chatScrollOffset < 0 {
		a.state.chatScrollOffset = 0
	}

	// Auto-scroll to bottom when streaming
	if a.state.chatAutoScroll {
		a.state.chatScrollOffset = 0
	}

	// Calculate visible range (scroll from bottom)
	endIdx := totalLines - a.state.chatScrollOffset
	startIdx := endIdx - availableHeight
	if startIdx < 0 {
		startIdx = 0
	}
	if endIdx > totalLines {
		endIdx = totalLines
	}

	// Get visible lines
	var visibleLines []string
	if startIdx < endIdx && len(messageLines) > 0 {
		visibleLines = messageLines[startIdx:endIdx]
	}

	// === BUILD FOOTER ===
	var footerLines []string

	// Input prompt (always visible, but shows streaming indicator when busy)
	prompt := lipgloss.NewStyle().
		Foreground(colorSecondary).
		Bold(true).
		Render("> ")

	if a.state.chatStreaming {
		// Show streaming indicator instead of input
		spinner := spinnerFrames[a.state.spinnerFrame%len(spinnerFrames)]
		streamingText := lipgloss.NewStyle().
			Foreground(colorMuted).
			Italic(true).
			Render(fmt.Sprintf("%s Streaming...", spinner))
		footerLines = append(footerLines, indent+prompt+streamingText)
	} else {
		inputStyle := lipgloss.NewStyle().
			Foreground(colorWhite)
		inputLine := indent + prompt + inputStyle.Render(a.state.input.View())
		footerLines = append(footerLines, inputLine)
	}

	// Status bar
	var statusParts []string
	if a.state.chatStreaming {
		statusParts = append(statusParts, a.buildStreamStatus())
		statusParts = append(statusParts, "[Esc] Cancel")
	} else {
		if a.state.chatScrollOffset > 0 {
			statusParts = append(statusParts, fmt.Sprintf("scroll: %d", a.state.chatScrollOffset))
		}
		statusParts = append(statusParts, "[Ctrl+U/D] Scroll  [Esc] Back")
	}

	statusLine := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(indent + strings.Join(statusParts, "  "))
	footerLines = append(footerLines, statusLine)

	// === COMBINE LAYOUT ===
	// Build message area
	var messageArea strings.Builder
	for i, line := range visibleLines {
		messageArea.WriteString(line)
		if i < len(visibleLines)-1 {
			messageArea.WriteString("\n")
		}
	}

	// Pad message area to fill available height (push footer to bottom)
	displayedLines := len(visibleLines)
	messagePadding := availableHeight - displayedLines
	for i := 0; i < messagePadding; i++ {
		if displayedLines > 0 || i > 0 {
			messageArea.WriteString("\n")
		}
	}

	// Combine all parts
	var output strings.Builder

	// Header
	output.WriteString(indent + header)
	output.WriteString("\n")

	// Messages (fills available space)
	output.WriteString(messageArea.String())
	output.WriteString("\n")

	// Footer (fixed at bottom)
	output.WriteString(strings.Join(footerLines, "\n"))

	return output.String()
}

// wrapText wraps text to fit within maxWidth, preserving words
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 60
	}

	var result strings.Builder
	lines := strings.Split(text, "\n")

	for lineIdx, line := range lines {
		if lineIdx > 0 {
			result.WriteString("\n")
		}

		if len(line) <= maxWidth {
			result.WriteString(line)
			continue
		}

		words := strings.Fields(line)
		lineLen := 0

		for i, word := range words {
			if i > 0 {
				if lineLen+1+len(word) > maxWidth {
					result.WriteString("\n")
					lineLen = 0
				} else {
					result.WriteString(" ")
					lineLen++
				}
			}
			result.WriteString(word)
			lineLen += len(word)
		}
	}

	return result.String()
}

// buildStreamStatus builds the dynamic status line during streaming
func (a *App) buildStreamStatus() string {
	var parts []string

	elapsed := time.Since(a.state.streamStart).Seconds()

	spinner := spinnerFrames[a.state.spinnerFrame%len(spinnerFrames)]
	switch a.state.streamPhase {
	case "connecting":
		msgIdx := int(elapsed*2) % len(loadingMessages)
		parts = append(parts, fmt.Sprintf("%s %s", spinner, loadingMessages[msgIdx]))
	case "streaming":
		if elapsed > 0 && a.state.streamTokens > 0 {
			tokPerSec := float64(a.state.streamTokens) / elapsed
			parts = append(parts, fmt.Sprintf("%s %.0f tok/s", spinner, tokPerSec))
		} else {
			parts = append(parts, fmt.Sprintf("%s Streaming...", spinner))
		}
	case "complete":
		if elapsed > 0 && a.state.streamTokens > 0 {
			tokPerSec := float64(a.state.streamTokens) / elapsed
			parts = append(parts, fmt.Sprintf("%d tokens (%.0f tok/s)", a.state.streamTokens, tokPerSec))
		} else {
			parts = append(parts, fmt.Sprintf("%d tokens", a.state.streamTokens))
		}
	default:
		parts = append(parts, spinner)
	}

	if a.state.contextLimit > 0 {
		pct := float64(a.state.contextUsed) / float64(a.state.contextLimit) * 100
		parts = append(parts, fmt.Sprintf("%.1fk/%.0fk (%.0f%%)",
			float64(a.state.contextUsed)/1000,
			float64(a.state.contextLimit)/1000,
			pct))
	}

	if elapsed > 0 {
		parts = append(parts, fmt.Sprintf("%.1fs", elapsed))
	}

	return strings.Join(parts, "  ")
}

// buildIdleStats shows abbreviated stats when not streaming
func (a *App) buildIdleStats() string {
	var parts []string

	if a.state.contextLimit > 0 && a.state.contextUsed > 0 {
		pct := float64(a.state.contextUsed) / float64(a.state.contextLimit) * 100
		parts = append(parts, fmt.Sprintf("%.1fk/%.0fk (%.0f%%)",
			float64(a.state.contextUsed)/1000,
			float64(a.state.contextLimit)/1000,
			pct))
	}

	if a.state.chatSkill != nil {
		parts = append(parts, fmt.Sprintf("[%s]", a.state.chatSkill.Name))
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "  ")
}

// getModelDisplayName returns a friendly model name for display
func (a *App) getModelDisplayName() string {
	if a.state.config == nil {
		return ""
	}
	model := a.state.config.Model
	provider := a.state.config.Provider

	displayModel := model
	switch {
	case strings.Contains(model, "claude-3-5-sonnet"):
		displayModel = "Claude 3.5 Sonnet"
	case strings.Contains(model, "claude-3-opus"):
		displayModel = "Claude 3 Opus"
	case strings.Contains(model, "claude-3-sonnet"):
		displayModel = "Claude 3 Sonnet"
	case strings.Contains(model, "claude-3-haiku"):
		displayModel = "Claude 3 Haiku"
	case strings.Contains(model, "gpt-4o"):
		displayModel = "GPT-4o"
	case strings.Contains(model, "gpt-4-turbo"):
		displayModel = "GPT-4 Turbo"
	case strings.Contains(model, "gpt-4"):
		displayModel = "GPT-4"
	case strings.Contains(model, "llama-3"):
		displayModel = "Llama 3"
	case strings.Contains(model, "mixtral"):
		displayModel = "Mixtral"
	case strings.Contains(model, "gemini"):
		displayModel = "Gemini"
	}

	if provider != "" && !strings.Contains(strings.ToLower(displayModel), strings.ToLower(provider)) {
		return fmt.Sprintf("%s via %s", displayModel, provider)
	}
	return displayModel
}
