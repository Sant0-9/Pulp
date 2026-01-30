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
