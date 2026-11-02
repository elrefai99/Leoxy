package core

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/elrefai99/Leoxy/app/internal/file"
)

func TestHTTPRouteRateLimitIsolation(t *testing.T) {
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer frontend.Close()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer backend.Close()

	serverSecurity := file.Security{RateLimit: 60, RateLimitBurst: 1, GlobalRateLimit: 6000, GlobalRateBurst: 10}
	mux := http.NewServeMux()
	registerHTTP(mux, serverSecurity, file.Upstream{
		Name: "Backend", Path: "/api", ServerURL: backend.URL,
		Security: file.Security{RouteRateLimit: 6000, RouteRateBurst: 10},
	}, nil)
	registerHTTP(mux, serverSecurity, file.Upstream{
		Name: "Frontend", Path: "/", ServerURL: frontend.URL,
		Security: file.Security{RateLimit: 60, RateLimitBurst: 2},
	}, nil)
	root := newHTTPRoot(mux, mux, serverSecurity)

	for _, test := range []struct {
		path string
		want int
	}{
		{path: "/api", want: http.StatusOK},
		{path: "/", want: http.StatusOK},
		{path: "/", want: http.StatusOK},
		{path: "/", want: http.StatusTooManyRequests},
		{path: "/api", want: http.StatusTooManyRequests},
	} {
		if got := requestStatus(root, test.path, "192.0.2.10:1000"); got != test.want {
			t.Fatalf("GET %s status = %d, want %d", test.path, got, test.want)
		}
	}
}

func TestHTTPRoutePoliciesDoNotConsumeOtherRouteBuckets(t *testing.T) {
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer frontend.Close()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer backend.Close()

	serverSecurity := file.Security{GlobalRateLimit: 6000, GlobalRateBurst: 20}
	mux := http.NewServeMux()
	registerHTTP(mux, serverSecurity, file.Upstream{
		Name: "Backend", Path: "/api", ServerURL: backend.URL,
		Security: file.Security{RateLimit: 60, RateLimitBurst: 1, RouteRateLimit: 60, RouteRateBurst: 2},
	}, nil)
	registerHTTP(mux, serverSecurity, file.Upstream{Name: "Frontend", Path: "/", ServerURL: frontend.URL}, nil)
	root := newHTTPRoot(mux, mux, serverSecurity)

	for _, test := range []struct {
		path   string
		remote string
		want   int
	}{
		{path: "/api", remote: "192.0.2.1:1000", want: http.StatusOK},
		{path: "/api", remote: "192.0.2.1:1000", want: http.StatusTooManyRequests},
		{path: "/api", remote: "192.0.2.2:1000", want: http.StatusOK},
		{path: "/api", remote: "192.0.2.3:1000", want: http.StatusTooManyRequests},
		{path: "/", remote: "192.0.2.1:1000", want: http.StatusOK},
		{path: "/", remote: "192.0.2.1:1000", want: http.StatusOK},
	} {
		if got := requestStatus(root, test.path, test.remote); got != test.want {
			t.Fatalf("GET %s from %s status = %d, want %d", test.path, test.remote, got, test.want)
		}
	}
}

func TestHTTPGlobalRateLimitAppliesOnceAcrossRoutes(t *testing.T) {
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer frontend.Close()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer backend.Close()

	serverSecurity := file.Security{GlobalRateLimit: 60, GlobalRateBurst: 2}
	mux := http.NewServeMux()
	registerHTTP(mux, serverSecurity, file.Upstream{Name: "Backend", Path: "/api", ServerURL: backend.URL}, nil)
	registerHTTP(mux, serverSecurity, file.Upstream{Name: "Frontend", Path: "/", ServerURL: frontend.URL}, nil)
	root := newHTTPRoot(mux, mux, serverSecurity)

	if got := requestStatus(root, "/", "192.0.2.1:1000"); got != http.StatusOK {
		t.Fatalf("frontend status = %d, want %d", got, http.StatusOK)
	}
	if got := requestStatus(root, "/api", "192.0.2.2:1000"); got != http.StatusOK {
		t.Fatalf("API status = %d, want %d", got, http.StatusOK)
	}
	if got := requestStatus(root, "/ping", "192.0.2.3:1000"); got != http.StatusTooManyRequests {
		t.Fatalf("third global request status = %d, want %d", got, http.StatusTooManyRequests)
	}
}

func TestConcurrentHTTPFrontendAndAPIRequests(t *testing.T) {
	frontend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer frontend.Close()
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	defer backend.Close()

	serverSecurity := file.Security{GlobalRateLimit: 60000, GlobalRateBurst: 200}
	mux := http.NewServeMux()
	registerHTTP(mux, serverSecurity, file.Upstream{
		Name: "Backend", Path: "/api", ServerURL: backend.URL,
		Security: file.Security{RateLimit: 60000, RateLimitBurst: 100, RouteRateLimit: 60000, RouteRateBurst: 100},
	}, nil)
	registerHTTP(mux, serverSecurity, file.Upstream{
		Name: "Frontend", Path: "/", ServerURL: frontend.URL,
		Security: file.Security{RateLimit: 60000, RateLimitBurst: 100},
	}, nil)
	root := newHTTPRoot(mux, mux, serverSecurity)

	var wait sync.WaitGroup
	errors := make(chan string, 100)
	for i := 0; i < 100; i++ {
		wait.Add(1)
		go func(i int) {
			defer wait.Done()
			path := "/"
			if i%2 == 0 {
				path = "/api"
			}
			remote := fmt.Sprintf("192.0.2.%d:1000", i%4+1)
			if got := requestStatus(root, path, remote); got != http.StatusOK {
				errors <- fmt.Sprintf("GET %s from %s status = %d", path, remote, got)
			}
		}(i)
	}
	wait.Wait()
	close(errors)
	for message := range errors {
		t.Error(message)
	}
}

func requestStatus(handler http.Handler, path, remote string) int {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	request.RemoteAddr = remote
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code
}
