package skill

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SkillMetadata is the lightweight index entry loaded at startup
type SkillMetadata struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Path        string `yaml:"-"` // Full path to SKILL.md
	DirPath     string `yaml:"-"` // Directory containing the skill
}

// Skill is the full skill loaded on-demand
type Skill struct {
	SkillMetadata
	Body string // Markdown body (instructions)
}

// LoadMetadata reads only YAML frontmatter (fast startup)
func LoadMetadata(skillPath string) (*SkillMetadata, error) {
	file, err := os.Open(skillPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var frontmatter strings.Builder
	scanner := bufio.NewScanner(file)
	inFrontmatter := false
	lineCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineCount++

		if lineCount == 1 && line == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter {
			if line == "---" {
				break // End of frontmatter
			}
			frontmatter.WriteString(line)
			frontmatter.WriteString("\n")
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	var meta SkillMetadata
	if err := yaml.Unmarshal([]byte(frontmatter.String()), &meta); err != nil {
		return nil, err
	}

	meta.Path = skillPath
	meta.DirPath = filepath.Dir(skillPath)

	return &meta, nil
}

// LoadFull reads the entire skill including body
func LoadFull(meta *SkillMetadata) (*Skill, error) {
	content, err := os.ReadFile(meta.Path)
	if err != nil {
		return nil, err
	}

	// Parse frontmatter and body
	parts := strings.SplitN(string(content), "---", 3)
	if len(parts) < 3 {
		// No frontmatter or malformed
		return &Skill{
			SkillMetadata: *meta,
			Body:          string(content),
		}, nil
	}

	// parts[0] is empty (before first ---)
	// parts[1] is frontmatter
	// parts[2] is body

	body := strings.TrimSpace(parts[2])

	return &Skill{
		SkillMetadata: *meta,
		Body:          body,
	}, nil
}
