package models

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"glossias/src/pkg/generated/db"
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

// StoryContentReadiness is the per-phase report for one story. Video covers
// the story-level link every phase flow starts from; it is edited on the
// metadata page.
type StoryContentReadiness struct {
	Video    PhaseReadiness `json:"video"`
	Identify PhaseReadiness `json:"identify"`
	Produce  PhaseReadiness `json:"produce"`
	Recall   PhaseReadiness `json:"recall"`
}

// AllReady reports whether every new phase is fully authored.
func (r StoryContentReadiness) AllReady() bool {
	return r.Video.Ready && r.Identify.Ready && r.Produce.Ready && r.Recall.Ready
}

// MissingPhases lists the phases that are not ready, in phase order. Empty
// when AllReady.
func (r StoryContentReadiness) MissingPhases() []string {
	var missing []string
	for _, p := range []PhaseReadiness{r.Video, r.Identify, r.Produce, r.Recall} {
		if !p.Ready {
			missing = append(missing, p.Phase)
		}
	}
	return missing
}

// newReadiness builds a report from the issues collected for a phase.
func newReadiness(phase string, issues []ContentIssue) PhaseReadiness {
	return PhaseReadiness{
		Phase:  phase,
		Ready:  len(issues) == 0,
		Issues: issues,
	}
}

// ValidateVideo checks that the story has a video link.
func ValidateVideo(videoURL string) PhaseReadiness {
	var issues []ContentIssue
	if strings.TrimSpace(videoURL) == "" {
		issues = append(issues, ContentIssue{Field: "videoUrl", Message: "story has no video link"})
	}
	return newReadiness("video", issues)
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

		if strings.TrimSpace(segment.ReferenceEnglish) == "" {
			issues = append(issues, ContentIssue{Field: field, Message: "reference English is empty"})
		}
		if strings.TrimSpace(segment.HebrewText) == "" {
			issues = append(issues, ContentIssue{Field: field, Message: "Hebrew segment is empty"})
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
// sentences filling positions 1-5, each with text, a picture, playable audio
// (uploaded override or story-line narration covering the sentence), and a
// distinct target word from this story.
func ValidateRecallSentences(sentences []RecallSentence, storyTargetVocabIDs map[int]bool, lines []StoryLine, linesWithAudio map[int]bool) PhaseReadiness {
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

		if !RecallSentenceHasAudio(sentence, lines, linesWithAudio) {
			issues = append(issues, ContentIssue{Field: field, Message: "sentence has no audio"})
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

// GetStoryContentReadiness returns a story's phase readiness report, computing
// it from the story's Summer 2026 content on a cache miss. The report is
// cached per story and invalidated by InvalidateStoryContentReadiness on every
// write that can change it.
func GetStoryContentReadiness(ctx context.Context, storyID int) (StoryContentReadiness, error) {
	reports, err := GetStoriesContentReadiness(ctx, []int{storyID})
	if err != nil {
		return StoryContentReadiness{}, err
	}
	report, ok := reports[storyID]
	if !ok {
		return StoryContentReadiness{}, ErrNotFound
	}
	return report, nil
}

// GetStoriesContentReadiness returns phase readiness for many stories in a
// constant number of queries. Cache hits are skipped; misses share eight
// `WHERE story_id = ANY($1)` fetches instead of looping GetStoryContentReadiness.
func GetStoriesContentReadiness(ctx context.Context, storyIDs []int) (map[int]StoryContentReadiness, error) {
	out := make(map[int]StoryContentReadiness, len(storyIDs))
	if len(storyIDs) == 0 {
		return out, nil
	}

	misses := make([]int, 0, len(storyIDs))
	seen := make(map[int]bool, len(storyIDs))
	for _, id := range storyIDs {
		if seen[id] {
			continue
		}
		seen[id] = true
		if cacheInstance != nil && keyBuilder != nil {
			var cached StoryContentReadiness
			if err := cacheInstance.GetJSON(keyBuilder.StoryContentReadiness(id), &cached); err == nil {
				out[id] = cached
				continue
			}
		}
		misses = append(misses, id)
	}
	if len(misses) == 0 {
		return out, nil
	}

	built, err := buildStoriesContentReadiness(ctx, misses)
	if err != nil {
		return nil, err
	}
	for id, report := range built {
		if cacheInstance != nil && keyBuilder != nil {
			_ = cacheInstance.SetJSON(keyBuilder.StoryContentReadiness(id), report)
		}
		out[id] = report
	}
	return out, nil
}

func buildStoriesContentReadiness(ctx context.Context, storyIDs []int) (map[int]StoryContentReadiness, error) {
	if queries == nil {
		return nil, errors.New("database not initialized")
	}

	ids := toInt32IDs(storyIDs)

	videoRows, err := queries.GetStoriesVideoURLs(ctx, ids)
	if err != nil {
		return nil, err
	}
	wordRows, err := queries.GetStoriesTargetVocabulary(ctx, ids)
	if err != nil {
		return nil, err
	}
	countRows, err := queries.GetStoriesLexicalFormCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	segmentRows, err := queries.GetStoriesProduceSegments(ctx, ids)
	if err != nil {
		return nil, err
	}
	explanationRows, err := queries.GetStoriesProduceExplanations(ctx, ids)
	if err != nil {
		return nil, err
	}
	sentenceRows, err := queries.GetStoriesRecallSentences(ctx, ids)
	if err != nil {
		return nil, err
	}
	lineRows, err := queries.GetStoriesLines(ctx, ids)
	if err != nil {
		return nil, err
	}
	audioRows, err := queries.GetStoriesAudioFilesByLabel(ctx, db.GetStoriesAudioFilesByLabelParams{
		StoryIds: ids,
		Label:    "complete",
	})
	if err != nil {
		return nil, err
	}

	videos := make(map[int]string, len(videoRows))
	for _, row := range videoRows {
		videos[int(row.StoryID)] = row.VideoUrl.String
	}

	wordsByStory := make(map[int][]TargetVocabulary, len(videos))
	for _, row := range wordRows {
		sid := int(row.StoryID)
		wordsByStory[sid] = append(wordsByStory[sid], TargetVocabulary{
			ID:               int(row.ID),
			StoryID:          sid,
			LexicalForm:      row.LexicalForm,
			AudioPath:        row.AudioPath.String,
			AudioBucket:      row.AudioBucket.String,
			CorrectImagePath: row.CorrectImagePath.String,
			ImageBucket:      row.ImageBucket.String,
		})
	}

	countsByStory := make(map[int]map[string]int, len(videos))
	for _, row := range countRows {
		if !row.StoryID.Valid {
			continue
		}
		sid := int(row.StoryID.Int32)
		if countsByStory[sid] == nil {
			countsByStory[sid] = make(map[string]int)
		}
		countsByStory[sid][row.LexicalForm] = int(row.Occurrences)
	}

	segmentsByStory := make(map[int][]ProduceSegment, len(videos))
	for _, row := range segmentRows {
		sid := int(row.StoryID)
		segmentsByStory[sid] = append(segmentsByStory[sid], ProduceSegment{
			ID:               int(row.ID),
			StoryID:          sid,
			SegmentOrder:     int(row.SegmentOrder),
			ReferenceEnglish: row.ReferenceEnglish,
			HebrewText:       row.HebrewText,
			GrammarPointName: row.GrammarPointName.String,
			GrammarPointID:   optionalInt(row.GrammarPointID),
			LineStart:        optionalInt(row.LineStart),
			LineEnd:          optionalInt(row.LineEnd),
		})
	}

	explanations := make(map[int]string, len(explanationRows))
	for _, row := range explanationRows {
		explanations[int(row.StoryID)] = row.ExplanationText
	}

	sentencesByStory := make(map[int][]RecallSentence, len(videos))
	for _, row := range sentenceRows {
		sid := int(row.StoryID)
		sentencesByStory[sid] = append(sentencesByStory[sid], recallSentenceFromRow(
			row.ID, row.StoryID, row.SequenceOrder, row.HebrewText,
			row.TargetVocabID, row.ImagePath, row.ImageBucket, row.AudioPath, row.AudioBucket,
		))
	}

	linesByStory := make(map[int][]StoryLine, len(videos))
	for _, row := range lineRows {
		sid := int(row.StoryID)
		linesByStory[sid] = append(linesByStory[sid], StoryLine{LineNumber: int(row.LineNumber), Text: row.Text})
	}

	audioByStory := make(map[int]map[int]bool, len(videos))
	for _, row := range audioRows {
		if !row.StoryID.Valid || !row.LineNumber.Valid {
			continue
		}
		sid := int(row.StoryID.Int32)
		if audioByStory[sid] == nil {
			audioByStory[sid] = make(map[int]bool)
		}
		audioByStory[sid][int(row.LineNumber.Int32)] = true
	}

	out := make(map[int]StoryContentReadiness, len(videos))
	for id, videoURL := range videos {
		words := wordsByStory[id]
		occurrences := countsByStory[id]
		if occurrences == nil {
			occurrences = map[string]int{}
		}
		targetVocabIDs := make(map[int]bool, len(words))
		for _, word := range words {
			targetVocabIDs[word.ID] = true
		}
		linesWithAudio := audioByStory[id]
		if linesWithAudio == nil {
			linesWithAudio = map[int]bool{}
		}
		out[id] = StoryContentReadiness{
			Video:    ValidateVideo(videoURL),
			Identify: ValidateTargetVocabulary(words, occurrences),
			Produce:  ValidateProduceContent(segmentsByStory[id], explanations[id]),
			Recall:   ValidateRecallSentences(sentencesByStory[id], targetVocabIDs, linesByStory[id], linesWithAudio),
		}
	}
	return out, nil
}

func toInt32IDs(ids []int) []int32 {
	out := make([]int32, len(ids))
	for i, id := range ids {
		out[i] = int32(id)
	}
	return out
}

func RecallAudioContext(ctx context.Context, storyID int) ([]StoryLine, map[int]bool, error) {
	dbLines, err := queries.GetStoryLines(ctx, int32(storyID))
	if err != nil {
		return nil, nil, err
	}
	lines := make([]StoryLine, 0, len(dbLines))
	for _, line := range dbLines {
		lines = append(lines, StoryLine{LineNumber: int(line.LineNumber), Text: line.Text})
	}

	audioFiles, err := GetStoryAudioFilesByLabel(ctx, storyID, "complete")
	if err != nil {
		return nil, nil, err
	}
	linesWithAudio := make(map[int]bool, len(audioFiles))
	for _, file := range audioFiles {
		linesWithAudio[file.LineNumber] = true
	}
	return lines, linesWithAudio, nil
}
