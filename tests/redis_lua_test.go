package tests

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	shield "github.com/tanmaij/the-shield"
)

// TestRedisLimiter validates the Lua Script logic inside a real Redis environment.
func TestRedisLimiter(t *testing.T) {
	// 1. Setup: Initialize real Redis client for integration testing
	rdb := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
	ctx := context.Background()

	// 2. Health Check: Skip the test if Redis is not available locally
	// This prevents the entire test suite from failing in environments without Redis
	if err := rdb.Ping(ctx).Err(); err != nil {
		t.Skip("Skipping Redis integration test: localhost:6379 not reachable")
	}

	cfg := shield.Config{
		Limit:  2,
		Window: 1 * time.Second,
	}

	// 3. Define Test Cases: Using a slice of structs for ordered execution
	tests := []struct {
		name       string
		identifier string
		requestCnt int
		sleep      time.Duration
		wantAllow  bool
	}{
		{
			name:       "Allow valid sequence within limit",
			identifier: "user_1",
			requestCnt: 2,
			sleep:      0,
			wantAllow:  true,
		},
		{
			name:       "Block request exceeding burst limit",
			identifier: "user_2",
			requestCnt: 3,
			sleep:      0,
			wantAllow:  false,
		},
		{
			name:       "Reset quota after window expiration",
			identifier: "user_3",
			requestCnt: 3, // 2 to fill limit, then 1 after sleep
			sleep:      1100 * time.Millisecond,
			wantAllow:  true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Ensure test isolation by cleaning up the specific key before and after
			key := shield.KeyPrefix + tc.identifier
			rdb.Del(ctx, key)
			t.Cleanup(func() {
				rdb.Del(ctx, key)
			})

			limiter := shield.NewRedisLimiter(rdb, cfg)
			var gotAllow bool

			for i := 0; i < tc.requestCnt; i++ {
				// If a sleep is required, trigger it before the final request
				// to verify window reset logic.
				if i == tc.requestCnt-1 && tc.sleep > 0 {
					time.Sleep(tc.sleep)
				}

				allow, _, err := limiter.Allow(ctx, tc.identifier)
				if err != nil {
					t.Fatalf("Unexpected Redis error at request %d: %v", i+1, err)
				}
				gotAllow = allow
			}

			// 4. Assertion
			if gotAllow != tc.wantAllow {
				t.Errorf("Result mismatch: got %v, want %v", gotAllow, tc.wantAllow)
			}
		})
	}
}
