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
var spinnerFrames = []string{"*", "*", "*", "*"}

func (a *App) renderChat() string {
	boxWidth := min(70, a.width-4)
	leftPad := (a.width - boxWidth) / 2
	if leftPad < 2 {
		leftPad = 2
	}
	indent := strings.Repeat(" ", leftPad)

	// Calculate fixed heights
	headerHeight := 3 // Title + Model + blank line
	inputHeight := 4  // Input box + status bar
	if a.state.chatStreaming {
		inputHeight = 2 // Just status bar when streaming
	}

	// Available height for messages
	availableHeight := a.height - headerHeight - inputHeight
	if availableHeight < 5 {
		availableHeight = 5
	}

	// === BUILD HEADER ===
	var header strings.Builder
	titleText := "Chat"
	if a.state.chatSkill != nil {
		titleText = "Chat [" + a.state.chatSkill.Name + "]"
	}
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(titleText)
	header.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	header.WriteString("\n")

	modelInfo := a.getModelDisplayName()
	modelLine := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(modelInfo)
	header.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, modelLine))
	header.WriteString("\n\n")

	// === BUILD ALL MESSAGE LINES ===
	var messageLines []string

	for i, msg := range a.state.chatHistory {
		// Skip the last assistant message if streaming (shown separately)
		if a.state.chatStreaming && i == len(a.state.chatHistory)-1 && msg.role == "assistant" {
			continue
		}

		if msg.role == "user" {
			content := wrapText(msg.content, boxWidth-4)
			lines := strings.Split(content, "\n")
			for j, line := range lines {
				prefix := "> "
				if j > 0 {
					prefix = "  "
				}
				styled := lipgloss.NewStyle().
					Foreground(colorSecondary).
					Render(prefix + line)
				messageLines = append(messageLines, indent+styled)
			}
		} else {
			content := wrapText(msg.content, boxWidth-4)
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
			// Show loading message
			spinner := spinnerFrames[a.state.spinnerFrame%len(spinnerFrames)]
			elapsed := time.Since(a.state.streamStart).Seconds()
			msgIdx := int(elapsed*2) % len(loadingMessages)
			loadingText := lipgloss.NewStyle().
				Foreground(colorPrimary).
				Render(fmt.Sprintf("%s %s", spinner, loadingMessages[msgIdx]))
			messageLines = append(messageLines, indent+loadingText)
		} else {
			// Show streaming response
			content := wrapText(a.state.chatResult, boxWidth-4)
			lines := strings.Split(content, "\n")
			for _, line := range lines {
				styled := lipgloss.NewStyle().
					Foreground(colorWhite).
					Render("  " + line)
				messageLines = append(messageLines, indent+styled)
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

	// === BUILD INPUT/STATUS ===
	var footer strings.Builder

	if !a.state.chatStreaming {
		if a.state.chatSkill != nil {
			a.state.input.Placeholder = "Chat with " + a.state.chatSkill.Name + " skill..."
		} else {
			a.state.input.Placeholder = "Continue the conversation..."
		}
		inputBox := styleBox.Copy().
			Width(boxWidth).
			BorderForeground(colorMuted).
			Render(a.state.input.View())
		footer.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
		footer.WriteString("\n")
	}

	// Status bar with scroll indicator
	var status string
	if a.state.chatStreaming {
		streamStatus := a.buildStreamStatus()
		a.state.lastStats = streamStatus
		status = styleStatusBar.Render(streamStatus + "  [Esc] Cancel")
	} else {
		var statusParts []string

		// Scroll indicator
		if a.state.chatScrollOffset > 0 {
			statusParts = append(statusParts, fmt.Sprintf("[scroll: %d]", a.state.chatScrollOffset))
		}

		if a.state.lastStats != "" && len(a.state.chatHistory) > 0 {
			statusParts = append(statusParts, a.buildIdleStats())
		}
		statusParts = append(statusParts, "[j/k] Scroll  [n] New  [Esc] Back")
		status = styleStatusBar.Render(strings.Join(statusParts, "  "))
	}
	footer.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	// === COMBINE WITH FIXED LAYOUT ===
	headerContent := header.String()
	footerContent := footer.String()

	// Build message area with exact height
	var messageArea strings.Builder
	for i, line := range visibleLines {
		messageArea.WriteString(line)
		if i < len(visibleLines)-1 {
			messageArea.WriteString("\n")
		}
	}

	// Pad message area to fill available height
	displayedLines := len(visibleLines)
	messagePadding := availableHeight - displayedLines
	if messagePadding > 0 {
		if displayedLines > 0 {
			messageArea.WriteString("\n")
		}
		messageArea.WriteString(strings.Repeat("\n", messagePadding-1))
	}

	// Combine: header + messages (fixed height) + footer
	return headerContent + messageArea.String() + "\n" + footerContent
}

// wrapText wraps text to fit within maxWidth, preserving words
func wrapText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		maxWidth = 60
	}
	if len(text) <= maxWidth {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
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
		parts = append(parts, "...")
	}

	if a.state.contextLimit > 0 {
		pct := float64(a.state.contextUsed) / float64(a.state.contextLimit) * 100
		parts = append(parts, fmt.Sprintf("%.1fk/%.0fk ctx (%.0f%%)",
			float64(a.state.contextUsed)/1000,
			float64(a.state.contextLimit)/1000,
			pct))
	}

	if elapsed > 0 {
		parts = append(parts, fmt.Sprintf("%.1fs", elapsed))
	}

	if a.state.chatSkill != nil {
		parts = append(parts, fmt.Sprintf("[%s]", a.state.chatSkill.Name))
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
