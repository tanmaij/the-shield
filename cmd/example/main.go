package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
	shield "github.com/tanmaij/the-shield"
)

func main() {
	// 1. Configuration from Environment Variables
	redisAddr := os.Getenv("REDIS_ADDR")
	limit := 100
	window := 1 * time.Minute

	var limiter shield.Limiter

	// 2. Decide which provider to use
	if redisAddr != "" {
		// Use Redis Distributed Limiter
		rdb := redis.NewClient(&redis.Options{
			Addr: redisAddr,
		})
		limiter = shield.NewRedisLimiter(rdb, shield.Config{
			Limit:  limit,
			Window: window,
		})
		log.Printf("The Shield is guarding via Redis at %s (Limit: %d, Window: %v)", redisAddr, limit, window)
	} else {
		// Fallback to In-Memory Limiter
		limiter = shield.NewMemoryLimiter(shield.Config{
			Limit:  limit,
			Window: window,
		})
		log.Printf("The Shield is guarding via Memory (Limit: %d, Window: %v)", limit, window)
	}

	// 3. Define how to identify users (e.g., by IP Address)
	keyFunc := func(r *http.Request) string {
		// In production, consider headers like X-Forwarded-For if behind a proxy
		return r.RemoteAddr
	}

	// 4. Create a simple API handler
	mux := http.NewServeMux()
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	// 5. Wrap the mux with The Shield middleware
	shieldMiddleware := shield.Middleware(limiter, keyFunc)

	serverAddr := ":8080"
	log.Printf("Server starting at %s", serverAddr)

	// Start server with protected routes
	if err := http.ListenAndServe(serverAddr, shieldMiddleware(mux)); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
