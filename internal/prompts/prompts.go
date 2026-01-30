package prompts

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed chat.md
var ChatBase string

//go:embed extraction.md
var Extraction string

// BuildChatPrompt constructs the full chat system prompt
// If skill is provided, it appends the skill instructions
func BuildChatPrompt(skillName, skillBody string) string {
	base := strings.TrimSpace(ChatBase)

	if skillName != "" && skillBody != "" {
		return fmt.Sprintf("%s\n\n---\n\nActive Skill: %s\n\nFollow these specialized instructions:\n\n%s",
			base,
			skillName,
			skillBody,
		)
	}

	return base
}

// BuildSkillPrompt wraps skill body for document processing
func BuildSkillPrompt(skillBody string) string {
	return fmt.Sprintf("Follow these instructions when processing the document:\n\n%s", skillBody)
}
