package watcher

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"sort"
	"time"
)

const keepReports = 20

type Report struct {
	Project       string    `json:"project"`
	Hostname      string    `json:"hostname"`
	PID           int       `json:"pid"`
	At            time.Time `json:"at"`
	UptimeSeconds int64     `json:"uptime_seconds"`
	LimitBytes    uint64    `json:"limit_bytes"`
	MemoryBytes   uint64    `json:"memory_bytes"`
	HeapBytes     uint64    `json:"heap_bytes"`
	Goroutines    uint64    `json:"goroutines"`
	CPUSeconds    float64   `json:"cpu_seconds"`
	DiskBytes     uint64    `json:"disk_bytes"`
	HeapProfile   string    `json:"heap_profile,omitempty"`
}

type Writer struct {
	Dir         string
	Project     string
	Hostname    string
	Started     time.Time
	HeapProfile bool
}

func (w *Writer) Write(sample Sample, limit uint64) (Report, error) {
	stamp := sample.At.UTC().Format("20060102T150405Z")
	report := Report{
		Project:       w.Project,
		Hostname:      w.Hostname,
		PID:           os.Getpid(),
		At:            sample.At.UTC(),
		UptimeSeconds: int64(sample.At.Sub(w.Started).Seconds()),
		LimitBytes:    limit,
		MemoryBytes:   sample.MemoryBytes,
		HeapBytes:     sample.HeapBytes,
		Goroutines:    sample.Goroutines,
		CPUSeconds:    sample.CPUSeconds,
		DiskBytes:     directorySize(w.Dir),
	}

	if w.HeapProfile {
		profilePath := filepath.Join(w.Dir, "heap-"+stamp+".pprof")
		if err := writeHeapProfile(profilePath); err != nil {
			log.Printf("heap profile failed: %v", err)
		} else {
			report.HeapProfile = profilePath
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return report, err
	}
	if err := writeAtomic(filepath.Join(w.Dir, "memory-report-"+stamp+".json"), data); err != nil {
		return report, err
	}

	prune(w.Dir, "memory-report-*.json")
	prune(w.Dir, "heap-*.pprof")
	return report, nil
}

func directorySize(dir string) uint64 {
	var size uint64
	_ = filepath.WalkDir(dir, func(_ string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err == nil && info.Size() > 0 {
			size += uint64(info.Size())
		}
		return nil
	})
	return size
}

func writeHeapProfile(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	if err := pprof.Lookup("heap").WriteTo(file, 0); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".report-*")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func prune(dir, pattern string) {
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil || len(matches) <= keepReports {
		return
	}
	sort.Strings(matches)
	for _, path := range matches[:len(matches)-keepReports] {
		_ = os.Remove(path)
	}
}
