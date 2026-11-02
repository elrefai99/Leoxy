package security

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestRedisTokenBucketRateLimit(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
			t.Errorf("close fake Redis listener: %v", err)
		}
	})
	var requests atomic.Int64
	errors := make(chan error, 1)
	go serveFakeRedis(listener, &requests, errors)

	handler := RedisTokenBucketRateLimit(listener.Addr().String(), "", 0, 60, 1, "leoxy:test", false, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	if got, _ := statusOf(handler, "192.0.2.1:1000"); got != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", got, http.StatusNoContent)
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.RemoteAddr = "192.0.2.1:1000"
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests || response.Header().Get("Retry-After") != "60" {
		t.Fatalf("second response = %d Retry-After=%q, want 429 and 60", response.Code, response.Header().Get("Retry-After"))
	}
	if got := requests.Load(); got != 2 {
		t.Fatalf("Redis calls = %d, want 2", got)
	}
	select {
	case err := <-errors:
		t.Fatal(err)
	default:
	}
}

func TestRedisTokenBucketFailOpenUsesLocalBucket(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	handler := RedisTokenBucketRateLimit(address, "", 0, 60, 1, "leoxy:test", true, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	if got, _ := statusOf(handler, "192.0.2.1:1000"); got != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", got, http.StatusNoContent)
	}
	if got, _ := statusOf(handler, "192.0.2.1:1000"); got != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", got, http.StatusTooManyRequests)
	}
}

func serveFakeRedis(listener net.Listener, requests *atomic.Int64, errors chan<- error) {
	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			reader := bufio.NewReader(conn)
			for {
				args, err := readRedisCommand(reader)
				if err != nil {
					if err != io.EOF {
						select {
						case errors <- err:
						default:
						}
					}
					return
				}
				if len(args) != 6 || args[0] != "EVAL" || args[2] != "1" || args[3] != "leoxy:test:token-bucket" || args[4] != "60" || args[5] != "1" || !strings.Contains(args[1], "HMGET") {
					select {
					case errors <- fmt.Errorf("unexpected Redis command: %q", args):
					default:
					}
					_, _ = io.WriteString(conn, "-ERR unexpected command\r\n")
					return
				}
				result := int64(0)
				if requests.Add(1) == 1 {
					result = 1
				}
				if _, err := fmt.Fprintf(conn, ":%d\r\n", result); err != nil {
					return
				}
			}
		}()
	}
}

func readRedisCommand(reader *bufio.Reader) ([]string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	if len(line) < 3 || line[0] != '*' {
		return nil, fmt.Errorf("invalid RESP command header %q", line)
	}
	count, err := strconv.Atoi(strings.TrimSpace(line[1:]))
	if err != nil {
		return nil, err
	}
	args := make([]string, count)
	for i := range args {
		line, err = reader.ReadString('\n')
		if err != nil {
			return nil, err
		}
		if len(line) < 3 || line[0] != '$' {
			return nil, fmt.Errorf("invalid RESP bulk header %q", line)
		}
		size, err := strconv.Atoi(strings.TrimSpace(line[1:]))
		if err != nil {
			return nil, err
		}
		data := make([]byte, size+2)
		if _, err := io.ReadFull(reader, data); err != nil {
			return nil, err
		}
		args[i] = string(data[:size])
	}
	return args, nil
}
