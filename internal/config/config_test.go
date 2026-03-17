package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.Enabled {
		t.Error("default Enabled should be true")
	}
	if cfg.Volume != 0.8 {
		t.Errorf("default Volume = %v, want 0.8", cfg.Volume)
	}
	if !cfg.AutoUpdate {
		t.Error("default AutoUpdate should be true")
	}
	if !cfg.Notifications {
		t.Error("default Notifications should be true")
	}
	if cfg.NotificationPosition != "top-right" {
		t.Errorf("default NotificationPosition = %q, want %q", cfg.NotificationPosition, "top-right")
	}
	if cfg.NotificationDuration != 7 {
		t.Errorf("default NotificationDuration = %v, want 7", cfg.NotificationDuration)
	}
}

func TestValidPositions(t *testing.T) {
	expected := []string{
		"top-left", "top-center", "top-right",
		"bottom-left", "bottom-center", "bottom-right",
		"center",
	}

	if len(ValidPositions) != len(expected) {
		t.Fatalf("ValidPositions has %d entries, want %d", len(ValidPositions), len(expected))
	}

	for i, pos := range expected {
		if ValidPositions[i] != pos {
			t.Errorf("ValidPositions[%d] = %q, want %q", i, ValidPositions[i], pos)
		}
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp directory to avoid touching real config
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.json")

	cfg := Config{
		Enabled:              false,
		Volume:               0.5,
		AutoUpdate:           false,
		Notifications:        false,
		NotificationPosition: "bottom-left",
		NotificationDuration: 3,
	}

	// Write directly to temp file
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatal(err)
	}

	// Read it back
	var loaded Config
	raw, err := os.ReadFile(tmpFile)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.Enabled != cfg.Enabled {
		t.Errorf("Enabled = %v, want %v", loaded.Enabled, cfg.Enabled)
	}
	if loaded.Volume != cfg.Volume {
		t.Errorf("Volume = %v, want %v", loaded.Volume, cfg.Volume)
	}
	if loaded.NotificationPosition != cfg.NotificationPosition {
		t.Errorf("NotificationPosition = %q, want %q", loaded.NotificationPosition, cfg.NotificationPosition)
	}
	if loaded.NotificationDuration != cfg.NotificationDuration {
		t.Errorf("NotificationDuration = %v, want %v", loaded.NotificationDuration, cfg.NotificationDuration)
	}
}

func TestConfigJSONFieldNames(t *testing.T) {
	cfg := DefaultConfig()
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}

	expectedKeys := []string{
		"enabled", "volume", "auto_update", "notifications",
		"notification_position", "notification_duration_seconds",
	}

	for _, key := range expectedKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("JSON missing key %q", key)
		}
	}
}

func TestLoadMissingFile(t *testing.T) {
	// Unmarshal from a non-existent file should keep defaults
	def := DefaultConfig()
	cfg := def

	// Simulate Load() behavior with missing file: unmarshal returns error, defaults stay
	err := json.Unmarshal([]byte(""), &cfg)
	if err == nil {
		t.Fatal("expected error from empty unmarshal")
	}

	if cfg.Volume != def.Volume {
		t.Errorf("Volume = %v, want default %v", cfg.Volume, def.Volume)
	}
	if cfg.NotificationPosition != def.NotificationPosition {
		t.Errorf("NotificationPosition = %q, want default %q",
			cfg.NotificationPosition, def.NotificationPosition)
	}
}

func TestConfigPartialJSON(t *testing.T) {
	// Partial JSON should merge with defaults
	partial := `{"volume": 0.3}`

	var cfg Config
	cfg = DefaultConfig()
	json.Unmarshal([]byte(partial), &cfg)

	if cfg.Volume != 0.3 {
		t.Errorf("Volume = %v, want 0.3", cfg.Volume)
	}
	// Other fields should keep defaults
	if !cfg.Enabled {
		t.Error("Enabled should remain true from defaults")
	}
	if cfg.NotificationPosition != "top-right" {
		t.Errorf("NotificationPosition = %q, want default", cfg.NotificationPosition)
	}
}

func TestDataDir(t *testing.T) {
	dir := DataDir()
	if dir == "" {
		t.Error("DataDir() returned empty string")
	}
}

func TestConfigPath(t *testing.T) {
	p := ConfigPath()
	if filepath.Base(p) != "config.json" {
		t.Errorf("ConfigPath() base = %q, want config.json", filepath.Base(p))
	}
}

func TestSoundsDir(t *testing.T) {
	dir := SoundsDir()
	if filepath.Base(dir) != "sounds" {
		t.Errorf("SoundsDir() base = %q, want sounds", filepath.Base(dir))
	}
}

func TestIconPath(t *testing.T) {
	p := IconPath()
	if filepath.Base(p) != "icon.png" {
		t.Errorf("IconPath() base = %q, want icon.png", filepath.Base(p))
	}
}
