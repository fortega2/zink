package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/fortega2/zink/internal/config"
	"github.com/fortega2/zink/internal/middleware"
)

// Router is an HTTP handler that routes requests to backend services
// through reverse proxies with per-service and global middleware chains.
type Router struct {
	mux     *http.ServeMux
	handler http.Handler
}

// NewRouter builds a Router from cfg, applying per-service middlewares via registry.
func NewRouter(ctx context.Context, cfg *config.Config, logger *slog.Logger, registry *middleware.Registry) (*Router, error) {
	mux := http.NewServeMux()

	for _, svc := range cfg.Services {
		targets, err := parseServiceTargets(svc.Target, svc.Name)
		if err != nil {
			return nil, err
		}

		lbType := svc.LoadBalancer
		if lbType == "" {
			logger.Warn("load_balancer not set, defaulting to round_robin", "service", svc.Name)
			lbType = config.LoadBalancerRoundRobin
		}

		proxyHandler, err := createProxy(targets, lbType)
		if err != nil {
			return nil, fmt.Errorf("service '%s': %w", svc.Name, err)
		}

		svcHandler, err := applyServiceMiddlewares(ctx, proxyHandler, svc.Middlewares, logger, registry)
		if err != nil {
			return nil, fmt.Errorf("service '%s': %w", svc.Name, err)
		}

		exactPath := strings.TrimSuffix(svc.PathPrefix, "/")
		prefixPath := exactPath + "/"

		if exactPath != "" {
			mux.Handle(exactPath, http.StripPrefix(exactPath, svcHandler))
		}
		mux.Handle(prefixPath, http.StripPrefix(exactPath, svcHandler))
	}

	return &Router{mux: mux, handler: mux}, nil
}

func parseServiceTargets(targetStrs []string, svcName string) ([]*url.URL, error) {
	targets := make([]*url.URL, 0, len(targetStrs))
	for _, targetStr := range targetStrs {
		u, err := url.ParseRequestURI(targetStr)
		if err != nil {
			return nil, fmt.Errorf("invalid target URL '%s' for service '%s': %w", targetStr, svcName, err)
		}
		targets = append(targets, u)
	}
	return targets, nil
}

func applyServiceMiddlewares(ctx context.Context, h http.Handler, mws []config.MiddlewareConfig, logger *slog.Logger, registry *middleware.Registry) (http.Handler, error) {
	entries := make([]middleware.Entry, 0, len(mws))
	for _, mw := range mws {
		entries = append(entries, middleware.Entry{
			TypeName: string(mw.TypeName),
			Value:    mw.Value,
		})
	}

	chain, err := registry.Build(ctx, entries, logger)
	if err != nil {
		return nil, err
	}

	return middleware.Chain(h, chain...), nil
}

// Use wraps the entire mux with one or more global middlewares.
func (r *Router) Use(m ...middleware.Middleware) {
	r.handler = middleware.Chain(r.mux, m...)
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}
