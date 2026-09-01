package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"glossias/src/pkg/database"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestGradeProduceLive runs the real grader over testdata/produce_grading_samples.json
// and checks each score lands in the range a human grader would accept. It is
// the prompt-iteration loop for T13: skipped unless GRADING_LIVE=1 and an API
// key are set, because it costs money and needs the network. With DATABASE_URL
// set it grades with the prompt version active in that database (the one
// edited on the admin System page); without it, the built-in default.
//
// GRADING_LIVE_RUNS=N grades each case N times and checks the median score,
// so a single sampling outlier near a band edge doesn't fail the suite.
// Cases run in parallel, so N rounds still take about one round's wall time.
//
//	set -a; source .env; set +a
//	GRADING_LIVE=1 GRADING_LIVE_RUNS=3 go test ./src/pkg/models/ -run TestGradeProduceLive -v
func TestGradeProduceLive(t *testing.T) {
	if os.Getenv("GRADING_LIVE") != "1" {
		t.Skip("set GRADING_LIVE=1 (and ANTHROPIC_API_KEY) to run live grading samples")
	}
	grader, ok := NewAnthropicGraderFromEnv()
	if !ok {
		t.Skip("ANTHROPIC_API_KEY not set")
	}

	samples := loadGradingSamples(t)
	systemPrompt := liveGradingPrompt(t)
	runs := liveGradingRuns(t)

	// Subtests run in parallel, so the summary must wait for them: Cleanup on
	// the parent fires only after every parallel subtest has finished.
	var failures atomic.Int32
	t.Cleanup(func() {
		t.Logf("%d/%d cases in range (median of %d run(s) each)", len(samples.Cases)-int(failures.Load()), len(samples.Cases), runs)
	})

	for _, c := range samples.Cases {
		seg, ok := samples.Segments[c.Segment]
		if !ok {
			t.Fatalf("case references unknown segment %q", c.Segment)
		}
		t.Run(c.Note, func(t *testing.T) {
			t.Parallel()
			scores := make([]int, 0, runs)
			var feedback string
			for range runs {
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				grade, _, err := grader.GradeProduce(ctx, ProduceGradeRequest{
					SystemPrompt:            systemPrompt,
					ReferenceEnglish:        seg.Reference,
					HebrewText:              seg.Hebrew,
					StudentText:             c.Attempt,
					GrammarPointName:        seg.GrammarPoint,
					GrammarPointDescription: seg.Description,
				})
				cancel()
				if err != nil {
					t.Fatalf("grade: %v", err)
				}
				scores = append(scores, grade.Score)
				feedback = grade.Feedback
			}
			slices.Sort(scores)
			score := scores[len(scores)/2]
			inRange := score >= c.Min && score <= c.Max
			mark := "ok  "
			if !inRange {
				mark = "MISS"
				failures.Add(1)
			}
			detail := ""
			if runs > 1 {
				detail = fmt.Sprintf(" (runs: %v)", scores)
			}
			t.Logf("%s score=%3d want %3d–%3d%s  %q\n      feedback: %s", mark, score, c.Min, c.Max, detail, c.Attempt, feedback)
			if !inRange {
				t.Errorf("median score %d outside %d–%d", score, c.Min, c.Max)
			}
		})
	}
}

// liveGradingRuns reads GRADING_LIVE_RUNS: how many times to grade each case
// (the median score is checked). Defaults to 1.
func liveGradingRuns(t *testing.T) int {
	t.Helper()
	v := strings.TrimSpace(os.Getenv("GRADING_LIVE_RUNS"))
	if v == "" {
		return 1
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		t.Fatalf("GRADING_LIVE_RUNS must be a positive integer, got %q", v)
	}
	return n
}

// liveGradingPrompt returns the system prompt the live run should use: the
// version currently active in the database when DATABASE_URL is set (so an
// edit made on the System page can be checked against the samples before or
// after activating it), otherwise the built-in default.
func liveGradingPrompt(t *testing.T) string {
	t.Helper()
	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		t.Log("DATABASE_URL not set; grading with the built-in default prompt")
		return DefaultGradingSystemPrompt
	}

	store, err := database.InitDBWithReconnect(dbURL)
	if err != nil {
		t.Fatalf("connect to database for active prompt: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	SetDB(store.RawConn())

	prompt, err := GetActiveProduceGradingPrompt(context.Background())
	if errors.Is(err, ErrNotFound) {
		t.Log("no prompt stored in the database; grading with the built-in default prompt")
		return DefaultGradingSystemPrompt
	}
	if err != nil {
		t.Fatalf("read active grading prompt: %v", err)
	}
	t.Logf("grading with prompt version %d (%s)", prompt.ID, prompt.CreatedAt.Format(time.RFC3339))
	return prompt.Text
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
