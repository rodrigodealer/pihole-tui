package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Host != "http://192.168.0.70" {
		t.Errorf("expected default host, got %s", cfg.Host)
	}
}

func TestSaveAndLoad(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	cfg := &Config{Host: "http://10.0.0.1", Password: "secret"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	path := filepath.Join(tmp, ".config", "pihole-tui", "config.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config file not created")
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}
	if loaded.Host != "http://10.0.0.1" {
		t.Errorf("expected http://10.0.0.1, got %s", loaded.Host)
	}
	if loaded.Password != "secret" {
		t.Errorf("expected secret, got %s", loaded.Password)
	}
}
