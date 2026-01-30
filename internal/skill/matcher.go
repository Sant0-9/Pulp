package skill

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sant0-9/pulp/internal/llm"
)

// Matcher finds the best skill for a given instruction
type Matcher struct {
	provider llm.Provider
	model    string
	index    *SkillIndex
}

// NewMatcher creates a new skill matcher
func NewMatcher(provider llm.Provider, model string, index *SkillIndex) *Matcher {
	return &Matcher{
		provider: provider,
		model:    model,
		index:    index,
	}
}

// MatchResult contains the matching result
type MatchResult struct {
	Skill      *SkillMetadata
	Confidence float64
}

// Match finds the best skill for the given instruction
func (m *Matcher) Match(ctx context.Context, instruction string) (*MatchResult, error) {
	if m.index == nil || m.index.Count() == 0 {
		return nil, nil
	}

	allSkills := m.index.GetAll()
	if len(allSkills) == 0 {
		return nil, nil
	}

	prompt := m.buildMatchingPrompt(instruction, allSkills)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	resp, err := m.provider.Complete(ctx, &llm.CompletionRequest{
		Model: m.model,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   100,
		Temperature: 0.1,
	})
	if err != nil {
		return nil, err
	}

	return m.parseResponse(resp.Content, allSkills)
}

func (m *Matcher) buildMatchingPrompt(instruction string, skills []*SkillMetadata) string {
	var sb strings.Builder
	sb.WriteString("Match this document processing instruction to the best skill.\n\n")
	sb.WriteString(fmt.Sprintf("Instruction: \"%s\"\n\n", instruction))
	sb.WriteString("Available skills:\n")

	for _, s := range skills {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", s.Name, s.Description))
	}

	sb.WriteString("\nRespond with JSON only: {\"skill\": \"name-or-none\", \"confidence\": 0.0-1.0}")
	sb.WriteString("\nUse \"none\" if no skill matches well (confidence < 0.5)")

	return sb.String()
}

func (m *Matcher) parseResponse(content string, skills []*SkillMetadata) (*MatchResult, error) {
	// Clean up response
	content = strings.TrimSpace(content)

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

	var result struct {
		Skill      string  `json:"skill"`
		Confidence float64 `json:"confidence"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		return nil, nil // Return nil on parse error, not an error
	}

	if result.Skill == "none" || result.Skill == "" || result.Confidence < 0.5 {
		return nil, nil
	}

	// Find the skill
	skill := m.index.Get(result.Skill)
	if skill == nil {
		return nil, nil
	}

	return &MatchResult{
		Skill:      skill,
		Confidence: result.Confidence,
	}, nil
}
