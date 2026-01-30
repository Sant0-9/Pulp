package intent

import (
	"context"

	"github.com/sant0-9/pulp/internal/llm"
)

// Parser wraps user instructions into Intent
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

// Parse wraps the instruction in an Intent (no parsing needed for pure LLM interpretation)
func (p *Parser) Parse(ctx context.Context, instruction string) (*Intent, error) {
	return New(instruction), nil
}
