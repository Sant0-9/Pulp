package pipeline

import (
	"strings"
	"unicode/utf8"
)

// Chunk represents a piece of document content
type Chunk struct {
	ID       int
	Content  string
	Section  string
	Position int
}

// ChunkDocument splits document into semantic chunks
func ChunkDocument(content string, maxChunkSize int) []Chunk {
	if maxChunkSize <= 0 {
		maxChunkSize = 1500 // Default ~375 tokens
	}

	var chunks []Chunk

	// Split by double newlines (paragraphs/sections)
	sections := strings.Split(content, "\n\n")

	var currentChunk strings.Builder
	var currentSection string
	chunkID := 0

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		// Check if this is a header
		if strings.HasPrefix(section, "#") {
			lines := strings.SplitN(section, "\n", 2)
			currentSection = strings.TrimLeft(lines[0], "# ")
			if len(lines) > 1 {
				section = lines[1]
			} else {
				continue
			}
		}

		// Check if adding this would exceed max size
		if currentChunk.Len()+len(section)+2 > maxChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			chunks = append(chunks, Chunk{
				ID:       chunkID,
				Content:  currentChunk.String(),
				Section:  currentSection,
				Position: chunkID,
			})
			chunkID++
			currentChunk.Reset()
		}

		// Add section to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(section)
	}

	// Don't forget the last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			ID:       chunkID,
			Content:  currentChunk.String(),
			Section:  currentSection,
			Position: chunkID,
		})
	}

	return chunks
}

// EstimateTokens estimates token count (rough: 4 chars per token)
func EstimateTokens(text string) int {
	return utf8.RuneCountInString(text) / 4
}
