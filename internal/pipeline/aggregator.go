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
