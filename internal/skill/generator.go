package skill

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sant0-9/pulp/internal/llm"
)

// Generator creates new skills using LLM
type Generator struct {
	provider  llm.Provider
	model     string
	skillsDir string
}

// NewGenerator creates a new skill generator
func NewGenerator(provider llm.Provider, model string) *Generator {
	home, _ := os.UserHomeDir()
	return &Generator{
		provider:  provider,
		model:     model,
		skillsDir: filepath.Join(home, ".config", "pulp", "skills"),
	}
}

// Generate creates a new skill from a description
func (g *Generator) Generate(ctx context.Context, description string) (*Skill, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	prompt := buildGeneratorPrompt(description)

	resp, err := g.provider.Complete(ctx, &llm.CompletionRequest{
		Model: g.model,
		Messages: []llm.Message{
			{Role: "user", Content: prompt},
		},
		MaxTokens:   2000,
		Temperature: 0.7,
	})
	if err != nil {
		return nil, err
	}

	// Parse the generated skill
	skill, err := parseGeneratedSkill(resp.Content)
	if err != nil {
		return nil, err
	}

	// Save to disk
	if err := g.save(skill); err != nil {
		return nil, err
	}

	return skill, nil
}

func buildGeneratorPrompt(description string) string {
	return fmt.Sprintf(`Create a skill for document processing based on this description:

"%s"

Generate a complete SKILL.md file with YAML frontmatter and markdown body.

Requirements:
1. The "name" should be lowercase with hyphens (e.g., extract-dates, legal-summary)
2. The "description" should be 1-2 sentences explaining when to use this skill
3. The body should have clear guidelines and output format instructions

Respond with ONLY the SKILL.md content, starting with --- and ending with the markdown body.

Example format:
---
name: skill-name
description: When to use this skill. Keywords that trigger it.
---

# Skill Title

Instructions for the LLM when processing documents with this skill.

## Guidelines
- Point 1
- Point 2

## Output Format
How to structure the output.`, description)
}

func parseGeneratedSkill(content string) (*Skill, error) {
	content = strings.TrimSpace(content)

	// Remove markdown code blocks if present
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		var cleaned []string
		inBlock := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				inBlock = !inBlock
				continue
			}
			if !inBlock || !strings.HasPrefix(line, "```") {
				cleaned = append(cleaned, line)
			}
		}
		content = strings.Join(cleaned, "\n")
	}

	// Parse frontmatter
	parts := strings.SplitN(content, "---", 3)
	if len(parts) < 3 {
		return nil, fmt.Errorf("invalid skill format: missing frontmatter")
	}

	frontmatter := strings.TrimSpace(parts[1])
	body := strings.TrimSpace(parts[2])

	// Extract name and description from frontmatter
	var name, description string
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
			name = strings.Trim(name, "\"'")
		} else if strings.HasPrefix(line, "description:") {
			description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			description = strings.Trim(description, "\"'")
		}
	}

	if name == "" {
		return nil, fmt.Errorf("skill name not found in frontmatter")
	}

	// Sanitize name
	name = sanitizeName(name)

	return &Skill{
		SkillMetadata: SkillMetadata{
			Name:        name,
			Description: description,
		},
		Body: body,
	}, nil
}

func sanitizeName(name string) string {
	// Convert to lowercase
	name = strings.ToLower(name)
	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")
	// Remove invalid characters
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	name = reg.ReplaceAllString(name, "")
	// Remove multiple hyphens
	reg = regexp.MustCompile(`-+`)
	name = reg.ReplaceAllString(name, "-")
	// Trim hyphens from ends
	name = strings.Trim(name, "-")
	return name
}

func (g *Generator) save(skill *Skill) error {
	skillDir := filepath.Join(g.skillsDir, skill.Name)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return err
	}

	content := fmt.Sprintf(`---
name: %s
description: %s
---

%s`, skill.Name, skill.Description, skill.Body)

	skillPath := filepath.Join(skillDir, "SKILL.md")
	skill.Path = skillPath
	skill.DirPath = skillDir

	return os.WriteFile(skillPath, []byte(content), 0644)
}
