package shield

import (
	"context"
	"time"
)

// Limiter defines the interface for rate limiting logic.
type Limiter interface {
	// Allow checks if a request from a specific identifier is allowed.
	// Returns (allowed, remaining, error)
	Allow(ctx context.Context, identifier string) (bool, int, error)
	// Close releases any resources held by the limiter.
	Close(ctx context.Context) error
}

// Config holds the rate limit settings.
type Config struct {
	Limit  int           // Maximum number of requests
	Window time.Duration // Time window (e.g., 1 minute)
}
