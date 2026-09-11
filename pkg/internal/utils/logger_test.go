package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRequestLoggerWritesRequestLog(t *testing.T) {
	t.Chdir(t.TempDir())

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

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

	req := httptest.NewRequest(http.MethodPost, "/items", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

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

func TestRequestLoggerSkipsNodeModules(t *testing.T) {
	t.Chdir(t.TempDir())

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/node_modules/app.js", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)

	if _, err := os.Stat("log/request.log"); !os.IsNotExist(err) {
		t.Fatalf("node_modules request should not be logged")
	}
}
