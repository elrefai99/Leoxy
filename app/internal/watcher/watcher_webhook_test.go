package watcher

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNotifyPostsReport(t *testing.T) {
	var received Report
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	if err := Notify(server.Client(), server.URL, Report{Project: "demo", MemoryBytes: 42}); err != nil {
		t.Fatal(err)
	}
	if received.Project != "demo" || received.MemoryBytes != 42 {
		t.Fatalf("unexpected payload: %+v", received)
	}
}

func TestNotifyReportsServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	if err := Notify(server.Client(), server.URL, Report{}); err == nil {
		t.Fatal("expected error for http 500")
	}
}
