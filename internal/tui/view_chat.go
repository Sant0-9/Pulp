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

	// === TOP SECTION: Title + Model + History + Streaming ===
	var top strings.Builder

	// Title with optional skill indicator
	titleText := "Chat"
	if a.state.chatSkill != nil {
		titleText = "Chat [" + a.state.chatSkill.Name + "]"
	}
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(titleText)
	top.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	top.WriteString("\n")

	// Model info line
	modelInfo := a.getModelDisplayName()
	modelLine := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(modelInfo)
	top.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, modelLine))
	top.WriteString("\n")

	// Build conversation history
	leftPad := (a.width - boxWidth) / 2
	if leftPad < 2 {
		leftPad = 2
	}
	indent := strings.Repeat(" ", leftPad)

	for i, msg := range a.state.chatHistory {
		// Skip the last assistant message if streaming (shown in stream box)
		if a.state.chatStreaming && i == len(a.state.chatHistory)-1 && msg.role == "assistant" {
			continue
		}

		if msg.role == "user" {
			// User message with prompt indicator
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
				top.WriteString(indent + styled + "\n")
			}
		} else {
			// Assistant response with bullet indicator
			content := wrapText(msg.content, boxWidth-4)
			lines := strings.Split(content, "\n")
			for j, line := range lines {
				prefix := "  "
				if j == 0 {
					prefix = "  " // Indent to align with user messages
				}
				styled := lipgloss.NewStyle().
					Foreground(colorWhite).
					Render(prefix + line)
				top.WriteString(indent + styled + "\n")
			}
		}
		top.WriteString("\n") // Space between messages
	}

	// Streaming response box
	if a.state.chatStreaming {
		currentResponse := a.state.chatResult

		if currentResponse == "" {
			// Show animated loading message
			spinner := spinnerFrames[a.state.spinnerFrame%len(spinnerFrames)]
			elapsed := time.Since(a.state.streamStart).Seconds()
			msgIdx := int(elapsed*2) % len(loadingMessages)
			loadingText := lipgloss.NewStyle().
				Foreground(colorPrimary).
				Render(fmt.Sprintf("%s %s", spinner, loadingMessages[msgIdx]))
			top.WriteString(indent + loadingText + "\n")
		} else {
			// Show streaming response
			resultLines := strings.Split(currentResponse, "\n")
			maxResultLines := 15
			if len(resultLines) > maxResultLines {
				resultLines = resultLines[len(resultLines)-maxResultLines:]
			}

			// Render response lines with left alignment
			for i, line := range resultLines {
				wrappedLine := wrapText(line, boxWidth-4)
				prefix := "  "
				if i == 0 {
					prefix = "  "
				}
				styled := lipgloss.NewStyle().
					Foreground(colorWhite).
					Render(prefix + wrappedLine)
				top.WriteString(indent + styled + "\n")
			}
		}
	}

	// === BOTTOM SECTION: Input + Status ===
	var bottom strings.Builder

	// Input box (only when not streaming)
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
		bottom.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
		bottom.WriteString("\n")
	}

	// Status bar - show stats persistently
	var status string
	if a.state.chatStreaming {
		streamStatus := a.buildStreamStatus()
		a.state.lastStats = streamStatus // Save for after streaming
		status = styleStatusBar.Render(streamStatus + "  [Esc] Cancel")
	} else {
		// Show last stats if available, plus controls
		var statusParts []string
		if a.state.lastStats != "" && len(a.state.chatHistory) > 0 {
			// Show abbreviated stats
			statusParts = append(statusParts, a.buildIdleStats())
		}
		statusParts = append(statusParts, "[Enter] Send  [n] New chat  [Esc] Back")
		status = styleStatusBar.Render(strings.Join(statusParts, "  "))
	}
	bottom.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	// === COMBINE: Top grows down, bottom anchored ===
	topContent := top.String()
	bottomContent := bottom.String()

	topLines := strings.Count(topContent, "\n") + 1
	bottomLines := strings.Count(bottomContent, "\n") + 1
	usedHeight := topLines + bottomLines

	// Add flexible space between top and bottom
	padding := a.height - usedHeight
	if padding < 1 {
		padding = 1
	}

	return topContent + strings.Repeat("\n", padding) + bottomContent
}

// wrapText wraps text to fit within maxWidth, preserving words
func wrapText(text string, maxWidth int) string {
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

	// Spinner + Phase
	spinner := spinnerFrames[a.state.spinnerFrame%len(spinnerFrames)]
	switch a.state.streamPhase {
	case "connecting":
		// Show fun loading message
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
		// Final stats
		if elapsed > 0 && a.state.streamTokens > 0 {
			tokPerSec := float64(a.state.streamTokens) / elapsed
			parts = append(parts, fmt.Sprintf("%d tokens (%.0f tok/s)", a.state.streamTokens, tokPerSec))
		} else {
			parts = append(parts, fmt.Sprintf("%d tokens", a.state.streamTokens))
		}
	default:
		parts = append(parts, "...")
	}

	// Context usage - always show
	if a.state.contextLimit > 0 {
		pct := float64(a.state.contextUsed) / float64(a.state.contextLimit) * 100
		parts = append(parts, fmt.Sprintf("%.1fk/%.0fk ctx (%.0f%%)",
			float64(a.state.contextUsed)/1000,
			float64(a.state.contextLimit)/1000,
			pct))
	}

	// Elapsed time
	if elapsed > 0 {
		parts = append(parts, fmt.Sprintf("%.1fs", elapsed))
	}

	// Skill indicator
	if a.state.chatSkill != nil {
		parts = append(parts, fmt.Sprintf("[%s]", a.state.chatSkill.Name))
	}

	return strings.Join(parts, "  ")
}

// buildIdleStats shows abbreviated stats when not streaming
func (a *App) buildIdleStats() string {
	var parts []string

	// Context usage
	if a.state.contextLimit > 0 && a.state.contextUsed > 0 {
		pct := float64(a.state.contextUsed) / float64(a.state.contextLimit) * 100
		parts = append(parts, fmt.Sprintf("%.1fk/%.0fk (%.0f%%)",
			float64(a.state.contextUsed)/1000,
			float64(a.state.contextLimit)/1000,
			pct))
	}

	// Skill indicator
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

	// Shorten common model names
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

	// Add provider if not obvious from model name
	if provider != "" && !strings.Contains(strings.ToLower(displayModel), strings.ToLower(provider)) {
		return fmt.Sprintf("%s via %s", displayModel, provider)
	}
	return displayModel
}
