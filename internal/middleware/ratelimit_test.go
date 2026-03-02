package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const fakeRemoteAddr = "1.2.3.4:1234"

func TestRateLimitAlwaysEnforces(t *testing.T) {
	handler := RateLimit(t.Context(), 1, 1)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = fakeRemoteAddr
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = fakeRemoteAddr
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTooManyRequests, w.Code)
}

func TestRateLimitAllowsUnderLimit(t *testing.T) {
	handler := RateLimit(t.Context(), 100, 10)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := range 10 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = fakeRemoteAddr
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code, "request %d should be allowed", i+1)
	}
}

func TestRateLimitRejectsOverBurst(t *testing.T) {
	handler := RateLimit(t.Context(), 1, 3)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	allowed := 0
	rejected := 0
	for range 10 {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:9999"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		if w.Code == http.StatusOK {
			allowed++
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
			rejected++
		}
	}

	assert.Equal(t, 3, allowed, "only burst=3 requests should be allowed")
	assert.Equal(t, 7, rejected, "remaining 7 requests should be rejected")
}

func TestRateLimitIsolatesClients(t *testing.T) {
	handler := RateLimit(t.Context(), 1, 1)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	ips := []string{"192.168.1.1:1000", "192.168.1.2:1000", "192.168.1.3:1000"}
	for _, ip := range ips {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "first request from %s should be allowed", ip)
	}

	for _, ip := range ips {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = ip
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, http.StatusTooManyRequests, w.Code, "second request from %s should be rejected", ip)
	}
}

func TestRateLimitInvalidRemoteAddr(t *testing.T) {
	handler := RateLimit(t.Context(), 100, 5)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "not-a-valid-addr"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitIntegrationWithChain(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	handler := Chain(mux, RateLimit(t.Context(), 1, 2))

	for i, wantStatus := range []int{http.StatusAccepted, http.StatusAccepted, http.StatusTooManyRequests} {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "5.5.5.5:5555"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		assert.Equal(t, wantStatus, w.Code, "request %d: unexpected status", i+1)
	}
}

func TestRateLimitCleanupStopsOnContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	goroutinesBefore := runtime.NumGoroutine()
	_ = newRateLimiter(ctx, 100, 10)

	goroutinesAfterSpawn := runtime.NumGoroutine()
	assert.Greater(t, goroutinesAfterSpawn, goroutinesBefore, "cleanupLoop goroutine should have been spawned")

	cancel()

	assert.Eventually(t, func() bool {
		return runtime.NumGoroutine() <= goroutinesBefore
	}, time.Second, 10*time.Millisecond, "cleanupLoop goroutine should exit after context cancellation")
}
