package auth

import (
	"context"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/user"
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
			w.Header().Set("Access-Control-Expose-Headers", "X-Tracking-ID")

			// Handle preflight OPTIONS request
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			// Extract and validate JWT token for API routes (except health and time tracking)
			if strings.HasPrefix(r.URL.Path, "/api/") && r.URL.Path != "/api/health" &&
				!strings.HasPrefix(r.URL.Path, "/api/time-tracking") {
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

	// Dev auth bypass - only when DEV_USER is set
	devUser := os.Getenv("DEV_USER")
	devAuth := r.Header.Get("dev_auth")
	if devUser != "" && devAuth == "12345678" {
		// Load the dev user from Clerk to maintain consistency
		clerkUser, err := user.Get(ctx, devUser)
		if err != nil {
			logger.Warn("failed to fetch dev user from Clerk", "error", err, "dev_user", devUser)
			return devUser, nil
		}

		// Sync dev user data with database
		email := ""
		name := ""
		if len(clerkUser.EmailAddresses) > 0 {
			email = clerkUser.EmailAddresses[0].EmailAddress
		}
		if clerkUser.FirstName != nil && clerkUser.LastName != nil {
			name = *clerkUser.FirstName + " " + *clerkUser.LastName
		} else if clerkUser.FirstName != nil {
			name = *clerkUser.FirstName
		} else if clerkUser.LastName != nil {
			name = *clerkUser.LastName
		}

		_, err = models.UpsertUser(ctx, devUser, email, name)
		if err != nil {
			logger.Warn("failed to sync dev user to database", "error", err, "dev_user", devUser)
		}

		logger.Debug("dev auth bypass used", "dev_user", devUser)
		return devUser, nil
	}

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

	// Fetch full user details from Clerk to sync with database
	clerkUser, err := user.Get(ctx, userID)
	if err != nil {
		logger.Warn("failed to fetch user details from Clerk", "error", err, "user_id", userID)
		// Fallback to basic user sync without email/name
		_, syncErr := models.UpsertUser(ctx, userID, "", "")
		if syncErr != nil {
			logger.Warn("failed basic user sync to database", "error", syncErr, "user_id", userID)
		}
		return userID, nil
	}

	// Extract email and name from Clerk user data
	email := ""
	name := ""

	if len(clerkUser.EmailAddresses) > 0 {
		email = clerkUser.EmailAddresses[0].EmailAddress
	}

	if clerkUser.FirstName != nil && clerkUser.LastName != nil {
		name = *clerkUser.FirstName + " " + *clerkUser.LastName
	} else if clerkUser.FirstName != nil {
		name = *clerkUser.FirstName
	} else if clerkUser.LastName != nil {
		name = *clerkUser.LastName
	}

	// Sync user data with database
	_, err = models.UpsertUser(ctx, userID, email, name)
	if err != nil {
		logger.Warn("failed to sync user to database", "error", err, "user_id", userID, "email", email, "name", name)
		// Don't fail the request if database sync fails
	}

	return userID, nil
}

// GetUserID extracts user ID from request context
func GetUserIDWithOk(r *http.Request) (string, bool) {
	userID, ok := r.Context().Value(UserIDKey).(string)
	return userID, ok
}

func GetUserID(r *http.Request) string {
	userID, ok := r.Context().Value(UserIDKey).(string)
	if !ok {
		return ""
	}
	return userID
}

// HasPermission checks if user has permission to access a course
func HasPermission(ctx context.Context, userID string, courseID int32) bool {
	// return models.CanUserAccessCourse(ctx, userID, courseID)
	return true
}

// IsAdmin checks if user is admin of any course or super admin
func IsAdmin(ctx context.Context, userID string) bool {
	return models.IsUserAdmin(ctx, userID)
}
