package apis

import (
	"encoding/json"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// TimeTrackingHandler handles time tracking API endpoints
type TimeTrackingHandler struct {
	logger *slog.Logger
}

// NewTimeTrackingHandler creates a new time tracking handler
func NewTimeTrackingHandler(logger *slog.Logger) *TimeTrackingHandler {
	return &TimeTrackingHandler{
		logger: logger,
	}
}

// RegisterRoutes registers time tracking routes
func (h *TimeTrackingHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/time-tracking/start", h.startTimeTracking).Methods("POST")
	router.HandleFunc("/time-tracking/end", h.endTimeTracking).Methods("POST")
}

type StartTimeTrackingRequest struct {
	Route   string `json:"route"`
	StoryID *int32 `json:"story_id,omitempty"`
}

type EndTimeTrackingRequest struct {
	TrackingID int32 `json:"tracking_id"`
}

type StartTimeTrackingResponse struct {
	TrackingID int32  `json:"tracking_id"`
	Status     string `json:"status"`
}

// startTimeTracking handles starting a time tracking session
func (h *TimeTrackingHandler) startTimeTracking(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	if userID == "" {
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		h.logger.Warn("time tracking start without user", "ip", clientIP)
		userID = "anonymous"
	}

	var req StartTimeTrackingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Route == "" {
		http.Error(w, "route is required", http.StatusBadRequest)
		return
	}

	clientIP := r.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = r.RemoteAddr
	}

	trackingID, err := models.StartTimeTracking(r.Context(), userID, req.Route, req.StoryID, clientIP)
	if err != nil {
		h.logger.Error("failed to start time tracking", "error", err, "user_id", userID)
		http.Error(w, "Failed to start time tracking", http.StatusInternalServerError)
		return
	}

	response := StartTimeTrackingResponse{
		TrackingID: trackingID,
		Status:     "success",
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// endTimeTracking handles ending a time tracking session
func (h *TimeTrackingHandler) endTimeTracking(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	if userID == "" {
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		h.logger.Warn("time tracking end without user", "ip", clientIP)
	}

	var trackingID int32

	// Handle both JSON and FormData (from beacon)
	contentType := r.Header.Get("Content-Type")

	if contentType == "application/json" {
		var req EndTimeTrackingRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.Error("failed to decode JSON", "error", err)
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		trackingID = req.TrackingID
	} else {
		// Handle FormData from beacon
		if err := r.ParseForm(); err != nil {
			h.logger.Error("failed to parse form", "error", err)
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		trackingIDStr := r.FormValue("tracking_id")

		if trackingIDStr == "" {
			http.Error(w, "tracking_id is required", http.StatusBadRequest)
			return
		}

		id, err := strconv.Atoi(trackingIDStr)
		if err != nil {
			h.logger.Error("invalid tracking_id format", "error", err, "tracking_id_str", trackingIDStr)
			http.Error(w, "Invalid tracking_id", http.StatusBadRequest)
			return
		}
		trackingID = int32(id)
	}

	if trackingID == 0 {
		http.Error(w, "tracking_id is required", http.StatusBadRequest)
		return
	}

	err := models.EndTimeTrackingByID(r.Context(), trackingID)
	if err != nil {
		h.logger.Error("failed to end time tracking", "error", err, "tracking_id", trackingID)
		http.Error(w, "Failed to end time tracking", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
