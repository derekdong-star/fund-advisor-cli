package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteExampleIncludesGitBookWorkflowDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "portfolio.yaml")
	if err := WriteExample(path, false); err != nil {
		t.Fatalf("WriteExample() error = %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := cfg.Publishing.GitBook.DocsRoot; got != filepath.Join("..", "docs", "gitbook") {
		t.Fatalf("DocsRoot = %s, want ../docs/gitbook", got)
	}
	if got := cfg.Publishing.GitBook.ProjectDirectory; got != filepath.ToSlash(filepath.Join("docs", "gitbook")) {
		t.Fatalf("ProjectDirectory = %s, want docs/gitbook", got)
	}
	if got := cfg.Publishing.GitBook.OrganizationID; got != "" {
		t.Fatalf("OrganizationID = %q, want empty", got)
	}
	if got := cfg.Publishing.GitBook.SiteID; got != "" {
		t.Fatalf("SiteID = %q, want empty", got)
	}
	if got := cfg.Publishing.GitBook.SpaceID; got != "" {
		t.Fatalf("SpaceID = %q, want empty", got)
	}
}

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
	if got := cfg.Strategy.Turnover.DCAFrequency; got != "monthly" {
		t.Fatalf("DCAFrequency = %s, want monthly", got)
	}
	if got := cfg.Strategy.Turnover.MinDCAFundAmount; got != 1000 {
		t.Fatalf("MinDCAFundAmount = %.0f, want 1000", got)
	}
	if got := cfg.Strategy.Turnover.MaxDCAFunds; got != 3 {
		t.Fatalf("MaxDCAFunds = %d, want 3", got)
	}
	if cfg.Strategy.Turnover.PauseDCAOnRisk == nil || !*cfg.Strategy.Turnover.PauseDCAOnRisk {
		t.Fatalf("PauseDCAOnRisk should default to true")
	}
	if !cfg.Publishing.GitBook.Enabled {
		t.Fatalf("GitBook publishing should default to enabled")
	}
	if got := cfg.Publishing.GitBook.Mode; got != "git-sync" {
		t.Fatalf("GitBook mode = %s, want git-sync", got)
	}
	if got := cfg.Publishing.GitBook.ProjectDirectory; got != filepath.ToSlash(filepath.Join("docs", "gitbook")) {
		t.Fatalf("ProjectDirectory = %s, want docs/gitbook", got)
	}
	if got := cfg.Publishing.GitBook.Visibility; got != "public" {
		t.Fatalf("Visibility = %s, want public", got)
	}
	if !cfg.Publishing.GitBook.HideBacktestWhenUnavailable {
		t.Fatalf("HideBacktestWhenUnavailable should default to true")
	}
	if got := cfg.Publishing.GitBook.BacktestDays; got != 120 {
		t.Fatalf("BacktestDays = %d, want 120", got)
	}
	if got := cfg.Publishing.GitBook.BacktestRebalanceEvery; got != 20 {
		t.Fatalf("BacktestRebalanceEvery = %d, want 20", got)
	}
}
