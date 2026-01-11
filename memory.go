package shield

import (
	"context"
	"sync"
	"time"
)

// memoryLimiter implements Limiter interface using local system memory.
// It is suitable for single-instance applications or local testing.
type memoryLimiter struct {
	// store maps an identifier to a slice of request timestamps (millisecond).
	store map[string][]int64

	mu      sync.Mutex
	stopCh  chan struct{}
	wg      sync.WaitGroup
	cleanup time.Duration
	cfg     Config
}

// NewMemoryLimiter initializes a new in-memory sliding window limiter.
// It also starts a background goroutine to periodically clean up stale data.
func NewMemoryLimiter(cfg Config) Limiter {
	l := &memoryLimiter{
		cfg:    cfg,
		store:  make(map[string][]int64),
		stopCh: make(chan struct{}), // Initialize the channel here
	}

	go l.cleanupWorker()
	return l
}

// Allow checks if the request is permitted based on the sliding window algorithm.
func (m *memoryLimiter) Allow(ctx context.Context, identifier string) (bool, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now().UnixMilli()
	windowMs := m.cfg.Window.Milliseconds()
	boundary := now - windowMs

	// Retrieve timestamps for the given identifier.
	timestamps, exists := m.store[identifier]
	if !exists {
		timestamps = []int64{}
	}

	// Slide the window: Remove timestamps older than the window boundary.
	// Optimization: Find the first index that is within the window.
	validIdx := 0
	for i, ts := range timestamps {
		if ts > boundary {
			validIdx = i
			break
		}
		// If all timestamps are expired.
		if i == len(timestamps)-1 {
			validIdx = len(timestamps)
		}
	}

	// Slice the array to keep only valid timestamps.
	if validIdx > 0 {
		if validIdx >= len(timestamps) {
			timestamps = []int64{}
		} else {
			timestamps = timestamps[validIdx:]
		}
	}

	// Check if the current count exceeds the limit.
	if len(timestamps) < m.cfg.Limit {
		// Allow request and record the current timestamp.
		timestamps = append(timestamps, now)
		m.store[identifier] = timestamps

		remaining := m.cfg.Limit - len(timestamps)
		return true, remaining, nil
	}

	// Update the store even if blocked to keep the sliced state.
	m.store[identifier] = timestamps
	return false, 0, nil
}

// Close stops the cleanup worker and releases resources.
func (m *memoryLimiter) Close(ctx context.Context) error {
	close(m.stopCh)
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// cleanupWorker periodically removes identifiers that haven't had any activity
// beyond the window duration to keep memory usage under control.
func (m *memoryLimiter) cleanupWorker() {
	// We add m.wg.Add(1) and defer m.wg.Done() to track the goroutine
	m.wg.Add(1)
	defer m.wg.Done()

	ticker := time.NewTicker(m.cfg.Window * 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.mu.Lock()
			now := time.Now().UnixMilli()
			boundary := now - m.cfg.Window.Milliseconds()

			for id, timestamps := range m.store {
				if len(timestamps) == 0 || timestamps[len(timestamps)-1] < boundary {
					delete(m.store, id)
				}
			}
			m.mu.Unlock()
		case <-m.stopCh: // Listen for the Close() signal
			return
		}
	}
}
