package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"glossias/src/apis/types"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gorilla/mux"
)

// produceTimeLimitSeconds is the writing limit per segment (SUMMER_2026.md,
// Phase 4: 90 seconds).
const produceTimeLimitSeconds = 90

// maxProduceStudentTextLen bounds a submission; segments are 5–10 words.
const maxProduceStudentTextLen = 1000

// GetProducePage returns the Produce phase payload: the story text, the two
// segments (without their references), the authored explanation, and the
// student's submissions so far.
//
// A story with no segments authored yet returns an empty list rather than an
// error, so the page degrades to a "nothing to do here" state.
func (h *Handler) GetProducePage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID format", http.StatusBadRequest)
		return
	}
	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	// GetStoryData enforces course access.
	story, err := models.GetStoryData(ctx, id, userID)
	if err == models.ErrNotFound {
		h.sendError(w, "Story not found", http.StatusNotFound)
		return
	}
	if err != nil {
		h.log.Error("Failed to fetch story data", "error", err)
		h.sendError(w, "Failed to fetch story data", http.StatusInternalServerError)
		return
	}

	segments, err := models.GetStoryProduceSegments(ctx, id)
	if err != nil {
		h.log.Error("Failed to fetch produce segments", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch produce segments", http.StatusInternalServerError)
		return
	}

	explanation, err := models.GetStoryProduceExplanation(ctx, id)
	if err != nil && !errors.Is(err, models.ErrNotFound) {
		h.log.Error("Failed to fetch produce explanation", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch produce explanation", http.StatusInternalServerError)
		return
	}

	submissions, err := models.GetUserStoryProduceSubmissions(ctx, userID, id)
	if err != nil {
		h.log.Error("Failed to fetch produce submissions", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Failed to fetch produce progress", http.StatusInternalServerError)
		return
	}

	lines := make([]types.LineText, 0, len(story.Content.Lines))
	lineTexts := make([]string, 0, len(story.Content.Lines))
	for _, line := range story.Content.Lines {
		lines = append(lines, types.LineText{Text: line.Text})
		lineTexts = append(lineTexts, line.Text)
	}

	views := make([]types.ProduceSegmentView, 0, len(segments))
	for _, s := range segments {
		views = append(views, types.ProduceSegmentView{
			ID:               s.ID,
			SegmentOrder:     s.SegmentOrder,
			EnglishText:      s.EnglishText,
			GrammarPointName: s.GrammarPointName,
			Slot:             findProduceSlot(lineTexts, s.ReferenceHebrew),
		})
	}

	data := types.ProducePageData{
		PageData: types.PageData{
			StoryID:    storyID,
			StoryTitle: story.Metadata.Title["en"],
			Language:   story.Metadata.Language,
		},
		Lines:            lines,
		Segments:         views,
		Explanation:      explanation,
		Submissions:      produceSubmissionViews(segments, submissions),
		Completed:        produceCompleted(segments, submissions),
		TimeLimitSeconds: produceTimeLimitSeconds,
	}

	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: data})
}

// SubmitProduce stores a student's attempt at one segment and reveals the
// reference. Submissions are stored ungraded; AI grading (T13) fills in
// ai_score later and must never block this response.
//
// A segment already answered is not re-recorded — the exercise is timed and
// one-shot — the existing attempt is returned instead, so a retried request
// is harmless.
func (h *Handler) SubmitProduce(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req types.SubmitProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in SubmitProduce", "error", err, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	if utf8.RuneCountInString(req.StudentText) > maxProduceStudentTextLen {
		h.sendError(w, "Submission is too long", http.StatusBadRequest)
		return
	}

	storyID := mux.Vars(r)["id"]
	id, err := strconv.Atoi(storyID)
	if err != nil {
		h.sendError(w, "Invalid story ID", http.StatusBadRequest)
		return
	}
	userID := auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "Authentication required", http.StatusUnauthorized)
		return
	}

	// Access check (course membership) — the returned story is otherwise unused.
	if _, err := models.GetStoryData(ctx, id, userID); err != nil {
		if err == models.ErrNotFound {
			h.sendError(w, "Story not found", http.StatusNotFound)
			return
		}
		h.log.Error("Failed to fetch story in SubmitProduce", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return
	}

	segments, err := models.GetStoryProduceSegments(ctx, id)
	if err != nil {
		h.log.Error("Failed to fetch produce segments in SubmitProduce", "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch produce segments", http.StatusInternalServerError)
		return
	}
	idx := slices.IndexFunc(segments, func(s models.ProduceSegment) bool { return s.ID == req.SegmentID })
	if idx < 0 {
		h.log.Warn("Segment does not belong to story in SubmitProduce",
			"storyID", id, "segment", req.SegmentID, "ip", r.RemoteAddr)
		h.sendError(w, "Segment does not belong to this story", http.StatusBadRequest)
		return
	}
	segment := segments[idx]

	existing, err := models.GetUserStoryProduceSubmissions(ctx, userID, id)
	if err != nil {
		h.log.Error("Failed to fetch produce submissions in SubmitProduce", "error", err, "storyID", id, "userID", userID)
		h.sendError(w, "Failed to fetch produce progress", http.StatusInternalServerError)
		return
	}

	var studentText string
	if i := slices.IndexFunc(existing, func(s models.ProduceSubmission) bool { return s.SegmentID == segment.ID }); i >= 0 {
		studentText = existing[i].StudentText
	} else {
		saved, err := models.CreateProduceSubmission(ctx, userID, id, segment.ID, strings.TrimSpace(req.StudentText))
		if err != nil {
			h.log.Error("Failed to save produce submission", "error", err, "userID", userID, "storyID", id, "segment", segment.ID)
			h.sendError(w, "Failed to save submission", http.StatusInternalServerError)
			return
		}
		studentText = saved.StudentText
		existing = append(existing, *saved)
	}

	json.NewEncoder(w).Encode(types.APIResponse{
		Success: true,
		Data: types.SubmitProduceResponse{
			Submission: types.ProduceSubmissionView{
				SegmentID:       segment.ID,
				StudentText:     studentText,
				ReferenceHebrew: segment.ReferenceHebrew,
			},
			Completed: produceCompleted(segments, existing),
		},
	})
}

// produceSubmissionViews pairs each stored submission with its segment's
// reference, in segment order. Submissions whose segment no longer exists
// (re-authored content) are dropped.
func produceSubmissionViews(segments []models.ProduceSegment, submissions []models.ProduceSubmission) []types.ProduceSubmissionView {
	views := make([]types.ProduceSubmissionView, 0, len(submissions))
	for _, seg := range segments {
		for _, sub := range submissions {
			if sub.SegmentID == seg.ID {
				views = append(views, types.ProduceSubmissionView{
					SegmentID:       seg.ID,
					StudentText:     sub.StudentText,
					ReferenceHebrew: seg.ReferenceHebrew,
				})
				break
			}
		}
	}
	return views
}

// produceCompleted reports whether every authored segment has a submission.
// A story with no segments has nothing to do, so it counts as complete and
// navigation skips the phase (like isVocabCompleted).
func produceCompleted(segments []models.ProduceSegment, submissions []models.ProduceSubmission) bool {
	for _, seg := range segments {
		if !slices.ContainsFunc(submissions, func(s models.ProduceSubmission) bool { return s.SegmentID == seg.ID }) {
			return false
		}
	}
	return true
}

// isProduceCompleted reports whether the user has submitted every Produce
// segment for the story, for navigation's completion map.
func (h *Handler) isProduceCompleted(ctx context.Context, userID string, storyID int) (bool, error) {
	segments, err := models.GetStoryProduceSegments(ctx, storyID)
	if err != nil {
		return false, err
	}
	submissions, err := models.GetUserStoryProduceSubmissions(ctx, userID, storyID)
	if err != nil {
		return false, err
	}
	return produceCompleted(segments, submissions), nil
}

// findProduceSlot locates the first line containing the reference verbatim
// and returns its rune range, or nil when the reference does not appear in
// the text (authors may paraphrase). Offsets are runes, matching the vocab
// and identify segmenters.
func findProduceSlot(lines []string, reference string) *types.ProduceSlot {
	ref := strings.TrimSpace(reference)
	if ref == "" {
		return nil
	}
	for i, line := range lines {
		byteStart := strings.Index(line, ref)
		if byteStart < 0 {
			continue
		}
		start := utf8.RuneCountInString(line[:byteStart])
		return &types.ProduceSlot{
			LineIndex: i,
			Start:     start,
			End:       start + utf8.RuneCountInString(ref),
		}
	}
	return nil
}
