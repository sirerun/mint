// Package middleware provides HTTP middleware for the registry API.
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/sirerun/mint/registry/db"
	"github.com/sirerun/mint/registry/model"
)

type contextKey string

const publisherKey contextKey = "publisher"

// PublisherFromContext returns the authenticated publisher, if any.
func PublisherFromContext(ctx context.Context) *model.Publisher {
	p, _ := ctx.Value(publisherKey).(*model.Publisher)
	return p
}

// PublisherContextKey returns the context key used for the publisher.
// This is exported for use in tests.
func PublisherContextKey() contextKey {
	return publisherKey
}

// Auth returns middleware that authenticates requests via Bearer token.
// If required is true, unauthenticated requests are rejected with 401.
// If required is false, authentication is optional (for public endpoints
// where authenticated users get extra features like starring).
func Auth(store *db.DB, required bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearerToken(r)
			if token == "" {
				if required {
					http.Error(w, `{"error":"authentication required"}`, http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			hash := db.HashAPIKey(token)
			publisher, err := store.GetPublisherByAPIKeyHash(hash)
			if err != nil {
				if required {
					http.Error(w, `{"error":"invalid API key"}`, http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), publisherKey, publisher)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return ""
	}
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}
