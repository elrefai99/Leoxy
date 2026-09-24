package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
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
	trustedProxyCIDRs, err := utils.ParseCIDRs(cfg.Server.Security.TrustedProxyCIDRs)
	if err != nil {
		log.Fatalf("invalid trusted proxy CIDR: %v", err)
	}
	mux := http.NewServeMux()
	// Proxy routers
	mux.HandleFunc("/ping", server.Ping)
	mux.HandleFunc("/health/live", server.HealthProxy)
	metrics := utils.NewMetrics()
	mux.Handle("/metrics", http.HandlerFunc(metrics.Handler))
	mux.HandleFunc("/leoxy", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "docs/index.html")
	})

	for _, resource := range cfg.Upstream {
		serverURLs := resource.Servers
		if len(serverURLs) == 0 && resource.ServerURL != "" {
			serverURLs = []string{resource.ServerURL}
		}
		targets := make([]*url.URL, 0, len(serverURLs))
		for _, serverURL := range serverURLs {
			target, err := url.Parse(serverURL)
			if err != nil || target.Scheme == "" || target.Host == "" {
				log.Printf("invalid upstream %q: %v", serverURL, err)
				continue
			}
			targets = append(targets, target)
		}
		if len(targets) == 0 {
			log.Printf("upstream %q has no valid servers", resource.Name)
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

		var handler http.Handler
		if len(targets) == 1 {
			handler = server.ProxyHandler(prefix, server.NewProxy(targets[0], resource.IP, trustedProxyCIDRs))
		} else {
			handler = server.LoadBalancedProxyHandler(prefix, targets, resource.IP, trustedProxyCIDRs)
		}
		security := resource.Security
		if security.MaxBody <= 0 {
			security.MaxBody = resource.Body
		}
		secured, err := middleware.AccessControlWithJWT(
			security.AllowCIDRs,
			security.DenyCIDRs,
			security.AllowedMethods,
			security.AllowedPaths,
			security.APIKeys,
			security.JWTSecret,
			security.JWTIssuer,
			security.JWTAudience,
			security.RequireMTLS,
			handler,
			security.TrustedProxyCIDRs...,
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
			trustedProxyCIDRs,
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

	globalHandler, err := middleware.AccessControlWithJWT(
		cfg.Server.Security.AllowCIDRs,
		cfg.Server.Security.DenyCIDRs,
		cfg.Server.Security.AllowedMethods,
		cfg.Server.Security.AllowedPaths,
		cfg.Server.Security.APIKeys,
		cfg.Server.Security.JWTSecret,
		cfg.Server.Security.JWTIssuer,
		cfg.Server.Security.JWTAudience,
		cfg.Server.Security.RequireMTLS,
		mux,
		cfg.Server.Security.TrustedProxyCIDRs...,
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
		trustedProxyCIDRs,
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

	httpServer := &http.Server{
		Addr:              cfg.Server.Port,
		Handler:           utils.RequestLogger(metrics.Middleware(globalHandler)),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	if httpServer.Addr == "" {
		httpServer.Addr = ":8080"
	} else if !strings.Contains(httpServer.Addr, ":") {
		httpServer.Addr = ":" + httpServer.Addr
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Println("Leoxy Success Runner")
		serverErrors <- httpServer.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(shutdown)

	select {
	case err := <-serverErrors:
		if err != nil && err != http.ErrServerClosed {
			log.Printf("server stopped: %v", err)
		}
	case <-shutdown:
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Printf("graceful shutdown failed: %v", err)
		}
	}
}
