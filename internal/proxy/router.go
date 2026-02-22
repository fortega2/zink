package proxy

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/fortega2/zink/internal/config"
)

const defaultTimeout = 10 * time.Second

type Router struct {
	mux *http.ServeMux
}

func NewRouter(cfg *config.Config) (*Router, error) {
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

		proxyHandler := createProxy(targets)

		exactPath := strings.TrimSuffix(svc.PathPrefix, "/")
		prefixPath := exactPath + "/"

		if exactPath != "" {
			mux.Handle(exactPath, http.StripPrefix(exactPath, proxyHandler))
		}
		mux.Handle(prefixPath, http.StripPrefix(exactPath, proxyHandler))
	}

	return &Router{mux: mux}, nil
}

func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

func createProxy(targets []*url.URL) http.Handler {
	var current uint64

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			idx := atomic.AddUint64(&current, 1) % uint64(len(targets))
			target := targets[idx]

			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			w.WriteHeader(http.StatusBadGateway)
		},
	}

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx, cancel := context.WithTimeout(req.Context(), defaultTimeout)
		defer cancel()

		reqWithCtx := req.WithContext(ctx)
		proxy.ServeHTTP(w, reqWithCtx)
	})
}
