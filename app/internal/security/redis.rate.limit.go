package security

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const redisTimeout = time.Second

type redisConn struct {
	net.Conn
	reader *bufio.Reader
}

type RedisLimiter struct {
	addr     string
	password string
	database int
	pool     chan *redisConn
}

const redisTokenBucketScript = `local time = redis.call('TIME')
local now = tonumber(time[1]) + tonumber(time[2]) / 1000000
local rate = tonumber(ARGV[1]) / 60
local burst = tonumber(ARGV[2])
local values = redis.call('HMGET', KEYS[1], 'tokens', 'updated')
local tokens = tonumber(values[1]) or burst
local updated = tonumber(values[2]) or now
tokens = math.min(burst, tokens + math.max(0, now - updated) * rate)
local allowed = 0
if tokens >= 1 then
  tokens = tokens - 1
  allowed = 1
end
redis.call('HSET', KEYS[1], 'tokens', tokens, 'updated', now)
redis.call('EXPIRE', KEYS[1], math.max(1, math.ceil(burst * 120 / tonumber(ARGV[1]))))
return allowed`

func NewRedisLimiter(connect uint, addr, password string, database int) *RedisLimiter {
	return &RedisLimiter{addr: addr, password: password, database: database, pool: make(chan *redisConn, connect)}
}

func RedisRateLimit(connect uint, addr, password string, database, limit int, key string, next http.Handler) http.Handler {
	return RedisRateLimitWithFailOpen(connect, addr, password, database, limit, key, false, next)
}

func RedisRateLimitWithFailOpen(connect uint, addr, password string, database, limit int, key string, failOpen bool, next http.Handler) http.Handler {
	if strings.TrimSpace(addr) == "" || limit <= 0 {
		return next
	}
	limiter := NewRedisLimiter(connect, addr, password, database)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, err := limiter.Allow(key+":"+time.Now().UTC().Format("200601021504"), limit)
		if err != nil {
			if failOpen {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "rate limit store unavailable", http.StatusServiceUnavailable)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RedisTokenBucketRateLimit(connect uint, addr, password string, database, limit, burst int, key string, failOpen bool, next http.Handler) http.Handler {
	localFallback := RateLimit(0, 0, 0, 0, limit, burst, key, next)
	if strings.TrimSpace(addr) == "" || limit <= 0 {
		return localFallback
	}
	if burst <= 0 {
		burst = limit
	}
	limiter := NewRedisLimiter(connect, addr, password, database)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		allowed, err := limiter.AllowTokenBucket(key, limit, burst)
		if err != nil {
			if failOpen {
				localFallback.ServeHTTP(w, r)
				return
			}
			http.Error(w, "rate limit store unavailable", http.StatusServiceUnavailable)
			return
		}
		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (limiter *RedisLimiter) AllowTokenBucket(key string, rate, burst int) (bool, error) {
	conn, err := limiter.acquire()
	if err != nil {
		return false, err
	}
	if err := conn.SetDeadline(time.Now().Add(redisTimeout)); err != nil {
		_ = conn.Close()
		return false, err
	}
	err = conn.send([]string{
		"EVAL",
		redisTokenBucketScript,
		"1",
		key + ":token-bucket",
		strconv.Itoa(rate),
		strconv.Itoa(burst),
	})
	if err != nil {
		_ = conn.Close()
		return false, err
	}
	result, err := conn.reply()
	if err != nil {
		_ = conn.Close()
		return false, err
	}
	select {
	case limiter.pool <- conn:
	default:
		_ = conn.Close()
	}
	return result == 1, nil
}

func (limiter *RedisLimiter) Allow(key string, limit int) (bool, error) {
	conn, err := limiter.acquire()
	if err != nil {
		return false, err
	}
	count, err := conn.increment(key)
	if err != nil {
		_ = conn.Close()
		return false, err
	}
	select {
	case limiter.pool <- conn:
	default:
		_ = conn.Close()
	}
	return count <= int64(limit), nil
}

func (limiter *RedisLimiter) acquire() (*redisConn, error) {
	select {
	case conn := <-limiter.pool:
		return conn, nil
	default:
	}
	raw, err := net.DialTimeout("tcp", limiter.addr, redisTimeout)
	if err != nil {
		return nil, err
	}
	conn := &redisConn{Conn: raw, reader: bufio.NewReader(raw)}
	if err := conn.setup(limiter.password, limiter.database); err != nil {
		_ = raw.Close()
		return nil, err
	}
	return conn, nil
}

func (conn *redisConn) setup(password string, database int) error {
	if password != "" {
		if _, err := conn.do("AUTH", password); err != nil {
			return err
		}
	}
	if database > 0 {
		if _, err := conn.do("SELECT", strconv.Itoa(database)); err != nil {
			return err
		}
	}
	return nil
}

func (conn *redisConn) increment(key string) (int64, error) {
	if err := conn.SetDeadline(time.Now().Add(redisTimeout)); err != nil {
		return 0, err
	}
	if err := conn.send([]string{"INCR", key}, []string{"EXPIRE", key, "120"}); err != nil {
		return 0, err
	}
	count, err := conn.reply()
	if err != nil {
		return 0, err
	}
	if _, err := conn.reply(); err != nil {
		return 0, err
	}
	return count, nil
}

func (conn *redisConn) do(parts ...string) (int64, error) {
	if err := conn.SetDeadline(time.Now().Add(redisTimeout)); err != nil {
		return 0, err
	}
	if err := conn.send(parts); err != nil {
		return 0, err
	}
	return conn.reply()
}

func (conn *redisConn) send(commands ...[]string) error {
	var buffer bytes.Buffer
	for _, parts := range commands {
		fmt.Fprintf(&buffer, "*%d\r\n", len(parts))
		for _, part := range parts {
			fmt.Fprintf(&buffer, "$%d\r\n%s\r\n", len(part), part)
		}
	}
	_, err := conn.Write(buffer.Bytes())
	return err
}

func (conn *redisConn) reply() (int64, error) {
	line, err := conn.reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	line = strings.TrimRight(line, "\r\n")
	if line == "" {
		return 0, errors.New("redis: empty reply")
	}
	switch line[0] {
	case '-':
		return 0, fmt.Errorf("redis: %s", line[1:])
	case ':':
		return strconv.ParseInt(line[1:], 10, 64)
	case '+':
		return 0, nil
	default:
		return 0, fmt.Errorf("redis: unexpected reply %q", line)
	}
}
