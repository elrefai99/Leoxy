package utils

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

func RequestLogger(next http.Handler) http.Handler {
	err := os.MkdirAll("log", 0755)
	if err != nil {
		log.Fatal(err)
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		go func() {
			path := r.URL.Path
			if strings.HasPrefix(path, "/_nuxt") || strings.Contains(path, "/node_modules") {
				return
			}
			pathLog := "log/request.log"
			file, err := os.OpenFile(pathLog, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)

			if err != nil {
				log.Fatal(err)
			}

			defer file.Close()

			fileData := log.New(file, "", log.LstdFlags)

			fileData.Printf("%s %s %s %v",
				r.Method,
				r.URL.RequestURI(),
				r.RemoteAddr,
				time.Since(start),
			)
		}()

		next.ServeHTTP(w, r)
	})
}
