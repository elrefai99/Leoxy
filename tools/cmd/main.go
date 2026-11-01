package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elrefai99/Leoxy/tools/internal/fs"
)

var (
	mu sync.Mutex
	wg sync.WaitGroup
)

var (
	attempts atomic.Uint64
	request  atomic.Uint64
	failures atomic.Uint64
	response atomic.Uint64
)

var memBefore, memAfter runtime.MemStats

func main() {
	target := flag.String("url", "", "Leoxy URL")
	duration := flag.Duration("duration", 0, "Test duration")
	requestLimit := flag.Int("request", 0, "Requests count")
	concurrency := flag.Int("concurrency", 0, "Concurrent Worker")
	pprofURL := flag.String("pprof", "", "Optional Leoxy heap profile URL")
	profileOut := flag.String("profile-out", "docs/leoxy-heap.txt", "Heap profile output file")

	flag.Parse()

	if *concurrency < 1 || *duration <= 0 {
		fmt.Fprintln(os.Stderr, "Concurrency and duration must be positive")
		os.Exit(2)
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency,
			MaxConnsPerHost:     *concurrency,
			IdleConnTimeout:     90 * time.Second,
		},
	}
	defer client.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	latencies := make([]time.Duration, 0, 100000)

	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	start := time.Now()

	for i := 0; i < *concurrency; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for {
				if ctx.Err() != nil {
					return
				}

				if *requestLimit > 0 {
					slot := attempts.Add(1)
					if int(slot) > *requestLimit {
						return
					}
				} else {
					attempts.Add(1)
				}

				req, err := http.NewRequestWithContext(ctx, http.MethodGet, *target, nil)
				if err != nil {
					failures.Add(1)
					return
				}

				reqStart := time.Now()
				resp, err := client.Do(req)
				elapsed := time.Since(reqStart)

				if err != nil {
					if ctx.Err() != nil {
						return
					}
					failures.Add(1)
					mu.Lock()
					latencies = append(latencies, elapsed)
					mu.Unlock()
					continue
				}

				n, readErr := io.Copy(io.Discard, resp.Body)
				closeError := resp.Body.Close()

				if readErr != nil || closeError != nil || resp.StatusCode >= 500 {
					failures.Add(1)
				} else {
					request.Add(1)
					response.Add(uint64(n))
				}

				mu.Lock()
				latencies = append(latencies, elapsed)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&memAfter)
	sort.Slice(latencies, func(i, j int) bool {
		return latencies[i] < latencies[j]
	})

	failed := failures.Load()

	report := buildReport(*target, elapsed, *concurrency, latencies)
	fmt.Println(report)

	reportPath, err := writeReport(*concurrency, report)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Report write failed %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Report saved %s\n", reportPath)

	if *pprofURL != "" {
		if err := fs.SaveHeapProf(client, *pprofURL, *profileOut); err != nil {
			fmt.Fprintf(os.Stderr, "Heap profile failed %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("\nLeoxy heap profile saved %s\n", *profileOut)
	fmt.Printf("Analyze with: go tool pprof -top %s\n", *profileOut)
	if failed > 0 {
		os.Exit(1)
	}
}

func buildReport(target string, elapsed time.Duration, concurrency int, latencies []time.Duration) string {
	total := request.Load()

	var b strings.Builder
	row := func(metric, value string) {
		fmt.Fprintf(&b, "| %s | %s |\n", metric, value)
	}

	fmt.Fprintf(&b, "# Leoxy Benchmark - concurrency %d\n\n", concurrency)
	b.WriteString("## Network\n\n| Metric | Value |\n|---|---|\n")
	row("Target", target)
	row("Duration", elapsed.Round(time.Millisecond).String())
	row("Concurrency", fmt.Sprint(concurrency))
	row("Total attempts", fmt.Sprint(attempts.Load()))
	row("Success requests", fmt.Sprint(total))
	row("Failed requests", fmt.Sprint(failures.Load()))
	row("Throughput", fmt.Sprintf("%.2f req/s", float64(total)/elapsed.Seconds()))
	row("Response throughput", fmt.Sprintf("%.2f MB/s", float64(response.Load())/elapsed.Seconds()/1e6))

	if len(latencies) > 0 {
		row("Latency p50", percentile(latencies, 50).String())
		row("Latency p95", percentile(latencies, 95).String())
		row("Latency p99", percentile(latencies, 99).String())
		row("Latency max", latencies[len(latencies)-1].String())
	}

	b.WriteString("\n## Client memory\n\n| Metric | Value |\n|---|---|\n")
	row("Heap in use", fmt.Sprintf("%.2f MiB", float64(memAfter.HeapInuse)/1024/1024))
	row("Total allocated", fmt.Sprintf("%.2f MiB", float64(memAfter.TotalAlloc-memBefore.TotalAlloc)/1024/1024))
	row("Malloc allocations", fmt.Sprint(memAfter.Mallocs-memBefore.Mallocs))
	b.WriteString("\nThese memory values belong to the benchmark client, not Leoxy.\n")

	return b.String()
}

func writeReport(concurrency int, report string) (string, error) {
	dir := filepath.Join("docs", "baseline")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("report.%d.md", concurrency))
	return path, os.WriteFile(path, []byte(report), 0644)
}

func percentile(val []time.Duration, p int) time.Duration {
	index := (len(val)*p+99)/100 - 1

	if index < 0 {
		index = 0
	}

	if index >= len(val) {
		index = len(val) - 1
	}
	return val[index]
}
