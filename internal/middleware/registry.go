package middleware

import (
	"context"
	"fmt"
	"log/slog"
)

// Factory is a function that constructs a Middleware from an opaque config value.
// ctx is provided for middlewares that spawn background goroutines (e.g. rate limiter cleanup).
// logger is available for middlewares that need structured logging.
type Factory func(ctx context.Context, value any, logger *slog.Logger) (Middleware, error)

// Registry maps middleware type identifiers to their Factory functions.
// Register factories at startup; the router uses Build to construct per-service chains.
type Registry struct {
	factories map[string]Factory
}

// Entry holds a single middleware config value together with its type name.
type Entry struct {
	TypeName string
	Value    any
}

// NewRegistry returns an empty Registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register associates a Factory with the given type identifier.
func (r *Registry) Register(typeName string, f Factory) {
	r.factories[typeName] = f
}

// Build constructs a chain of Middlewares from a slice of raw config entries.
// Each entry must have a Value recognised by a registered Factory; an unregistered
// type causes Build to return an error immediately.
func (r *Registry) Build(ctx context.Context, entries []Entry, logger *slog.Logger) ([]Middleware, error) {
	chain := make([]Middleware, 0, len(entries))
	for _, e := range entries {
		factory, ok := r.factories[e.TypeName]
		if !ok {
			return nil, fmt.Errorf("middleware %q: no factory registered", e.TypeName)
		}
		mw, err := factory(ctx, e.Value, logger)
		if err != nil {
			return nil, fmt.Errorf("middleware %q: %w", e.TypeName, err)
		}
		chain = append(chain, mw)
	}
	return chain, nil
}
