package proxy

import (
	"net/http"
	"net/url"
	"sync/atomic"
)

func roundRobin(targets []*url.URL) func(*http.Request) {
	var current uint64

	return func(req *http.Request) {
		idx := atomic.AddUint64(&current, 1) % uint64(len(targets))
		target := targets[idx]

		if target.Scheme != "http" && target.Scheme != "https" {
			req.URL.Scheme = ""
			req.URL.Host = ""
			return
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Host = target.Host
	}
}
