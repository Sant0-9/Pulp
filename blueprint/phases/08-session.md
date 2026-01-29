# Phase 8: Session

## Goal
Enable follow-up conversations. After initial result, user can say "make it shorter" or "add more detail" and the context is preserved.

## Success Criteria
- Conversation history maintained
- Follow-up instructions work naturally
- "make it shorter" understands "it" = previous output
- Can revise multiple times
- Input shown after result for easy follow-up

---

## Changes Required

### 1. Update State for History

Already have `history []message` in state. Ensure it's properly used:

```go
// In state.go, ensure:
type message struct {
	role    string // "user" or "assistant"
	content string
}

type state struct {
	// ... existing fields ...

	// History for follow-ups
	history []message

	// Track if this is a follow-up
	isFollowUp bool
}
```

---

### 2. Update Result View with Input

Show the input box in the result view for follow-ups:

```
pulp/internal/tui/view_result.go
```

```go
func (a *App) renderResult() string {
	var b strings.Builder

	// Document info (small)
	if a.state.document != nil {
		docInfo := styleSubtitle.Render(a.state.document.Metadata.Title)
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, docInfo))
		b.WriteString("\n")
	}

	// Show what was asked (user message)
	if a.state.currentIntent != nil {
		asked := styleSubtitle.Render(fmt.Sprintf("> %s", a.state.currentIntent.RawPrompt))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, asked))
		b.WriteString("\n\n")
	}

	// Result box
	result := a.state.result
	if result == "" && a.state.streaming {
		result = "Thinking..."
	}

	// Calculate max height for result
	maxResultHeight := a.height - 14
	resultLines := strings.Split(result, "\n")
	if len(resultLines) > maxResultHeight && maxResultHeight > 0 {
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

	// Input for follow-up (only show when not streaming)
	if !a.state.streaming {
		a.state.input.Placeholder = "Follow-up or revision..."
		inputBox := styleBox.Copy().
			Width(min(70, a.width-4)).
			BorderForeground(colorMuted).
			Render(a.state.input.View())
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, inputBox))
		b.WriteString("\n\n")
	}

	// Status bar
	var status string
	if a.state.streaming {
		status = styleStatusBar.Render("Generating... [Esc] Cancel")
	} else {
		status = styleStatusBar.Render("[Enter] Submit  [c] Copy  [s] Save  [n] New document  [Esc] Quit")
	}
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, status))

	return a.centerVertically(b.String())
}
```

---

### 3. Handle Follow-Up Input

In `app.go`, handle Enter in result view:

```go
case key.Matches(msg, keys.Enter):
	// ... existing document view handling ...

	// Handle result view follow-up
	if a.view == viewResult && !a.state.streaming {
		instruction := strings.TrimSpace(a.state.input.Value())
		if instruction != "" {
			// Add user message to history
			a.state.history = append(a.state.history, message{
				role:    "user",
				content: instruction,
			})
			a.state.isFollowUp = true
			a.state.input.Reset()
			return a, a.parseIntent(instruction)
		}
	}
```

---

### 4. Update Writer to Include History

Update the writer to include conversation context:

```
pulp/internal/writer/writer.go
```

```go
// Update WriteRequest:
type WriteRequest struct {
	Aggregated  *pipeline.AggregatedContent
	Intent      *intent.Intent
	DocTitle    string
	History     []Message // Add this
	IsFollowUp  bool      // Add this
	PreviousResult string // Add this
}

type Message struct {
	Role    string
	Content string
}

// Update buildPrompt to handle follow-ups:
func (w *Writer) buildPrompt(req *WriteRequest) *prompt {
	var system strings.Builder
	i := req.Intent

	if req.IsFollowUp && req.PreviousResult != "" {
		// Follow-up mode
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

	// Original prompt building logic (unchanged)
	// ... rest of existing buildPrompt code ...
}
```

---

### 5. Update App to Pass History

When starting the writer, pass the history:

```go
func (a *App) startWriter() tea.Cmd {
	return func() tea.Msg {
		w := writer.NewWriter(a.state.provider, a.state.config.Model)

		// Convert history to writer format
		var history []writer.Message
		for _, m := range a.state.history {
			history = append(history, writer.Message{
				Role:    m.role,
				Content: m.content,
			})
		}

		// Get previous result for follow-ups
		var previousResult string
		if a.state.isFollowUp && len(a.state.history) > 0 {
			// Find last assistant message
			for i := len(a.state.history) - 1; i >= 0; i-- {
				if a.state.history[i].role == "assistant" {
					previousResult = a.state.history[i].content
					break
				}
			}
		}

		req := &writer.WriteRequest{
			Aggregated:     a.state.pipelineResult.Aggregated,
			Intent:         a.state.currentIntent,
			DocTitle:       a.state.document.Metadata.Title,
			History:        history,
			IsFollowUp:     a.state.isFollowUp,
			PreviousResult: previousResult,
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

### 6. Skip Pipeline for Follow-Ups

For follow-ups, we don't need to re-run the full pipeline:

```go
case intentParsedMsg:
	a.state.parsingIntent = false
	a.state.currentIntent = msg.intent

	if a.state.isFollowUp {
		// Skip pipeline, go straight to writer
		a.state.streaming = true
		a.state.result = ""
		a.view = viewResult
		return a, a.startWriter()
	}

	// First time: run full pipeline
	a.view = viewProcessing
	return a, a.runPipeline()
```

---

### 7. Reset Follow-Up Flag on New Document

```go
case msg.String() == "n":
	if a.view == viewDocument || a.view == viewResult {
		a.state.document = nil
		a.state.documentPath = ""
		a.state.history = nil       // Clear history
		a.state.isFollowUp = false  // Reset flag
		a.state.result = ""
		a.state.pipelineResult = nil
		a.state.currentIntent = nil
		a.state.input.Reset()
		a.state.input.Placeholder = "Drop a file or type a path..."
		a.view = viewWelcome
		return a, nil
	}
```

---

### 8. Focus Input After Stream Done

```go
case streamDoneMsg:
	a.state.streaming = false
	a.state.history = append(a.state.history, message{
		role:    "assistant",
		content: a.state.result,
	})
	a.state.input.Focus()  // Focus input for follow-up
	return a, textinput.Blink
```

---

## Test

```bash
go build -o pulp ./cmd/pulp
./pulp

# Load a document
# Type: "summarize"
# Wait for result

# Then type: "make it shorter"
# Should revise the summary

# Then type: "add bullet points"
# Should add bullets to the shorter version

# Then type: "more formal tone"
# Should make it more formal
```

---

## Example Flow

```
> research-paper.pdf
> summarize the key findings

[Summary generated]

> make it shorter, just the main 3 points

[Shorter version with 3 points]

> now format as an executive email

[Email format generated]
```

---

## Done Checklist

- [ ] History stored correctly
- [ ] Follow-up skips pipeline (uses cached extraction)
- [ ] "make it shorter" works
- [ ] "add bullets" works
- [ ] Input shown in result view
- [ ] Multiple revisions work
- [ ] New document clears history

---

## Commit Message

```
feat: add session support for follow-up conversations

- Maintain conversation history for context
- Skip pipeline on follow-ups (reuse extraction)
- Writer handles revision requests
- Show input in result view for easy follow-up
- Clear history on new document
```
