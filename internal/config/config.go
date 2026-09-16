package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

// Config is the typed runtime configuration.
// Precedence: defaults → file → flags.
type Config struct {
	Port int `json:"port"`

	// BindAddress is always a literal IPv4 loopback address.
	BindAddress string `json:"bindAddress"`

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

// WSURL is the default browser endpoint.
func (c Config) WSURL() string {
	return fmt.Sprintf("ws://%s/v1/ws", c.Addr())
}

// ManagerURL is the local HTML manager served by the bridge.
func (c Config) ManagerURL() string {
	return fmt.Sprintf("http://%s/", c.Addr())
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

// UserConfigPath returns the OS user-config path for OpenThaiID.
func UserConfigPath() (string, error) {
	if x := os.Getenv("OPENTHAIID_CONFIG"); x != "" {
		return x, nil
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "openthaiid", "config.json"), nil
}
