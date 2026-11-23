package auth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"glossias/src/pkg/models"
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	clerkjwt "github.com/clerk/clerk-sdk-go/v2/jwt"
	"github.com/clerk/clerk-sdk-go/v2/user"
	"github.com/gorilla/mux"
)

type contextKey string

const UserIDKey contextKey = "userID"

// StripeWebhookPath receives Stripe's signed deliveries. It is authenticated
// by the webhook signature, so it bypasses Clerk auth and rate limiting.
const StripeWebhookPath = "/api/webhooks/stripe"

// jwtLeeway is the clock skew allowed when checking a session token's expiry
// and not-before times.
const jwtLeeway = 60 * time.Second

// fetchClerkUser loads a user's profile from Clerk. A variable so tests can
// stub Clerk.
var fetchClerkUser = user.Get

var byPassURLS = []string{
	"/api/health",
	"/api/db-health",
	"/api/time-tracking/record",
	StripeWebhookPath,
}

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
			if strings.HasPrefix(r.URL.Path, "/api/") && !slices.Contains(byPassURLS, r.URL.Path) {
				userID, err := extractAndValidateUser(r, logger)
				if err != nil {
					// WARN, not ERROR: anyone can trigger this, and it must not
					// page or flood the error log.
					logger.Warn("auth failed", "error", err, "path", r.URL.Path)
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
		if !models.UserRecentlySynced(devUser) {
			// Load the dev user from Clerk to maintain consistency
			clerkUser, err := fetchClerkUser(ctx, devUser)
			if err != nil {
				logger.Warn("failed to fetch dev user from Clerk", "error", err, "dev_user", devUser)
				return devUser, nil
			}

			// Sync dev user data with database
			email, name := clerkProfile(clerkUser)
			if _, err := models.UpsertUser(ctx, devUser, email, name); err != nil {
				logger.Error("failed to sync dev user to database", "error", err, "user_id", devUser)
			} else {
				models.MarkUserSynced(devUser)
			}
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

	// Verify the JWT. Clerk session tokens live about 60 seconds, so the clock
	// tolerance stays small: every second of leeway extends how long a leaked
	// token is accepted after it expires.
	claims, err := clerkjwt.Verify(r.Context(), &clerkjwt.VerifyParams{
		Token:  token,
		Leeway: jwtLeeway,
	})
	if err != nil {
		return "", err
	}

	// Extract user ID from claims
	userID := claims.Subject
	if userID == "" {
		return "", &clerk.APIErrorResponse{HTTPStatusCode: 401}
	}

	syncUser(ctx, userID, logger)
	return userID, nil
}

// syncUser copies the user's Clerk profile into the users table. A sync done
// on this instance within the last few minutes (models.UserRecentlySynced) is
// trusted, so most requests make no Clerk call and no write. The sync is
// recorded only when both the Clerk fetch and the upsert succeeded.
func syncUser(ctx context.Context, userID string, logger *slog.Logger) {
	if models.UserRecentlySynced(userID) {
		return
	}

	// Fetch full user details from Clerk to sync with database
	clerkUser, err := fetchClerkUser(ctx, userID)
	if err != nil {
		logger.Warn("failed to fetch user details from Clerk", "error", err, "user_id", userID)
		// Make sure a row exists for a first-time user, but never overwrite a
		// stored email and name with the empty values we have here.
		if syncErr := models.EnsureUserExists(ctx, userID); syncErr != nil {
			logger.Warn("failed basic user sync to database", "error", syncErr, "user_id", userID)
		}
		return
	}

	// Sync user data with database
	email, name := clerkProfile(clerkUser)
	if _, err := models.UpsertUser(ctx, userID, email, name); err != nil {
		// Don't fail the request if database sync fails, but make it loud: a
		// unique-email conflict here means the same address is on two users rows.
		// Logs the user id only: emails and names stay out of the logs.
		logger.Error("failed to sync user to database", "error", err, "user_id", userID)
		return
	}
	models.MarkUserSynced(userID)
}

// clerkProfile extracts the email and display name to store for a Clerk user.
// It prefers the primary email address; Clerk does not guarantee the order of
// EmailAddresses, so index 0 may be a stale or secondary address.
func clerkProfile(u *clerk.User) (email, name string) {
	if u.PrimaryEmailAddressID != nil {
		for _, ea := range u.EmailAddresses {
			if ea.ID == *u.PrimaryEmailAddressID {
				email = ea.EmailAddress
				break
			}
		}
	}
	if email == "" && len(u.EmailAddresses) > 0 {
		email = u.EmailAddresses[0].EmailAddress
	}

	switch {
	case u.FirstName != nil && u.LastName != nil:
		name = *u.FirstName + " " + *u.LastName
	case u.FirstName != nil:
		name = *u.FirstName
	case u.LastName != nil:
		name = *u.LastName
	}
	return email, name
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

// userIDForLog is for diagnostic logs only — not authorization. Rate limiting
// runs before auth, so context is usually empty; peek at DEV_USER or the JWT
// subject when present.
func userIDForLog(r *http.Request) string {
	if id := GetUserID(r); id != "" {
		return id
	}
	if devUser := os.Getenv("DEV_USER"); devUser != "" && r.Header.Get("dev_auth") == "12345678" {
		return devUser
	}
	return jwtSubject(r.Header.Get("Authorization"))
}

func jwtSubject(authHeader string) string {
	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader || token == "" {
		return ""
	}
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return ""
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ""
	}
	var claims struct {
		Sub string `json:"sub"`
	}
	if json.Unmarshal(payload, &claims) != nil {
		return ""
	}
	return claims.Sub
}

// HasPermission checks if user has permission to access a course (non-admin)
func HasReadPermission(ctx context.Context, userID string, courseID int32) bool {
	return models.CanUserAccessCourse(ctx, userID, courseID)
}

// IsAnyAdmin checks if user is admin of any course or super admin
func IsAnyAdmin(ctx context.Context, userID string) bool {
	return models.IsUserAnyAdmin(ctx, userID)
}

// IsCourseOrSuperAdmin checks if user is admin of one single course or super admin
func IsCourseOrSuperAdmin(ctx context.Context, userID string, courseID int32) bool {
	return models.IsUserCourseOrSuperAdmin(ctx, userID, courseID)
}

// IsCourseAddmin checks if user is admin of one single course (no super admin override)
func IsCourseAdmin(ctx context.Context, userID string, courseID int32) bool {
	return models.IsUserOnlyCourseAdmin(ctx, userID, courseID)
}
