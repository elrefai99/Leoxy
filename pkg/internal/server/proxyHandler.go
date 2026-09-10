package server

import (
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/elrefai99/Leoxy/pkg/internal/utils"
)

func NewProxy(target *url.URL, forwardIP bool) *httputil.ReverseProxy {
	proxy := httputil.NewSingleHostReverseProxy(target)

	if forwardIP {
		originalDirector := proxy.Director
		proxy.Director = func(r *http.Request) {
			clientIP := utils.GetIpAddress(r)
			originalDirector(r)
			r.Header.Set("X-Real-IP", clientIP)
			r.Header.Set("X-Forwarded-For", clientIP)
		}
	}
	return proxy
}

func ProxyHandler(_ string, proxy *httputil.ReverseProxy) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		proxy.ServeHTTP(w, r)
	}
}
