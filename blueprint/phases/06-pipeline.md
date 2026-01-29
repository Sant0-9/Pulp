# Phase 6: Pipeline

## Goal
Create processing pipeline that takes document + intent and extracts key information. Show animated progress in TUI.

## Success Criteria
- Pipeline orchestrates: chunk -> extract -> aggregate
- Progress view shows animated stages
- Extraction uses local LLM
- Aggregated content ready for writer

---

## Files to Create

### 1. Chunker

```
pulp/internal/pipeline/chunker.go
```

```go
package pipeline

import (
	"strings"
	"unicode/utf8"
)

// Chunk represents a piece of document content
type Chunk struct {
	ID       int
	Content  string
	Section  string
	Position int
}

// ChunkDocument splits document into semantic chunks
func ChunkDocument(content string, maxChunkSize int) []Chunk {
	if maxChunkSize <= 0 {
		maxChunkSize = 1500 // Default ~375 tokens
	}

	var chunks []Chunk

	// Split by double newlines (paragraphs/sections)
	sections := strings.Split(content, "\n\n")

	var currentChunk strings.Builder
	var currentSection string
	chunkID := 0

	for _, section := range sections {
		section = strings.TrimSpace(section)
		if section == "" {
			continue
		}

		// Check if this is a header
		if strings.HasPrefix(section, "#") {
			lines := strings.SplitN(section, "\n", 2)
			currentSection = strings.TrimLeft(lines[0], "# ")
			if len(lines) > 1 {
				section = lines[1]
			} else {
				continue
			}
		}

		// Check if adding this would exceed max size
		if currentChunk.Len()+len(section)+2 > maxChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			chunks = append(chunks, Chunk{
				ID:       chunkID,
				Content:  currentChunk.String(),
				Section:  currentSection,
				Position: chunkID,
			})
			chunkID++
			currentChunk.Reset()
		}

		// Add section to current chunk
		if currentChunk.Len() > 0 {
			currentChunk.WriteString("\n\n")
		}
		currentChunk.WriteString(section)
	}

	// Don't forget the last chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, Chunk{
			ID:       chunkID,
			Content:  currentChunk.String(),
			Section:  currentSection,
			Position: chunkID,
		})
	}

	return chunks
}

// EstimateTokens estimates token count (rough: 4 chars per token)
func EstimateTokens(text string) int {
	return utf8.RuneCountInString(text) / 4
}
```

---

### 2. Extractor

```
pulp/internal/pipeline/extractor.go
```

```go
package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/sant0-9/pulp/internal/llm"
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

const extractionPrompt = `Extract key information from this text. Return JSON only:
{
  "key_points": ["point 1", "point 2", "point 3"],
  "entities": ["names", "organizations", "dates", "numbers mentioned"],
  "facts": ["specific factual claims"],
  "summary": "one sentence summary"
}

Be specific. Include names, numbers, dates. No generic statements.
Return ONLY valid JSON.`

func (e *Extractor) Extract(ctx context.Context, chunk Chunk) (*Extraction, error) {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	req := &llm.CompletionRequest{
		Model: e.model,
		Messages: []llm.Message{
			{Role: "system", Content: extractionPrompt},
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
```

---

### 3. Aggregator

```
pulp/internal/pipeline/aggregator.go
```

```go
package pipeline

import (
	"strings"
)

// AggregatedContent contains all extracted information
type AggregatedContent struct {
	KeyPoints []string
	Entities  []string
	Facts     []string
	Summaries []string
	WordCount int
}

// Aggregate combines extractions from all chunks
func Aggregate(extractions []*Extraction) *AggregatedContent {
	agg := &AggregatedContent{}

	seen := make(map[string]bool) // For deduplication

	for _, ext := range extractions {
		// Add key points (dedupe)
		for _, kp := range ext.KeyPoints {
			kp = strings.TrimSpace(kp)
			if kp != "" && !seen[strings.ToLower(kp)] {
				agg.KeyPoints = append(agg.KeyPoints, kp)
				seen[strings.ToLower(kp)] = true
			}
		}

		// Add entities (dedupe)
		for _, ent := range ext.Entities {
			ent = strings.TrimSpace(ent)
			if ent != "" && !seen["ent:"+strings.ToLower(ent)] {
				agg.Entities = append(agg.Entities, ent)
				seen["ent:"+strings.ToLower(ent)] = true
			}
		}

		// Add facts (dedupe)
		for _, fact := range ext.Facts {
			fact = strings.TrimSpace(fact)
			if fact != "" && !seen["fact:"+strings.ToLower(fact)] {
				agg.Facts = append(agg.Facts, fact)
				seen["fact:"+strings.ToLower(fact)] = true
			}
		}

		// Add summary
		if ext.Summary != "" {
			agg.Summaries = append(agg.Summaries, ext.Summary)
		}
	}

	// Estimate word count
	for _, kp := range agg.KeyPoints {
		agg.WordCount += len(strings.Fields(kp))
	}
	for _, s := range agg.Summaries {
		agg.WordCount += len(strings.Fields(s))
	}

	return agg
}

// FormatForWriter formats aggregated content for the writer
func (a *AggregatedContent) FormatForWriter() string {
	var b strings.Builder

	if len(a.Summaries) > 0 {
		b.WriteString("SECTION SUMMARIES:\n")
		for _, s := range a.Summaries {
			b.WriteString("- " + s + "\n")
		}
		b.WriteString("\n")
	}

	if len(a.KeyPoints) > 0 {
		b.WriteString("KEY POINTS:\n")
		for _, kp := range a.KeyPoints {
			b.WriteString("- " + kp + "\n")
		}
		b.WriteString("\n")
	}

	if len(a.Facts) > 0 {
		b.WriteString("FACTS:\n")
		for _, f := range a.Facts {
			b.WriteString("- " + f + "\n")
		}
		b.WriteString("\n")
	}

	if len(a.Entities) > 0 {
		b.WriteString("KEY ENTITIES: " + strings.Join(a.Entities, ", ") + "\n")
	}

	return b.String()
}
```

---

### 4. Pipeline Orchestrator

```
pulp/internal/pipeline/pipeline.go
```

```go
package pipeline

import (
	"context"
	"fmt"

	"github.com/sant0-9/pulp/internal/document"
	"github.com/sant0-9/pulp/internal/intent"
	"github.com/sant0-9/pulp/internal/llm"
)

// Stage represents a pipeline stage
type Stage int

const (
	StageChunking Stage = iota
	StageExtracting
	StageAggregating
	StageDone
)

func (s Stage) String() string {
	switch s {
	case StageChunking:
		return "Chunking"
	case StageExtracting:
		return "Extracting"
	case StageAggregating:
		return "Aggregating"
	case StageDone:
		return "Done"
	default:
		return "Unknown"
	}
}

// Progress represents pipeline progress
type Progress struct {
	Stage       Stage
	StageIndex  int
	TotalStages int
	ItemIndex   int
	TotalItems  int
	Message     string
}

// Result contains pipeline output
type Result struct {
	Aggregated *AggregatedContent
	Chunks     []Chunk
}

// Pipeline processes documents
type Pipeline struct {
	extractor *Extractor
	onProgress func(Progress)
}

// NewPipeline creates a new pipeline
func NewPipeline(provider llm.Provider, model string) *Pipeline {
	return &Pipeline{
		extractor: NewExtractor(provider, model),
	}
}

// SetProgressCallback sets the progress callback
func (p *Pipeline) SetProgressCallback(fn func(Progress)) {
	p.onProgress = fn
}

func (p *Pipeline) progress(pr Progress) {
	if p.onProgress != nil {
		p.onProgress(pr)
	}
}

// Process runs the pipeline
func (p *Pipeline) Process(ctx context.Context, doc *document.Document, intent *intent.Intent) (*Result, error) {
	// Stage 1: Chunking
	p.progress(Progress{
		Stage:       StageChunking,
		StageIndex:  0,
		TotalStages: 3,
		Message:     "Splitting document into chunks...",
	})

	chunks := ChunkDocument(doc.Content, 1500)
	if len(chunks) == 0 {
		return nil, fmt.Errorf("no content to process")
	}

	// Stage 2: Extraction
	p.progress(Progress{
		Stage:       StageExtracting,
		StageIndex:  1,
		TotalStages: 3,
		TotalItems:  len(chunks),
		Message:     fmt.Sprintf("Extracting from %d chunks...", len(chunks)),
	})

	var extractions []*Extraction
	for i, chunk := range chunks {
		p.progress(Progress{
			Stage:       StageExtracting,
			StageIndex:  1,
			TotalStages: 3,
			ItemIndex:   i + 1,
			TotalItems:  len(chunks),
			Message:     fmt.Sprintf("Extracting chunk %d/%d", i+1, len(chunks)),
		})

		ext, err := p.extractor.Extract(ctx, chunk)
		if err != nil {
			// Log but continue
			continue
		}
		extractions = append(extractions, ext)
	}

	// Stage 3: Aggregation
	p.progress(Progress{
		Stage:       StageAggregating,
		StageIndex:  2,
		TotalStages: 3,
		Message:     "Aggregating results...",
	})

	aggregated := Aggregate(extractions)

	// Done
	p.progress(Progress{
		Stage:       StageDone,
		StageIndex:  3,
		TotalStages: 3,
		Message:     "Processing complete",
	})

	return &Result{
		Aggregated: aggregated,
		Chunks:     chunks,
	}, nil
}
```

---

### 5. Progress View

```
pulp/internal/tui/view_processing.go
```

```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sant0-9/pulp/internal/pipeline"
)

func (a *App) renderProcessing() string {
	var b strings.Builder

	// Title
	title := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render("Processing")
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, title))
	b.WriteString("\n\n")

	// Document info
	if a.state.document != nil {
		docInfo := styleSubtitle.Render(a.state.document.Metadata.Title)
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, docInfo))
		b.WriteString("\n\n")
	}

	// Intent info
	if a.state.currentIntent != nil {
		intentInfo := styleSubtitle.Render(fmt.Sprintf(
			"Task: %s | Tone: %s",
			a.state.currentIntent.Action,
			a.state.currentIntent.Tone,
		))
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, intentInfo))
		b.WriteString("\n\n")
	}

	// Progress stages
	stages := []string{"Chunking", "Extracting", "Aggregating"}
	currentStage := 0
	if a.state.pipelineProgress != nil {
		currentStage = a.state.pipelineProgress.StageIndex
	}

	var stageLines []string
	for i, stage := range stages {
		var icon string
		var style lipgloss.Style

		if i < currentStage {
			// Completed
			icon = "●"
			style = lipgloss.NewStyle().Foreground(colorSuccess)
		} else if i == currentStage {
			// Current
			icon = "◐"
			style = lipgloss.NewStyle().Foreground(colorSecondary).Bold(true)
		} else {
			// Pending
			icon = "○"
			style = lipgloss.NewStyle().Foreground(colorMuted)
		}

		// Progress bar for extraction
		var progressBar string
		if i == currentStage && a.state.pipelineProgress != nil {
			p := a.state.pipelineProgress
			if p.TotalItems > 0 {
				pct := float64(p.ItemIndex) / float64(p.TotalItems)
				filled := int(pct * 30)
				empty := 30 - filled
				progressBar = "  " +
					lipgloss.NewStyle().Foreground(colorSecondary).Render(strings.Repeat("━", filled)) +
					lipgloss.NewStyle().Foreground(colorMuted).Render(strings.Repeat("░", empty)) +
					fmt.Sprintf("  %d/%d", p.ItemIndex, p.TotalItems)
			}
		}

		line := style.Render(fmt.Sprintf("  %s  %-12s", icon, stage)) + progressBar
		stageLines = append(stageLines, line)
	}

	stagesBox := styleBox.Copy().
		Width(min(60, a.width-4)).
		Render(strings.Join(stageLines, "\n"))
	b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, stagesBox))
	b.WriteString("\n\n")

	// Message
	if a.state.pipelineProgress != nil && a.state.pipelineProgress.Message != "" {
		msg := styleSubtitle.Render(a.state.pipelineProgress.Message)
		b.WriteString(lipgloss.PlaceHorizontal(a.width, lipgloss.Center, msg))
	}

	return a.centerVertically(b.String())
}
```

---

### 6. Update State

Add to `state.go`:

```go
import (
	"github.com/sant0-9/pulp/internal/pipeline"
)

type state struct {
	// ... existing fields ...

	// Pipeline
	pipelineProgress *pipeline.Progress
	pipelineResult   *pipeline.Result
	processingError  error
}
```

---

### 7. Wire Up Pipeline

In `app.go`, add messages and handlers:

```go
type pipelineProgressMsg struct {
	progress pipeline.Progress
}

type pipelineDoneMsg struct {
	result *pipeline.Result
}

type pipelineErrorMsg struct {
	error
}

// After intent is parsed, start pipeline:
case intentParsedMsg:
	a.state.parsingIntent = false
	a.state.currentIntent = msg.intent
	a.view = viewProcessing
	return a, a.runPipeline()

func (a *App) runPipeline() tea.Cmd {
	return func() tea.Msg {
		pipe := pipeline.NewPipeline(a.state.provider, a.state.config.Model)

		// Set up progress callback that sends messages
		// (We'll need to pass program reference for this)
		// For now, we'll handle this differently

		ctx := context.Background()
		result, err := pipe.Process(ctx, a.state.document, a.state.currentIntent)
		if err != nil {
			return pipelineErrorMsg{err}
		}

		return pipelineDoneMsg{result}
	}
}

// Handle in Update:
case pipelineProgressMsg:
	a.state.pipelineProgress = &msg.progress
	return a, nil

case pipelineDoneMsg:
	a.state.pipelineResult = msg.result
	a.view = viewResult
	return a, nil

case pipelineErrorMsg:
	a.state.processingError = msg.error
	a.view = viewDocument
	return a, nil
```

---

### 8. Update View Switch

```go
case viewProcessing:
	return a.renderProcessing()
```

---

## Test

```bash
# Make sure Ollama is running
ollama serve

# Build and run
go build -o pulp ./cmd/pulp
./pulp

# Load a document
# Type an instruction: "summarize"
# Watch the progress view

# Expected:
# - See chunking stage
# - See extraction progress (X/Y chunks)
# - See aggregation stage
# - Transition to result view (empty for now)
```

---

## Done Checklist

- [ ] Chunker splits document into chunks
- [ ] Extractor extracts key points from chunks using LLM
- [ ] Aggregator combines and dedupes extractions
- [ ] Pipeline orchestrates all stages
- [ ] Progress view shows animated stages
- [ ] Extraction progress shows X/Y
- [ ] Errors handled gracefully

---

## Commit Message

```
feat: add processing pipeline with extraction

- Create semantic chunker for documents
- Add LLM-based extractor for key points
- Build aggregator to combine extractions
- Create pipeline orchestrator with stages
- Add animated progress view with stage indicators
```
