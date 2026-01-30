package pipeline

import (
	"context"
	"fmt"

	"github.com/sant0-9/pulp/internal/converter"
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
	extractor  *Extractor
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
func (p *Pipeline) Process(ctx context.Context, doc *converter.Document, _ *intent.Intent) (*Result, error) {
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
