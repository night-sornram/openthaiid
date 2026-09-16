package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/night-sornram/openthaiid/internal/card"
)

func TestDefaultValidate(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 17891 {
		t.Fatalf("port %d", cfg.Port)
	}
	if cfg.ManagerURL() != "http://127.0.0.1:17891/" {
		t.Fatalf("manager url %s", cfg.ManagerURL())
	}
	if cfg.WSURL() != "ws://127.0.0.1:17891/v1/ws" {
		t.Fatalf("ws url %s", cfg.WSURL())
	}
	if cfg.TLSEnabled() || cfg.WSSURL() != "" || cfg.TLSPort != DefaultTLSPort {
		t.Fatalf("tls defaults %+v", cfg)
	}
}

func TestValidateRejectsUnsafe(t *testing.T) {
	cfg := Default()
	cfg.BindAddress = "0.0.0.0"
	if err := cfg.Validate(); err == nil {
		t.Fatal("wildcard bind must fail")
	}
	cfg = Default()
	cfg.OriginAllowList = []string{"*"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("wildcard origin must fail")
	}
	cfg = Default()
	cfg.TLSCertFile = "loopback.pem"
	if err := cfg.Validate(); err == nil {
		t.Fatal("tls cert without key must fail")
	}
	cfg = Default()
	cfg.TLSCertFile = "loopback.pem"
	cfg.TLSKeyFile = "loopback-key.pem"
	cfg.TLSPort = cfg.Port
	if err := cfg.Validate(); err == nil {
		t.Fatal("tlsPort == port must fail")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	body := `{
  "port": 18080,
  "trustDuration": "2h",
  "logLevel": "debug",
  "fields": ["citizenId"]
}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 18080 || cfg.TrustDuration != 2*time.Hour || cfg.LogLevel != "debug" {
		t.Fatalf("got %+v", cfg)
	}
	if len(cfg.Fields) != 1 || cfg.Fields[0] != card.FieldCitizenID {
		t.Fatalf("fields %v", cfg.Fields)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg, err := LoadFile(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 17891 {
		t.Fatal("expected defaults")
	}
}

func TestUserDirOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("OPENTHAIID_HOME", dir)
	got, err := UserDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != dir {
		t.Fatalf("UserDir %q", got)
	}
	p, err := UserConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if p != filepath.Join(dir, "config.json") {
		t.Fatalf("config path %q", p)
	}
}

func TestLoadTLSFileAndEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	body := `{
  "tlsCertFile": "loopback.pem",
  "tlsKeyFile": "loopback-key.pem",
  "tlsPort": 18443
}`
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.TLSEnabled() || cfg.TLSPort != 18443 {
		t.Fatalf("file tls %+v", cfg)
	}
	if cfg.WSSURL() != "wss://127.0.0.1:18443/v1/ws" {
		t.Fatalf("wss %s", cfg.WSSURL())
	}
	if cfg.HTTPSURL() != "https://127.0.0.1:18443/" {
		t.Fatalf("https %s", cfg.HTTPSURL())
	}
	if cfg.WSURL() != "ws://127.0.0.1:17891/v1/ws" {
		t.Fatalf("plain ws must stay default, got %s", cfg.WSURL())
	}

	t.Setenv("OPENTHAIID_HOME", dir)
	if got := cfg.ResolvedTLSCertFile(); got != filepath.Join(dir, "loopback.pem") {
		t.Fatalf("resolved cert %q", got)
	}

	t.Setenv("OPENTHAIID_TLS_CERT", filepath.Join(dir, "other.pem"))
	t.Setenv("OPENTHAIID_TLS_KEY", filepath.Join(dir, "other-key.pem"))
	t.Setenv("OPENTHAIID_TLS_PORT", "19000")
	OverlayEnv(&cfg)
	if cfg.TLSCertFile != filepath.Join(dir, "other.pem") || cfg.TLSPort != 19000 {
		t.Fatalf("env overlay %+v", cfg)
	}
	if cfg.WSSURL() != "wss://127.0.0.1:19000/v1/ws" {
		t.Fatalf("env wss %s", cfg.WSSURL())
	}
}
