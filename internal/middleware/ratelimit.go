package middleware

import (
	"context"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	rateLimitCleanupInterval = 5 * time.Minute
	rateLimitClientTTL       = 10 * time.Minute
)

type rateLimitClient struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type rateLimiter struct {
	mu      sync.Mutex
	clients map[string]*rateLimitClient
	rate    float64
	burst   int
}

func RateLimit(ctx context.Context, rate float64, burst int) Middleware {
	rl := newRateLimiter(ctx, rate, burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}

			limiter := rl.getLimiter(ip)
			if !limiter.Allow() {
				reservation := limiter.Reserve()
				retryAfter := int(math.Ceil(reservation.Delay().Seconds()))
				reservation.Cancel()
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func newRateLimiter(ctx context.Context, rate float64, burst int) *rateLimiter {
	rl := &rateLimiter{
		clients: make(map[string]*rateLimitClient),
		rate:    rate,
		burst:   burst,
	}
	go rl.cleanupLoop(ctx)
	return rl
}

func (rl *rateLimiter) getLimiter(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	client, ok := rl.clients[ip]
	if !ok {
		client = &rateLimitClient{
			limiter:  rate.NewLimiter(rate.Limit(rl.rate), rl.burst),
			lastSeen: time.Now(),
		}
		rl.clients[ip] = client
	}
	client.lastSeen = time.Now()
	return client.limiter
}

func (rl *rateLimiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(rateLimitCleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			rl.cleanup()
		}
	}
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-rateLimitClientTTL)
	for ip, client := range rl.clients {
		if client.lastSeen.Before(threshold) {
			delete(rl.clients, ip)
		}
	}
}
