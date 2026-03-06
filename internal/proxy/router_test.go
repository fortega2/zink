package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fortega2/zink/internal/config"
	"github.com/fortega2/zink/internal/middleware"
)

var logger = slog.New(slog.DiscardHandler)

func emptyRegistry() *middleware.Registry { return middleware.NewRegistry() }

func setupMockBackend(t *testing.T, name string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := fmt.Sprintf("%s received path: %s", name, r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(response)) //nolint:gosec // G705: test-only mock backend, not production code
		assert.NoError(t, err)
	}))
}

func TestRouterRouting(t *testing.T) {
	backend1 := setupMockBackend(t, "Backend 1")
	defer backend1.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:         8080,
			Host:         "localhost",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		Services: []config.ServiceConfig{
			{
				Name:       "test-service",
				PathPrefix: "/api/test",
				Target:     []string{backend1.URL},
			},
			{
				Name:       "admin-service",
				PathPrefix: "/api/admin/",
				Target:     []string{backend1.URL},
			},
		},
	}

	router, err := NewRouter(context.Background(), cfg, logger, emptyRegistry())
	require.NoError(t, err)

	tests := []struct {
		name           string
		requestURL     string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Exact match on non-trailing slash prefix",
			requestURL:     "/api/test",
			expectedStatus: http.StatusOK,
			expectedBody:   "Backend 1 received path: /",
		},
		{
			name:           "Subpath match on non-trailing slash prefix",
			requestURL:     "/api/test/users",
			expectedStatus: http.StatusOK,
			expectedBody:   "Backend 1 received path: /users",
		},
		{
			name:           "Exact match on trailing slash prefix",
			requestURL:     "/api/admin/",
			expectedStatus: http.StatusOK,
			expectedBody:   "Backend 1 received path: /",
		},
		{
			name:           "Subpath match on trailing slash prefix",
			requestURL:     "/api/admin/dashboard",
			expectedStatus: http.StatusOK,
			expectedBody:   "Backend 1 received path: /dashboard",
		},
		{
			name:           "Path not found",
			requestURL:     "/api/unknown",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.requestURL, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()

			assert.Equal(t, tt.expectedStatus, res.StatusCode)

			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedBody, string(bodyBytes))
		})
	}
}

func TestRouterRoundRobin(t *testing.T) {
	backendA := setupMockBackend(t, "Backend A")
	defer backendA.Close()

	backendB := setupMockBackend(t, "Backend B")
	defer backendB.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Port: 8080},
		Services: []config.ServiceConfig{
			{
				Name:       "balance-service",
				PathPrefix: "/api/balance",
				Target:     []string{backendA.URL, backendB.URL},
			},
		},
	}

	router, err := NewRouter(context.Background(), cfg, logger, emptyRegistry())
	require.NoError(t, err)

	tests := []struct {
		name         string
		requestURL   string
		expectedBody string
	}{
		{"Request 1 goes to backend B", "/api/balance", "Backend B received path: /"},
		{"Request 2 goes to backend A", "/api/balance/foo", "Backend A received path: /foo"},
		{"Request 3 goes to backend B", "/api/balance/bar", "Backend B received path: /bar"},
		{"Request 4 goes to backend A", "/api/balance/baz", "Backend A received path: /baz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.requestURL, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			res := w.Result()
			defer res.Body.Close()
			assert.Equal(t, http.StatusOK, res.StatusCode)

			bodyBytes, err := io.ReadAll(res.Body)
			require.NoError(t, err)

			assert.True(t, strings.Contains(string(bodyBytes), tt.expectedBody), "Expected '%s' to be in response '%s'", tt.expectedBody, string(bodyBytes))
		})
	}
}

func TestRouterInvalidTargetURL(t *testing.T) {
	cfg := &config.Config{
		Services: []config.ServiceConfig{
			{
				Name:       "invalid-target",
				PathPrefix: "/api",
				Target:     []string{"http://[::1]:namedport"},
			},
		},
	}

	router, err := NewRouter(context.Background(), cfg, logger, emptyRegistry())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target URL")
	assert.Nil(t, router)
}
