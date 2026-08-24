package core

import (
	"net/http"
	"sync"
	"time"
)

// RateLimiter implements per-tenant token bucket rate limiting.
type RateLimiter struct {
	buckets sync.Map // tenantID -> *tokenBucket
	rate    int      // tokens per interval
	burst   int      // max burst size
}

type tokenBucket struct {
	tokens   float64
	lastTime time.Time
	mu       sync.Mutex
}

// NewRateLimiter creates a rate limiter with the given rate (requests/second) and burst size.
func NewRateLimiter(ratePerSecond int, burst int) *RateLimiter {
	return &RateLimiter{
		rate:  ratePerSecond,
		burst: burst,
	}
}

// Allow checks if a request from the given tenant is allowed.
func (rl *RateLimiter) Allow(tenantID string) bool {
	b := rl.getBucket(tenantID)
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastTime).Seconds()
	b.lastTime = now

	// Refill tokens
	b.tokens += elapsed * float64(rl.rate)
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

func (rl *RateLimiter) getBucket(tenantID string) *tokenBucket {
	if v, ok := rl.buckets.Load(tenantID); ok {
		return v.(*tokenBucket)
	}
	b := &tokenBucket{
		tokens:   float64(rl.burst),
		lastTime: time.Now(),
	}
	actual, _ := rl.buckets.LoadOrStore(tenantID, b)
	return actual.(*tokenBucket)
}

// Cleanup removes buckets for tenants that haven't made requests recently.
func (rl *RateLimiter) Cleanup(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	rl.buckets.Range(func(key, value any) bool {
		b := value.(*tokenBucket)
		b.mu.Lock()
		if b.lastTime.Before(cutoff) {
			rl.buckets.Delete(key)
		}
		b.mu.Unlock()
		return true
	})
}

// RateLimitMiddleware returns a Gin middleware that enforces per-tenant rate limiting.
func (rl *RateLimiter) RateLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-ID")
			if tenantID == "" {
				tenantID = "__anonymous__"
			}

			if !rl.Allow(tenantID) {
				w.Header().Set("Retry-After", "1")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`{"error":"rate limit exceeded","code":"RATE_LIMITED"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
