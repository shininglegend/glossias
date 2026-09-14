package billing

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"glossias/src/auth"
	"glossias/src/pkg/models"

	"github.com/gorilla/mux"
)

type Handler struct {
	log     *slog.Logger
	gateway Gateway
}

func NewHandler(logger *slog.Logger, gateway Gateway) *Handler {
	return &Handler{log: logger, gateway: gateway}
}

func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/pricing", h.GetPricing).Methods("GET", "OPTIONS")
	router.HandleFunc("/checkout", h.CreateCheckout).Methods("POST", "OPTIONS")
}

type pricingCourse struct {
	CourseID     int    `json:"course_id"`
	CourseNumber string `json:"course_number"`
	Name         string `json:"name"`
	StoryCount   int    `json:"story_count"`
	HasAccess    bool   `json:"has_access"`
	ExpiresAt    string `json:"expires_at,omitempty"`
}

func (h *Handler) GetPricing(w http.ResponseWriter, r *http.Request) {
	if h.gateway == nil {
		http.Error(w, "Payments are not configured", http.StatusServiceUnavailable)
		return
	}
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	price, err := h.gateway.GetPrice(r.Context())
	if err != nil {
		h.log.Error("failed to retrieve price", "error", err)
		http.Error(w, "Failed to load pricing", http.StatusBadGateway)
		return
	}

	courses, err := models.GetCoursesForUser(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to list enrollments", "error", err)
		http.Error(w, "Failed to load pricing", http.StatusInternalServerError)
		return
	}
	ids := make([]int32, 0, len(courses))
	for _, c := range courses {
		if !c.IsTrial {
			ids = append(ids, int32(c.CourseID))
		}
	}
	counts, err := models.StoryCountsForCourses(r.Context(), ids)
	if err != nil {
		h.log.Error("failed to count stories", "error", err)
		http.Error(w, "Failed to load pricing", http.StatusInternalServerError)
		return
	}

	qCourse := r.URL.Query().Get("course_id")
	var selected *pricingCourse
	var payable []pricingCourse
	for _, c := range courses {
		if c.IsTrial {
			continue
		}
		pc := pricingCourse{
			CourseID:     c.CourseID,
			CourseNumber: c.CourseNumber,
			Name:         c.Name,
			StoryCount:   counts[int32(c.CourseID)],
			HasAccess:    c.HasAccess,
		}
		if c.ExpiresAt != nil {
			pc.ExpiresAt = c.ExpiresAt.UTC().Format(time.RFC3339)
		}
		if qCourse != "" && strconv.Itoa(c.CourseID) == qCourse {
			cp := pc
			selected = &cp
		}
		if !c.HasAccess {
			payable = append(payable, pc)
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"amount_cents":    price.AmountCents,
		"currency":        price.Currency,
		"name":            price.Name,
		"course":          selected,
		"payable_courses": payable,
	})
}

type checkoutRequest struct {
	CourseID int32 `json:"course_id"`
}

func (h *Handler) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	if h.gateway == nil {
		http.Error(w, "Payments are not configured", http.StatusServiceUnavailable)
		return
	}
	userID, ok := auth.GetUserIDWithOk(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	var req checkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.CourseID == 0 {
		http.Error(w, "course_id is required", http.StatusBadRequest)
		return
	}

	course, err := models.GetCourse(r.Context(), req.CourseID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			http.Error(w, "Course not found", http.StatusNotFound)
			return
		}
		h.log.Error("failed to load course", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	}
	if course.IsTrial {
		http.Error(w, "Trial courses do not require payment", http.StatusBadRequest)
		return
	}
	enrolled, err := models.IsUserEnrolledInCourse(r.Context(), userID, req.CourseID)
	if err != nil {
		h.log.Error("failed to check enrollment", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	}
	if !enrolled {
		http.Error(w, "Not enrolled in this course", http.StatusBadRequest)
		return
	}
	if exp, err := models.ActiveAccessExpiresAt(r.Context(), userID, req.CourseID); err != nil {
		h.log.Error("failed to check entitlement", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	} else if exp != nil {
		writeJSON(w, http.StatusOK, map[string]any{
			"already_active": true,
			"expires_at":     exp.UTC().Format(time.RFC3339),
		})
		return
	}

	user, err := models.GetUser(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to load user", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	}
	customerID, err := models.UserStripeCustomerID(r.Context(), userID)
	if err != nil {
		h.log.Error("failed to load stripe customer", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
		return
	}
	if customerID == "" {
		customerID, err = h.gateway.CreateCustomer(r.Context(), user.Email, user.Name, userID)
		if err != nil {
			h.log.Error("failed to create stripe customer", "error", err)
			http.Error(w, "Failed to start checkout", http.StatusBadGateway)
			return
		}
		if err := models.SetUserStripeCustomerID(r.Context(), userID, customerID); err != nil {
			h.log.Error("failed to store stripe customer", "error", err)
			http.Error(w, "Failed to start checkout", http.StatusInternalServerError)
			return
		}
	}

	appURL := os.Getenv("PUBLIC_APP_URL")
	if appURL == "" {
		appURL = "http://localhost:5173"
	}
	session, err := h.gateway.CreateCheckoutSession(r.Context(), CheckoutParams{
		CustomerID: customerID,
		UserID:     userID,
		CourseID:   req.CourseID,
		SuccessURL: appURL + "/checkout/success?session_id={CHECKOUT_SESSION_ID}",
		CancelURL:  fmt.Sprintf("%s/pricing?course=%d", appURL, req.CourseID),
	})
	if err != nil {
		h.log.Error("failed to create checkout session", "error", err)
		http.Error(w, "Failed to start checkout", http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": session.URL})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
