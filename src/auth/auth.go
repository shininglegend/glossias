package auth

import (
	"context"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/gorilla/mux"
)

type contextKey string

const UserIDKey contextKey = "userID"

// Middleware combines CORS and Clerk authentication
func Middleware(logger *slog.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			// Handle preflight OPTIONS request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Extract and validate JWT token for API routes
			if strings.HasPrefix(r.URL.Path, "/api/") {
				userID, err := extractAndValidateUser(r, logger)
				if err != nil {
					logger.Error("auth failed", "error", err, "path", r.URL.Path)
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}
				// Add user ID to request context
				ctx := context.WithValue(r.Context(), UserIDKey, userID)
				r = r.WithContext(ctx)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractAndValidateUser extracts user info from JWT and syncs with database
func extractAndValidateUser(r *http.Request, logger *slog.Logger) (string, error) {
	ctx := r.Context()
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", &clerk.APIErrorResponse{HTTPStatusCode: 401}
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return "", &clerk.APIErrorResponse{HTTPStatusCode: 401}
	}

	// Verify JWT token
	claims, err := clerkjwt.Verify(r.Context(), &clerkjwt.VerifyParams{
		Token: token,
	})
	if err != nil {
		return "", err
	}

	// Extract user ID from claims
	userID := claims.Subject
	if userID == "" {
		return "", &clerk.APIErrorResponse{HTTPStatusCode: 401}
	}

	// Basic user sync - just ensure user exists in database
	// TODO: Additional user info can be fetched separately
	_, err = models.UpsertUser(ctx, userID, "", "")
	if err != nil {
		logger.Warn("failed to sync user to database", "error", err, "user_id", userID)
		// Don't fail the request if database sync fails
	}

	return userID, nil
}

// GetUserID extracts user ID from request context
func GetUserID(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

// HasPermission checks if user has permission to access a course
func HasPermission(ctx context.Context, userID string, courseID int32) bool {
	return models.CanUserAccessCourse(ctx, userID, courseID)
}

// IsAdmin checks if user is admin of any course or super admin
func IsAdmin(ctx context.Context, userID string) bool {
	return models.IsUserAdmin(ctx, userID)
}
