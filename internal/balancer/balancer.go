package balancer

import (
	"fmt"
	"net/http"
	"net/url"
	"sync/atomic"

	"github.com/fortega2/zink/internal/config"
)

// NewDirector returns a Director function for use with httputil.ReverseProxy
// based on the given load balancer type and target URLs.
func NewDirector(lbType config.LoadBalancer, targets []*url.URL) (func(*http.Request), error) {
	switch lbType {
	case config.LoadBalancerRoundRobin:
		return RoundRobin(targets), nil
	default:
		return nil, fmt.Errorf("unknown load_balancer type %q for service", lbType)
	}
}

// RoundRobin returns a Director function that distributes requests across
// targets using an atomic round-robin counter.
func RoundRobin(targets []*url.URL) func(*http.Request) {
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
