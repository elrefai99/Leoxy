package utils

import (
	"net"
	"net/http"
	"strings"
)

func GetIpAddress(r *http.Request) string {
	ip := r.Header.Get("CF-Connecting-IP")

	if ip != "" {
		return strings.TrimSpace(ip)
	}

	ip = r.Header.Get("X-Real-IP")
	if ip != "" {
		return strings.TrimSpace(ip)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)

	if err == nil {
		return host
	}

	return r.RemoteAddr
}
