package models

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Content readiness for the Summer 2026 phases.
//
// The counts and per-item asset requirements are authoring rules the schema
// cannot express: the DB enforces segment_order IN (1,2) and sequence_order
// BETWEEN 1 AND 5, but "exactly five target words, each appearing at least
// twice, each with audio and a picture" has to be checked here. The admin
// editors surface this report while authoring, and navigation can use the same
// Ready flags to skip phases whose content is absent rather than serving a
// broken page.

// ContentIssue is one reason a phase's authored content is not yet usable.
type ContentIssue struct {
	Field   string `json:"field,omitempty"`
	Message string `json:"message"`
}

// PhaseReadiness reports whether one phase has everything it needs.
type PhaseReadiness struct {
	Phase  string         `json:"phase"`
	Ready  bool           `json:"ready"`
	Issues []ContentIssue `json:"issues"`
}

// StoryContentReadiness is the per-phase report for one story.
type StoryContentReadiness struct {
	Identify PhaseReadiness `json:"identify"`
	Produce  PhaseReadiness `json:"produce"`
	Recall   PhaseReadiness `json:"recall"`
}

// AllReady reports whether every new phase is fully authored.
func (r StoryContentReadiness) AllReady() bool {
	return r.Identify.Ready && r.Produce.Ready && r.Recall.Ready
}

// newReadiness builds a report from the issues collected for a phase.
func newReadiness(phase string, issues []ContentIssue) PhaseReadiness {
	return PhaseReadiness{
		Phase:  phase,
		Ready:  len(issues) == 0,
		Issues: issues,
	}
}

// ValidateTargetVocabulary checks the Identify phase's authoring rules against a
// story's target words. occurrences maps a lexical form to how many times it
// appears in the story text (see GetStoryLexicalFormCounts).
func ValidateTargetVocabulary(words []TargetVocabulary, occurrences map[string]int) PhaseReadiness {
	issues := make([]ContentIssue, 0)

	if len(words) != TargetWordsPerStory {
		issues = append(issues, ContentIssue{
			Field:   "targetVocabulary",
			Message: fmt.Sprintf("story has %d target words; exactly %d are required", len(words), TargetWordsPerStory),
		})
	}

	for _, word := range words {
		field := fmt.Sprintf("targetVocabulary[%d]", word.ID)

		if strings.TrimSpace(word.LexicalForm) == "" {
			issues = append(issues, ContentIssue{Field: field, Message: "lexical form is empty"})
			continue
		}

		if count := occurrences[word.LexicalForm]; count < MinTargetWordOccurrences {
			issues = append(issues, ContentIssue{
				Field: field,
				Message: fmt.Sprintf("%q appears %d time(s) in the story text; at least %d annotated occurrences are required",
					word.LexicalForm, count, MinTargetWordOccurrences),
			})
		}

		if word.AudioPath == "" || word.AudioBucket == "" {
			issues = append(issues, ContentIssue{
				Field:   field,
				Message: fmt.Sprintf("%q has no pronunciation audio", word.LexicalForm),
			})
		}

		if word.CorrectImagePath == "" || word.ImageBucket == "" {
			issues = append(issues, ContentIssue{
				Field:   field,
				Message: fmt.Sprintf("%q has no picture", word.LexicalForm),
			})
		}
	}

	return newReadiness("identify", issues)
}

// ValidateProduceContent checks the Produce phase's authoring rules: both
// ordered segments present and complete, and the contrastive explanation
// authored.
func ValidateProduceContent(segments []ProduceSegment, explanation string) PhaseReadiness {
	issues := make([]ContentIssue, 0)

	if len(segments) != ProduceSegmentsPerStory {
		issues = append(issues, ContentIssue{
			Field:   "produceSegments",
			Message: fmt.Sprintf("story has %d produce segments; exactly %d are required", len(segments), ProduceSegmentsPerStory),
		})
	}

	seenOrder := make(map[int]bool, len(segments))
	for _, segment := range segments {
		field := fmt.Sprintf("produceSegments[%d]", segment.SegmentOrder)

		if seenOrder[segment.SegmentOrder] {
			issues = append(issues, ContentIssue{
				Field:   field,
				Message: fmt.Sprintf("more than one segment at position %d", segment.SegmentOrder),
			})
		}
		seenOrder[segment.SegmentOrder] = true

		if strings.TrimSpace(segment.EnglishText) == "" {
			issues = append(issues, ContentIssue{Field: field, Message: "English prompt is empty"})
		}
		if strings.TrimSpace(segment.ReferenceHebrew) == "" {
			issues = append(issues, ContentIssue{Field: field, Message: "reference translation is empty"})
		}
		if segment.GrammarPointID == nil {
			issues = append(issues, ContentIssue{
				Field:   field,
				Message: "no grammar point selected; AI grading needs one for context",
			})
		}
	}

	for order := 1; order <= ProduceSegmentsPerStory; order++ {
		if !seenOrder[order] {
			issues = append(issues, ContentIssue{
				Field:   fmt.Sprintf("produceSegments[%d]", order),
				Message: fmt.Sprintf("no segment authored at position %d", order),
			})
		}
	}

	if strings.TrimSpace(explanation) == "" {
		issues = append(issues, ContentIssue{
			Field:   "produceExplanation",
			Message: "the contrastive grammar explanation is empty",
		})
	}

	return newReadiness("produce", issues)
}

// ValidateRecallSentences checks the Recall phase's authoring rules: five
// sentences filling positions 1-5, each with text, a picture, and a distinct
// target word from this story.
func ValidateRecallSentences(sentences []RecallSentence, storyTargetVocabIDs map[int]bool) PhaseReadiness {
	issues := make([]ContentIssue, 0)

	if len(sentences) != RecallSentencesPerStory {
		issues = append(issues, ContentIssue{
			Field:   "recallSentences",
			Message: fmt.Sprintf("story has %d recall sentences; exactly %d are required", len(sentences), RecallSentencesPerStory),
		})
	}

	seenOrder := make(map[int]bool, len(sentences))
	seenTargetVocab := make(map[int]bool, len(sentences))
	for _, sentence := range sentences {
		field := fmt.Sprintf("recallSentences[%d]", sentence.SequenceOrder)

		if seenOrder[sentence.SequenceOrder] {
			issues = append(issues, ContentIssue{
				Field:   field,
				Message: fmt.Sprintf("more than one sentence at position %d", sentence.SequenceOrder),
			})
		}
		seenOrder[sentence.SequenceOrder] = true

		if strings.TrimSpace(sentence.HebrewText) == "" {
			issues = append(issues, ContentIssue{Field: field, Message: "sentence text is empty"})
		}

		if sentence.ImagePath == "" || sentence.ImageBucket == "" {
			issues = append(issues, ContentIssue{Field: field, Message: "sentence has no picture"})
		}

		switch {
		case sentence.TargetVocabID == nil:
			issues = append(issues, ContentIssue{Field: field, Message: "no target word linked to this sentence"})
		case !storyTargetVocabIDs[*sentence.TargetVocabID]:
			issues = append(issues, ContentIssue{
				Field:   field,
				Message: "the linked target word does not belong to this story",
			})
		case seenTargetVocab[*sentence.TargetVocabID]:
			issues = append(issues, ContentIssue{
				Field:   field,
				Message: "this target word is already used by another sentence; each of the five words needs its own sentence",
			})
		default:
			seenTargetVocab[*sentence.TargetVocabID] = true
		}
	}

	for order := 1; order <= RecallSentencesPerStory; order++ {
		if !seenOrder[order] {
			issues = append(issues, ContentIssue{
				Field:   fmt.Sprintf("recallSentences[%d]", order),
				Message: fmt.Sprintf("no sentence authored at position %d", order),
			})
		}
	}

	return newReadiness("recall", issues)
}

// GetStoryContentReadiness loads a story's Summer 2026 content and validates
// every phase.
func GetStoryContentReadiness(ctx context.Context, storyID int) (StoryContentReadiness, error) {
	if queries == nil {
		return StoryContentReadiness{}, errors.New("database not initialized")
	}

	words, err := GetStoryTargetVocabulary(ctx, storyID)
	if err != nil {
		return StoryContentReadiness{}, err
	}

	counts, err := GetStoryLexicalFormCounts(ctx, storyID)
	if err != nil {
		return StoryContentReadiness{}, err
	}
	occurrences := make(map[string]int, len(counts))
	for _, count := range counts {
		occurrences[count.LexicalForm] = count.Occurrences
	}

	segments, err := GetStoryProduceSegments(ctx, storyID)
	if err != nil {
		return StoryContentReadiness{}, err
	}

	explanation, err := GetStoryProduceExplanation(ctx, storyID)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return StoryContentReadiness{}, err
	}

	sentences, err := GetStoryRecallSentences(ctx, storyID)
	if err != nil {
		return StoryContentReadiness{}, err
	}

	targetVocabIDs := make(map[int]bool, len(words))
	for _, word := range words {
		targetVocabIDs[word.ID] = true
	}

	return StoryContentReadiness{
		Identify: ValidateTargetVocabulary(words, occurrences),
		Produce:  ValidateProduceContent(segments, explanation),
		Recall:   ValidateRecallSentences(sentences, targetVocabIDs),
	}, nil
}
