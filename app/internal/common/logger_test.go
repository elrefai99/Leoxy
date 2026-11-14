package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequestLoggerWritesRequestLog(t *testing.T) {
	t.Chdir(t.TempDir())

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() {
		if err := CloseRequestLogger(handler); err != nil {
			t.Errorf("close request logger: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	closeRequestLoggerForTest(t, handler)

	requestLog, err := os.ReadFile("log/request.log")
	if err != nil {
		t.Fatalf("read request log: %v", err)
	}
	if !strings.Contains(string(requestLog), "GET /health") {
		t.Fatalf("request log does not contain request: %s", requestLog)
	}
	if _, err := os.Stat("log/error.log"); !os.IsNotExist(err) {
		t.Fatalf("error log should not exist for a successful request")
	}
}

func TestRequestLoggerWritesErrorsToBothLogs(t *testing.T) {
	t.Chdir(t.TempDir())

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "bad request", http.StatusBadRequest)
	}))
	t.Cleanup(func() {
		if err := CloseRequestLogger(handler); err != nil {
			t.Errorf("close request logger: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	closeRequestLoggerForTest(t, handler)

	for _, path := range []string{"log/request.log", "log/error.log"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if !strings.Contains(string(data), "POST /items") {
			t.Fatalf("%s does not contain request: %s", path, data)
		}
	}
}

func TestRequestLoggerAddsRequestIDAndJSONFields(t *testing.T) {
	t.Chdir(t.TempDir())

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	t.Cleanup(func() {
		if err := CloseRequestLogger(handler); err != nil {
			t.Errorf("close request logger: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodPut, "/items?id=secret", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	closeRequestLoggerForTest(t, handler)

	requestID := response.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Fatal("request ID was not returned")
	}
	data, err := os.ReadFile("log/request.log")
	if err != nil {
		t.Fatalf("read request log: %v", err)
	}
	var entry map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))[20:]), &entry); err != nil {
		t.Fatalf("parse request log: %v", err)
	}
	if entry["path"] != "/items" || entry["status"] != float64(http.StatusCreated) {
		t.Fatalf("unexpected log entry: %v", entry)
	}
	if entry["request_id"] != requestID {
		t.Fatalf("request ID = %v, want %s", entry["request_id"], requestID)
	}
}

func BenchmarkLogFileWrite(b *testing.B) {
	for _, test := range []struct {
		name  string
		close bool
	}{
		{name: "persistent"},
		{name: "open_close_per_write", close: true},
	} {
		b.Run(test.name, func(b *testing.B) {
			file := logFile{path: filepath.Join(b.TempDir(), "request.log")}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				file.write(`{"method":"GET","path":"/app.css","status":200}`)
				if test.close {
					_ = file.close()
				}
			}
			b.StopTimer()
			_ = file.close()
		})
	}
}

func TestRequestLoggerDoesNotSkipByDefault(t *testing.T) {
	t.Chdir(t.TempDir())

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(func() { _ = CloseRequestLogger(handler) })

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/node_modules/x", nil))
	closeRequestLoggerForTest(t, handler)

	data, err := os.ReadFile("log/request.log")
	if err != nil {
		t.Fatalf("read request log: %v", err)
	}
	if !strings.Contains(string(data), "/api/node_modules/x") {
		t.Fatalf("request was not logged: %s", data)
	}
}

func TestRequestLoggerSkipPaths(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		status int
		logged bool
	}{
		{"exact prefix", "/static", http.StatusOK, false},
		{"nested path", "/static/app.js", http.StatusOK, false},
		{"traversal is cleaned", "/static/../api", http.StatusOK, true},
		{"similar prefix is logged", "/staticfoo", http.StatusOK, true},
		{"unrelated path", "/api", http.StatusOK, true},
		{"errors are always logged", "/static/missing.js", http.StatusNotFound, true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			handler := RequestLoggerWithSkip(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
			}), nil, []string{"/static"})
			t.Cleanup(func() { _ = CloseRequestLogger(handler) })

			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			closeRequestLoggerForTest(t, handler)

			if response.Header().Get("X-Request-ID") == "" {
				t.Fatal("request ID missing on skipped or logged request")
			}
			_, err := os.Stat("log/request.log")
			if logged := err == nil; logged != test.logged {
				t.Fatalf("logged = %v, want %v", logged, test.logged)
			}
		})
	}
}

func closeRequestLoggerForTest(t *testing.T, handler http.Handler) {
	t.Helper()
	if err := CloseRequestLogger(handler); err != nil {
		t.Fatalf("close request logger: %v", err)
	}
}
