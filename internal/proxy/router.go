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

type Router struct {
	mux     *http.ServeMux
	handler http.Handler
}

func NewRouter(ctx context.Context, cfg *config.Config, logger *slog.Logger) (*Router, error) {
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

		svcHandler, err := applyServiceMiddlewares(ctx, proxyHandler, svc.Middlewares, logger)
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

func applyServiceMiddlewares(ctx context.Context, h http.Handler, mws []config.MiddlewareConfig, logger *slog.Logger) (http.Handler, error) {
	chain := make([]middleware.Middleware, 0, len(mws))
	for _, mw := range mws {
		switch cfg := mw.Value.(type) {
		case config.RateLimitMiddleware:
			chain = append(chain, middleware.RateLimit(ctx, cfg.Rate, cfg.Burst))
		case config.AuthMiddleware:
			authMw, err := middleware.Auth(cfg)
			if err != nil {
				return nil, fmt.Errorf("auth middleware: %w", err)
			}
			chain = append(chain, authMw)
		default:
			logger.Warn("unknown middleware type, skipping", "type", fmt.Sprintf("%T", mw.Value))
		}
	}
	return middleware.Chain(h, chain...), nil
}

func (r *Router) Use(m ...middleware.Middleware) {
	r.handler = middleware.Chain(r.mux, m...)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}
