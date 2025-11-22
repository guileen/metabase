package middleware

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiterConfig struct for configurable rate limiting
type RateLimiterConfig struct {
	Limit  int           // requests allowed per window
	Burst  int           // maximum burst size
	Window time.Duration // time window in seconds
}

// RateLimiter represents a token bucket rate limiter
type RateLimiter struct {
	limit    int
	burst    int
	window   time.Duration
	tokens   int
	lastSeen time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit, burst int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		limit:  limit,
		burst:  burst,
		window: window,
		tokens: burst,
	}
}

// Allow checks if a request should be allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastSeen)

	// Add tokens based on elapsed time
	tokensToAdd := int(elapsed.Seconds() * float64(rl.limit) / rl.window.Seconds())
	rl.tokens += tokensToAdd

	if rl.tokens > rl.burst {
		rl.tokens = rl.burst
	}

	rl.lastSeen = now

	if rl.tokens >= 1 {
		rl.tokens--
		return true
	}

	return false
}

// RateLimiterHandler creates a rate limiting middleware
func RateLimiterHandler(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !rl.Allow() {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RateLimit middleware function with config
func RateLimitConfig(config RateLimiterConfig) func(http.Handler) http.Handler {
	rl := NewRateLimiter(config.Limit, config.Burst, config.Window)
	return RateLimiterHandler(rl)
}

// Global rate limiter instance (simple implementation)
var globalRateLimiter = NewRateLimiter(100, 20, 60*time.Second)

// Simple rate limiter middleware
func RateLimit(next http.Handler) http.Handler {
	return RateLimiterHandler(globalRateLimiter)(next)
}
