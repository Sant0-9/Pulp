# Phase 9: All Providers

## Goal
Implement all LLM providers: Groq, OpenAI, Anthropic, OpenRouter, and Custom (OpenAI-compatible).

## Success Criteria
- All providers implement the Provider interface
- Can switch providers in settings
- Each provider handles its API correctly
- Error messages are provider-specific and helpful

---

## Files to Create

### 1. Groq Provider

```
pulp/internal/llm/groq.go
```

```go
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GroqProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewGroqProvider(apiKey, model string) *GroqProvider {
	if model == "" {
		model = "llama-3.1-70b-versatile"
	}
	return &GroqProvider{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (g *GroqProvider) Name() string {
	return "groq"
}

func (g *GroqProvider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.groq.com/openai/v1/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to Groq API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("Groq API error: status %d", resp.StatusCode)
	}

	return nil
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature float64         `json:"temperature,omitempty"`
	Stream      bool            `json:"stream"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponse struct {
	ID      string `json:"id"`
	Choices []struct {
		Message      openAIMessage `json:"message"`
		FinishReason string        `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`
}

func (g *GroqProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = g.model
	}

	apiReq := openAIRequest{
		Model:       model,
		Messages:    toOpenAIMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	body, _ := json.Marshal(apiReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.groq.com/openai/v1/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Groq request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Groq error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from Groq")
	}

	return &CompletionResponse{
		Content:      apiResp.Choices[0].Message.Content,
		Model:        model,
		FinishReason: apiResp.Choices[0].FinishReason,
		Usage: Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
	}, nil
}

func (g *GroqProvider) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = g.model
	}

	apiReq := openAIRequest{
		Model:       model,
		Messages:    toOpenAIMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	body, _ := json.Marshal(apiReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.groq.com/openai/v1/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+g.apiKey)

	resp, err := g.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Groq request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Groq error (status %d): %s", resp.StatusCode, string(body))
	}

	events := make(chan StreamEvent)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !bytes.HasPrefix([]byte(line), []byte("data: ")) {
				continue
			}

			data := line[6:]
			if data == "[DONE]" {
				events <- StreamEvent{Done: true}
				return
			}

			var chunk openAIStreamResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 {
				if chunk.Choices[0].FinishReason != nil {
					events <- StreamEvent{Done: true}
					return
				}
				events <- StreamEvent{Chunk: chunk.Choices[0].Delta.Content}
			}
		}
	}()

	return events, nil
}

func toOpenAIMessages(msgs []Message) []openAIMessage {
	result := make([]openAIMessage, len(msgs))
	for i, m := range msgs {
		result[i] = openAIMessage{Role: m.Role, Content: m.Content}
	}
	return result
}
```

---

### 2. OpenAI Provider

```
pulp/internal/llm/openai.go
```

```go
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIProvider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

func NewOpenAIProvider(apiKey, model string) *OpenAIProvider {
	if model == "" {
		model = "gpt-4o-mini"
	}
	return &OpenAIProvider{
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://api.openai.com/v1",
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (o *OpenAIProvider) Name() string {
	return "openai"
}

func (o *OpenAIProvider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", o.baseURL+"/models", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to OpenAI API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("OpenAI API error: status %d", resp.StatusCode)
	}

	return nil
}

func (o *OpenAIProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}

	apiReq := openAIRequest{
		Model:       model,
		Messages:    toOpenAIMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	body, _ := json.Marshal(apiReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		o.baseURL+"/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OpenAI error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Choices) == 0 {
		return nil, fmt.Errorf("no response from OpenAI")
	}

	return &CompletionResponse{
		Content:      apiResp.Choices[0].Message.Content,
		Model:        model,
		FinishReason: apiResp.Choices[0].FinishReason,
		Usage: Usage{
			PromptTokens:     apiResp.Usage.PromptTokens,
			CompletionTokens: apiResp.Usage.CompletionTokens,
			TotalTokens:      apiResp.Usage.TotalTokens,
		},
	}, nil
}

func (o *OpenAIProvider) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}

	apiReq := openAIRequest{
		Model:       model,
		Messages:    toOpenAIMessages(req.Messages),
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	body, _ := json.Marshal(apiReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		o.baseURL+"/chat/completions",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.apiKey)

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("OpenAI request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("OpenAI error (status %d): %s", resp.StatusCode, string(body))
	}

	events := make(chan StreamEvent)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !bytes.HasPrefix([]byte(line), []byte("data: ")) {
				continue
			}

			data := line[6:]
			if data == "[DONE]" {
				events <- StreamEvent{Done: true}
				return
			}

			var chunk openAIStreamResponse
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) > 0 {
				if chunk.Choices[0].FinishReason != nil {
					events <- StreamEvent{Done: true}
					return
				}
				events <- StreamEvent{Chunk: chunk.Choices[0].Delta.Content}
			}
		}
	}()

	return events, nil
}
```

---

### 3. Anthropic Provider

```
pulp/internal/llm/anthropic.go
```

```go
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AnthropicProvider struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewAnthropicProvider(apiKey, model string) *AnthropicProvider {
	if model == "" {
		model = "claude-3-5-sonnet-20241022"
	}
	return &AnthropicProvider{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (a *AnthropicProvider) Name() string {
	return "anthropic"
}

func (a *AnthropicProvider) Ping(ctx context.Context) error {
	// Anthropic doesn't have a simple ping endpoint, so we do a minimal request
	req, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader([]byte(`{"model":"claude-3-5-sonnet-20241022","max_tokens":1,"messages":[{"role":"user","content":"hi"}]}`)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return fmt.Errorf("invalid API key")
	}
	// 200 or 400 (bad request) both mean we connected successfully
	if resp.StatusCode != 200 && resp.StatusCode != 400 {
		return fmt.Errorf("Anthropic API error: status %d", resp.StatusCode)
	}

	return nil
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Stream    bool               `json:"stream"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (a *AnthropicProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = a.model
	}

	// Extract system message
	var system string
	var messages []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	apiReq := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Stream:    false,
	}

	body, _ := json.Marshal(apiReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Anthropic error (status %d): %s", resp.StatusCode, string(body))
	}

	var apiResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("no response from Anthropic")
	}

	return &CompletionResponse{
		Content:      apiResp.Content[0].Text,
		Model:        model,
		FinishReason: apiResp.StopReason,
		Usage: Usage{
			PromptTokens:     apiResp.Usage.InputTokens,
			CompletionTokens: apiResp.Usage.OutputTokens,
			TotalTokens:      apiResp.Usage.InputTokens + apiResp.Usage.OutputTokens,
		},
	}, nil
}

func (a *AnthropicProvider) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = a.model
	}

	var system string
	var messages []anthropicMessage
	for _, m := range req.Messages {
		if m.Role == "system" {
			system = m.Content
		} else {
			messages = append(messages, anthropicMessage{
				Role:    m.Role,
				Content: m.Content,
			})
		}
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 2048
	}

	apiReq := anthropicRequest{
		Model:     model,
		MaxTokens: maxTokens,
		System:    system,
		Messages:  messages,
		Stream:    true,
	}

	body, _ := json.Marshal(apiReq)

	httpReq, err := http.NewRequestWithContext(ctx, "POST",
		"https://api.anthropic.com/v1/messages",
		bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("Anthropic request failed: %w", err)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("Anthropic error (status %d): %s", resp.StatusCode, string(body))
	}

	events := make(chan StreamEvent)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Text()
			if !bytes.HasPrefix([]byte(line), []byte("data: ")) {
				continue
			}

			data := line[6:]

			var event struct {
				Type  string `json:"type"`
				Delta struct {
					Text string `json:"text"`
				} `json:"delta"`
			}
			if err := json.Unmarshal([]byte(data), &event); err != nil {
				continue
			}

			switch event.Type {
			case "content_block_delta":
				events <- StreamEvent{Chunk: event.Delta.Text}
			case "message_stop":
				events <- StreamEvent{Done: true}
				return
			}
		}
	}()

	return events, nil
}
```

---

### 4. OpenRouter Provider

```
pulp/internal/llm/openrouter.go
```

```go
package llm

import (
	"net/http"
	"time"
)

type OpenRouterProvider struct {
	*OpenAIProvider
}

func NewOpenRouterProvider(apiKey, model string) *OpenRouterProvider {
	if model == "" {
		model = "meta-llama/llama-3.1-70b-instruct"
	}
	return &OpenRouterProvider{
		OpenAIProvider: &OpenAIProvider{
			apiKey:  apiKey,
			model:   model,
			baseURL: "https://openrouter.ai/api/v1",
			httpClient: &http.Client{
				Timeout: 5 * time.Minute,
			},
		},
	}
}

func (o *OpenRouterProvider) Name() string {
	return "openrouter"
}
```

---

### 5. Custom Provider (OpenAI-Compatible)

```
pulp/internal/llm/custom.go
```

```go
package llm

import (
	"net/http"
	"time"
)

type CustomProvider struct {
	*OpenAIProvider
}

func NewCustomProvider(baseURL, apiKey, model string) *CustomProvider {
	return &CustomProvider{
		OpenAIProvider: &OpenAIProvider{
			apiKey:  apiKey,
			model:   model,
			baseURL: baseURL,
			httpClient: &http.Client{
				Timeout: 5 * time.Minute,
			},
		},
	}
}

func (c *CustomProvider) Name() string {
	return "custom"
}
```

---

### 6. Update Factory

```
pulp/internal/llm/factory.go
```

```go
package llm

import (
	"fmt"

	"github.com/sant0-9/pulp/internal/config"
)

func NewProvider(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case "ollama":
		host := "http://localhost:11434"
		if cfg.BaseURL != "" {
			host = cfg.BaseURL
		}
		return NewOllamaProvider(host, cfg.Model), nil

	case "groq":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("groq requires an API key")
		}
		return NewGroqProvider(cfg.APIKey, cfg.Model), nil

	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openai requires an API key")
		}
		return NewOpenAIProvider(cfg.APIKey, cfg.Model), nil

	case "anthropic":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic requires an API key")
		}
		return NewAnthropicProvider(cfg.APIKey, cfg.Model), nil

	case "openrouter":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openrouter requires an API key")
		}
		return NewOpenRouterProvider(cfg.APIKey, cfg.Model), nil

	case "custom":
		if cfg.BaseURL == "" {
			return nil, fmt.Errorf("custom provider requires base_url")
		}
		return NewCustomProvider(cfg.BaseURL, cfg.APIKey, cfg.Model), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}
```

---

## Test

```bash
# Test with Groq
export GROQ_API_KEY="gsk_..."
# Edit ~/.config/pulp/config.yaml to use groq
./pulp

# Test with OpenAI
export OPENAI_API_KEY="sk-..."
# Edit config to use openai
./pulp

# Test with Anthropic
export ANTHROPIC_API_KEY="sk-ant-..."
# Edit config to use anthropic
./pulp
```

---

## Done Checklist

- [ ] Groq provider works
- [ ] OpenAI provider works
- [ ] Anthropic provider works
- [ ] OpenRouter provider works
- [ ] Custom provider works
- [ ] All providers stream correctly
- [ ] Error messages are helpful
- [ ] Can switch providers in config

---

## Commit Message

```
feat: add all LLM providers

- Implement Groq provider with streaming
- Implement OpenAI provider with streaming
- Implement Anthropic provider with streaming
- Add OpenRouter provider (OpenAI-compatible)
- Add Custom provider for self-hosted LLMs
- Update factory to create all provider types
```
