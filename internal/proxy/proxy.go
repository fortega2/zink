package proxy

import (
	"context"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/fortega2/zink/internal/balancer"
	"github.com/fortega2/zink/internal/config"
)

const defaultTimeout = 5 * time.Second

func createProxy(targets []*url.URL, lbType config.LoadBalancer) (http.Handler, error) {
	director, err := balancer.NewDirector(lbType, targets)
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
