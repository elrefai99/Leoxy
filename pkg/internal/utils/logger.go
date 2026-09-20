package utils

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(data)
}

func (w *statusWriter) Flush() {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

type requestLogger struct {
	mutex sync.Mutex
}

func RequestLogger(next http.Handler) http.Handler {
	if err := os.MkdirAll("log", 0755); err != nil {
		log.Fatal(err)
	}
	logger := &requestLogger{}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/node_modules") {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		writer := &statusWriter{ResponseWriter: w}

		next.ServeHTTP(writer, r)

		message := logEntry(r, writer.status, time.Since(start))
		logger.write(message, writer.status >= http.StatusBadRequest)
	})
}

func logEntry(r *http.Request, status int, duration time.Duration) string {
	return r.Method + " " +
		r.URL.Path + " " +
		r.RemoteAddr + " " +
		http.StatusText(status) + " " +
		duration.String() + " request_id=" + strconv.Quote(r.Header.Get("X-Request-ID"))
}

func (logger *requestLogger) write(message string, isError bool) {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()
	writeLog("log/request.log", message)
	if !isError {
		return
	}
	writeLog("log/error.log", message)
}

func writeLog(path string, message string) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("open log file %s: %v", path, err)
		return
	}
	defer file.Close()
	log.New(file, "", log.LstdFlags).Println(message)
}
