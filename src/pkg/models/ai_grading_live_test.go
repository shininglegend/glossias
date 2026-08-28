package models

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestGradeProduceLive runs the real grader over testdata/produce_grading_samples.json
// and checks each score lands in the range a human grader would accept. It is
// the prompt-iteration loop for T13: skipped unless GRADING_LIVE=1 and an API
// key are set, because it costs money and needs the network.
//
//	ANTHROPIC_API_KEY=... GRADING_LIVE=1 go test ./src/pkg/models/ -run TestGradeProduceLive -v
func TestGradeProduceLive(t *testing.T) {
	if os.Getenv("GRADING_LIVE") != "1" {
		t.Skip("set GRADING_LIVE=1 (and ANTHROPIC_API_KEY) to run live grading samples")
	}
	grader, ok := NewAnthropicGraderFromEnv()
	if !ok {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	samples := loadGradingSamples(t)

	var failures int
	for _, c := range samples.Cases {
		seg, ok := samples.Segments[c.Segment]
		if !ok {
			t.Fatalf("case references unknown segment %q", c.Segment)
		}
		t.Run(c.Note, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			grade, _, err := grader.GradeProduce(ctx, ProduceGradeRequest{
				ReferenceEnglish:        seg.Reference,
				HebrewText:              seg.Hebrew,
				StudentText:             c.Attempt,
				GrammarPointName:        seg.GrammarPoint,
				GrammarPointDescription: seg.Description,
			})
			if err != nil {
				t.Fatalf("grade: %v", err)
			}
			inRange := grade.Score >= c.Min && grade.Score <= c.Max
			mark := "ok  "
			if !inRange {
				mark = "MISS"
				failures++
			}
			t.Logf("%s score=%3d want %3d–%3d  %q\n      feedback: %s", mark, grade.Score, c.Min, c.Max, c.Attempt, grade.Feedback)
			if !inRange {
				t.Errorf("score %d outside %d–%d", grade.Score, c.Min, c.Max)
			}
		})
	}
	t.Logf("%d/%d cases in range", len(samples.Cases)-failures, len(samples.Cases))
}

type gradingSamples struct {
	Segments map[string]struct {
		Hebrew       string `json:"hebrew"`
		Reference    string `json:"reference"`
		GrammarPoint string `json:"grammarPoint"`
		Description  string `json:"description"`
	} `json:"segments"`
	Cases []struct {
		Segment string `json:"segment"`
		Attempt string `json:"attempt"`
		Min     int    `json:"min"`
		Max     int    `json:"max"`
		Note    string `json:"note"`
	} `json:"cases"`
}

func loadGradingSamples(t *testing.T) gradingSamples {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "produce_grading_samples.json"))
	if err != nil {
		t.Fatalf("read samples: %v", err)
	}
	var s gradingSamples
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatalf("parse samples: %v", err)
	}
	return s
}

// TestGradingSamplesWellFormed keeps the samples file valid even when the
// live test is skipped.
func TestGradingSamplesWellFormed(t *testing.T) {
	s := loadGradingSamples(t)
	if len(s.Cases) == 0 {
		t.Fatal("no cases")
	}
	for _, c := range s.Cases {
		if _, ok := s.Segments[c.Segment]; !ok {
			t.Errorf("case %q references unknown segment %q", c.Note, c.Segment)
		}
		if c.Min < 0 || c.Max > 100 || c.Min > c.Max {
			t.Errorf("case %q has bad range %d–%d", c.Note, c.Min, c.Max)
		}
	}
}
