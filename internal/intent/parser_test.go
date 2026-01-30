package intent

import (
	"testing"
)

func TestQuickParse(t *testing.T) {
	p := &Parser{}

	tests := []struct {
		name        string
		instruction string
		wantAction  Action
		wantTone    string
		wantAudience string
		wantFormat  string
	}{
		{
			name:        "simple summarize",
			instruction: "summarize this",
			wantAction:  ActionSummarize,
			wantTone:    "neutral",
			wantAudience: "general",
			wantFormat:  "prose",
		},
		{
			name:        "summarize for boss",
			instruction: "summarize for my boss",
			wantAction:  ActionSummarize,
			wantTone:    "professional",
			wantAudience: "executive",
			wantFormat:  "prose",
		},
		{
			name:        "explain like I'm 5",
			instruction: "explain like I'm 5",
			wantAction:  ActionExplain,
			wantTone:    "simple",
			wantAudience: "child",
			wantFormat:  "prose",
		},
		{
			name:        "bullet points",
			instruction: "give me bullet points",
			wantAction:  ActionSummarize,
			wantTone:    "neutral",
			wantAudience: "general",
			wantFormat:  "bullets",
		},
		{
			name:        "action items",
			instruction: "extract action items",
			wantAction:  ActionExtract,
			wantTone:    "neutral",
			wantAudience: "general",
			wantFormat:  "bullets",
		},
		{
			name:        "key points for executive",
			instruction: "key points for my boss",
			wantAction:  ActionExtract,
			wantTone:    "professional",
			wantAudience: "executive",
			wantFormat:  "bullets",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.quickParse(tt.instruction)
			if got == nil {
				t.Fatal("quickParse returned nil, expected intent")
			}
			if got.Action != tt.wantAction {
				t.Errorf("Action = %v, want %v", got.Action, tt.wantAction)
			}
			if got.Tone != tt.wantTone {
				t.Errorf("Tone = %v, want %v", got.Tone, tt.wantTone)
			}
			if got.Audience != tt.wantAudience {
				t.Errorf("Audience = %v, want %v", got.Audience, tt.wantAudience)
			}
			if got.Format != tt.wantFormat {
				t.Errorf("Format = %v, want %v", got.Format, tt.wantFormat)
			}
		})
	}
}

func TestQuickParseReturnsNilForUnknown(t *testing.T) {
	p := &Parser{}

	// Unknown pattern should return nil to trigger LLM parsing
	got := p.quickParse("do something creative with this")
	if got != nil {
		t.Error("expected nil for unknown pattern, got intent")
	}
}
