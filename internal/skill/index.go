package skill

import (
	"os"
	"path/filepath"
)

// SkillIndex manages all available skills
type SkillIndex struct {
	skills    map[string]*SkillMetadata
	skillsDir string
}

// NewSkillIndex loads all skill metadata from ~/.config/pulp/skills/
func NewSkillIndex() (*SkillIndex, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	skillsDir := filepath.Join(home, ".config", "pulp", "skills")

	idx := &SkillIndex{
		skills:    make(map[string]*SkillMetadata),
		skillsDir: skillsDir,
	}

	// Create skills directory if it doesn't exist
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return nil, err
	}

	// Scan for skills
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return idx, nil // Return empty index if can't read
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(skillsDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillPath); os.IsNotExist(err) {
			continue
		}

		meta, err := LoadMetadata(skillPath)
		if err != nil {
			continue // Skip invalid skills
		}

		// Use directory name as fallback if no name in frontmatter
		if meta.Name == "" {
			meta.Name = entry.Name()
		}

		idx.skills[meta.Name] = meta
	}

	return idx, nil
}

// Get returns metadata by name
func (idx *SkillIndex) Get(name string) *SkillMetadata {
	if idx == nil {
		return nil
	}
	return idx.skills[name]
}

// GetAll returns all metadata for semantic matching
func (idx *SkillIndex) GetAll() []*SkillMetadata {
	if idx == nil {
		return nil
	}
	result := make([]*SkillMetadata, 0, len(idx.skills))
	for _, meta := range idx.skills {
		result = append(result, meta)
	}
	return result
}

// List returns all skill names
func (idx *SkillIndex) List() []string {
	if idx == nil {
		return nil
	}
	result := make([]string, 0, len(idx.skills))
	for name := range idx.skills {
		result = append(result, name)
	}
	return result
}

// SkillsDir returns the skills directory path
func (idx *SkillIndex) SkillsDir() string {
	return idx.skillsDir
}

// Count returns the number of loaded skills
func (idx *SkillIndex) Count() int {
	if idx == nil {
		return 0
	}
	return len(idx.skills)
}
