package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/elrefai99/Leoxy/tools/internal/fs"
)

type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlags) Set(value string) error {
	if !strings.Contains(value, ":") {
		return fmt.Errorf("header %q must be in Key: Value form", value)
	}
	*h = append(*h, value)
	return nil
}

type config struct {
	target  string
	method  string
	headers []string
	payload []byte
	chunked bool
	limit   int64
}

type result struct {
	latencies []time.Duration
	statuses  map[int]uint64
	bytes     uint64
}

func parseExpect(value string) (map[int]bool, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	expected := map[int]bool{}
	for _, part := range strings.Split(value, ",") {
		code, err := strconv.Atoi(strings.TrimSpace(part))
		if err != nil {
			return nil, fmt.Errorf("invalid status %q", part)
		}
		expected[code] = true
	}
	return expected, nil
}

func accepted(status int, expected map[int]bool) bool {
	if expected != nil {
		return expected[status]
	}
	return status >= 200 && status < 400
}

func newRequest(ctx context.Context, cfg config) (*http.Request, error) {
	var body io.Reader
	if len(cfg.payload) > 0 {
		body = bytes.NewReader(cfg.payload)
	}
	req, err := http.NewRequestWithContext(ctx, cfg.method, cfg.target, body)
	if err != nil {
		return nil, err
	}
	if cfg.chunked && len(cfg.payload) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(cfg.payload))
		req.ContentLength = -1
		req.GetBody = nil
	}
	for _, header := range cfg.headers {
		key, value, _ := strings.Cut(header, ":")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if strings.EqualFold(key, "Host") {
			req.Host = value
			continue
		}
		req.Header.Add(key, value)
	}
	return req, nil
}

func worker(ctx context.Context, client *http.Client, cfg config, attempts *atomic.Int64, out *result) {
	for ctx.Err() == nil {
		if cfg.limit > 0 && attempts.Add(1) > cfg.limit {
			return
		}
		req, err := newRequest(ctx, cfg)
		if err != nil {
			out.statuses[0]++
			return
		}
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			out.statuses[0]++
			out.latencies = append(out.latencies, time.Since(start))
			continue
		}
		n, readErr := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		out.latencies = append(out.latencies, time.Since(start))
		if readErr != nil {
			out.statuses[0]++
			continue
		}
		out.statuses[resp.StatusCode]++
		out.bytes += uint64(n)
	}
}

func percentile(values []time.Duration, p int) time.Duration {
	index := (len(values)*p+99)/100 - 1
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}

func run() int {
	var headers headerFlags
	target := flag.String("url", "", "Leoxy URL")
	method := flag.String("method", http.MethodGet, "HTTP method")
	body := flag.String("body", "", "Request body")
	bodySize := flag.Int("body-size", 0, "Generated request body size in bytes")
	chunked := flag.Bool("chunked", false, "Send the body with unknown length (chunked)")
	duration := flag.Duration("duration", 0, "Test duration")
	requestLimit := flag.Int("request", 0, "Total request limit")
	concurrency := flag.Int("concurrency", 0, "Concurrent workers")
	expect := flag.String("expect", "", "Comma separated accepted statuses (default: 2xx and 3xx)")
	pprofURL := flag.String("pprof", "", "Optional Leoxy heap profile URL")
	profileOut := flag.String("profile-out", "docs/leoxy-heap.pb.gz", "Heap profile output file")
	flag.Var(&headers, "header", "Request header as 'Key: Value' (repeatable)")
	flag.Parse()

	expected, err := parseExpect(*expect)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	if *target == "" || *concurrency < 1 || *duration <= 0 {
		fmt.Fprintln(os.Stderr, "url, concurrency and duration are required")
		return 2
	}

	payload := []byte(*body)
	if *bodySize > 0 {
		payload = bytes.Repeat([]byte("a"), *bodySize)
	}

	cfg := config{
		target:  *target,
		method:  strings.ToUpper(*method),
		headers: headers,
		payload: payload,
		chunked: *chunked,
		limit:   int64(*requestLimit),
	}

	client := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        *concurrency * 2,
			MaxIdleConnsPerHost: *concurrency,
			MaxConnsPerHost:     *concurrency,
			IdleConnTimeout:     90 * time.Second,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	defer client.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	var memBefore, memAfter runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&memBefore)

	var attempts atomic.Int64
	results := make([]result, *concurrency)
	var wg sync.WaitGroup
	start := time.Now()
	for i := range results {
		results[i].statuses = map[int]uint64{}
		wg.Add(1)
		go func(out *result) {
			defer wg.Done()
			worker(ctx, client, cfg, &attempts, out)
		}(&results[i])
	}
	wg.Wait()
	elapsed := time.Since(start)
	runtime.ReadMemStats(&memAfter)

	statuses := map[int]uint64{}
	latencies := make([]time.Duration, 0, 1024)
	var totalBytes uint64
	for i := range results {
		for code, count := range results[i].statuses {
			statuses[code] += count
		}
		latencies = append(latencies, results[i].latencies...)
		totalBytes += results[i].bytes
	}
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	var total, okCount, failed uint64
	codes := make([]int, 0, len(statuses))
	for code, count := range statuses {
		codes = append(codes, code)
		total += count
		if accepted(code, expected) {
			okCount += count
		} else {
			failed += count
		}
	}
	sort.Ints(codes)

	fmt.Println("\n========== Leoxy Network ==========")
	fmt.Printf("Target: %s %s\n", cfg.method, cfg.target)
	fmt.Printf("Duration: %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Concurrency: %d\n", *concurrency)
	fmt.Printf("Total requests: %d\n", total)
	fmt.Printf("Accepted: %d\n", okCount)
	fmt.Printf("Rejected: %d\n", failed)
	fmt.Printf("Throughput: %.2f req/s\n", float64(okCount)/elapsed.Seconds())
	fmt.Printf("Response Throughput: %.2f MB/s\n", float64(totalBytes)/elapsed.Seconds()/1e6)

	fmt.Println("\nStatus codes:")
	for _, code := range codes {
		label := strconv.Itoa(code)
		if code == 0 {
			label = "error"
		}
		fmt.Printf("  %-6s %d\n", label, statuses[code])
	}

	if len(latencies) > 0 {
		fmt.Println("\nLatency:")
		fmt.Printf("  p50: %s\n", percentile(latencies, 50))
		fmt.Printf("  p95: %s\n", percentile(latencies, 95))
		fmt.Printf("  p99: %s\n", percentile(latencies, 99))
		fmt.Printf("  max: %s\n", latencies[len(latencies)-1])
	}

	fmt.Println("\n========== Client Memory ==========")
	fmt.Printf("Heap in use: %.2f MiB\n", float64(memAfter.HeapInuse)/1024/1024)
	fmt.Printf("Total allocated: %.2f MiB\n", float64(memAfter.TotalAlloc-memBefore.TotalAlloc)/1024/1024)
	fmt.Printf("Malloc allocations: %d\n", memAfter.Mallocs-memBefore.Mallocs)
	fmt.Println("(These values belong to this tool, not Leoxy)")

	code := 0
	if failed > 0 {
		code = 1
	}

	if *pprofURL != "" {
		if err := fs.SaveHeapProf(client, *pprofURL, *profileOut); err != nil {
			fmt.Fprintf(os.Stderr, "Heap profile failed: %v\n", err)
			return 1
		}
		fmt.Printf("\nLeoxy heap profile saved %s\n", *profileOut)
		fmt.Printf("Analyze with: go tool pprof -top %s\n", *profileOut)
	}

	return code
}

func main() {
	os.Exit(run())
}
