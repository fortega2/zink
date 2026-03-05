package ratelimit

import (
	"context"
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/fortega2/zink/internal/middleware"
)

const (
	cleanupInterval = 5 * time.Minute
	clientTTL       = 10 * time.Minute
)

type client struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type limiter struct {
	mu      sync.Mutex
	clients map[string]*client
	rate    float64
	burst   int
}

// New returns a per-IP token bucket rate limiting middleware.
// Each unique client IP gets its own rate.Limiter with the given rate (req/s)
// and burst capacity. Stale client entries are cleaned up every 5 minutes.
// The cleanup goroutine respects ctx cancellation for graceful shutdown.
func New(ctx context.Context, r float64, burst int) middleware.Middleware {
	rl := newLimiter(ctx, r, burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ip, _, err := net.SplitHostPort(req.RemoteAddr)
			if err != nil {
				ip = req.RemoteAddr
			}

			l := rl.get(ip)
			if !l.Allow() {
				reservation := l.Reserve()
				retryAfter := int(math.Ceil(reservation.Delay().Seconds()))
				reservation.Cancel()
				w.Header().Set("Retry-After", strconv.Itoa(retryAfter))
				http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, req)
		})
	}
}

func newLimiter(ctx context.Context, r float64, burst int) *limiter {
	rl := &limiter{
		clients: make(map[string]*client),
		rate:    r,
		burst:   burst,
	}
	go rl.cleanupLoop(ctx)
	return rl
}

func (rl *limiter) get(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	c, ok := rl.clients[ip]
	if !ok {
		c = &client{
			limiter:  rate.NewLimiter(rate.Limit(rl.rate), rl.burst),
			lastSeen: time.Now(),
		}
		rl.clients[ip] = c
	}
	c.lastSeen = time.Now()
	return c.limiter
}

func (rl *limiter) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(cleanupInterval)
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

func (rl *limiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	threshold := time.Now().Add(-clientTTL)
	for ip, c := range rl.clients {
		if c.lastSeen.Before(threshold) {
			delete(rl.clients, ip)
		}
	}
}
