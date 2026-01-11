package tests

import (
	"context"
	"testing"
	"time"

	shield "github.com/tanmaij/the-shield"
)

// TestMemoryLimiter ensures the in-memory sliding window logic works correctly.
func TestMemoryLimiter(t *testing.T) {
	cfg := shield.Config{
		Limit:  2,
		Window: 100 * time.Millisecond,
	}
	ctx := context.Background()

	tests := []struct {
		name       string
		identifier string
		requestCnt int
		sleep      time.Duration
		wantAllow  bool
		wantRemain int
	}{
		{
			name:       "Allow first request",
			identifier: "user_1",
			requestCnt: 1,
			wantAllow:  true,
			wantRemain: 1,
		},
		{
			name:       "Block when exceeding limit",
			identifier: "user_2",
			requestCnt: 3, // 1st: OK (rem:1), 2nd: OK (rem:0), 3rd: Blocked
			wantAllow:  false,
			wantRemain: 0,
		},
		{
			name:       "Recover after window",
			identifier: "user_3",
			requestCnt: 3,
			sleep:      120 * time.Millisecond,
			wantAllow:  true,
			wantRemain: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Isolation: New instance for every sub-test
			limiter := shield.NewMemoryLimiter(cfg)

			// Clean up resources after sub-test finishes
			t.Cleanup(func() {
				_ = limiter.Close(ctx)
			})

			var gotAllow bool
			var gotRemain int

			for i := 0; i < tc.requestCnt; i++ {
				if i == tc.requestCnt-1 && tc.sleep > 0 {
					time.Sleep(tc.sleep)
				}

				var err error
				gotAllow, gotRemain, err = limiter.Allow(ctx, tc.identifier)
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			if gotAllow != tc.wantAllow {
				t.Errorf("gotAllow = %v, want %v", gotAllow, tc.wantAllow)
			}
			if gotRemain != tc.wantRemain {
				t.Errorf("gotRemain = %d, want %d", gotRemain, tc.wantRemain)
			}
		})
	}
}
