package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndValidateDefaultConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "portfolio.yaml")
	if err := WriteExample(path, false); err != nil {
		t.Fatalf("WriteExample() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file missing: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got, want := len(cfg.Funds), 10; got != want {
		t.Fatalf("fund count = %d, want %d", got, want)
	}
	if got, want := len(cfg.Candidates), 10; got != want {
		t.Fatalf("candidate count = %d, want %d", got, want)
	}
	if cfg.ResolveStorageDSN() == cfg.Storage.DSN {
		t.Fatalf("ResolveStorageDSN() should return an absolute or joined path")
	}
}
