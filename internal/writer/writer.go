package writer

import (
	"context"
	"fmt"

	"github.com/sant0-9/pulp/internal/intent"
	"github.com/sant0-9/pulp/internal/llm"
	"github.com/sant0-9/pulp/internal/pipeline"
)

// Writer generates final output from aggregated content
type Writer struct {
	provider llm.Provider
	model    string
}

// NewWriter creates a new writer
func NewWriter(provider llm.Provider, model string) *Writer {
	return &Writer{
		provider: provider,
		model:    model,
	}
}

// Message represents a conversation message
type Message struct {
	Role    string
	Content string
}

// WriteRequest contains everything needed to generate output
type WriteRequest struct {
	Aggregated     *pipeline.AggregatedContent
	Intent         *intent.Intent
	DocTitle       string
	History        []Message
	IsFollowUp     bool
	PreviousResult string
}

// Write generates the final output (non-streaming)
func (w *Writer) Write(ctx context.Context, req *WriteRequest) (string, error) {
	messages := w.buildMessages(req)

	llmReq := &llm.CompletionRequest{
		Model:       w.model,
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	resp, err := w.provider.Complete(ctx, llmReq)
	if err != nil {
		return "", err
	}

	return resp.Content, nil
}

// Stream generates output with streaming
func (w *Writer) Stream(ctx context.Context, req *WriteRequest) (<-chan llm.StreamEvent, error) {
	messages := w.buildMessages(req)

	llmReq := &llm.CompletionRequest{
		Model:       w.model,
		Messages:    messages,
		MaxTokens:   4096,
		Temperature: 0.7,
	}

	return w.provider.Stream(ctx, llmReq)
}

func (w *Writer) buildMessages(req *WriteRequest) []llm.Message {
	// Get document content
	docContent := req.Aggregated.FormatForWriter()
	if req.DocTitle != "" {
		docContent = fmt.Sprintf("Document: %s\n\n%s", req.DocTitle, docContent)
	}

	if req.IsFollowUp && req.PreviousResult != "" {
		// Follow-up: include previous result for revision
		return []llm.Message{
			{
				Role:    "user",
				Content: fmt.Sprintf("Here is my previous response:\n\n%s\n\n---\n\n%s", req.PreviousResult, req.Intent.RawPrompt),
			},
		}
	}

	// First request: document + instruction
	return []llm.Message{
		{
			Role:    "user",
			Content: fmt.Sprintf("%s\n\n---\n\n%s", docContent, req.Intent.RawPrompt),
		},
	}
}
