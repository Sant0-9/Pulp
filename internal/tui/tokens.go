package tui

import "strings"

// estimateTokens returns approximate token count (~4 chars per token)
func estimateTokens(text string) int {
	return (len(text) + 3) / 4
}

// getContextLimit returns the context window size for a model
func getContextLimit(model string) int {
	model = strings.ToLower(model)

	// Claude models
	if strings.Contains(model, "claude") {
		return 200000
	}

	// GPT-4 variants
	if strings.Contains(model, "gpt-4o") || strings.Contains(model, "gpt-4-turbo") {
		return 128000
	}
	if strings.Contains(model, "gpt-4-32k") {
		return 32000
	}
	if strings.Contains(model, "gpt-4") {
		return 8000
	}

	// Llama variants
	if strings.Contains(model, "llama-3") || strings.Contains(model, "llama3") {
		return 128000
	}
	if strings.Contains(model, "llama") {
		return 8000
	}

	// Groq models
	if strings.Contains(model, "mixtral") {
		return 32000
	}

	// Gemini
	if strings.Contains(model, "gemini") {
		return 1000000
	}

	// Default fallback
	return 8000
}
