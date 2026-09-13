package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/elrefai99/Leoxy/pkg/internal/config"
	"github.com/elrefai99/Leoxy/pkg/internal/file"
	"github.com/elrefai99/Leoxy/pkg/internal/middleware"
	"github.com/elrefai99/Leoxy/pkg/internal/server"
	"github.com/elrefai99/Leoxy/pkg/internal/utils"
)

var once sync.Once

func runServer() {
	once.Do(func() {
		if err := file.CreateConfig(); err != nil {
			log.Fatal(err)
		}
	})

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	mux := http.NewServeMux()
	// Proxy routers
	mux.HandleFunc("/ping", server.Ping)
	mux.HandleFunc("/health/live", server.HealthProxy)
	mux.HandleFunc("/leoxy", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/index.html")
	})

	for _, resource := range cfg.Upstream {
		target, err := url.Parse(resource.ServerURL)
		if err != nil || target.Scheme == "" || target.Host == "" {
			log.Printf("invalid upstream %q: %v", resource.ServerURL, err)
			continue
		}

		prefix := strings.TrimSpace(resource.Path)
		if prefix == "" {
			prefix = "/"
		}
		if !strings.HasPrefix(prefix, "/") {
			prefix = "/" + prefix
		}
		if prefix != "/" {
			prefix = strings.TrimRight(prefix, "/")
			if prefix == "" {
				prefix = "/"
			}
		}

		proxy := server.NewProxy(target, resource.IP)
		var handler http.Handler = server.ProxyHandler(prefix, proxy)
		security := resource.Security
		if security.MaxBody <= 0 {
			security.MaxBody = resource.Body
		}
		secured, err := middleware.AccessControl(
			security.AllowCIDRs,
			security.DenyCIDRs,
			security.AllowedMethods,
			security.AllowedPaths,
			security.APIKeys,
			security.JWTSecret,
			security.RequireMTLS,
			handler,
		)
		if err != nil {
			log.Fatalf("invalid security configuration for %q: %v", resource.Name, err)
		}
		handler = secured
		handler = middleware.RateLimit(
			security.RateLimit,
			security.RateLimitBurst,
			security.RouteRateLimit,
			security.RouteRateBurst,
			0,
			0,
			prefix,
			handler,
		)
		if security.MaxBody > 0 {
			handler = middleware.Body(security.MaxBody, handler)
		}

		if prefix == "/" {
			mux.Handle("/", handler)
		} else {
			mux.Handle(prefix, handler)
			mux.Handle(prefix+"/", handler)
		}
	}

	globalHandler, err := middleware.AccessControl(
		cfg.Server.Security.AllowCIDRs,
		cfg.Server.Security.DenyCIDRs,
		cfg.Server.Security.AllowedMethods,
		cfg.Server.Security.AllowedPaths,
		cfg.Server.Security.APIKeys,
		cfg.Server.Security.JWTSecret,
		cfg.Server.Security.RequireMTLS,
		mux,
	)
	if err != nil {
		log.Fatal(err)
	}
	globalHandler = middleware.RateLimit(
		cfg.Server.Security.RateLimit,
		cfg.Server.Security.RateLimitBurst,
		0,
		0,
		cfg.Server.Security.GlobalRateLimit,
		cfg.Server.Security.GlobalRateBurst,
		"global",
		globalHandler,
	)
	globalHandler = middleware.RedisRateLimit(
		cfg.Server.Security.RedisAddr,
		cfg.Server.Security.RedisPassword,
		cfg.Server.Security.RedisDB,
		cfg.Server.Security.GlobalRateLimit,
		"leoxy:global",
		globalHandler,
	)
	if cfg.Server.Security.MaxBody > 0 {
		globalHandler = middleware.Body(cfg.Server.Security.MaxBody, globalHandler)
	}

	server := &http.Server{
		Addr:              cfg.Server.Port,
		Handler:           utils.RequestLogger(globalHandler),
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
