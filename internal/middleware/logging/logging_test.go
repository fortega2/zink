package logging

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fortega2/zink/internal/middleware"
)

func TestLogging(t *testing.T) {
	tests := []struct {
		name           string
		handler        http.Handler
		expectedStatus int
	}{
		{
			name: "Logs 200 OK",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusOK)
			}),
			expectedStatus: http.StatusOK,
		},
		{
			name: "Logs 404 Not Found",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}),
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "Defaults to 200 when WriteHeader is never called",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte("ok"))
				assert.NoError(t, err)
			}),
			expectedStatus: http.StatusOK,
		},
	}

	logger := slog.New(slog.DiscardHandler)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := New(logger)(tt.handler)

			req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
			w := httptest.NewRecorder()

			handler.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestResponseWriterCapturesStatusCode(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: inner, statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusUnauthorized)

	assert.Equal(t, http.StatusUnauthorized, rw.statusCode)
	assert.Equal(t, http.StatusUnauthorized, inner.Code)
}

func TestLoggingGlobalMiddleware(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	handler := middleware.Chain(mux, New(logger))

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	require.NoError(t, res.Body.Close())
	assert.Equal(t, http.StatusAccepted, res.StatusCode)
}
