package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSavePreapprovedOriginsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"port": 18080, "logLevel": "debug"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	origins := []string{"https://erp.clinic.example", "http://127.0.0.1:5173"}
	if err := SavePreapprovedOrigins(path, origins); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm %o", st.Mode().Perm())
	}
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != 18080 || cfg.LogLevel != "debug" {
		t.Fatalf("other keys lost: %+v", cfg)
	}
	if len(cfg.PreapprovedOrigins) != 2 || cfg.PreapprovedOrigins[0] != "https://erp.clinic.example" {
		t.Fatalf("origins %v", cfg.PreapprovedOrigins)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["port"].(float64) != 18080 {
		t.Fatalf("port %v", obj["port"])
	}
}

func TestSavePreapprovedOriginsCreatesDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	if err := SavePreapprovedOrigins(path, []string{"https://erp.clinic.example"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.PreapprovedOrigins) != 1 {
		t.Fatalf("origins %v", cfg.PreapprovedOrigins)
	}
}
