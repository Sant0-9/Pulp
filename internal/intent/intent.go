package intent

// Intent holds the user's instruction
type Intent struct {
	// The raw instruction from the user
	RawPrompt string
}

// New creates a new intent from a raw prompt
func New(prompt string) *Intent {
	return &Intent{
		RawPrompt: prompt,
	}
}
