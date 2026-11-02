package file

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/viper"
)

func config(upstreams ...Upstream) *ServerConfig {
	cfg := &ServerConfig{Upstream: upstreams}
	return cfg
}

func TestValidateAcceptsSimpleConfig(t *testing.T) {
	cfg := config(Upstream{Name: "app", Path: "/api", ServerURL: "http://localhost:3000"})
	if err := Validate(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsReservedAndDuplicatePaths(t *testing.T) {
	cfg := config(
		Upstream{Name: "a", Path: "/metrics", ServerURL: "http://localhost:1"},
		Upstream{Name: "b", Path: "/api", ServerURL: "http://localhost:2"},
		Upstream{Name: "c", Path: "/api/", ServerURL: "http://localhost:3"},
	)
	err := Validate(cfg)
	if err == nil || !strings.Contains(err.Error(), "reserved") || !strings.Contains(err.Error(), "already used") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateRequiresTLSForMTLS(t *testing.T) {
	cfg := config()
	cfg.Server.Security.RequireMTLS = true
	if err := Validate(cfg); err == nil {
		t.Fatal("expected mtls error")
	}
}

func TestValidateRejectsShortJWTSecretAndBadCIDR(t *testing.T) {
	cfg := config()
	cfg.Server.Security.JWTSecret = "short"
	cfg.Server.Security.AllowCIDRs = []string{"not-a-cidr"}
	if err := Validate(cfg); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateRejectsNegativeUpstreamRateLimit(t *testing.T) {
	cfg := config(Upstream{
		Name:      "frontend",
		Path:      "/",
		ServerURL: "http://localhost:3000",
		Security:  Security{RateLimit: -1, RateLimitBurst: -1},
	})
	if err := Validate(cfg); err == nil || !strings.Contains(err.Error(), "rate_limit") {
		t.Fatalf("validation error = %v, want negative rate limit error", err)
	}
}

func TestLoadRateLimitPolicies(t *testing.T) {
	tests := []struct {
		name      string
		config    string
		rateLimit string
		server    int
		frontend  Security
		backend   Security
	}{
		{
			name: "legacy configuration",
			config: `server:
  security:
    rate_limit: 600
    rate_limit_burst: 100
upstream:
  - name: backend
    path: /api
    server_url: http://localhost:9000
    security:
      route_rate_limit: 200
      route_rate_burst: 20
  - name: frontend
    path: /
    server_url: http://localhost:3000
`,
			server:   600,
			frontend: Security{},
			backend:  Security{RouteRateLimit: 200, RouteRateBurst: 20},
		},
		{
			name: "independent upstream policies",
			config: `server:
  security:
    global_rate_limit: 10000
    global_rate_burst: 1000
upstream:
  - name: backend
    path: /api
    server_url: http://localhost:9000
    security:
      rate_limit: 1000
      rate_limit_burst: 100
      route_rate_limit: 200
      route_rate_burst: 20
  - name: frontend
    path: /
    server_url: http://localhost:3000
    security:
      rate_limit: 1000
      rate_limit_burst: 500
`,
			frontend: Security{RateLimit: 1000, RateLimitBurst: 500},
			backend:  Security{RateLimit: 1000, RateLimitBurst: 100, RouteRateLimit: 200, RouteRateBurst: 20},
		},
		{
			name: "environment override",
			config: `server:
  security:
    rate_limit: 600
    rate_limit_burst: 100
upstream:
  - name: backend
    path: /api
    server_url: http://localhost:9000
`,
			rateLimit: "777",
			server:    777,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			oldPath, oldConfig := ConfigPath, Config
			ConfigPath = filepath.Join(t.TempDir(), "config.yaml")
			viper.Reset()
			t.Setenv("RATE_LIMIT", test.rateLimit)
			t.Cleanup(func() {
				ConfigPath, Config = oldPath, oldConfig
				viper.Reset()
			})
			if err := os.WriteFile(ConfigPath, []byte(test.config), 0600); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load()
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Server.Security.RateLimit != test.server {
				t.Fatalf("server rate_limit = %d, want %d", cfg.Server.Security.RateLimit, test.server)
			}
			if len(cfg.Upstream) < 1 {
				t.Fatal("expected configured upstreams")
			}
			var frontend, backend Security
			for _, upstream := range cfg.Upstream {
				switch upstream.Name {
				case "frontend":
					frontend = upstream.Security
				case "backend":
					backend = upstream.Security
				}
			}
			if !reflect.DeepEqual(frontend, test.frontend) || !reflect.DeepEqual(backend, test.backend) {
				t.Fatalf("loaded policies frontend=%+v backend=%+v, want frontend=%+v backend=%+v", frontend, backend, test.frontend, test.backend)
			}
		})
	}
}

func TestCreateConfigSetsGlobalRateLimitDefaults(t *testing.T) {
	oldPath, oldConfig := ConfigPath, Config
	ConfigPath = filepath.Join(t.TempDir(), "leoxy", "config.yaml")
	viper.Reset()
	t.Setenv("RATE_LIMIT", "")
	t.Cleanup(func() {
		ConfigPath, Config = oldPath, oldConfig
		viper.Reset()
	})
	if err := CreateConfig(); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Security.GlobalRateLimit != 600 || cfg.Server.Security.GlobalRateBurst != 500 {
		t.Fatalf("global defaults = %d/%d, want 600/500", cfg.Server.Security.GlobalRateLimit, cfg.Server.Security.GlobalRateBurst)
	}
	if len(cfg.Upstream) != 1 || cfg.Upstream[0].Security.RateLimit != 0 {
		t.Fatalf("default frontend policy = %+v, want no per-IP limit", cfg.Upstream)
	}
}

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{"": "/", "api": "/api", "/api/": "/api", "/": "/", "//": "/"}
	for input, want := range cases {
		if got := NormalizePath(input); got != want {
			t.Fatalf("NormalizePath(%q) = %q, want %q", input, got, want)
		}
	}
}
