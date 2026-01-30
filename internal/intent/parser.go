package intent

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/sant0-9/pulp/internal/llm"
)

// Parser parses natural language into Intent
type Parser struct {
	provider llm.Provider
	model    string
}

// NewParser creates a new intent parser
func NewParser(provider llm.Provider, model string) *Parser {
	return &Parser{
		provider: provider,
		model:    model,
	}
}

// Parse parses a natural language instruction
func (p *Parser) Parse(ctx context.Context, instruction string) (*Intent, error) {
	// Quick pattern matching for common cases (fast path)
	if intent := p.quickParse(instruction); intent != nil {
		return intent, nil
	}

	// Use LLM for complex parsing
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	req := &llm.CompletionRequest{
		Model: p.model,
		Messages: []llm.Message{
			{Role: "system", Content: parserSystemPrompt},
			{Role: "user", Content: buildParserPrompt(instruction)},
		},
		MaxTokens:   500,
		Temperature: 0.1, // Low temperature for consistent parsing
	}

	resp, err := p.provider.Complete(ctx, req)
	if err != nil {
		// Fall back to defaults on error
		return DefaultIntent(instruction), nil
	}

	// Parse JSON response
	intent := &Intent{RawPrompt: instruction}
	content := strings.TrimSpace(resp.Content)

	// Try to extract JSON if wrapped in markdown
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		var jsonLines []string
		inJSON := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inJSON = !inJSON
				continue
			}
			if inJSON {
				jsonLines = append(jsonLines, line)
			}
		}
		content = strings.Join(jsonLines, "\n")
	}

	if err := json.Unmarshal([]byte(content), intent); err != nil {
		// Fall back to defaults
		return DefaultIntent(instruction), nil
	}

	intent.RawPrompt = instruction
	return intent, nil
}

// quickParse handles common patterns without LLM
func (p *Parser) quickParse(instruction string) *Intent {
	lower := strings.ToLower(instruction)

	intent := &Intent{
		Action:    ActionSummarize,
		Tone:      "neutral",
		Audience:  "general",
		Format:    "prose",
		RawPrompt: instruction,
	}

	// Action detection
	switch {
	case strings.Contains(lower, "action item") || strings.Contains(lower, "todo"):
		intent.Action = ActionExtract
		intent.ExtractType = "action_items"
		intent.Format = "bullets"
	case strings.Contains(lower, "key point") || strings.Contains(lower, "main point"):
		intent.Action = ActionExtract
		intent.ExtractType = "key_points"
		intent.Format = "bullets"
	case strings.Contains(lower, "explain"):
		intent.Action = ActionExplain
	case strings.Contains(lower, "rewrite") || strings.Contains(lower, "turn into") || strings.Contains(lower, "make it"):
		intent.Action = ActionRewrite
	case strings.Contains(lower, "shorter") || strings.Contains(lower, "condense"):
		intent.Action = ActionCondense
	}

	// Audience detection
	switch {
	case strings.Contains(lower, "boss") || strings.Contains(lower, "executive") || strings.Contains(lower, "ceo"):
		intent.Audience = "executive"
		intent.Tone = "professional"
	case strings.Contains(lower, "5") || strings.Contains(lower, "child") || strings.Contains(lower, "simple"):
		intent.Audience = "child"
		intent.Tone = "simple"
	case strings.Contains(lower, "technical") || strings.Contains(lower, "engineer"):
		intent.Audience = "expert"
		intent.Tone = "technical"
	}

	// Format detection
	switch {
	case strings.Contains(lower, "bullet"):
		intent.Format = "bullets"
	case strings.Contains(lower, "outline"):
		intent.Format = "outline"
	}

	// Length hints
	if strings.Contains(lower, "brief") || strings.Contains(lower, "short") {
		maxWords := 150
		intent.MaxWords = &maxWords
	}

	// Detect if this is a simple/common pattern
	commonPatterns := []string{
		"summarize",
		"summary",
		"action item",
		"key point",
		"bullet",
		"boss",
		"executive",
		"explain",
	}

	for _, pattern := range commonPatterns {
		if strings.Contains(lower, pattern) {
			return intent // Use quick parse result
		}
	}

	// Not a common pattern, return nil to use LLM
	return nil
}
