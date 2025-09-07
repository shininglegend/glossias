// logos-stories/src/admin/courses/handler.go
package courses

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type Handler struct {
	log *slog.Logger
}

func NewHandler(log *slog.Logger) *Handler {
	return &Handler{
		log: log,
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router) {
	// Base: /api/admin/courses
	courses := r.PathPrefix("/courses").Subrouter()

	// Course CRUD endpoints
	courses.HandleFunc("", h.listCoursesHandler).Methods("GET", "OPTIONS")
	courses.HandleFunc("", h.createCourseHandler).Methods("POST", "OPTIONS")
	courses.HandleFunc("/{id:[0-9]+}", h.getCourseHandler).Methods("GET", "OPTIONS")
	courses.HandleFunc("/{id:[0-9]+}", h.updateCourseHandler).Methods("PUT", "OPTIONS")
	courses.HandleFunc("/{id:[0-9]+}", h.deleteCourseHandler).Methods("DELETE", "OPTIONS")

	// User-course assignment endpoints
	courses.HandleFunc("/{id:[0-9]+}/admins", h.listCourseAdminsHandler).Methods("GET", "OPTIONS")
	courses.HandleFunc("/{id:[0-9]+}/admins", h.addCourseAdminHandler).Methods("POST", "OPTIONS")
	courses.HandleFunc("/{id:[0-9]+}/admins/{user_id}", h.removeCourseAdminHandler).Methods("DELETE", "OPTIONS")
}

// Course CRUD handlers
func (h *Handler) listCoursesHandler(w http.ResponseWriter, r *http.Request) {
	h.handleCoursesList(w, r)
}

func (h *Handler) createCourseHandler(w http.ResponseWriter, r *http.Request) {
	h.handleCourseCreate(w, r)
}

func (h *Handler) getCourseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}
	h.handleCourseGet(w, r, int32(courseID))
}

func (h *Handler) updateCourseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}
	h.handleCourseUpdate(w, r, int32(courseID))
}

func (h *Handler) deleteCourseHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}
	h.handleCourseDelete(w, r, int32(courseID))
}

// Course admin assignment handlers
func (h *Handler) listCourseAdminsHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}
	h.handleCourseAdminsList(w, r, int32(courseID))
}

func (h *Handler) addCourseAdminHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}
	h.handleCourseAdminAdd(w, r, int32(courseID))
}

func (h *Handler) removeCourseAdminHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	courseID, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, "Invalid course ID", http.StatusBadRequest)
		return
	}
	userID := vars["user_id"]
	h.handleCourseAdminRemove(w, r, int32(courseID), userID)
}
