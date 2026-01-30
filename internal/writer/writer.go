package writer

import (
	"context"
	"fmt"

	"github.com/sant0-9/pulp/internal/intent"
	"github.com/sant0-9/pulp/internal/llm"
	"github.com/sant0-9/pulp/internal/pipeline"
	"github.com/sant0-9/pulp/internal/prompts"
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
	var messages []llm.Message

	// Build system prompt with skill instructions if present
	if req.Intent.HasSkill() {
		messages = append(messages, llm.Message{
			Role:    "system",
			Content: prompts.BuildSkillPrompt(req.Intent.MatchedSkill.Body),
		})
	}

	// Get document content
	docContent := req.Aggregated.FormatForWriter()
	if req.DocTitle != "" {
		docContent = fmt.Sprintf("Document: %s\n\n%s", req.DocTitle, docContent)
	}

	if req.IsFollowUp && req.PreviousResult != "" {
		// Follow-up: include previous result for revision
		userContent := fmt.Sprintf("Previous response:\n\n%s\n\n---\n\n%s", req.PreviousResult, req.Intent.RawPrompt)
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: userContent,
		})
	} else {
		// First request: document + instruction
		userContent := docContent
		if req.Intent.RawPrompt != "" {
			userContent = fmt.Sprintf("%s\n\n---\n\n%s", docContent, req.Intent.RawPrompt)
		}
		messages = append(messages, llm.Message{
			Role:    "user",
			Content: userContent,
		})
	}

	return messages
}
