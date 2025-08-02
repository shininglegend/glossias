// internal/pkg/auth/clerk.go
package auth

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/clerk/clerk-sdk-go/v2"
)

type ClerkMiddleware struct {
	client clerk.Client
}

func NewClerkMiddleware() (*ClerkMiddleware, error) {
	key := os.Getenv("CLERK_SECRET_KEY")
	if key == "" {
		return nil, errors.New("CLERK_SECRET_KEY is not set")
	}

	clerk.SetKey(key)
	return &ClerkMiddleware{client: clerk.Client{}}, nil
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
		if token == "" {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		// Add token to context for downstream handlers
		ctx := context.WithValue(r.Context(), "clerk_token", token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
