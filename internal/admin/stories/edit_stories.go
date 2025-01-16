package stories

import (
	"logos-stories/internal/pkg/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// Example for edit story handler
func (h *Handler) editStoryHandler(w http.ResponseWriter, r *http.Request) {
	storyIDStr := mux.Vars(r)["id"]
	if storyIDStr == "" {
		http.Error(w, "Story ID is required", http.StatusBadRequest)
		return
	}
	storyID, err := strconv.Atoi(storyIDStr)
	if err != nil {
		http.Error(w, "Invalid story ID", http.StatusBadRequest)
		return
	}

	story, err := models.GetStoryData(storyID)
	if err != nil {
		h.log.Error("Failed to load story", "error", err)
		http.Error(w, "Failed to load story", http.StatusInternalServerError)
		return
	}

	if r.Method == "GET" {
		if err := h.te.Render(w, "admin/editStory.html", story); err != nil {
			h.log.Error("Failed to render edit story form", "error", err)
			http.Error(w, "Failed to render edit story form", http.StatusInternalServerError)
			return
		}
		return
	}
	// TODO: Handle PUT requests to update story
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed) // :)
}
