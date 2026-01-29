# Phase 3: Provider System

## Goal
Create the LLM provider abstraction and implement Ollama provider. Able to send a message and get a response.

## Success Criteria
- Provider interface defined
- Ollama provider implemented
- Can ping Ollama to check connection
- Can send a completion request and get response
- Error handling for connection failures

---

## Files to Create

### 1. Provider Interface

```
pulp/internal/llm/provider.go
```

```go
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
```

---

### 2. Ollama Provider

```
pulp/internal/llm/ollama.go
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

type OllamaProvider struct {
	host       string
	model      string
	httpClient *http.Client
}

func NewOllamaProvider(host, model string) *OllamaProvider {
	if host == "" {
		host = "http://localhost:11434"
	}
	return &OllamaProvider{
		host:  host,
		model: model,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
}

func (o *OllamaProvider) Name() string {
	return "ollama"
}

func (o *OllamaProvider) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", o.host+"/api/tags", nil)
	if err != nil {
		return err
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("cannot connect to Ollama at %s: %w", o.host, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama returned status %d", resp.StatusCode)
	}

	return nil
}

type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type ollamaChatResponse struct {
	Model     string        `json:"model"`
	Message   ollamaMessage `json:"message"`
	Done      bool          `json:"done"`
	DoneReason string       `json:"done_reason,omitempty"`
}

func (o *OllamaProvider) Complete(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}

	ollamaReq := ollamaChatRequest{
		Model:    model,
		Messages: convertMessages(req.Messages),
		Stream:   false,
		Options: &ollamaOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.host+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &CompletionResponse{
		Content:      ollamaResp.Message.Content,
		Model:        ollamaResp.Model,
		FinishReason: ollamaResp.DoneReason,
	}, nil
}

func (o *OllamaProvider) Stream(ctx context.Context, req *CompletionRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = o.model
	}

	ollamaReq := ollamaChatRequest{
		Model:    model,
		Messages: convertMessages(req.Messages),
		Stream:   true,
		Options: &ollamaOptions{
			Temperature: req.Temperature,
			NumPredict:  req.MaxTokens,
		},
	}

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", o.host+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := o.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("ollama error (status %d): %s", resp.StatusCode, string(body))
	}

	events := make(chan StreamEvent)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk ollamaChatResponse
			if err := json.Unmarshal(line, &chunk); err != nil {
				events <- StreamEvent{Error: err}
				return
			}

			if chunk.Done {
				events <- StreamEvent{Done: true}
				return
			}

			events <- StreamEvent{Chunk: chunk.Message.Content}
		}

		if err := scanner.Err(); err != nil {
			events <- StreamEvent{Error: err}
		}
	}()

	return events, nil
}

func convertMessages(msgs []Message) []ollamaMessage {
	result := make([]ollamaMessage, len(msgs))
	for i, m := range msgs {
		result[i] = ollamaMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}
	return result
}
```

---

### 3. Provider Factory

```
pulp/internal/llm/factory.go
```

```go
package llm

import (
	"fmt"

	"github.com/sant0-9/pulp/internal/config"
)

// NewProvider creates a provider from config
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
		// Will be implemented in Phase 9
		return nil, fmt.Errorf("groq provider not yet implemented")

	case "openai":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("openai requires an API key")
		}
		return nil, fmt.Errorf("openai provider not yet implemented")

	case "anthropic":
		if cfg.APIKey == "" {
			return nil, fmt.Errorf("anthropic requires an API key")
		}
		return nil, fmt.Errorf("anthropic provider not yet implemented")

	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}

// NewLocalProvider creates a provider for local extraction
func NewLocalProvider(cfg *config.Config) (Provider, error) {
	if cfg.Local == nil || !cfg.Local.Enabled {
		return nil, nil
	}

	switch cfg.Local.Provider {
	case "ollama":
		return NewOllamaProvider(cfg.Local.Host, cfg.Local.Model), nil
	default:
		return nil, fmt.Errorf("unknown local provider: %s", cfg.Local.Provider)
	}
}
```

---

### 4. Test Provider on Startup

Update `pulp/internal/tui/app.go` to test provider connection:

Add to imports:
```go
import (
	"github.com/sant0-9/pulp/internal/llm"
)
```

Add new message types:
```go
type providerReadyMsg struct{}
type providerErrorMsg struct{ error }
```

Add to state in `state.go`:
```go
type state struct {
	// ... existing fields ...

	// Provider
	provider      llm.Provider
	localProvider llm.Provider
	providerReady bool
	providerError error
}
```

Update `Init()` in app.go:
```go
func (a *App) Init() tea.Cmd {
	if a.state.needsSetup {
		a.view = viewSetup
		return textinput.Blink
	}

	// Test provider connection
	return tea.Batch(
		textinput.Blink,
		a.testProvider(),
	)
}

func (a *App) testProvider() tea.Cmd {
	return func() tea.Msg {
		provider, err := llm.NewProvider(a.state.config)
		if err != nil {
			return providerErrorMsg{err}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := provider.Ping(ctx); err != nil {
			return providerErrorMsg{err}
		}

		return providerReadyMsg{}
	}
}
```

Handle messages in `Update()`:
```go
case providerReadyMsg:
	a.state.providerReady = true
	provider, _ := llm.NewProvider(a.state.config)
	a.state.provider = provider
	return a, nil

case providerErrorMsg:
	a.state.providerError = msg.error
	return a, nil
```

---

### 5. Show Provider Status in Welcome

Update `view_welcome.go` to show connection status:

```go
func (a *App) renderWelcome() string {
	var b strings.Builder

	// Logo
	logoRendered := styleLogo.Render(logo)

	// Subtitle
	subtitle := styleSubtitle.Render("Document Intelligence")

	// Provider status
	var status string
	if a.state.providerError != nil {
		status = lipgloss.NewStyle().
			Foreground(colorError).
			Render(fmt.Sprintf("Provider error: %s", a.state.providerError))
	} else if a.state.providerReady {
		providerName := a.state.config.Provider
		status = lipgloss.NewStyle().
			Foreground(colorSuccess).
			Render(fmt.Sprintf("Connected to %s", providerName))
	} else {
		status = styleSubtitle.Render("Connecting...")
	}

	// Input (only show if ready)
	var inputSection string
	if a.state.providerReady {
		inputSection = styleBox.Copy().
			Width(60).
			BorderForeground(colorSecondary).
			Render(a.state.input.View())
	}

	// Instructions
	instructions := styleStatusBar.Render("[s] Settings  [?] Help  [Esc] Quit")

	// Combine
	content := lipgloss.JoinVertical(
		lipgloss.Center,
		logoRendered,
		subtitle,
		"",
		status,
		"",
		inputSection,
	)

	// Center and add status bar
	content = a.centerVertically(content)

	// Add status bar at bottom
	lines := strings.Count(content, "\n")
	padding := a.height - lines - 2
	if padding > 0 {
		content += strings.Repeat("\n", padding)
	}
	content += lipgloss.PlaceHorizontal(a.width, lipgloss.Center, instructions)

	return lipgloss.PlaceHorizontal(a.width, lipgloss.Center, content)
}
```

---

## Test

```bash
# Make sure Ollama is running
ollama serve

# In another terminal, pull a model if needed
ollama pull llama3.1:8b

# Build and run
go build -o pulp ./cmd/pulp
./pulp

# Expected:
# 1. Shows "Connecting..." briefly
# 2. Then shows "Connected to ollama"
# 3. Input field appears when connected

# Test without Ollama running:
# Stop ollama, run pulp
# Should show error message
```

---

## Done Checklist

- [ ] Provider interface defined
- [ ] Ollama provider implements interface
- [ ] Ping checks Ollama connection
- [ ] Complete sends request and gets response
- [ ] Stream returns channel of chunks
- [ ] Factory creates provider from config
- [ ] Welcome view shows connection status
- [ ] Error handling for connection failures

---

## Commit Message

```
feat: add LLM provider system with Ollama support

- Define Provider interface (Complete, Stream, Ping)
- Implement Ollama provider with streaming support
- Add provider factory for config-based creation
- Show provider connection status in welcome view
- Handle connection errors gracefully
```
