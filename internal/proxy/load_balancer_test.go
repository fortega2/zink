package proxy

import (
	"context"
	"net/http"
	"net/url"
	"testing"
)

const (
	testBackend1         = "backend1.example.com"
	testBackend2         = "backend2.example.com"
	testBackend3         = "backend3.example.com"
	testBackendHTTP      = "backend.example.com:8080"
	testBackendHTTPS     = "secure.example.com"
	testBackendSingle    = "single.example.com:8443"
	testOriginalURL      = "http://original.com/path"
	testSchemeHTTP       = "http"
	testSchemeHTTPS      = "https"
	testErrSchemeFormat  = "got scheme %q, want %q"
	testErrHostFormat    = "got host %q, want %q"
	testErrURLHostFormat = "got URL.Host %q, want %q"
)

func TestRoundRobinDistribution(t *testing.T) {
	target1, _ := url.Parse(testSchemeHTTP + "://" + testBackend1)
	target2, _ := url.Parse(testSchemeHTTP + "://" + testBackend2)
	target3, _ := url.Parse(testSchemeHTTP + "://" + testBackend3)
	targets := []*url.URL{target1, target2, target3}

	director := roundRobin(targets)
	testCases := []struct {
		expectedHost string
	}{
		{testBackend2},
		{testBackend3},
		{testBackend1},
		{testBackend2},
		{testBackend3},
	}

	for i, tc := range testCases {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testOriginalURL, nil)
		director(req)

		if req.URL.Host != tc.expectedHost {
			t.Errorf("request %d: "+testErrHostFormat, i, req.URL.Host, tc.expectedHost)
		}
		if req.Host != tc.expectedHost {
			t.Errorf("request %d: got req.Host %q, want %q", i, req.Host, tc.expectedHost)
		}
		if req.URL.Scheme != testSchemeHTTP {
			t.Errorf("request %d: "+testErrSchemeFormat, i, req.URL.Scheme, testSchemeHTTP)
		}
	}
}

func TestRoundRobinSingleTarget(t *testing.T) {
	target, _ := url.Parse(testSchemeHTTPS + "://" + testBackendSingle)
	targets := []*url.URL{target}

	director := roundRobin(targets)

	for range 5 {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testOriginalURL, nil)
		director(req)

		if req.URL.Host != testBackendSingle {
			t.Errorf(testErrHostFormat, req.URL.Host, testBackendSingle)
		}
		if req.URL.Scheme != testSchemeHTTPS {
			t.Errorf(testErrSchemeFormat, req.URL.Scheme, testSchemeHTTPS)
		}
	}
}

func TestRoundRobinHTTPScheme(t *testing.T) {
	target, _ := url.Parse(testSchemeHTTP + "://" + testBackendHTTP)
	targets := []*url.URL{target}

	director := roundRobin(targets)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testOriginalURL, nil)
	director(req)

	if req.URL.Scheme != testSchemeHTTP {
		t.Errorf(testErrSchemeFormat, req.URL.Scheme, testSchemeHTTP)
	}
	if req.URL.Host != testBackendHTTP {
		t.Errorf(testErrURLHostFormat, req.URL.Host, testBackendHTTP)
	}
	if req.Host != testBackendHTTP {
		t.Errorf("got req.Host %q, want %q", req.Host, testBackendHTTP)
	}
}

func TestRoundRobinHTTPSScheme(t *testing.T) {
	target, _ := url.Parse(testSchemeHTTPS + "://" + testBackendHTTPS)
	targets := []*url.URL{target}

	director := roundRobin(targets)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testOriginalURL, nil)
	director(req)

	if req.URL.Scheme != testSchemeHTTPS {
		t.Errorf(testErrSchemeFormat, req.URL.Scheme, testSchemeHTTPS)
	}
	if req.URL.Host != testBackendHTTPS {
		t.Errorf(testErrURLHostFormat, req.URL.Host, testBackendHTTPS)
	}
	if req.Host != testBackendHTTPS {
		t.Errorf("got req.Host %q, want %q", req.Host, testBackendHTTPS)
	}
}

func TestRoundRobinInvalidScheme(t *testing.T) {
	target := &url.URL{
		Scheme: "ftp",
		Host:   "ftp.example.com",
	}
	targets := []*url.URL{target}

	director := roundRobin(targets)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testOriginalURL, nil)

	req.URL.Scheme = testSchemeHTTP
	req.URL.Host = "original.com"

	director(req)

	if req.URL.Scheme != "" {
		t.Errorf("got scheme %q, want empty string", req.URL.Scheme)
	}
	if req.URL.Host != "" {
		t.Errorf("got URL.Host %q, want empty string", req.URL.Host)
	}
}

func TestRoundRobinMixedSchemes(t *testing.T) {
	target1, _ := url.Parse(testSchemeHTTP + "://" + testBackend1)
	target2, _ := url.Parse(testSchemeHTTPS + "://" + testBackend2)
	targets := []*url.URL{target1, target2}

	director := roundRobin(targets)

	req1, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testOriginalURL, nil)
	director(req1)
	if req1.URL.Scheme != testSchemeHTTPS || req1.URL.Host != testBackend2 {
		t.Errorf("first request: got %s://%s, want %s://%s",
			req1.URL.Scheme, req1.URL.Host, testSchemeHTTPS, testBackend2)
	}

	req2, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, testOriginalURL, nil)
	director(req2)
	if req2.URL.Scheme != testSchemeHTTP || req2.URL.Host != testBackend1 {
		t.Errorf("second request: got %s://%s, want %s://%s",
			req2.URL.Scheme, req2.URL.Host, testSchemeHTTP, testBackend1)
	}
}

func TestRoundRobinPreservesPathAndQuery(t *testing.T) {
	target, _ := url.Parse(testSchemeHTTP + "://backend.example.com")
	targets := []*url.URL{target}

	director := roundRobin(targets)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://original.com/api/v1/users?limit=10", nil)

	originalPath := req.URL.Path
	originalQuery := req.URL.RawQuery

	director(req)

	if req.URL.Path != originalPath {
		t.Errorf("path changed: got %q, want %q", req.URL.Path, originalPath)
	}
	if req.URL.RawQuery != originalQuery {
		t.Errorf("query changed: got %q, want %q", req.URL.RawQuery, originalQuery)
	}
}
