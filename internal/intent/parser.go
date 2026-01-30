package intent

import (
	"context"
	"strings"

	"github.com/sant0-9/pulp/internal/llm"
	"github.com/sant0-9/pulp/internal/skill"
)

// Parser wraps user instructions into Intent with skill matching
type Parser struct {
	provider   llm.Provider
	model      string
	skillIndex *skill.SkillIndex
	matcher    *skill.Matcher
}

// NewParser creates a new intent parser
func NewParser(provider llm.Provider, model string, skillIndex *skill.SkillIndex) *Parser {
	var matcher *skill.Matcher
	if skillIndex != nil && skillIndex.Count() > 0 {
		matcher = skill.NewMatcher(provider, model, skillIndex)
	}

	return &Parser{
		provider:   provider,
		model:      model,
		skillIndex: skillIndex,
		matcher:    matcher,
	}
}

// Parse wraps the instruction in an Intent with optional skill matching
func (p *Parser) Parse(ctx context.Context, instruction string) (*Intent, error) {
	// Check for explicit skill invocation (/skill-name)
	if strings.HasPrefix(instruction, "/") {
		parts := strings.SplitN(instruction, " ", 2)
		skillName := strings.TrimPrefix(parts[0], "/")

		if meta := p.skillIndex.Get(skillName); meta != nil {
			// Load the full skill
			s, err := skill.LoadFull(meta)
			if err != nil {
				// Fall back to no skill
				return New(instruction), nil
			}

			// Extract remaining instruction after /skill-name
			remaining := ""
			if len(parts) > 1 {
				remaining = strings.TrimSpace(parts[1])
			}

			intent := New(remaining)
			intent.WithSkill(s, true)
			return intent, nil
		}
	}

	// Try semantic matching if we have skills
	if p.matcher != nil {
		result, err := p.matcher.Match(ctx, instruction)
		if err == nil && result != nil {
			// Load the full skill
			s, err := skill.LoadFull(result.Skill)
			if err == nil {
				intent := New(instruction)
				intent.WithSkill(s, false)
				return intent, nil
			}
		}
	}

	// No skill matched, return plain intent
	return New(instruction), nil
}
