package writer

import (
	"context"
	"fmt"
	"strings"

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
	History        []Message // Conversation history
	IsFollowUp     bool      // Whether this is a follow-up request
	PreviousResult string    // Previous result for revisions
}

// Write generates the final output (non-streaming)
func (w *Writer) Write(ctx context.Context, req *WriteRequest) (string, error) {
	prompt := w.buildPrompt(req)

	llmReq := &llm.CompletionRequest{
		Model: w.model,
		Messages: []llm.Message{
			{Role: "system", Content: prompt.System},
			{Role: "user", Content: prompt.User},
		},
		MaxTokens:   2048,
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
	prompt := w.buildPrompt(req)

	llmReq := &llm.CompletionRequest{
		Model: w.model,
		Messages: []llm.Message{
			{Role: "system", Content: prompt.System},
			{Role: "user", Content: prompt.User},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
	}

	return w.provider.Stream(ctx, llmReq)
}

type prompt struct {
	System string
	User   string
}

func (w *Writer) buildPrompt(req *WriteRequest) *prompt {
	var system strings.Builder
	i := req.Intent

	// Handle follow-up mode
	if req.IsFollowUp && req.PreviousResult != "" {
		system.WriteString("You are a skilled writer helping revise content. ")
		system.WriteString("The user has already received a response and wants changes. ")
		system.WriteString(fmt.Sprintf("\nWrite in a %s style for %s. ",
			i.ToneDescription(), i.AudienceDescription()))

		if i.MaxWords != nil {
			system.WriteString(fmt.Sprintf("Keep the response under %d words. ", *i.MaxWords))
		}

		system.WriteString(fmt.Sprintf("\n\nUser's revision request: \"%s\"", i.RawPrompt))

		// User content includes previous result
		user := fmt.Sprintf("Previous response to revise:\n\n%s", req.PreviousResult)

		return &prompt{
			System: system.String(),
			User:   user,
		}
	}

	// Original mode - base instruction
	system.WriteString("You are a skilled writer. ")

	// Action
	switch i.Action {
	case intent.ActionSummarize:
		system.WriteString("Summarize the provided content clearly and concisely. ")
	case intent.ActionRewrite:
		system.WriteString("Rewrite the provided content in a new style. ")
	case intent.ActionExtract:
		system.WriteString(fmt.Sprintf("Extract %s from the provided content. ", i.ExtractType))
	case intent.ActionExplain:
		system.WriteString("Explain the provided content clearly. ")
	case intent.ActionCondense:
		system.WriteString("Make the provided content more concise while keeping key information. ")
	}

	// Tone
	system.WriteString(fmt.Sprintf("\nWrite in a %s style. ", i.ToneDescription()))

	// Audience
	system.WriteString(fmt.Sprintf("The reader is %s. ", i.AudienceDescription()))

	// Format
	switch i.Format {
	case "bullets":
		system.WriteString("Use bullet points. ")
	case "outline":
		system.WriteString("Structure as an outline with headers. ")
	}

	// Constraints
	if i.MaxWords != nil {
		system.WriteString(fmt.Sprintf("Keep the response under %d words. ", *i.MaxWords))
	}

	// Additional context
	system.WriteString(fmt.Sprintf("\n\nOriginal request: \"%s\"", i.RawPrompt))

	// User content
	user := req.Aggregated.FormatForWriter()
	if req.DocTitle != "" {
		user = fmt.Sprintf("Document: %s\n\n%s", req.DocTitle, user)
	}

	return &prompt{
		System: system.String(),
		User:   user,
	}
}
