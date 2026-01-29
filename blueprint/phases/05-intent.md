# Phase 5: Intent Parser

## Goal
Parse natural language instructions into structured Intent using local LLM. "summarize for my boss" becomes actionable parameters.

## Success Criteria
- Intent struct defined with action, tone, audience, format
- Parser uses local LLM to extract intent from natural language
- Common phrases correctly parsed
- Works without internet (Ollama)

---

## Files to Create

### 1. Intent Types

```
pulp/internal/intent/intent.go
```

```go
package intent

// Intent represents parsed user instruction
type Intent struct {
	// Action: what to do
	Action Action `json:"action"`

	// Style
	Tone     string `json:"tone"`     // professional, casual, technical, academic
	Audience string `json:"audience"` // executive, expert, general, child

	// Format
	Format string `json:"format"` // prose, bullets, outline, table

	// Constraints
	MaxWords *int `json:"max_words,omitempty"`

	// For extraction
	ExtractType string `json:"extract_type,omitempty"` // action_items, key_points, quotes

	// Style hints from the original prompt
	StyleHints []string `json:"style_hints,omitempty"`

	// Original prompt for context
	RawPrompt string `json:"raw_prompt"`
}

type Action string

const (
	ActionSummarize Action = "summarize"
	ActionRewrite   Action = "rewrite"
	ActionExtract   Action = "extract"
	ActionExplain   Action = "explain"
	ActionCondense  Action = "condense"
)

// DefaultIntent returns sensible defaults
func DefaultIntent(prompt string) *Intent {
	return &Intent{
		Action:    ActionSummarize,
		Tone:      "neutral",
		Audience:  "general",
		Format:    "prose",
		RawPrompt: prompt,
	}
}

// ToneDescription returns description for prompt building
func (i *Intent) ToneDescription() string {
	switch i.Tone {
	case "professional", "executive":
		return "professional and concise, suitable for business communication"
	case "casual", "friendly":
		return "casual and conversational, like explaining to a friend"
	case "technical":
		return "technical and detailed, suitable for experts"
	case "academic":
		return "formal and academic, suitable for scholarly work"
	case "simple":
		return "simple and easy to understand, avoiding jargon"
	default:
		return "clear and well-organized"
	}
}

// AudienceDescription returns description for prompt building
func (i *Intent) AudienceDescription() string {
	switch i.Audience {
	case "executive", "boss", "manager":
		return "a busy executive who wants the key points quickly"
	case "expert", "technical":
		return "a domain expert who appreciates technical details"
	case "child", "simple":
		return "someone with no background, using simple language"
	case "general":
		return "a general audience with moderate knowledge"
	default:
		return "a general reader"
	}
}
```

---

### 2. Parser Prompt

```
pulp/internal/intent/prompt.go
```

```go
package intent

const parserSystemPrompt = `You are an intent parser. Given a user's instruction about a document, extract their intent.

Return ONLY valid JSON with this structure:
{
  "action": "summarize|rewrite|extract|explain|condense",
  "tone": "professional|casual|technical|academic|simple|neutral",
  "audience": "executive|expert|general|child",
  "format": "prose|bullets|outline",
  "max_words": null or number,
  "extract_type": null or "action_items|key_points|quotes|facts",
  "style_hints": ["list", "of", "hints"]
}

Rules:
- "for my boss" or "executive" -> tone: professional, audience: executive
- "like I'm 5" or "simple" -> tone: simple, audience: child
- "bullet points" or "bullets" -> format: bullets
- "keep it short" or "brief" -> max_words: 150
- "detailed" or "thorough" -> max_words: null (no limit)
- "action items" or "todos" -> action: extract, extract_type: action_items
- "key points" or "main points" -> action: extract, extract_type: key_points

Return ONLY the JSON, no explanation.`

func buildParserPrompt(instruction string) string {
	return `Parse this instruction: "` + instruction + `"`
}
```

---

### 3. Parser Implementation

```
pulp/internal/intent/parser.go
```

```go
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
```

---

### 4. Update State

Add to `state.go`:

```go
import (
	"github.com/sant0-9/pulp/internal/intent"
)

type state struct {
	// ... existing fields ...

	// Intent
	currentIntent *intent.Intent
	parsingIntent bool
}
```

---

### 5. Wire Up in App

Add intent parsing when user submits instruction. In `app.go`:

```go
import (
	"github.com/sant0-9/pulp/internal/intent"
)

type intentParsedMsg struct {
	intent *intent.Intent
}

// In handleKey, when Enter is pressed on document view:
case key.Matches(msg, keys.Enter):
	if a.view == viewDocument {
		instruction := strings.TrimSpace(a.state.input.Value())
		if instruction != "" {
			a.state.parsingIntent = true
			return a.parseIntent(instruction)
		}
	}
	// ... rest of enter handling

func (a *App) parseIntent(instruction string) tea.Cmd {
	return func() tea.Msg {
		parser := intent.NewParser(a.state.provider, a.state.config.Model)
		ctx := context.Background()

		parsed, err := parser.Parse(ctx, instruction)
		if err != nil {
			// Use defaults on error
			parsed = intent.DefaultIntent(instruction)
		}

		return intentParsedMsg{parsed}
	}
}

// In Update:
case intentParsedMsg:
	a.state.parsingIntent = false
	a.state.currentIntent = msg.intent
	// For now, just show what we parsed (processing comes in Phase 6)
	return a, nil
```

---

### 6. Show Parsed Intent (Debug)

Update document view to show parsed intent when available:

```go
// In renderDocument, after input box but before status:
if a.state.currentIntent != nil {
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
```

---

## Test

```bash
# Make sure Ollama is running with a model
ollama run qwen2.5:3b

# Build and run
go build -o pulp ./cmd/pulp
./pulp

# Load a document, then try these instructions:
# - "summarize"
# - "summarize for my boss"
# - "explain like I'm 5"
# - "bullet points"
# - "what are the action items"
# - "make it shorter"

# Each should show parsed intent at bottom
```

---

## Done Checklist

- [ ] Intent struct defined
- [ ] Parser system prompt created
- [ ] Quick parse handles common patterns
- [ ] LLM parse handles complex instructions
- [ ] Parser falls back to defaults on error
- [ ] Intent shown in TUI after parsing
- [ ] Works with Ollama (no internet)

---

## Commit Message

```
feat: add natural language intent parser

- Define Intent struct with action, tone, audience, format
- Create parser with quick-match for common patterns
- Use local LLM for complex instruction parsing
- Show parsed intent in document view
- Fall back to sensible defaults on parse errors
```
