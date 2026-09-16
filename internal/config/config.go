package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

// DefaultTLSPort is the optional loopback HTTPS/WSS listener.
// Plain HTTP/WS stays on Port (17891). Never ship a private key for this port.
const DefaultTLSPort = 17892

// Config is the typed runtime configuration.
// Precedence: defaults → file → env → flags.
type Config struct {
	Port int `json:"port"`

	// BindAddress is always a literal IPv4 loopback address.
	BindAddress string `json:"bindAddress"`

	// TLSCertFile and TLSKeyFile enable an optional second loopback listener
	// (HTTPS + WSS) for Firefox/Safari Fill from an https ERP. Paths are
	// technician-managed (mkcert or a local self-signed cert). Empty = HTTP/WS
	// only. Never commit these files; never ship a shared private key.
	TLSCertFile string `json:"tlsCertFile"`
	TLSKeyFile  string `json:"tlsKeyFile"`
	// TLSPort is the HTTPS/WSS listen port. Default 17892. Ignored unless both
	// certificate paths are set. Must differ from Port.
	TLSPort int `json:"tlsPort"`

	// EligibleHTTPLoopback allows http://127.0.0.1 and http://localhost origins
	// for development. HTTPS origins are always eligible.
	EligibleHTTPLoopback bool `json:"eligibleHttpLoopback"`

	// OriginAllowList, when non-empty, restricts which origins may request
	// approval. It does not grant trust by itself.
	OriginAllowList []string `json:"originAllowList"`

	TrustDuration   time.Duration `json:"trustDuration"`
	SessionDuration time.Duration `json:"sessionDuration"`
	ApprovalTimeout time.Duration `json:"approvalTimeout"`
	ReadTimeout     time.Duration `json:"readTimeout"`

	MaxActiveReads      int `json:"maxActiveReads"`
	MaxConnections      int `json:"maxConnections"`
	MaxPendingApprovals int `json:"maxPendingApprovals"`
	MaxFrameBytes       int `json:"maxFrameBytes"`

	LogLevel string `json:"logLevel"`

	// Fields is the default requested field set for docs/doctor.
	Fields []card.Field `json:"fields"`
}

// Default returns the MVP defaults from the engineering plan.
func Default() Config {
	return Config{
		Port:                 17891,
		BindAddress:          "127.0.0.1",
		TLSPort:              DefaultTLSPort,
		EligibleHTTPLoopback: true,
		TrustDuration:        8 * time.Hour,
		SessionDuration:      15 * time.Minute,
		ApprovalTimeout:      120 * time.Second,
		ReadTimeout:          10 * time.Second,
		MaxActiveReads:       1,
		MaxConnections:       16,
		MaxPendingApprovals:  4,
		MaxFrameBytes:        64 * 1024,
		LogLevel:             "info",
		Fields:               append([]card.Field(nil), card.DefaultFields...),
	}
}

// Addr returns the listen address.
func (c Config) Addr() string {
	return net.JoinHostPort(c.BindAddress, strconv.Itoa(c.Port))
}

// WSURL is the default browser endpoint (plain HTTP helper).
func (c Config) WSURL() string {
	return fmt.Sprintf("ws://%s/v1/ws", c.Addr())
}

// TLSEnabled is true when a technician configured loopback HTTPS/WSS.
func (c Config) TLSEnabled() bool {
	return strings.TrimSpace(c.TLSCertFile) != "" && strings.TrimSpace(c.TLSKeyFile) != ""
}

// TLSAddr is the optional HTTPS/WSS listen address (127.0.0.1 only).
func (c Config) TLSAddr() string {
	port := c.TLSPort
	if port < 1 {
		port = DefaultTLSPort
	}
	return net.JoinHostPort(c.BindAddress, strconv.Itoa(port))
}

// WSSURL is the optional TLS browser endpoint. Empty when TLS is off.
func (c Config) WSSURL() string {
	if !c.TLSEnabled() {
		return ""
	}
	return fmt.Sprintf("wss://%s/v1/ws", c.TLSAddr())
}

// HTTPSURL is the optional HTTPS manager URL. Ready stays on ManagerURL (http).
func (c Config) HTTPSURL() string {
	if !c.TLSEnabled() {
		return ""
	}
	return fmt.Sprintf("https://%s/", c.TLSAddr())
}

// ManagerURL is the local HTML manager served by the bridge (plain HTTP).
func (c Config) ManagerURL() string {
	return fmt.Sprintf("http://%s/", c.Addr())
}

// ResolvedTLSCertFile returns an absolute certificate path. Relative paths
// resolve against the per-user OpenThaiID directory.
func (c Config) ResolvedTLSCertFile() string {
	return resolveMaybeRelative(c.TLSCertFile)
}

// ResolvedTLSKeyFile returns an absolute private-key path. Relative paths
// resolve against the per-user OpenThaiID directory.
func (c Config) ResolvedTLSKeyFile() string {
	return resolveMaybeRelative(c.TLSKeyFile)
}

func resolveMaybeRelative(p string) string {
	p = strings.TrimSpace(p)
	if p == "" || filepath.IsAbs(p) {
		return p
	}
	dir, err := UserDir()
	if err != nil || dir == "" {
		return p
	}
	return filepath.Join(dir, p)
}

// OverlayEnv applies OPENTHAIID_TLS_* (and only those) on top of file defaults.
// Flags still win after this. Does not read card data.
func OverlayEnv(cfg *Config) {
	if cfg == nil {
		return
	}
	if v := strings.TrimSpace(os.Getenv("OPENTHAIID_TLS_CERT")); v != "" {
		cfg.TLSCertFile = v
	}
	if v := strings.TrimSpace(os.Getenv("OPENTHAIID_TLS_KEY")); v != "" {
		cfg.TLSKeyFile = v
	}
	if v := strings.TrimSpace(os.Getenv("OPENTHAIID_TLS_PORT")); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			cfg.TLSPort = n
		}
	}
}

// Validate rejects unsafe values before the process listens.
func (c Config) Validate() error {
	ip := net.ParseIP(c.BindAddress)
	if ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("bindAddress must be a literal loopback address, got %q", c.BindAddress)
	}
	if ip.To4() == nil {
		return fmt.Errorf("bindAddress must be IPv4 loopback, got %q", c.BindAddress)
	}
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port out of range: %d", c.Port)
	}
	c.TLSCertFile = strings.TrimSpace(c.TLSCertFile)
	c.TLSKeyFile = strings.TrimSpace(c.TLSKeyFile)
	certSet := c.TLSCertFile != ""
	keySet := c.TLSKeyFile != ""
	if certSet != keySet {
		return fmt.Errorf("tlsCertFile and tlsKeyFile must be set together")
	}
	if c.TLSPort < 1 || c.TLSPort > 65535 {
		return fmt.Errorf("tlsPort out of range: %d", c.TLSPort)
	}
	if certSet && c.TLSPort == c.Port {
		return fmt.Errorf("tlsPort must differ from port (%d)", c.Port)
	}
	if c.TrustDuration <= 0 || c.SessionDuration <= 0 || c.ApprovalTimeout <= 0 || c.ReadTimeout <= 0 {
		return fmt.Errorf("durations must be positive")
	}
	if c.MaxActiveReads < 1 || c.MaxConnections < 1 || c.MaxPendingApprovals < 1 {
		return fmt.Errorf("connection/read limits must be at least 1")
	}
	if c.MaxFrameBytes < 1024 || c.MaxFrameBytes > 1<<20 {
		return fmt.Errorf("maxFrameBytes out of range")
	}
	switch c.LogLevel {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid logLevel %q", c.LogLevel)
	}
	if _, err := card.NormalizeFields(c.Fields); err != nil {
		return err
	}
	for _, o := range c.OriginAllowList {
		if o == "*" || o == "" {
			return fmt.Errorf("wildcard or empty origin in allow list is forbidden")
		}
	}
	return nil
}

// UserDir returns the per-user OpenThaiID directory (config, logs, pid).
func UserDir() (string, error) {
	if x := os.Getenv("OPENTHAIID_HOME"); x != "" {
		return x, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "openthaiid"), nil
}

// UserConfigPath returns the OS user-config path for OpenThaiID.
func UserConfigPath() (string, error) {
	if x := os.Getenv("OPENTHAIID_CONFIG"); x != "" {
		return x, nil
	}
	dir, err := UserDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}
