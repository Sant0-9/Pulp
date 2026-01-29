package llm

import (
	"context"
)

// Provider is the interface all LLM providers must implement
type Provider interface {
	// Name returns the provider name
	Name() string

	// Complete sends a completion request and returns the full response
	Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)

	// Stream sends a completion request and streams the response
	Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamEvent, error)

	// Ping checks if the provider is reachable
	Ping(ctx context.Context) error
}

// CompletionRequest represents a request to the LLM
type CompletionRequest struct {
	Model       string
	Messages    []Message
	MaxTokens   int
	Temperature float64
}

// Message represents a chat message
type Message struct {
	Role    string
	Content string
}

// CompletionResponse represents the full response
type CompletionResponse struct {
	Content      string
	Model        string
	FinishReason string
	Usage        Usage
}

// Usage tracks token usage
type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

// StreamEvent represents a streaming chunk or completion
type StreamEvent struct {
	Chunk string
	Done  bool
	Error error
	Usage *Usage
}

// NewRequest creates a simple completion request
func NewRequest(model string, systemPrompt, userPrompt string) *CompletionRequest {
	return &CompletionRequest{
		Model: model,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   2048,
		Temperature: 0.7,
	}
}
