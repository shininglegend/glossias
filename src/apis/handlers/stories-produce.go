package handlers

import (
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
// segments (without their reference English), the authored explanation, and the
// student's progress — submissions so far and any countdown already running.
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

	starts, err := models.GetUserStoryProduceAttemptStarts(ctx, userID, id)
	if err != nil {
		h.log.Error("Failed to fetch produce attempt starts", "error", err, "storyID", id, "userID", userID)
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
			ReferenceEnglish: s.ReferenceEnglish,
			GrammarPointName: s.GrammarPointName,
			Slot:             produceSlot(lineTexts, s),
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
		Starts:           produceStartViews(segments, submissions, starts),
		Completed:        produceCompleted(segments, submissions),
		TimeLimitSeconds: produceTimeLimitSeconds,
	}

	json.NewEncoder(w).Encode(types.APIResponse{Success: true, Data: data})
}

// StartProduce records that the student began writing a segment and returns
// the countdown's remaining time. Calling it again for the same segment (a
// reload) returns the original countdown rather than restarting it.
func (h *Handler) StartProduce(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req types.StartProduceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.log.Warn("Invalid request body in StartProduce", "error", err, "ip", r.RemoteAddr)
		h.sendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	id, userID, segments, ok := h.produceRequestContext(w, r, "StartProduce")
	if !ok {
		return
	}
	if !slices.ContainsFunc(segments, func(s models.ProduceSegment) bool { return s.ID == req.SegmentID }) {
		h.log.Warn("Segment does not belong to story in StartProduce",
			"storyID", id, "segment", req.SegmentID, "ip", r.RemoteAddr)
		h.sendError(w, "Segment does not belong to this story", http.StatusBadRequest)
		return
	}

	start, err := models.StartProduceAttempt(ctx, userID, id, req.SegmentID)
	if err != nil {
		h.log.Error("Failed to record produce attempt start", "error", err, "userID", userID, "storyID", id, "segment", req.SegmentID)
		h.sendError(w, "Failed to start segment", http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(types.APIResponse{
		Success: true,
		Data: types.ProduceAttemptStartView{
			SegmentID:   start.SegmentID,
			SecondsLeft: secondsLeft(start.ElapsedSeconds),
		},
	})
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

	id, userID, segments, ok := h.produceRequestContext(w, r, "SubmitProduce")
	if !ok {
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
		// Grade in the background; the student gets the reference right away
		// and a grading failure only leaves ai_score NULL.
		h.produceGrading.Enqueue(userID, *saved, segment)
	}

	json.NewEncoder(w).Encode(types.APIResponse{
		Success: true,
		Data: types.SubmitProduceResponse{
			Submission: types.ProduceSubmissionView{
				SegmentID:   segment.ID,
				StudentText: studentText,
				HebrewText:  segment.HebrewText,
			},
			Completed: produceCompleted(segments, existing),
		},
	})
}

// produceRequestContext does the shared prologue of the Produce write
// endpoints: parse the story ID, require a user, enforce course access, and
// load the story's segments. It writes the error response itself and returns
// ok=false when the caller should stop.
func (h *Handler) produceRequestContext(w http.ResponseWriter, r *http.Request, op string) (id int, userID string, segments []models.ProduceSegment, ok bool) {
	ctx := r.Context()
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		h.sendError(w, "Invalid story ID", http.StatusBadRequest)
		return 0, "", nil, false
	}
	userID = auth.GetUserID(r)
	if userID == "" {
		h.sendError(w, "Authentication required", http.StatusUnauthorized)
		return 0, "", nil, false
	}

	// Access check (course membership) — the returned story is otherwise unused.
	if _, err := models.GetStoryData(ctx, id, userID); err != nil {
		if err == models.ErrNotFound {
			h.sendError(w, "Story not found", http.StatusNotFound)
			return 0, "", nil, false
		}
		h.log.Error("Failed to fetch story in "+op, "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch story", http.StatusInternalServerError)
		return 0, "", nil, false
	}

	segments, err = models.GetStoryProduceSegments(ctx, id)
	if err != nil {
		h.log.Error("Failed to fetch produce segments in "+op, "error", err, "storyID", id)
		h.sendError(w, "Failed to fetch produce segments", http.StatusInternalServerError)
		return 0, "", nil, false
	}
	return id, userID, segments, true
}

// secondsLeft converts elapsed time into the countdown remaining, floored at 0.
func secondsLeft(elapsedSeconds int) int {
	return max(0, produceTimeLimitSeconds-elapsedSeconds)
}

// produceStartViews reports, in segment order, the segments the student has
// started but not submitted, with their remaining time. Starts for submitted
// or deleted segments are omitted — they no longer drive anything.
func produceStartViews(segments []models.ProduceSegment, submissions []models.ProduceSubmission, starts []models.ProduceAttemptStart) []types.ProduceAttemptStartView {
	views := make([]types.ProduceAttemptStartView, 0, len(starts))
	for _, seg := range segments {
		if slices.ContainsFunc(submissions, func(s models.ProduceSubmission) bool { return s.SegmentID == seg.ID }) {
			continue
		}
		for _, st := range starts {
			if st.SegmentID == seg.ID {
				views = append(views, types.ProduceAttemptStartView{
					SegmentID:   seg.ID,
					SecondsLeft: secondsLeft(st.ElapsedSeconds),
				})
				break
			}
		}
	}
	return views
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
					SegmentID:   seg.ID,
					StudentText: sub.StudentText,
					HebrewText:  seg.HebrewText,
				})
				break
			}
		}
	}
	return views
}

// produceCompleted reports whether every authored segment has a submission.
// A story with no segments has nothing to do, so it counts as complete and
// navigation skips the phase (like vocab with no items).
func produceCompleted(segments []models.ProduceSegment, submissions []models.ProduceSubmission) bool {
	for _, seg := range segments {
		if !slices.ContainsFunc(submissions, func(s models.ProduceSubmission) bool { return s.SegmentID == seg.ID }) {
			return false
		}
	}
	return true
}

// produceSlot locates a segment in the story text. The authored line range
// wins: the Hebrew text is looked for on each line within it so the page can
// highlight exactly it, and if it isn't found verbatim on any single line the
// whole range is marked. With no authored range (content from before the
// columns existed) the Hebrew is searched for across every line, and nil
// means it was not found.
func produceSlot(lines []string, segment models.ProduceSegment) *types.ProduceSlot {
	ref := strings.TrimSpace(segment.HebrewText)

	if segment.LineStart != nil && segment.LineEnd != nil {
		startIdx := *segment.LineStart - 1
		endIdx := *segment.LineEnd - 1
		if startIdx < 0 || endIdx >= len(lines) || startIdx > endIdx {
			return nil
		}
		for i := startIdx; i <= endIdx; i++ {
			if start, end, found := runeRange(lines[i], ref); found {
				return &types.ProduceSlot{LineIndex: i, LineEnd: i, Exact: true, Start: start, End: end}
			}
		}
		return &types.ProduceSlot{LineIndex: startIdx, LineEnd: endIdx}
	}

	if ref == "" {
		return nil
	}
	for i, line := range lines {
		if start, end, found := runeRange(line, ref); found {
			return &types.ProduceSlot{LineIndex: i, LineEnd: i, Exact: true, Start: start, End: end}
		}
	}
	return nil
}

// runeRange finds needle in line and returns its rune offsets, matching the
// vocab and identify segmenters.
func runeRange(line, needle string) (start, end int, found bool) {
	if needle == "" {
		return 0, 0, false
	}
	before, _, ok := strings.Cut(line, needle)
	if !ok {
		return 0, 0, false
	}
	start = utf8.RuneCountInString(before)
	return start, start + utf8.RuneCountInString(needle), true
}
