// internal/pkg/auth/clerk.go
package auth

import (
	"net/http"
	"os"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
)

type ClerkMiddleware struct {
	client clerk.Client
}

func NewClerkMiddleware() (*ClerkMiddleware, error) {
	client, err := clerk.NewClient(os.Getenv("CLERK_SECRET_KEY"))
	if err != nil {
		return nil, err
	}
	return &ClerkMiddleware{client: client}, nil
}

func (cm *ClerkMiddleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get token from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		// Verify the session
		sess, err := cm.client.Sessions().Verify(token)
		if err != nil {
			http.Error(w, "Invalid session", http.StatusUnauthorized)
			return
		}

		// Add session to context
		ctx := clerk.WithSession(r.Context(), sess)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
