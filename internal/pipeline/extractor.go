package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sant0-9/pulp/internal/llm"
	"github.com/sant0-9/pulp/internal/prompts"
)

// Extraction contains extracted information from a chunk
type Extraction struct {
	ChunkID   int
	KeyPoints []string
	Entities  []string
	Facts     []string
	Summary   string
}

// Extractor extracts key information from chunks
type Extractor struct {
	provider llm.Provider
	model    string
}

func NewExtractor(provider llm.Provider, model string) *Extractor {
	return &Extractor{
		provider: provider,
		model:    model,
	}
}

func (e *Extractor) Extract(ctx context.Context, chunk Chunk) (*Extraction, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req := &llm.CompletionRequest{
		Model: e.model,
		Messages: []llm.Message{
			{Role: "system", Content: prompts.Extraction},
			{Role: "user", Content: chunk.Content},
		},
		MaxTokens:   500,
		Temperature: 0.3,
	}

	resp, err := e.provider.Complete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("extraction failed: %w", err)
	}

	// Parse JSON
	content := strings.TrimSpace(resp.Content)

	// Handle markdown-wrapped JSON
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		var jsonLines []string
		in := false
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				in = !in
				continue
			}
			if in {
				jsonLines = append(jsonLines, line)
			}
		}
		content = strings.Join(jsonLines, "\n")
	}

	var result struct {
		KeyPoints []string `json:"key_points"`
		Entities  []string `json:"entities"`
		Facts     []string `json:"facts"`
		Summary   string   `json:"summary"`
	}

	if err := json.Unmarshal([]byte(content), &result); err != nil {
		// If JSON parsing fails, use the content as a summary
		return &Extraction{
			ChunkID: chunk.ID,
			Summary: resp.Content,
		}, nil
	}

	return &Extraction{
		ChunkID:   chunk.ID,
		KeyPoints: result.KeyPoints,
		Entities:  result.Entities,
		Facts:     result.Facts,
		Summary:   result.Summary,
	}, nil
}
