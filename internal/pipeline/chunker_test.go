package pipeline

import (
	"strings"
	"testing"
)

func TestChunkDocument(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		maxChunkSize int
		wantChunks   int
	}{
		{
			name:         "empty content",
			content:      "",
			maxChunkSize: 1000,
			wantChunks:   0,
		},
		{
			name:         "single paragraph",
			content:      "This is a single paragraph of text.",
			maxChunkSize: 1000,
			wantChunks:   1,
		},
		{
			name:         "multiple paragraphs within limit",
			content:      "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
			maxChunkSize: 1000,
			wantChunks:   1,
		},
		{
			name:         "paragraphs exceed limit",
			content:      strings.Repeat("word ", 100) + "\n\n" + strings.Repeat("word ", 100),
			maxChunkSize: 200,
			wantChunks:   2,
		},
		{
			name:         "with headers",
			content:      "# Header 1\nContent under header 1.\n\n# Header 2\nContent under header 2.",
			maxChunkSize: 1000,
			wantChunks:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := ChunkDocument(tt.content, tt.maxChunkSize)
			if len(chunks) != tt.wantChunks {
				t.Errorf("ChunkDocument() returned %d chunks, want %d", len(chunks), tt.wantChunks)
			}
		})
	}
}

func TestEstimateTokens(t *testing.T) {
	// Rough estimate: 4 chars per token
	text := "This is a test string with some words."
	tokens := EstimateTokens(text)

	// 39 characters / 4 = ~9-10 tokens
	if tokens < 5 || tokens > 15 {
		t.Errorf("EstimateTokens() = %d, want roughly 9-10", tokens)
	}
}
