package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
)

// Config holds grove-street settings.
type Config struct {
	Enabled       bool    `json:"enabled"`
	Volume        float64 `json:"volume"`
	AutoUpdate    bool    `json:"auto_update"`
	Notifications bool    `json:"notifications"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Enabled:       true,
		Volume:        0.8,
		AutoUpdate:    true,
		Notifications: true,
	}
}

// IconPath returns the path to the notification icon.
func IconPath() string {
	return filepath.Join(DataDir(), "icon.png")
}

// Load reads config from disk, falling back to defaults.
func Load() Config {
	cfg := DefaultConfig()
	data, err := os.ReadFile(ConfigPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

// Save writes config to disk.
func Save(cfg Config) error {
	os.MkdirAll(filepath.Dir(ConfigPath()), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(), data, 0644)
}

// DataDir returns the base directory for grove-street data.
func DataDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "grove-street")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".grove-street")
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return filepath.Join(DataDir(), "config.json")
}

// SoundsDir returns the path to the sounds directory.
func SoundsDir() string {
	return filepath.Join(DataDir(), "sounds")
}
