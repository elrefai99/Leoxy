package security

import (
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-that-is-at-least-32-bytes"

func pasthrough() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func signToken(t *testing.T, method jwt.SigningMethod, secret string, claims jwt.MapClaims) string {
	t.Helper()
	token, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func codeFor(handler http.Handler, remote string, configure func(*http.Request)) int {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remote
	if configure != nil {
		configure(req)
	}
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, req)
	return response.Code
}

func TestJWTValidation(t *testing.T) {
	handler, err := AccessControlWithJWT(nil, nil, nil, nil, nil, testSecret, "leoxy", "api", false, pasthrough())
	if err != nil {
		t.Fatal(err)
	}

	future := time.Now().Add(time.Hour).Unix()
	past := time.Now().Add(-time.Hour).Unix()
	valid := jwt.MapClaims{"exp": future, "iss": "leoxy", "aud": "api"}

	none, err := jwt.NewWithClaims(jwt.SigningMethodNone, valid).SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatal(err)
	}

	validToken := signToken(t, jwt.SigningMethodHS256, testSecret, valid)

	tests := []struct {
		name   string
		header string
		status int
	}{
		{"valid token", "Bearer " + validToken, http.StatusOK},
		{"valid token without bearer prefix", validToken, http.StatusOK},
		{"expired", "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{"exp": past, "iss": "leoxy", "aud": "api"}), http.StatusUnauthorized},
		{"missing exp", "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{"iss": "leoxy", "aud": "api"}), http.StatusUnauthorized},
		{"wrong secret", "Bearer " + signToken(t, jwt.SigningMethodHS256, "other-secret", valid), http.StatusUnauthorized},
		{"wrong issuer", "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{"exp": future, "iss": "evil", "aud": "api"}), http.StatusUnauthorized},
		{"wrong audience", "Bearer " + signToken(t, jwt.SigningMethodHS256, testSecret, jwt.MapClaims{"exp": future, "iss": "leoxy", "aud": "other"}), http.StatusUnauthorized},
		{"HS512 is rejected", "Bearer " + signToken(t, jwt.SigningMethodHS512, testSecret, valid), http.StatusUnauthorized},
		{"alg none is rejected", "Bearer " + none, http.StatusUnauthorized},
		{"garbage", "Bearer not-a-token", http.StatusUnauthorized},
		{"no header", "", http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := codeFor(handler, "192.0.2.1:1234", func(r *http.Request) {
				if test.header != "" {
					r.Header.Set("Authorization", test.header)
				}
			})
			if status != test.status {
				t.Fatalf("status = %d, want %d", status, test.status)
			}
		})
	}
}

func TestAPIKeyAcceptedAsHeaderOrBearer(t *testing.T) {
	handler, err := AccessControl(nil, nil, nil, nil, []string{"secret"}, "", false, pasthrough())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		configure func(*http.Request)
		status    int
	}{
		{"X-API-Key", func(r *http.Request) { r.Header.Set("X-API-Key", "secret") }, http.StatusOK},
		{"Authorization bearer", func(r *http.Request) { r.Header.Set("Authorization", "Bearer secret") }, http.StatusOK},
		{"wrong key", func(r *http.Request) { r.Header.Set("X-API-Key", "nope") }, http.StatusUnauthorized},
		{"missing key", nil, http.StatusUnauthorized},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if status := codeFor(handler, "192.0.2.1:1234", test.configure); status != test.status {
				t.Fatalf("status = %d, want %d", status, test.status)
			}
		})
	}
}

func TestDenyCIDRBlocksClient(t *testing.T) {
	handler, err := AccessControl([]string(nil), []string{"192.0.2.0/24"}, nil, nil, nil, "", false, pasthrough())
	if err != nil {
		t.Fatal(err)
	}

	if status := codeFor(handler, "192.0.2.5:1234", nil); status != http.StatusForbidden {
		t.Fatalf("denied network status = %d, want %d", status, http.StatusForbidden)
	}
	if status := codeFor(handler, "198.51.100.5:1234", nil); status != http.StatusOK {
		t.Fatalf("other network status = %d, want %d", status, http.StatusOK)
	}
}

func TestMethodCheckRunsBeforeAuthentication(t *testing.T) {
	handler, err := AccessControl(nil, nil, []string{"GET"}, nil, []string{"k"}, "", false, pasthrough())
	if err != nil {
		t.Fatal(err)
	}

	post := httptest.NewRecorder()
	handler.ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST status = %d, want %d", post.Code, http.StatusMethodNotAllowed)
	}

	if status := codeFor(handler, "192.0.2.1:1234", nil); status != http.StatusUnauthorized {
		t.Fatalf("GET without key status = %d, want %d", status, http.StatusUnauthorized)
	}
}

func TestAllowedPathsGlobDoesNotCrossSlash(t *testing.T) {
	handler, err := AccessControl(nil, nil, nil, []string{"/api/*"}, nil, "", false, pasthrough())
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path   string
		status int
	}{
		{"/api/items", http.StatusOK},
		{"/api/a/b", http.StatusNotFound},
		{"/api", http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
		})
	}
}

func TestMTLSRequiresVerifiedClientCertificate(t *testing.T) {
	handler, err := AccessControlWithJWT(nil, nil, nil, nil, nil, "", "", "", true, pasthrough())
	if err != nil {
		t.Fatal(err)
	}

	cert := &x509.Certificate{}
	tests := []struct {
		name   string
		state  *tls.ConnectionState
		status int
	}{
		{"plain http", nil, http.StatusUnauthorized},
		{"tls without client certificate", &tls.ConnectionState{}, http.StatusUnauthorized},
		{"certificate not verified", &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}}, http.StatusUnauthorized},
		{"verified certificate", &tls.ConnectionState{PeerCertificates: []*x509.Certificate{cert}, VerifiedChains: [][]*x509.Certificate{{cert}}}, http.StatusOK},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			status := codeFor(handler, "192.0.2.1:1234", func(r *http.Request) {
				r.TLS = test.state
			})
			if status != test.status {
				t.Fatalf("status = %d, want %d", status, test.status)
			}
		})
	}
}
