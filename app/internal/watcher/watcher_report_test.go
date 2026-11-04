package watcher

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriterWritesReportAndProfile(t *testing.T) {
	dir := t.TempDir()
	writer := &Writer{
		Dir:         dir,
		Project:     "demo",
		Hostname:    "host",
		Started:     time.Now().Add(-time.Minute),
		HeapProfile: true,
	}
	sample := Sample{At: time.Now(), MemoryBytes: 150 << 20, HeapBytes: 100 << 20, Goroutines: 7, CPUSeconds: 2.5}

	report, err := writer.Write(sample, 100<<20)
	if err != nil {
		t.Fatal(err)
	}

	reportPath := filepath.Join(dir, "memory-report-"+sample.At.UTC().Format("20060102T150405Z")+".json")
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Project != "demo" || decoded.MemoryBytes != sample.MemoryBytes || decoded.LimitBytes != 100<<20 || decoded.CPUSeconds != sample.CPUSeconds {
		t.Fatalf("unexpected report: %+v", decoded)
	}
	if report.HeapProfile == "" {
		t.Fatal("heap profile path missing")
	}
	if _, err := os.Stat(report.HeapProfile); err != nil {
		t.Fatalf("heap profile not written: %v", err)
	}
}

func TestPruneKeepsNewest(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < keepReports+5; i++ {
		name := fmt.Sprintf("memory-report-%04d.json", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	prune(dir, "memory-report-*.json")

	matches, err := filepath.Glob(filepath.Join(dir, "memory-report-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != keepReports {
		t.Fatalf("files = %d, want %d", len(matches), keepReports)
	}
	if _, err := os.Stat(filepath.Join(dir, "memory-report-0000.json")); !os.IsNotExist(err) {
		t.Fatal("oldest report should be pruned")
	}
}
