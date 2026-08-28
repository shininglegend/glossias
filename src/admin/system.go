package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"unicode/utf8"

	"glossias/src/auth"
	"glossias/src/pkg/models"
)

// System-page endpoints (super admin only): the Produce grader's versioned
// system prompt. Versions are append-only; PUT adds a new one and it becomes
// active for every grading run from then on.

type gradingPromptResponse struct {
	Active  models.ProduceGradingPrompt   `json:"active"`
	History []models.ProduceGradingPrompt `json:"history"`
	// Default is the built-in prompt, offered so an admin can restore it.
	Default string `json:"default"`
}

type gradingPromptRequest struct {
	Text string `json:"text"`
	Note string `json:"note"`
}

// requireSuperAdmin writes the error response and returns ok=false when the
// caller is not a super admin.
func (h *Handler) requireSuperAdmin(w http.ResponseWriter, r *http.Request, op string) (userID string, ok bool) {
	userID, ok = auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return "", false
	}
	if !models.IsUserSuperAdmin(r.Context(), userID) {
		h.log.Warn(op+" denied - not super admin", "user_id", userID)
		http.Error(w, "Forbidden - super admin required", http.StatusForbidden)
		return "", false
	}
	return userID, true
}

func (h *Handler) gradingPromptHandler(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireSuperAdmin(w, r, "grading prompt access")
	if !ok {
		return
	}
	ctx := r.Context()

	if r.Method == http.MethodPut {
		var req gradingPromptRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		text := strings.TrimSpace(req.Text)
		if text == "" {
			http.Error(w, "Prompt text is required", http.StatusBadRequest)
			return
		}
		if utf8.RuneCountInString(text) > models.MaxGradingPromptLen {
			http.Error(w, "Prompt is too long", http.StatusBadRequest)
			return
		}
		if _, err := models.CreateProduceGradingPrompt(ctx, text, req.Note, userID); err != nil {
			h.log.Error("failed to save grading prompt", "error", err, "user_id", userID)
			http.Error(w, "Failed to save prompt", http.StatusInternalServerError)
			return
		}
		h.log.Info("grading prompt updated", "user_id", userID)
	}

	history, err := models.ListProduceGradingPrompts(ctx)
	if err != nil {
		h.log.Error("failed to list grading prompts", "error", err)
		http.Error(w, "Failed to load prompts", http.StatusInternalServerError)
		return
	}
	active, err := models.GetActiveProduceGradingPrompt(ctx)
	if errors.Is(err, models.ErrNotFound) {
		active = models.ProduceGradingPrompt{Text: models.DefaultGradingSystemPrompt, IsDefault: true}
	} else if err != nil {
		h.log.Error("failed to load active grading prompt", "error", err)
		http.Error(w, "Failed to load prompts", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(gradingPromptResponse{
		Active:  active,
		History: history,
		Default: models.DefaultGradingSystemPrompt,
	})
}
