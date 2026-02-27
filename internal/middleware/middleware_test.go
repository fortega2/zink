package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChain(t *testing.T) {
	order := make([]string, 0, 3)

	makeMiddleware := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name+":before")
				next.ServeHTTP(w, r)
				order = append(order, name+":after")
			})
		}
	}

	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(finalHandler, makeMiddleware("A"), makeMiddleware("B"))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	chained.ServeHTTP(w, req)

	assert.Equal(t, []string{"A:before", "B:before", "handler", "B:after", "A:after"}, order)
}

func TestChainEmpty(t *testing.T) {
	finalHandler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	chained := Chain(finalHandler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	chained.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

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

	logger := slog.Default()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := Logging(logger)(tt.handler)

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
	logger := slog.Default()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	handler := Chain(mux, Logging(logger))

	req := httptest.NewRequest(http.MethodPost, "/api/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	res := w.Result()
	require.NoError(t, res.Body.Close())
	assert.Equal(t, http.StatusAccepted, res.StatusCode)
}
