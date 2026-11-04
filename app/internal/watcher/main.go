package watcher

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/metrics"
	"strings"
	"time"
)

type Sample struct {
	At          time.Time
	MemoryBytes uint64
	HeapBytes   uint64
	Goroutines  uint64
	CPUSeconds  float64
}

type Options struct {
	Project     string
	LimitMB     int
	Interval    time.Duration
	Cooldown    time.Duration
	HeapProfile bool
	WebhookURL  string
	Dir         string
}

var sampleNames = []string{
	"/memory/classes/total:bytes",
	"/memory/classes/heap/released:bytes",
	"/memory/classes/heap/objects:bytes",
	"/sched/goroutines:goroutines",
	"/cpu/classes/total:cpu-seconds",
}

func uintValue(sample metrics.Sample) uint64 {
	if sample.Value.Kind() != metrics.KindUint64 {
		return 0
	}
	return sample.Value.Uint64()
}

func floatValue(sample metrics.Sample) float64 {
	if sample.Value.Kind() != metrics.KindFloat64 {
		return 0
	}
	return sample.Value.Float64()
}

func ReadSample() Sample {
	samples := make([]metrics.Sample, len(sampleNames))
	for i, name := range sampleNames {
		samples[i].Name = name
	}
	metrics.Read(samples)

	total := uintValue(samples[0])
	released := uintValue(samples[1])
	var memory uint64
	if total > released {
		memory = total - released
	}

	return Sample{
		At:          time.Now(),
		MemoryBytes: memory,
		HeapBytes:   uintValue(samples[2]),
		Goroutines:  uintValue(samples[3]),
		CPUSeconds:  floatValue(samples[4]),
	}
}

func Start(options Options, onSample func(Sample), done <-chan struct{}) {
	if options.LimitMB <= 0 {
		return
	}
	if options.Interval <= 0 {
		options.Interval = 5 * time.Second
	}
	if options.Cooldown <= 0 {
		options.Cooldown = 5 * time.Minute
	}
	if options.Dir == "" {
		options.Dir = filepath.Join("leoxy", "log")
	}
	if err := os.MkdirAll(options.Dir, 0700); err != nil {
		log.Printf("memory monitor disabled: %v", err)
		return
	}

	project := strings.TrimSpace(options.Project)
	if project == "" {
		if workingDir, err := os.Getwd(); err == nil {
			project = filepath.Base(workingDir)
		}
	}
	hostname, _ := os.Hostname()
	webhook := validWebhook(options.WebhookURL)
	limit := uint64(options.LimitMB) * 1024 * 1024
	client := &http.Client{}

	writer := &Writer{
		Dir:         options.Dir,
		Project:     project,
		Hostname:    hostname,
		Started:     time.Now(),
		HeapProfile: options.HeapProfile,
	}

	watcher := &Watcher{
		limit:    limit,
		cooldown: options.Cooldown,
		onBreach: func(sample Sample) {
			report, err := writer.Write(sample, limit)
			if err != nil {
				log.Printf("memory report failed: %v", err)
				return
			}
			log.Printf("resource limit exceeded: project=%s memory_mb=%d limit_mb=%d cpu_seconds=%.2f", project, sample.MemoryBytes>>20, options.LimitMB, sample.CPUSeconds)
			if webhook == "" {
				return
			}
			go func() {
				if err := Notify(client, webhook, report); err != nil {
					log.Printf("memory webhook failed: %v", err)
				}
			}()
		},
	}

	go func() {
		ticker := time.NewTicker(options.Interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				sample := ReadSample()
				if onSample != nil {
					onSample(sample)
				}
				watcher.step(sample)
			}
		}
	}()

	log.Printf("memory monitor started: project=%s limit_mb=%d interval=%s cooldown=%s", project, options.LimitMB, options.Interval, options.Cooldown)
}

func validWebhook(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	target, err := url.Parse(value)
	if err != nil || (target.Scheme != "http" && target.Scheme != "https") || target.Host == "" {
		log.Printf("memory webhook ignored: url must use http or https and include a host")
		return ""
	}
	return target.String()
}
