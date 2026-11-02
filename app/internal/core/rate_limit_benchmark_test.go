package core

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/elrefai99/Leoxy/app/internal/file"
	securitymiddleware "github.com/elrefai99/Leoxy/app/internal/security"
)

type benchmarkResponseWriter struct {
	header http.Header
}

func (w *benchmarkResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = make(http.Header)
	}
	return w.header
}

func (*benchmarkResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (*benchmarkResponseWriter) WriteHeader(int)           {}

func BenchmarkRateLimitIsolation(b *testing.B) {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.1:1000"
	response := &benchmarkResponseWriter{}
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	const rate = 1_000_000_000

	b.Run("before-shared-server-ip-and-global", func(b *testing.B) {
		handler := securitymiddleware.RateLimit(rate, rate, 0, 0, rate, rate, "global", next)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.ServeHTTP(response, request)
		}
	})

	b.Run("after-global-and-route-ip", func(b *testing.B) {
		route := securitymiddleware.RateLimit(rate, rate, 0, 0, 0, 0, "/", next)
		handler := newHTTPRoot(route, route, securityForBenchmark(rate))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			handler.ServeHTTP(response, request)
		}
	})
}

func securityForBenchmark(rate int) file.Security {
	return file.Security{GlobalRateLimit: rate, GlobalRateBurst: rate}
}
