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

	mux.HandleFunc("/ping", server.CheckServer)

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

	log.Println("server is running")

	if err := http.ListenAndServe(
		cfg.MAIN_SERVER,
		requestLogger(mux),
	); err != nil {
		log.Fatal(err)
	}
	if err := http.ListenAndServe(cfg.MAIN_SERVER, mux); err != nil {
		log.Fatal(err)
	}
}
