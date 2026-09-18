package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

// fileShape mirrors Config with JSON-friendly duration strings.
type fileShape struct {
	Port                 int      `json:"port"`
	BindAddress          string   `json:"bindAddress"`
	EligibleHTTPLoopback *bool    `json:"eligibleHttpLoopback"`
	OriginAllowList      []string `json:"originAllowList"`
	PreapprovedOrigins   []string `json:"preapprovedOrigins"`
	TrustDuration        string   `json:"trustDuration"`
	SessionDuration      string   `json:"sessionDuration"`
	ApprovalTimeout      string   `json:"approvalTimeout"`
	ReadTimeout          string   `json:"readTimeout"`
	MaxActiveReads       int      `json:"maxActiveReads"`
	MaxConnections       int      `json:"maxConnections"`
	MaxPendingApprovals  int      `json:"maxPendingApprovals"`
	MaxFrameBytes        int      `json:"maxFrameBytes"`
	LogLevel             string   `json:"logLevel"`
	Fields               []string `json:"fields"`
	TLSCertFile          string   `json:"tlsCertFile"`
	TLSKeyFile           string   `json:"tlsKeyFile"`
	TLSPort              int      `json:"tlsPort"`
}

// LoadFile reads a versioned JSON config on top of defaults.
// Missing file returns defaults with no error.
func LoadFile(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	var f fileShape
	if err := json.Unmarshal(data, &f); err != nil {
		return cfg, err
	}
	if f.Port != 0 {
		cfg.Port = f.Port
	}
	if f.BindAddress != "" {
		cfg.BindAddress = f.BindAddress
	}
	if f.EligibleHTTPLoopback != nil {
		cfg.EligibleHTTPLoopback = *f.EligibleHTTPLoopback
	}
	if f.OriginAllowList != nil {
		cfg.OriginAllowList = append([]string(nil), f.OriginAllowList...)
	}
	if f.PreapprovedOrigins != nil {
		canon, err := CanonicalOriginList(f.PreapprovedOrigins)
		if err != nil {
			return cfg, err
		}
		cfg.PreapprovedOrigins = canon
	}
	var err2 error
	if cfg.TrustDuration, err2 = parseDur(f.TrustDuration, cfg.TrustDuration); err2 != nil {
		return cfg, err2
	}
	if cfg.SessionDuration, err2 = parseDur(f.SessionDuration, cfg.SessionDuration); err2 != nil {
		return cfg, err2
	}
	if cfg.ApprovalTimeout, err2 = parseDur(f.ApprovalTimeout, cfg.ApprovalTimeout); err2 != nil {
		return cfg, err2
	}
	if cfg.ReadTimeout, err2 = parseDur(f.ReadTimeout, cfg.ReadTimeout); err2 != nil {
		return cfg, err2
	}
	if f.MaxActiveReads != 0 {
		cfg.MaxActiveReads = f.MaxActiveReads
	}
	if f.MaxConnections != 0 {
		cfg.MaxConnections = f.MaxConnections
	}
	if f.MaxPendingApprovals != 0 {
		cfg.MaxPendingApprovals = f.MaxPendingApprovals
	}
	if f.MaxFrameBytes != 0 {
		cfg.MaxFrameBytes = f.MaxFrameBytes
	}
	if f.LogLevel != "" {
		cfg.LogLevel = f.LogLevel
	}
	if f.TLSCertFile != "" {
		cfg.TLSCertFile = f.TLSCertFile
	}
	if f.TLSKeyFile != "" {
		cfg.TLSKeyFile = f.TLSKeyFile
	}
	if f.TLSPort != 0 {
		cfg.TLSPort = f.TLSPort
	}
	if len(f.Fields) > 0 {
		cfg.Fields = cfg.Fields[:0]
		for _, s := range f.Fields {
			cfg.Fields = append(cfg.Fields, card.Field(s))
		}
	}
	return cfg, cfg.Validate()
}

func parseDur(s string, def time.Duration) (time.Duration, error) {
	if s == "" {
		return def, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, err
	}
	return d, nil
}
