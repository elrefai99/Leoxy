package middleware

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/elrefai99/Leoxy/pkg/internal/utils"
)

func AccessControl(allow, deny []string, methods, paths, apiKeys []string, jwtSecret string, requireMTLS bool, next http.Handler) (http.Handler, error) {
	allowed, err := networks(allow)
	if err != nil {
		return nil, err
	}
	denied, err := networks(deny)
	if err != nil {
		return nil, err
	}
	methodSet := set(methods)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := net.ParseIP(utils.GetIpAddress(r))
		for _, network := range denied {
			if ip != nil && network.Contains(ip) {
				http.Error(w, "access denied", http.StatusForbidden)
				return
			}
		}
		if len(allowed) > 0 {
			matched := false
			for _, network := range allowed {
				if ip != nil && network.Contains(ip) {
					matched = true
					break
				}
			}
			if !matched {
				http.Error(w, "access denied", http.StatusForbidden)
				return
			}
		}
		if len(methodSet) > 0 && !methodSet[r.Method] {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if len(paths) > 0 && !matchesPath(r.URL.Path, paths) {
			http.NotFound(w, r)
			return
		}
		apiKeyValid := validAPIKey(r.Header.Get("Authorization"), apiKeys) || validAPIKey(r.Header.Get("X-API-Key"), apiKeys)
		jwtValid := jwtSecret != "" && validJWT(r.Header.Get("Authorization"), jwtSecret)
		if (len(apiKeys) > 0 || jwtSecret != "") && !apiKeyValid && !jwtValid {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		if requireMTLS && (r.TLS == nil || len(r.TLS.PeerCertificates) == 0 || !validCertificate(r.TLS.PeerCertificates[0])) {
			http.Error(w, "client certificate required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	}), nil
}

func networks(values []string) ([]*net.IPNet, error) {
	result := make([]*net.IPNet, 0, len(values))
	for _, value := range values {
		_, network, err := net.ParseCIDR(strings.TrimSpace(value))
		if err != nil {
			return nil, err
		}
		result = append(result, network)
	}
	return result, nil
}

func set(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		result[strings.ToUpper(strings.TrimSpace(value))] = true
	}
	return result
}
func matchesPath(value string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := path.Match(pattern, value); matched || strings.HasPrefix(value, strings.TrimSuffix(pattern, "*")) {
			return true
		}
	}
	return false
}
func validAPIKey(value string, keys []string) bool {
	value = strings.TrimSpace(strings.TrimPrefix(value, "Bearer "))
	for _, key := range keys {
		if subtle.ConstantTimeCompare([]byte(value), []byte(key)) == 1 {
			return true
		}
	}
	return false
}
func validCertificate(cert *x509.Certificate) bool {
	return cert != nil && time.Now().After(cert.NotBefore) && time.Now().Before(cert.NotAfter)
}

func validJWT(value, secret string) bool {
	parts := strings.Split(strings.TrimSpace(strings.TrimPrefix(value, "Bearer ")), ".")
	if len(parts) != 3 {
		return false
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	var claims map[string]interface{}
	if json.Unmarshal(data, &claims) != nil {
		return false
	}
	if exp, ok := claims["exp"].(float64); ok && exp < float64(time.Now().Unix()) {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(parts[0] + "." + parts[1]))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	return subtle.ConstantTimeCompare(mac.Sum(nil), signature) == 1
}
