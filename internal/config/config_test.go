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
