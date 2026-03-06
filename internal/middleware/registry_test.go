package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var discardLogger = slog.New(slog.DiscardHandler)

func noopFactory(_ context.Context, _ any, _ *slog.Logger) (Middleware, error) {
	return func(next http.Handler) http.Handler { return next }, nil
}

func errorFactory(_ context.Context, _ any, _ *slog.Logger) (Middleware, error) {
	return nil, errors.New("factory failed")
}

func TestRegistryBuildSuccess(t *testing.T) {
	reg := NewRegistry()
	reg.Register("noop", noopFactory)

	entries := []Entry{{TypeName: "noop", Value: nil}}
	chain, err := reg.Build(context.Background(), entries, discardLogger)

	require.NoError(t, err)
	require.Len(t, chain, 1)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	Chain(http.HandlerFunc(func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
	}), chain...).ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRegistryBuildUnknownType(t *testing.T) {
	reg := NewRegistry()

	entries := []Entry{{TypeName: "unknown", Value: nil}}
	chain, err := reg.Build(context.Background(), entries, discardLogger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown")
	assert.Nil(t, chain)
}

func TestRegistryBuildFactoryError(t *testing.T) {
	reg := NewRegistry()
	reg.Register("bad", errorFactory)

	entries := []Entry{{TypeName: "bad", Value: nil}}
	chain, err := reg.Build(context.Background(), entries, discardLogger)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "factory failed")
	assert.Nil(t, chain)
}

func TestRegistryBuildEmpty(t *testing.T) {
	reg := NewRegistry()

	chain, err := reg.Build(context.Background(), nil, discardLogger)

	require.NoError(t, err)
	assert.Empty(t, chain)
}
