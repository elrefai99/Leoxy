package proxy

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

func TestProxyPreservesBrowserResponseHeadersAndCompressedBody(t *testing.T) {
	var compressed strings.Builder
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write([]byte("frontend asset")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.Header().Set("ETag", `"asset-v1"`)
		w.Header().Set("Last-Modified", "Mon, 02 Jan 2006 15:04:05 GMT")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Add("Set-Cookie", "session=one; Path=/; HttpOnly")
		w.Header().Set("Content-Length", fmt.Sprint(compressed.Len()))
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, compressed.String())
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/app.css", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()
	LoadBalancedProxyHandler("/", []*url.URL{target}, false).ServeHTTP(response, request)

	for name, want := range map[string]string{
		"Content-Type":     "text/css; charset=utf-8",
		"Content-Encoding": "gzip",
		"Cache-Control":    "public, max-age=3600",
		"ETag":             `"asset-v1"`,
		"Last-Modified":    "Mon, 02 Jan 2006 15:04:05 GMT",
		"Vary":             "Accept-Encoding",
	} {
		if got := response.Header().Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	if got := response.Result().Cookies(); len(got) != 1 || got[0].Name != "session" {
		t.Fatalf("cookies = %v, want session cookie", got)
	}
	reader, err := gzip.NewReader(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "frontend asset" {
		t.Fatalf("decompressed body = %q", body)
	}
}

func TestProxyStreamsResponseBeforeUpstreamCompletes(t *testing.T) {
	release := make(chan struct{})
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = io.WriteString(w, "data: ready\n\n")
		w.(http.Flusher).Flush()
		<-release
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewServer(LoadBalancedProxyHandler("/", []*url.URL{target}, false))
	defer proxy.Close()

	client := &http.Client{Timeout: 5 * time.Second}
	response, err := client.Get(proxy.URL + "/events")
	if err != nil {
		close(release)
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Errorf("close streamed response body: %v", err)
		}
	}()
	line, err := bufio.NewReader(response.Body).ReadString('\n')
	if err != nil {
		close(release)
		t.Fatalf("read streamed event: %v", err)
	}
	if line != "data: ready\n" {
		close(release)
		t.Fatalf("first streamed line = %q", line)
	}
	close(release)
}

func TestProxyPreservesRedirectLocation(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	}))
	defer upstream.Close()
	target, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	LoadBalancedProxyHandler("/", []*url.URL{target}, false).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/private", nil))
	if response.Code != http.StatusFound || response.Header().Get("Location") != "/login" {
		t.Fatalf("response = %d, Location = %q", response.Code, response.Header().Get("Location"))
	}
}
