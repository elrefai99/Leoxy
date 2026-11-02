package proxy

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
)

func countedBackend(t *testing.T, counter *atomic.Int64) *url.URL {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counter.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func deadBackend(t *testing.T) *url.URL {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	server.Close()
	return target
}

func statusFor(handler http.Handler, method string) int {
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(method, "/", nil))
	return response.Code
}

func TestRoundRobinDistributesEvenly(t *testing.T) {
	counters := make([]atomic.Int64, 3)
	targets := make([]*url.URL, 3)
	for i := range targets {
		targets[i] = countedBackend(t, &counters[i])
	}
	handler := LoadBalancedProxyHandler("/", targets, false)

	for i := 0; i < 30; i++ {
		if status := statusFor(handler, http.MethodGet); status != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i, status, http.StatusOK)
		}
	}

	for i := range counters {
		if got := counters[i].Load(); got != 10 {
			t.Fatalf("backend %d served %d requests, want 10", i, got)
		}
	}
}

func TestGetFailsOverToHealthyBackend(t *testing.T) {
	var alive atomic.Int64
	targets := []*url.URL{deadBackend(t), countedBackend(t, &alive)}
	handler := LoadBalancedProxyHandler("/", targets, false)

	for i := 0; i < 10; i++ {
		if status := statusFor(handler, http.MethodGet); status != http.StatusOK {
			t.Fatalf("request %d status = %d, want %d", i, status, http.StatusOK)
		}
	}
	if got := alive.Load(); got != 10 {
		t.Fatalf("healthy backend served %d requests, want 10", got)
	}
}

func TestPostToDeadBackendIsNotRetried(t *testing.T) {
	var alive atomic.Int64
	targets := []*url.URL{deadBackend(t), countedBackend(t, &alive)}
	handler := LoadBalancedProxyHandler("/", targets, false)

	want := []int{http.StatusBadGateway, http.StatusOK, http.StatusBadGateway, http.StatusOK}
	for i, expected := range want {
		if status := statusFor(handler, http.MethodPost); status != expected {
			t.Fatalf("request %d status = %d, want %d", i, status, expected)
		}
	}
	if got := alive.Load(); got != 2 {
		t.Fatalf("healthy backend served %d requests, want 2", got)
	}
}

func TestAllBackendsDownReturnsBadGateway(t *testing.T) {
	handler := LoadBalancedProxyHandler("/", []*url.URL{deadBackend(t), deadBackend(t)}, false)

	if status := statusFor(handler, http.MethodGet); status != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d", status, http.StatusBadGateway)
	}
}

func TestForwardedHeadersIgnoreClientSpoofing(t *testing.T) {
	seen := make(chan http.Header, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen <- r.Header.Clone()
	}))
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	for _, forward := range []bool{true, false} {
		handler := LoadBalancedProxyHandler("/", []*url.URL{target}, forward)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("X-Forwarded-For", "10.9.9.9")
		req.Header.Set("X-Real-IP", "10.9.9.9")
		handler.ServeHTTP(httptest.NewRecorder(), req)

		header := <-seen
		if strings.Contains(header.Get("X-Forwarded-For"), "10.9.9.9") {
			t.Fatalf("forged X-Forwarded-For reached the upstream: %q", header.Get("X-Forwarded-For"))
		}
		realIP := header.Get("X-Real-IP")
		if forward && realIP != "192.0.2.1" {
			t.Fatalf("X-Real-IP = %q, want 192.0.2.1", realIP)
		}
		if !forward && realIP != "" {
			t.Fatalf("X-Real-IP = %q, want empty", realIP)
		}
	}
}
