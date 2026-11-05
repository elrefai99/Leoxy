package file

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Upstream struct {
	Name          string   `mapstructure:"name"`
	Protocol      string   `mapstructure:"protocol"`
	Listen        string   `mapstructure:"listen"`
	Path          string   `mapstructure:"path"`
	IP            bool     `mapstructure:"ip"`
	ServerURL     string   `mapstructure:"server_url"`
	Servers       []string `mapstructure:"servers"`
	Limit_request int      `mapstructure:"limit_request"`
	Body          int      `mapstructure:"body"`
	Security      Security `mapstructure:"security"`
}

type Timeout struct {
	Connect uint `mapstructure:"connect"`
	Read    uint `mapstructure:"read"`
	Write   uint `mapstructure:"write"`
	Idle    uint `mapstructure:"idle"`
	Request uint `mapstructure:"request"`
}

type Watcher struct {
	Status          string `mapstructure:"status"`
	ProjectName     string `mapstructure:"project_name"`
	MemoryLimitMB   int    `mapstructure:"memory_limit_mb"`
	IntervalSeconds int    `mapstructure:"interval_seconds"`
	CooldownSeconds int    `mapstructure:"cooldown_seconds"`
	HeapProfile     bool   `mapstructure:"heap_profile"`
	WebhookURL      string `mapstructure:"webhook_url"`
}

type Security struct {
	RateLimit         int      `mapstructure:"rate_limit"`
	RateLimitBurst    int      `mapstructure:"rate_limit_burst"`
	GlobalRateLimit   int      `mapstructure:"global_rate_limit"`
	GlobalRateBurst   int      `mapstructure:"global_rate_burst"`
	RouteRateLimit    int      `mapstructure:"route_rate_limit"`
	RouteRateBurst    int      `mapstructure:"route_rate_burst"`
	APIKeys           []string `mapstructure:"api_keys"`
	JWTSecret         string   `mapstructure:"jwt_secret"`
	JWTIssuer         string   `mapstructure:"jwt_issuer"`
	JWTAudience       string   `mapstructure:"jwt_audience"`
	RequireMTLS       bool     `mapstructure:"require_mtls"`
	AllowCIDRs        []string `mapstructure:"allow_cidrs"`
	DenyCIDRs         []string `mapstructure:"deny_cidrs"`
	AllowedMethods    []string `mapstructure:"allowed_methods"`
	AllowedPaths      []string `mapstructure:"allowed_paths"`
	MaxBody           int      `mapstructure:"max_body"`
	MaxConnsPerIP     int      `mapstructure:"max_conns_per_ip"`
	RedisConnect      uint     `mapstructure:"redis_connect"`
	RedisAddr         string   `mapstructure:"redis_addr"`
	RedisPassword     string   `mapstructure:"redis_password"`
	RedisDB           int      `mapstructure:"redis_db"`
	RedisFailOpen     bool     `mapstructure:"redis_fail_open"`
	TrustedProxyCIDRs []string `mapstructure:"trusted_proxy_cidrs"`
	InternalCIDRs     []string `mapstructure:"internal_cidrs"`
	TLSCert           string   `mapstructure:"tls_cert"`
	TLSKey            string   `mapstructure:"tls_key"`
	ClientCA          string   `mapstructure:"client_ca"`
}

type ServerConfig struct {
	Server struct {
		Host     string   `mapstructure:"host"`
		Port     string   `mapstructure:"port"`
		Security Security `mapstructure:"security"`
		Timeout  Timeout  `mapstructure:"timeout"`
	} `mapstructure:"server"`
	Upstream []Upstream `mapstructure:"upstream"`
	Watcher  Watcher    `mapstructure:"watcher"`
}

var Config *ServerConfig

var ConfigPath = "leoxy/leoxy_config.yaml"

var reservedPaths = []string{"/ping", "/health", "/metrics", "/leoxy"}

func Load() (*ServerConfig, error) {
	viper.SetConfigFile(ConfigPath)
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(`.`, `_`))

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error loading config file: %s", err)
	}

	var cfg ServerConfig
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error reading config file: %s", err)
	}
	if cfg.Server.Timeout.Connect == 0 {
		cfg.Server.Timeout.Connect = 5
	}
	if cfg.Server.Timeout.Read == 0 {
		cfg.Server.Timeout.Read = 30
	}
	if cfg.Server.Timeout.Write == 0 {
		cfg.Server.Timeout.Write = 30
	}
	if cfg.Server.Timeout.Idle == 0 {
		cfg.Server.Timeout.Idle = 60
	}
	if cfg.Server.Timeout.Request == 0 {
		cfg.Server.Timeout.Request = 60
	}
	if cfg.Server.Security.RedisConnect == 0 {
		cfg.Server.Security.RedisConnect = 16
	}

	if value := os.Getenv("PORT"); value != "" {
		cfg.Server.Port = value
	}
	if value := os.Getenv("LEOXY_HOST"); value != "" {
		cfg.Server.Host = value
	}
	if value := os.Getenv("RATE_LIMIT"); value != "" {
		limit, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("RATE_LIMIT must be an integer: %w", err)
		}
		cfg.Server.Security.RateLimit = limit
	}
	if value := os.Getenv("JWT_SECRET"); value != "" {
		cfg.Server.Security.JWTSecret = value
	}
	if value := os.Getenv("JWT_ISSUER"); value != "" {
		cfg.Server.Security.JWTIssuer = value
	}
	if value := os.Getenv("JWT_AUDIENCE"); value != "" {
		cfg.Server.Security.JWTAudience = value
	}
	if value := os.Getenv("API_KEYS"); value != "" {
		cfg.Server.Security.APIKeys = splitSecrets(value)
	}
	if value := os.Getenv("REDIS_PASSWORD"); value != "" {
		cfg.Server.Security.RedisPassword = value
	}
	if value := os.Getenv("MEMORY_LIMIT_MB"); value != "" {
		if limit, err := strconv.Atoi(value); err == nil {
			cfg.Watcher.MemoryLimitMB = limit
		}
	}
	if value := os.Getenv("PROJECT_NAME"); value != "" {
		cfg.Watcher.ProjectName = value
	}
	if value := os.Getenv("Watcher_WEBHOOK_URL"); value != "" {
		cfg.Watcher.WebhookURL = value
	}

	Config = &cfg
	return Config, nil
}

func NormalizePath(value string) string {
	prefix := strings.TrimSpace(value)
	if prefix == "" {
		return "/"
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if prefix != "/" {
		prefix = strings.TrimRight(prefix, "/")
		if prefix == "" {
			return "/"
		}
	}
	return prefix
}

func Validate(cfg *ServerConfig) error {
	var errs []error
	server := cfg.Server.Security
	errs = append(errs, validateSecurity("server.security", server)...)

	needTLS := server.RequireMTLS
	paths := map[string]string{}
	listeners := map[string]string{}

	for i, upstream := range cfg.Upstream {
		name := upstream.Name
		if name == "" {
			name = fmt.Sprintf("#%d", i)
		}
		protocol := strings.ToLower(strings.TrimSpace(upstream.Protocol))
		switch protocol {
		case "tcp", "udp":
			listen := strings.TrimSpace(upstream.Listen)
			if listen == "" || strings.TrimSpace(upstream.ServerURL) == "" {
				errs = append(errs, fmt.Errorf("upstream %q: %s requires listen and server_url", name, protocol))
				continue
			}
			key := protocol + "|" + listen
			if previous, ok := listeners[key]; ok {
				errs = append(errs, fmt.Errorf("upstream %q: %s listener %s already used by %q", name, protocol, listen, previous))
			}
			listeners[key] = name
			errs = append(errs, validateSecurity("upstream "+name, upstream.Security)...)
		case "", "http", "https", "http2", "graphql", "websocket":
			path := NormalizePath(upstream.Path)
			if strings.ContainsAny(path, " \t\r\n{}") {
				errs = append(errs, fmt.Errorf("upstream %q: path %q contains unsupported characters", name, path))
			}
			if isReserved(path) {
				errs = append(errs, fmt.Errorf("upstream %q: path %q is reserved", name, path))
			}
			if previous, ok := paths[path]; ok {
				errs = append(errs, fmt.Errorf("upstream %q: path %q already used by %q", name, path, previous))
			}
			paths[path] = name
			if upstream.Security.RequireMTLS {
				needTLS = true
			}
			errs = append(errs, validateSecurity("upstream "+name, upstream.Security)...)
		default:
			errs = append(errs, fmt.Errorf("upstream %q: unsupported protocol %q", name, upstream.Protocol))
		}
	}

	if (server.TLSCert == "") != (server.TLSKey == "") {
		errs = append(errs, errors.New("server.security: tls_cert and tls_key must be set together"))
	}
	if needTLS && (server.TLSCert == "" || server.TLSKey == "" || server.ClientCA == "") {
		errs = append(errs, errors.New("require_mtls needs server.security tls_cert, tls_key and client_ca"))
	}
	return errors.Join(errs...)
}

func isReserved(path string) bool {
	for _, reserved := range reservedPaths {
		if path == reserved || strings.HasPrefix(path, reserved+"/") {
			return true
		}
	}
	return false
}

func validateSecurity(scope string, security Security) []error {
	var errs []error
	numbers := map[string]int{
		"rate_limit":        security.RateLimit,
		"rate_limit_burst":  security.RateLimitBurst,
		"global_rate_limit": security.GlobalRateLimit,
		"global_rate_burst": security.GlobalRateBurst,
		"route_rate_limit":  security.RouteRateLimit,
		"route_rate_burst":  security.RouteRateBurst,
		"max_body":          security.MaxBody,
		"max_conns_per_ip":  security.MaxConnsPerIP,
	}
	for name, value := range numbers {
		if value < 0 {
			errs = append(errs, fmt.Errorf("%s: %s must not be negative", scope, name))
		}
	}
	lists := map[string][]string{
		"allow_cidrs":         security.AllowCIDRs,
		"deny_cidrs":          security.DenyCIDRs,
		"trusted_proxy_cidrs": security.TrustedProxyCIDRs,
		"internal_cidrs":      security.InternalCIDRs,
	}
	for name, values := range lists {
		for _, value := range values {
			if _, _, err := net.ParseCIDR(strings.TrimSpace(value)); err != nil {
				errs = append(errs, fmt.Errorf("%s: %s has invalid CIDR %q", scope, name, value))
			}
		}
	}
	if security.JWTSecret != "" && len(security.JWTSecret) < 32 {
		errs = append(errs, fmt.Errorf("%s: jwt_secret must be at least 32 bytes", scope))
	}
	return errs
}

func splitSecrets(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}
