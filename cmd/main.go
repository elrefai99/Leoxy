package main

import (
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/elrefai99/Leoxy/internal/config"
	"github.com/elrefai99/Leoxy/internal/server"
)

func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		next.ServeHTTP(w, r)

		log.Printf("%s %s %s %v",
			r.Method,
			r.URL.RequestURI(),
			r.RemoteAddr,
			time.Since(start),
		)
	})
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()

	// Proxy routers
	mux.HandleFunc("/ping", server.Ping)
	mux.HandleFunc("/health/live", server.HealthProxy)

	servers := []string{
		cfg.PROXY_SERVER_1,
		cfg.PROXY_SERVER_2,
		cfg.PROXY_SERVER_3,
		cfg.PROXY_SERVER_4,
	}

	for index, resource := range servers {
		resource = strings.TrimSpace(resource)
		if resource == "" {
			continue
		}

		parsedURL, err := url.Parse(resource)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			log.Printf("invalid proxy server %q: %v", resource, err)
			continue
		}

		proxy := server.NewProxy(parsedURL)
		prefix := "/proxy/" + strconv.Itoa(index+1) + "/"
		mux.HandleFunc(prefix, server.ProxyHandler(prefix, proxy))
	}

	server := &http.Server{
		Addr:              cfg.PORT,
		Handler:           requestLogger(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if server.Addr == "" {
		server.Addr = ":8080"
	} else if !strings.Contains(server.Addr, ":") {
		server.Addr = ":" + server.Addr
	}

	done := make(chan struct{})
	go func() {
		log.Printf("server is running on %s", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	// keeps main alive forever
	<-done
}
