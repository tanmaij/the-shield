package shield

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// KeyFunc defines a function to extract the identifier (IP, API Key, UserID) from the request.
type KeyFunc func(r *http.Request) string

// Middleware is a standard Go net/http middleware.
// It can be used with any framework that supports the standard library.
func Middleware(limiter Limiter, keyFunc KeyFunc) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFunc(r)
			if key == "" {
				http.Error(w, "Bad Request", http.StatusBadRequest)
				return
			}

			ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
			defer cancel()

			allowed, remaining, err := limiter.Allow(ctx, key)
			if err != nil {
				// fail-open
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))

			if !allowed {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Retry-After", "60")
				w.WriteHeader(http.StatusTooManyRequests)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"error": "Rate limit exceeded",
				})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
