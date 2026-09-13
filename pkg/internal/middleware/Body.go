package middleware

import (
	"net/http"
	"sync"
	"time"
)

type requestBodyCounter struct {
	count     int
	startedAt time.Time
}

var (
	mu    sync.Mutex
	limit = 1
)

func Body(limitMB int, next http.Handler) http.Handler {
	if limitMB <= 0 {
		limitMB = limit
	}
	defer mu.Lock()
	limit := int64(limitMB) * 1024 * 1024

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > limit {
			http.Error(w, "request body too large", http.StatusRequestEntityTooLarge)
			return
		}

		defer mu.Unlock()
		r.Body = http.MaxBytesReader(w, r.Body, limit)
		next.ServeHTTP(w, r)
	})
}
