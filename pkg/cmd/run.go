package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/elrefai99/Leoxy/pkg/internal/config"
	"github.com/elrefai99/Leoxy/pkg/internal/middleware"
	"github.com/elrefai99/Leoxy/pkg/internal/server"
	"github.com/elrefai99/Leoxy/pkg/internal/utils"
)

func runServer() {
	if err := utils.CreateConfig(); err != nil {
		log.Fatal(err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()

	// Proxy routers
	mux.HandleFunc("/ping", server.Ping)
	mux.HandleFunc("/health/live", server.HealthProxy)

	for _, resource := range cfg.Upstream {
		target, err := url.Parse(resource.ServerURL)
		if err != nil || target.Scheme == "" || target.Host == "" {
			log.Printf("invalid upstream %q: %v", resource.ServerURL, err)
			continue
		}

		prefix := resource.Path
		if prefix == "" {
			prefix = "/"
		}

		proxy := server.NewProxy(target, resource.IP)
		var handler http.Handler = server.ProxyHandler(prefix, proxy)
		if resource.Body > 0 {
			handler = middleware.Body(resource.Body, handler)
		}
		if resource.Limit_request > 0 {
			handler = middleware.LimitRequest(resource.Limit_request, handler)
		}

		if prefix == "/" {
			mux.Handle("/", handler)
		} else {
			prefix = strings.TrimRight(prefix, "/")
			mux.Handle(prefix, handler)
			mux.Handle(prefix+"/", handler)
		}
	}

	server := &http.Server{
		Addr:              cfg.Server.Port,
		Handler:           utils.RequestLogger(mux),
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
		log.Println("Leoxy Success Runner")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()
	// keeps main alive forever
	<-done
}
