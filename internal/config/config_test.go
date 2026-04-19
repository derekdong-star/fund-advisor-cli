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
	if cfg.LLM.Enabled {
		t.Fatalf("LLM should default to disabled")
	}
	if got := cfg.LLM.Provider; got != "openai" {
		t.Fatalf("LLM provider = %s, want openai", got)
	}
	if got := cfg.LLM.BaseURL; got != "https://api.openai.com/v1" {
		t.Fatalf("LLM base URL = %s, want https://api.openai.com/v1", got)
	}
	if got := cfg.LLM.APIKeyEnv; got != "FUND_ADVISOR_LLM_API_KEY" {
		t.Fatalf("LLM api key env = %s, want FUND_ADVISOR_LLM_API_KEY", got)
	}
	if got := cfg.LLM.Mode; got != "rerank_only" {
		t.Fatalf("LLM mode = %s, want rerank_only", got)
	}
	if !cfg.MarketPool.Enabled {
		t.Fatalf("market pool should default to enabled")
	}
	if got := cfg.MarketPool.SelectionCount; got != 6 {
		t.Fatalf("market pool selection count = %d, want 6", got)
	}
	if got := cfg.MarketPool.MaxFundsPerTheme; got != 12 {
		t.Fatalf("market pool max funds per theme = %d, want 12", got)
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
	if got := cfg.LLM.Model; got != "gpt-5-mini" {
		t.Fatalf("LLM model = %s, want gpt-5-mini", got)
	}
	if got := cfg.LLM.TimeoutSeconds; got != 20 {
		t.Fatalf("LLM timeout = %d, want 20", got)
	}
	if got := cfg.LLM.MaxCandidatesPerCall; got != 8 {
		t.Fatalf("LLM max candidates = %d, want 8", got)
	}
	if got := cfg.MarketPool.SelectionCount; got != 6 {
		t.Fatalf("market pool selection count = %d, want 6", got)
	}
	if got := cfg.MarketPool.MaxFundsPerTheme; got != 12 {
		t.Fatalf("market pool max funds per theme = %d, want 12", got)
	}
	if got := cfg.MarketPool.MinReturn120D; got != 0.08 {
		t.Fatalf("market pool min 120d return = %.2f, want 0.08", got)
	}
}
