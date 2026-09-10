package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/pkg/internal/utils"
)

type requestCounter struct {
	count     int
	startedAt time.Time
}

func LimitRequest(limit int, next http.Handler) http.Handler {
	if limit <= 0 {
		return next
	}

	var mutex sync.Mutex
	clients := make(map[string]requestCounter)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := utils.GetIpAddress(r)
		now := time.Now()

		mutex.Lock()
		counter := clients[clientIP]
		if counter.startedAt.IsZero() || now.Sub(counter.startedAt) >= time.Minute {
			counter = requestCounter{startedAt: now}
		}
		counter.count++
		clients[clientIP] = counter
		allowed := counter.count <= limit
		defer mutex.Unlock()

		if !allowed {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "request limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
