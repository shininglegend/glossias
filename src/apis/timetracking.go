package apis

import (
	"encoding/json"
	"fmt"
	"glossias/src/auth"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"

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
	router.HandleFunc("/time-tracking/record", h.recordTimeTracking).Methods("POST")
}

type StartTimeTrackingRequest struct {
	Route   string `json:"route"`
	StoryID *int32 `json:"story_id,omitempty"`
}

type RecordTimeTrackingRequest struct {
	ElapsedMs  int32  `json:"elapsed_ms"`
	TrackingID string `json:"tracking_id"`
}

type StartTimeTrackingResponse struct {
	TrackingID string `json:"tracking_id"`
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

	trackingID, err := models.MakeTimeTrackingSession(r.Context(), userID, req.Route, req.StoryID)
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

// recordTimeTracking handles recording a complete time tracking session in one call
func (h *TimeTrackingHandler) recordTimeTracking(w http.ResponseWriter, r *http.Request) {
	var req RecordTimeTrackingRequest

	// Handle both JSON and multipart form data (for beacons)
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
	} else {
		// Parse multipart form data for beacon requests
		if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max memory
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		req.TrackingID = r.FormValue("tracking_id")
		elapsedStr := r.FormValue("elapsed_ms")
		if elapsedStr == "" {
			http.Error(w, "elapsed_ms is required", http.StatusBadRequest)
			return
		}

		var elapsed int32
		if _, err := fmt.Sscanf(elapsedStr, "%d", &elapsed); err != nil {
			http.Error(w, "elapsed_ms must be a valid integer", http.StatusBadRequest)
			return
		}
		req.ElapsedMs = elapsed
	}

	session, err := models.GetTimeTrackingBySessionID(r.Context(), req.TrackingID)
	if err != nil {
		h.logger.Error("failed to get time tracking session", "error", err, "tracking_id", req.TrackingID)
		http.Error(w, "Invalid tracking ID", http.StatusBadRequest)
		return
	}

	// If session not found, log and return error
	if session == nil {
		h.logger.Warn("time tracking session not found", "tracking_id", req.TrackingID)
		http.Error(w, "Tracking session not found or expired", http.StatusBadRequest)
		return
	}

	if req.ElapsedMs <= 0 {
		http.Error(w, "elapsed_ms must be positive", http.StatusBadRequest)
		return
	}

	fmt.Println("Session: ", session)

	err = models.RecordTimeTracking(r.Context(), session.UserID, session.Route, session.StoryID, req.ElapsedMs)
	if err != nil {
		h.logger.Error("failed to record time tracking", "error", err, "user_id", session.UserID)
		http.Error(w, "Failed to record time tracking", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
