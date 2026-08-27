package models

import (
	"strings"
	"testing"
)

// fullTargetWord builds a target word with both assets attached.
func fullTargetWord(id int, lexicalForm string) TargetVocabulary {
	return TargetVocabulary{
		ID:               id,
		StoryID:          1,
		LexicalForm:      lexicalForm,
		AudioPath:        "stories/1/word_audio_" + lexicalForm,
		AudioBucket:      "audio-files",
		CorrectImagePath: "stories/1/image_target_vocab_" + lexicalForm,
		ImageBucket:      "images",
	}
}

// fiveTargetWords is the shape a fully authored Identify phase has.
func fiveTargetWords() ([]TargetVocabulary, map[string]int) {
	forms := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	words := make([]TargetVocabulary, 0, len(forms))
	occurrences := make(map[string]int, len(forms))
	for i, form := range forms {
		words = append(words, fullTargetWord(i+1, form))
		occurrences[form] = MinTargetWordOccurrences
	}
	return words, occurrences
}

func TestValidateTargetVocabulary(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(words []TargetVocabulary, occurrences map[string]int) []TargetVocabulary
		wantReady bool
		wantIssue string
	}{
		{
			name:      "five complete words with enough occurrences is ready",
			mutate:    func(w []TargetVocabulary, _ map[string]int) []TargetVocabulary { return w },
			wantReady: true,
		},
		{
			name: "four words is not enough",
			mutate: func(w []TargetVocabulary, _ map[string]int) []TargetVocabulary {
				return w[:4]
			},
			wantIssue: "exactly 5 are required",
		},
		{
			name: "a sixth word is too many",
			mutate: func(w []TargetVocabulary, occurrences map[string]int) []TargetVocabulary {
				occurrences["zeta"] = MinTargetWordOccurrences
				return append(w, fullTargetWord(6, "zeta"))
			},
			wantIssue: "exactly 5 are required",
		},
		{
			name: "a word appearing once is rejected",
			mutate: func(w []TargetVocabulary, occurrences map[string]int) []TargetVocabulary {
				occurrences["beta"] = 1
				return w
			},
			wantIssue: "at least 2 annotated occurrences are required",
		},
		{
			name: "a word absent from the story text is rejected",
			mutate: func(w []TargetVocabulary, occurrences map[string]int) []TargetVocabulary {
				delete(occurrences, "gamma")
				return w
			},
			wantIssue: `"gamma" appears 0 time(s)`,
		},
		{
			name: "a word missing its audio is rejected",
			mutate: func(w []TargetVocabulary, _ map[string]int) []TargetVocabulary {
				w[0].AudioPath = ""
				return w
			},
			wantIssue: "has no pronunciation audio",
		},
		{
			name: "a path without a bucket does not count as an asset",
			mutate: func(w []TargetVocabulary, _ map[string]int) []TargetVocabulary {
				w[1].ImageBucket = ""
				return w
			},
			wantIssue: "has no picture",
		},
		{
			name: "a word missing its picture is rejected",
			mutate: func(w []TargetVocabulary, _ map[string]int) []TargetVocabulary {
				w[2].CorrectImagePath = ""
				return w
			},
			wantIssue: "has no picture",
		},
		{
			name: "an empty lexical form is rejected",
			mutate: func(w []TargetVocabulary, _ map[string]int) []TargetVocabulary {
				w[3].LexicalForm = "   "
				return w
			},
			wantIssue: "lexical form is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			words, occurrences := fiveTargetWords()
			got := ValidateTargetVocabulary(tt.mutate(words, occurrences), occurrences)

			if got.Phase != "identify" {
				t.Errorf("Phase = %q, want %q", got.Phase, "identify")
			}
			if got.Ready != tt.wantReady {
				t.Errorf("Ready = %v, want %v (issues: %v)", got.Ready, tt.wantReady, got.Issues)
			}
			if tt.wantIssue != "" && !hasIssueContaining(got.Issues, tt.wantIssue) {
				t.Errorf("expected an issue containing %q, got %v", tt.wantIssue, got.Issues)
			}
		})
	}
}

func TestValidateTargetVocabularyEmptyStory(t *testing.T) {
	got := ValidateTargetVocabulary(nil, nil)

	if got.Ready {
		t.Error("an unauthored story must not be ready")
	}
	if !hasIssueContaining(got.Issues, "story has 0 target words") {
		t.Errorf("expected a count issue, got %v", got.Issues)
	}
}

// twoProduceSegments is the shape a fully authored Produce phase has.
func twoProduceSegments() []ProduceSegment {
	grammarPointID := 7
	segments := make([]ProduceSegment, 0, ProduceSegmentsPerStory)
	for order := 1; order <= ProduceSegmentsPerStory; order++ {
		segments = append(segments, ProduceSegment{
			ID:              order,
			StoryID:         1,
			SegmentOrder:    order,
			EnglishText:     "the dog ran home",
			ReferenceHebrew: "הכלב רץ הביתה",
			GrammarPointID:  &grammarPointID,
		})
	}
	return segments
}

func TestValidateProduceContent(t *testing.T) {
	tests := []struct {
		name        string
		segments    func() []ProduceSegment
		explanation string
		wantReady   bool
		wantIssue   string
	}{
		{
			name:        "two complete segments plus an explanation is ready",
			segments:    twoProduceSegments,
			explanation: "Both segments use the construct state.",
			wantReady:   true,
		},
		{
			name:        "a missing explanation is rejected",
			segments:    twoProduceSegments,
			explanation: "  ",
			wantIssue:   "explanation is empty",
		},
		{
			name: "one segment is not enough",
			segments: func() []ProduceSegment {
				return twoProduceSegments()[:1]
			},
			explanation: "explained",
			wantIssue:   "no segment authored at position 2",
		},
		{
			name: "duplicate positions are rejected",
			segments: func() []ProduceSegment {
				segments := twoProduceSegments()
				segments[1].SegmentOrder = 1
				return segments
			},
			explanation: "explained",
			wantIssue:   "more than one segment at position 1",
		},
		{
			name: "an empty reference translation is rejected",
			segments: func() []ProduceSegment {
				segments := twoProduceSegments()
				segments[0].ReferenceHebrew = ""
				return segments
			},
			explanation: "explained",
			wantIssue:   "reference translation is empty",
		},
		{
			name: "an empty English prompt is rejected",
			segments: func() []ProduceSegment {
				segments := twoProduceSegments()
				segments[1].EnglishText = "\t"
				return segments
			},
			explanation: "explained",
			wantIssue:   "English prompt is empty",
		},
		{
			name: "a segment without a grammar point is rejected",
			segments: func() []ProduceSegment {
				segments := twoProduceSegments()
				segments[0].GrammarPointID = nil
				return segments
			},
			explanation: "explained",
			wantIssue:   "no grammar point selected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateProduceContent(tt.segments(), tt.explanation)

			if got.Phase != "produce" {
				t.Errorf("Phase = %q, want %q", got.Phase, "produce")
			}
			if got.Ready != tt.wantReady {
				t.Errorf("Ready = %v, want %v (issues: %v)", got.Ready, tt.wantReady, got.Issues)
			}
			if tt.wantIssue != "" && !hasIssueContaining(got.Issues, tt.wantIssue) {
				t.Errorf("expected an issue containing %q, got %v", tt.wantIssue, got.Issues)
			}
		})
	}
}

// fiveRecallSentences is the shape a fully authored Recall phase has, one
// sentence per target word.
func fiveRecallSentences() ([]RecallSentence, map[int]bool) {
	sentences := make([]RecallSentence, 0, RecallSentencesPerStory)
	targetVocabIDs := make(map[int]bool, RecallSentencesPerStory)
	for order := 1; order <= RecallSentencesPerStory; order++ {
		targetID := order
		targetVocabIDs[targetID] = true
		sentences = append(sentences, RecallSentence{
			ID:            100 + order,
			StoryID:       1,
			SequenceOrder: order,
			HebrewText:    "משפט",
			TargetVocabID: &targetID,
			ImagePath:     "stories/1/image_recall_" + string(rune('0'+order)),
			ImageBucket:   "images",
		})
	}
	return sentences, targetVocabIDs
}

func TestValidateRecallSentences(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(sentences []RecallSentence, ids map[int]bool) []RecallSentence
		wantReady bool
		wantIssue string
	}{
		{
			name:      "five complete sentences is ready",
			mutate:    func(s []RecallSentence, _ map[int]bool) []RecallSentence { return s },
			wantReady: true,
		},
		{
			name: "four sentences leaves a gap",
			mutate: func(s []RecallSentence, _ map[int]bool) []RecallSentence {
				return s[:4]
			},
			wantIssue: "no sentence authored at position 5",
		},
		{
			name: "a sentence without a picture is rejected",
			mutate: func(s []RecallSentence, _ map[int]bool) []RecallSentence {
				s[0].ImagePath = ""
				return s
			},
			wantIssue: "no picture",
		},
		{
			name: "an empty sentence is rejected",
			mutate: func(s []RecallSentence, _ map[int]bool) []RecallSentence {
				s[1].HebrewText = " "
				return s
			},
			wantIssue: "sentence text is empty",
		},
		{
			name: "an unlinked sentence is rejected",
			mutate: func(s []RecallSentence, _ map[int]bool) []RecallSentence {
				s[2].TargetVocabID = nil
				return s
			},
			wantIssue: "no target word linked",
		},
		{
			name: "a target word from another story is rejected",
			mutate: func(s []RecallSentence, _ map[int]bool) []RecallSentence {
				foreign := 999
				s[3].TargetVocabID = &foreign
				return s
			},
			wantIssue: "does not belong to this story",
		},
		{
			name: "reusing one target word leaves another word uncovered",
			mutate: func(s []RecallSentence, _ map[int]bool) []RecallSentence {
				s[4].TargetVocabID = s[0].TargetVocabID
				return s
			},
			wantIssue: "already used by another sentence",
		},
		{
			name: "duplicate positions are rejected",
			mutate: func(s []RecallSentence, _ map[int]bool) []RecallSentence {
				s[4].SequenceOrder = 1
				return s
			},
			wantIssue: "more than one sentence at position 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentences, ids := fiveRecallSentences()
			got := ValidateRecallSentences(tt.mutate(sentences, ids), ids)

			if got.Phase != "recall" {
				t.Errorf("Phase = %q, want %q", got.Phase, "recall")
			}
			if got.Ready != tt.wantReady {
				t.Errorf("Ready = %v, want %v (issues: %v)", got.Ready, tt.wantReady, got.Issues)
			}
			if tt.wantIssue != "" && !hasIssueContaining(got.Issues, tt.wantIssue) {
				t.Errorf("expected an issue containing %q, got %v", tt.wantIssue, got.Issues)
			}
		})
	}
}

func TestStoryContentReadinessAllReady(t *testing.T) {
	words, occurrences := fiveTargetWords()
	sentences, ids := fiveRecallSentences()

	ready := StoryContentReadiness{
		Video:    ValidateVideo("https://example.com/video"),
		Identify: ValidateTargetVocabulary(words, occurrences),
		Produce:  ValidateProduceContent(twoProduceSegments(), "explained"),
		Recall:   ValidateRecallSentences(sentences, ids),
	}
	if !ready.AllReady() {
		t.Errorf("fully authored story reported not ready: %+v", ready)
	}

	ready.Recall = ValidateRecallSentences(sentences[:2], ids)
	if ready.AllReady() {
		t.Error("AllReady must be false when one phase is incomplete")
	}
	if got := ready.MissingPhases(); len(got) != 1 || got[0] != "recall" {
		t.Errorf("MissingPhases = %v, want [recall]", got)
	}

	ready.Recall = ValidateRecallSentences(sentences, ids)
	ready.Video = ValidateVideo("  ")
	if ready.AllReady() {
		t.Error("AllReady must be false when the video link is missing")
	}
	if got := ready.MissingPhases(); len(got) != 1 || got[0] != "video" {
		t.Errorf("MissingPhases = %v, want [video]", got)
	}
}

func hasIssueContaining(issues []ContentIssue, want string) bool {
	for _, issue := range issues {
		if strings.Contains(issue.Message, want) {
			return true
		}
	}
	return false
}
