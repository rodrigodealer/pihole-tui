package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Type     string `json:"type"`
	Host     string `json:"host"`
	Password string `json:"password"`
	Username string `json:"username"`
}

func DefaultPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "pihole-tui", "config.json")
}

func Load() (*Config, error) {
	path := DefaultPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{Type: "pihole", Host: "http://192.168.0.70"}, nil
		}
		return nil, err
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if cfg.Type == "" {
		cfg.Type = "pihole"
	}
	return &cfg, nil
}

func (c *Config) Save() error {
	path := DefaultPath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
