package intent

import "github.com/sant0-9/pulp/internal/skill"

// Intent holds the user's instruction and matched skill
type Intent struct {
	// The raw instruction from the user
	RawPrompt string

	// Matched skill (if any)
	MatchedSkill *skill.Skill

	// True if user explicitly invoked with /skill-name
	ExplicitSkill bool
}

// New creates a new intent from a raw prompt
func New(prompt string) *Intent {
	return &Intent{
		RawPrompt: prompt,
	}
}

// WithSkill attaches a skill to the intent
func (i *Intent) WithSkill(s *skill.Skill, explicit bool) *Intent {
	i.MatchedSkill = s
	i.ExplicitSkill = explicit
	return i
}

// HasSkill returns true if a skill is attached
func (i *Intent) HasSkill() bool {
	return i.MatchedSkill != nil
}

// SkillName returns the skill name or empty string
func (i *Intent) SkillName() string {
	if i.MatchedSkill == nil {
		return ""
	}
	return i.MatchedSkill.Name
}
