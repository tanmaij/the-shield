# The Shield

The Shield is a lightweight Go library/application for API rate limiting, supporting both in-memory and Redis-based distributed modes.

## Description

The Shield provides a `net/http` middleware that applies sliding window rate limiting.
It is designed to be easy to integrate, minimal in dependencies, and flexible enough to run in:

- Single-instance mode using in-memory storage
- Multi-instance mode using Redis (via Lua scripts)

## Installation

Install as a dependency:

```bash
go get github.com/tanmaij/the-shield@latest
```

## Quick start (memory limiter)

```go
package main

import (
    "context"
    "net/http"
    "time"

    shield "github.com/tanmaij/the-shield"
)

func main() {
    cfg := shield.Config{Limit: 10, Window: 1 * time.Minute}
    ml := shield.NewMemoryLimiter(cfg)
    defer ml.Close(context.Background())

    mux := http.NewServeMux()
    mux.Handle("/ping", shield.Middleware(ml, shield.KeyFromRequest)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })))

    http.ListenAndServe(":8080", mux)
}
```

## Quick start (Redis limiter)

```go
package main

import (
    "context"
    "net/http"
    "time"

    "github.com/redis/go-redis/v9"

    shield "github.com/tanmaij/the-shield"
)

func main() {
    client := redis.NewClient(&redis.Options{Addr: "localhost:6379"})
    defer client.Close()

    cfg := shield.Config{Limit: 100, Window: 1 * time.Minute}
    rl := shield.NewRedisLimiter(client, cfg)
    defer rl.Close(context.Background())

    mux := http.NewServeMux()
    mux.Handle("/ping", shield.Middleware(rl, shield.KeyFromRequest)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Write([]byte("pong"))
    })))

    http.ListenAndServe(":8080", mux)
}
```

## Notes

- Always call `Close(ctx)` on a limiter when your app is shutting down to stop background goroutines (in-memory limiter).
- Redis keys use the exported prefix `shield.KeyPrefix` (default: `shield:`).
- Middleware uses a 5s request context timeout by default; adjust if your limiter calls may block longer.

## Main Features

- Rate limiting using a sliding time window
- Two limiter implementations:
  - `memory` — local, in-memory limiter
  - `redis` — distributed limiter using Redis + Lua
- Standard `net/http` middleware — easy to wrap any handler
- Sample endpoint `/ping` for quick testing

## Project Structure

- `pkg/shield` — core library:
- `limiter.go` — limiter interface and configuration
- `memory.go` — in-memory limiter implementation
- `redis_lua.go` — Redis-based limiter using Lua scripts (sliding window)
- `middleware.go` — `net/http` middleware
- `cmd/example/main.go` — example application using the middleware
- `docker-compose.yaml` — Redis setup and application runtime via Docker
