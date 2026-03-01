package proxy

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/fortega2/zink/internal/config"
	"github.com/fortega2/zink/internal/middleware"
)

const defaultTimeout = 5 * time.Second

type Router struct {
	mux     *http.ServeMux
	handler http.Handler
}

func NewRouter(cfg *config.Config, logger *slog.Logger) (*Router, error) {
	mux := http.NewServeMux()

	for _, svc := range cfg.Services {
		targets := make([]*url.URL, 0, len(svc.Target))
		for _, targetStr := range svc.Target {
			u, err := url.ParseRequestURI(targetStr)
			if err != nil {
				return nil, fmt.Errorf("invalid target URL '%s' for service '%s': %w", targetStr, svc.Name, err)
			}
			targets = append(targets, u)
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

		exactPath := strings.TrimSuffix(svc.PathPrefix, "/")
		prefixPath := exactPath + "/"

		if exactPath != "" {
			mux.Handle(exactPath, http.StripPrefix(exactPath, proxyHandler))
		}
		mux.Handle(prefixPath, http.StripPrefix(exactPath, proxyHandler))
	}

	return &Router{mux: mux, handler: mux}, nil
}

func (r *Router) Use(m ...middleware.Middleware) {
	r.handler = middleware.Chain(r.mux, m...)
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.handler.ServeHTTP(w, req)
}

func createProxy(targets []*url.URL, lbType config.LoadBalancer) (http.Handler, error) {
	director, err := NewDirector(lbType, targets)
	if err != nil {
		return nil, err
	}

	proxy := &httputil.ReverseProxy{
		Director: director,
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusBadGateway)
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), defaultTimeout)
		defer cancel()

		reqWithCtx := req.WithContext(ctx)
		proxy.ServeHTTP(w, reqWithCtx) //nolint:gosec // G704: targets are fixed from config, SSRF is not applicable for a reverse proxy
	}), nil
}
