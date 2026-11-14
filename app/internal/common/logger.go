package common

import (
	"encoding/json"
	"errors"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const maxLogBytes int64 = 50 << 20
const maxRequestIDLength = 64
const logQueueSize = 4096

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

type logFile struct {
	path string
	file *os.File
	size int64
}

func (l *logFile) close() error {
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

func (l *logFile) open() bool {
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log.Printf("open log file %s: %v", l.path, err)
		return false
	}
	l.size = 0
	if info, err := file.Stat(); err == nil {
		l.size = info.Size()
	}
	l.file = file
	return true
}

func (l *logFile) write(message string) {
	if l.file == nil && !l.open() {
		return
	}
	if l.size >= maxLogBytes {
		_ = l.close()
		_ = os.Rename(l.path, l.path+".1")
		if !l.open() {
			return
		}
	}
	n, err := l.file.WriteString(time.Now().Format("2006/01/02 15:04:05 ") + message + "\n")
	l.size += int64(n)
	if err != nil {
		_ = l.file.Close()
		l.file = nil
		log.Printf("write log file %s: %v", l.path, err)
	}
}

type requestLogger struct {
	mutex    sync.Mutex
	request  logFile
	failed   logFile
	entries  chan logEntryRecord
	done     chan struct{}
	closeErr error
	closed   bool
}

type logEntryRecord struct {
	message string
	isError bool
}

type requestLoggerHandler struct {
	next      http.Handler
	logger    *requestLogger
	trusted   []*net.IPNet
	skipPaths []string
}

func (h *requestLoggerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if !validRequestID(requestID) {
		requestID = newRequestID()
	}
	r.Header.Set("X-Request-ID", requestID)
	w.Header().Set("X-Request-ID", requestID)
	start := time.Now()
	writer := &statusWriter{ResponseWriter: w}

	h.next.ServeHTTP(writer, r)

	status := writer.status
	if status == 0 {
		status = http.StatusOK
	}
	if status < http.StatusBadRequest && skipLogging(r.URL.Path, h.skipPaths) {
		return
	}
	entry := logEntry(r, status, time.Since(start), h.trusted)
	h.logger.write(entry, status >= http.StatusBadRequest)
}

func skipLogging(value string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return false
	}
	cleaned := path.Clean(value)
	for _, prefix := range prefixes {
		if cleaned == prefix || strings.HasPrefix(cleaned, prefix+"/") {
			return true
		}
	}
	return false
}

func normalizeSkipPaths(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if !strings.HasPrefix(value, "/") {
			value = "/" + value
		}
		if cleaned := path.Clean(value); cleaned != "/" {
			result = append(result, cleaned)
		}
	}
	return result
}

func (h *requestLoggerHandler) Close() error {
	h.logger.mutex.Lock()
	if !h.logger.closed {
		h.logger.closed = true
		close(h.logger.entries)
	}
	h.logger.mutex.Unlock()
	<-h.logger.done
	h.logger.mutex.Lock()
	err := h.logger.closeErr
	h.logger.mutex.Unlock()
	return err
}

func CloseRequestLogger(handler http.Handler) error {
	if closer, ok := handler.(interface{ Close() error }); ok {
		return closer.Close()
	}
	return nil
}

var requestSequence uint64

func RequestLogger(next http.Handler) http.Handler {
	return RequestLoggerWithSkip(next, nil, nil)
}

func RequestLoggerTrusted(next http.Handler, trusted []*net.IPNet) http.Handler {
	return RequestLoggerWithSkip(next, trusted, nil)
}
func RequestLoggerWithSkip(next http.Handler, trusted []*net.IPNet, skipPaths []string) http.Handler {
	dir := "leoxy/log"
	if err := os.MkdirAll(dir, 0700); err != nil {
		log.Fatal(err)
	}
	logger := &requestLogger{
		request: logFile{path: filepath.Join(dir, "request.log")},
		failed:  logFile{path: filepath.Join(dir, "error.log")},
		entries: make(chan logEntryRecord, logQueueSize),
		done:    make(chan struct{}),
	}
	go logger.writeLoop()

	return &requestLoggerHandler{next: next, logger: logger, trusted: trusted, skipPaths: normalizeSkipPaths(skipPaths)}
}

func logEntry(r *http.Request, status int, duration time.Duration, trusted []*net.IPNet) string {
	entry := map[string]interface{}{
		"method":      r.Method,
		"path":        r.URL.Path,
		"remote_ip":   GetClientIP(r, trusted),
		"status":      status,
		"status_text": http.StatusText(status),
		"duration_ms": duration.Milliseconds(),
		"request_id":  r.Header.Get("X-Request-ID"),
		"error":       status >= http.StatusBadRequest,
		"message":     r.Method + " " + r.URL.Path,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return r.Method + " " + r.URL.Path
	}
	return string(data)
}

func (logger *requestLogger) write(message string, isError bool) {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()
	if logger.closed {
		return
	}
	logger.entries <- logEntryRecord{message: message, isError: isError}
}

func (logger *requestLogger) writeLoop() {
	for entry := range logger.entries {
		logger.request.write(entry.message)
		if entry.isError {
			logger.failed.write(entry.message)
		}
	}
	logger.closeErr = errors.Join(logger.request.close(), logger.failed.close())
	close(logger.done)
}

func validRequestID(value string) bool {
	if value == "" || len(value) > maxRequestIDLength {
		return false
	}
	for _, c := range value {
		letter := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
		digit := c >= '0' && c <= '9'
		if !letter && !digit && c != '-' && c != '_' && c != '.' {
			return false
		}
	}
	return true
}

func newRequestID() string {
	return strconv.FormatUint(atomic.AddUint64(&requestSequence, 1), 36)
}
