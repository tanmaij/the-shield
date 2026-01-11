package shield

import (
	"context"
	_ "embed"
	"time"

	"github.com/redis/go-redis/v9"
)

// KeyPrefix is the prefix used for all keys stored in Redis.
const KeyPrefix = "shield:"

// Use go:embed to load Lua script from a separate file for better management
// For simplicity, I'll define it as a string here.
const slidingWindowLua = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local window = tonumber(ARGV[2])
local limit = tonumber(ARGV[3])
local clear_before = now - window

-- Remove old entries outside the current sliding window
redis.call('ZREMRANGEBYSCORE', key, 0, clear_before)

-- Count elements in the current window
local current_count = redis.call('ZCARD', key)

if current_count < limit then
    -- Add the current request timestamp
    redis.call('ZADD', key, now, now)
    -- Set expiry to auto-clean memory after the window passes
    redis.call('PEXPIRE', key, window)
    return {1, limit - current_count - 1}
else
    return {0, 0}
end
`

type redisLimiter struct {
	client *redis.Client
	cfg    Config
	script *redis.Script
}

// NewRedisLimiter creates a new Redis-based sliding window limiter.
func NewRedisLimiter(client *redis.Client, cfg Config) Limiter {
	return &redisLimiter{
		client: client,
		cfg:    cfg,
		script: redis.NewScript(slidingWindowLua),
	}
}

// Allow checks if the request is permitted based on the sliding window algorithm.
func (r *redisLimiter) Allow(ctx context.Context, identifier string) (bool, int, error) {
	now := time.Now().UnixMilli()
	windowMs := r.cfg.Window.Milliseconds()

	// Keys: [shield:user_123]
	// Args: [current_timestamp, window_size_ms, max_limit]
	values, err := r.script.Run(ctx, r.client, []string{KeyPrefix + identifier}, now, windowMs, r.cfg.Limit).Result()
	if err != nil {
		return false, 0, err
	}

	res := values.([]interface{})
	allowed := res[0].(int64) == 1
	remaining := int(res[1].(int64))

	return allowed, remaining, nil
}

// Close releases any resources held by the limiter.
func (r *redisLimiter) Close(ctx context.Context) error {
	// no background goroutine to stop; implement to satisfy interface
	return nil
}
