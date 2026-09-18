package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// SavePreapprovedOrigins writes the explicit IT clinic-website list into the
// JSON config, preserving other keys. Creates the parent directory if needed.
// Origins must already be validated. Never writes card field values.
func SavePreapprovedOrigins(path string, origins []string) error {
	path = filepath.Clean(path)
	if path == "" || path == "." {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	obj := map[string]any{}
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		if err := json.Unmarshal(data, &obj); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	if origins == nil {
		origins = []string{}
	}
	copied := make([]string, len(origins))
	copy(copied, origins)
	obj["preapprovedOrigins"] = copied
	raw, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o600)
}
