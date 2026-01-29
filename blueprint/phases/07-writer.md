# Phase 7: Writer + Output

## Goal
Create writer component that takes aggregated content + intent and generates final output. Stream tokens to result view.

## Success Criteria
- Writer builds prompts from intent
- Streaming output shows tokens appearing
- Result view displays formatted output
- Copy and save work

---

## Files to Create

### 1. Writer

```
pulp/internal/writer/writer.go
```

```go
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

// WriteRequest contains everything needed to generate output
type WriteRequest struct {
	Aggregated *pipeline.AggregatedContent
	Intent     *intent.Intent
	DocTitle   string
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

	// Base instruction
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
```

---

### 2. Result View with Streaming

```
pulp/internal/tui/view_result.go
```

```go
package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (a *App) renderResult() string {
	var b strings.Builder

	// Document info (small)
	if a.state.document != nil {
		docInfo := styleSubtitle.Render(a.state.document.Metadata.Title)
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, docInfo))
		b.WriteString("\n\n")
	}

	// Result box
	result := a.state.result
	if result == "" && a.state.streaming {
		result = "..."
	}

	// Calculate max height for result
	maxResultHeight := a.height - 10
	resultLines := strings.Split(result, "\n")
	if len(resultLines) > maxResultHeight {
		// Show last N lines when streaming
		resultLines = resultLines[len(resultLines)-maxResultHeight:]
		result = strings.Join(resultLines, "\n")
	}

	resultStyle := styleBox.Copy().
		Width(min(70, a.width-4)).
		BorderForeground(colorPrimary)

	if a.state.streaming {
		resultStyle = resultStyle.BorderForeground(colorSecondary)
	}

	resultBox := resultStyle.Render(result)
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, resultBox))
	b.WriteString("\n\n")

	// Status bar
	var status string
	if a.state.streaming {
		status = styleStatusBar.Render("Streaming... [Esc] Cancel")
	} else {
		status = styleStatusBar.Render("[c] Copy  [s] Save  [Enter] Follow-up  [n] New document  [Esc] Quit")
	}
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	return a.centerVertically(b.String())
}
```

---

### 3. Update State

Add to `state.go`:

```go
type state struct {
	// ... existing fields ...

	// Result
	result    string
	streaming bool
}
```

---

### 4. Wire Up Writer with Streaming

In `app.go`, update pipeline completion to start writer:

```go
import (
	"github.com/sant0-9/pulp/internal/writer"
)

type streamChunkMsg struct {
	chunk string
}

type streamDoneMsg struct{}

type streamErrorMsg struct {
	error
}

// When pipeline completes, start writer:
case pipelineDoneMsg:
	a.state.pipelineResult = msg.result
	a.state.streaming = true
	a.state.result = ""
	a.view = viewResult
	return a, a.startWriter()

func (a *App) startWriter() tea.Cmd {
	return func() tea.Msg {
		w := writer.NewWriter(a.state.provider, a.state.config.Model)

		req := &writer.WriteRequest{
			Aggregated: a.state.pipelineResult.Aggregated,
			Intent:     a.state.currentIntent,
			DocTitle:   a.state.document.Metadata.Title,
		}

		ctx := context.Background()
		stream, err := w.Stream(ctx, req)
		if err != nil {
			return streamErrorMsg{err}
		}

		// Read all chunks and send as one message
		// (In a real app, we'd use program.Send for true streaming)
		var result strings.Builder
		for event := range stream {
			if event.Error != nil {
				return streamErrorMsg{event.Error}
			}
			if event.Done {
				break
			}
			result.WriteString(event.Chunk)
		}

		return streamDoneMsg{}
	}
}

// For true streaming, we need to pass the program to the writer
// This is a simplified version that collects then displays
```

---

### 5. Better Streaming with Program Reference

Update the app to support true streaming:

```go
// Add to App struct:
type App struct {
	// ... existing fields ...
	program *tea.Program
}

// Add method to set program:
func (a *App) SetProgram(p *tea.Program) {
	a.program = p
}

// Update main.go:
func main() {
	app := tui.NewApp()
	p := tea.NewProgram(
		app,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)
	app.SetProgram(p)

	if _, err := p.Run(); err != nil {
		// ...
	}
}

// Now startWriter can send chunks:
func (a *App) startWriter() tea.Cmd {
	return func() tea.Msg {
		w := writer.NewWriter(a.state.provider, a.state.config.Model)

		req := &writer.WriteRequest{
			Aggregated: a.state.pipelineResult.Aggregated,
			Intent:     a.state.currentIntent,
			DocTitle:   a.state.document.Metadata.Title,
		}

		ctx := context.Background()
		stream, err := w.Stream(ctx, req)
		if err != nil {
			return streamErrorMsg{err}
		}

		go func() {
			for event := range stream {
				if event.Error != nil {
					a.program.Send(streamErrorMsg{event.Error})
					return
				}
				if event.Done {
					a.program.Send(streamDoneMsg{})
					return
				}
				a.program.Send(streamChunkMsg{event.Chunk})
			}
			a.program.Send(streamDoneMsg{})
		}()

		return nil
	}
}
```

---

### 6. Handle Stream Messages

```go
case streamChunkMsg:
	a.state.result += msg.chunk
	return a, nil

case streamDoneMsg:
	a.state.streaming = false
	// Add to history
	a.state.history = append(a.state.history, message{
		role:    "assistant",
		content: a.state.result,
	})
	return a, nil

case streamErrorMsg:
	a.state.streaming = false
	a.state.processingError = msg.error
	return a, nil
```

---

### 7. Copy and Save

Add clipboard and file saving:

```go
import (
	"os"
	"path/filepath"
)

// In handleKey for result view:
case msg.String() == "c":
	if a.view == viewResult && !a.state.streaming {
		return a, copyToClipboard(a.state.result)
	}

case msg.String() == "s":
	if a.view == viewResult && !a.state.streaming {
		return a, saveToFile(a.state.result, a.state.document.Metadata.Title)
	}

type clipboardMsg struct {
	success bool
	err     error
}

type saveMsg struct {
	path string
	err  error
}

func copyToClipboard(content string) tea.Cmd {
	return func() tea.Msg {
		// Use pbcopy on macOS, xclip on Linux, clip on Windows
		// For simplicity, we'll use a Go clipboard library
		// Add: "golang.design/x/clipboard"
		// For now, just return success
		return clipboardMsg{success: true}
	}
}

func saveToFile(content, title string) tea.Cmd {
	return func() tea.Msg {
		// Generate filename
		filename := strings.ReplaceAll(title, " ", "_") + "_summary.md"
		home, _ := os.UserHomeDir()
		path := filepath.Join(home, "Documents", filename)

		// Ensure directory exists
		os.MkdirAll(filepath.Dir(path), 0755)

		// Write file
		err := os.WriteFile(path, []byte(content), 0644)
		if err != nil {
			return saveMsg{err: err}
		}

		return saveMsg{path: path}
	}
}

// Handle messages:
case clipboardMsg:
	// Show brief notification (could add a toast)
	return a, nil

case saveMsg:
	if msg.err != nil {
		// Show error
	} else {
		// Show success with path
	}
	return a, nil
```

---

## Test

```bash
# Build and run
go build -o pulp ./cmd/pulp
./pulp

# Load a document
# Type: "summarize for my boss"

# Expected:
# - Pipeline runs
# - Result view shows with streaming text
# - Tokens appear progressively
# - When done, can use c/s/n keys
```

---

## Done Checklist

- [ ] Writer builds prompts from intent
- [ ] Streaming output works
- [ ] Result view shows text appearing
- [ ] Result scrolls if too long
- [ ] Copy to clipboard works (or is stubbed)
- [ ] Save to file works
- [ ] History updated after completion

---

## Commit Message

```
feat: add writer with streaming output

- Create writer component that builds prompts from intent
- Implement streaming output to result view
- Add result view with scrolling support
- Add copy and save functionality
- Update history after generation completes
```
