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
	if got := cfg.Publishing.GitBook.MomentumSensitivityDays; got != 1200 {
		t.Fatalf("MomentumSensitivityDays = %d, want 1200", got)
	}
	if got := cfg.Publishing.GitBook.MomentumSensitivitySeeds; got != 5 {
		t.Fatalf("MomentumSensitivitySeeds = %d, want 5", got)
	}
	if got := cfg.Publishing.GitBook.MomentumParameterGridDays; got != 1200 {
		t.Fatalf("MomentumParameterGridDays = %d, want 1200", got)
	}
	if got := cfg.Publishing.GitBook.MomentumParameterGridSeeds; got != 5 {
		t.Fatalf("MomentumParameterGridSeeds = %d, want 5", got)
	}
	if got := cfg.Publishing.GitBook.MomentumStageDays; got != 1200 {
		t.Fatalf("MomentumStageDays = %d, want 1200", got)
	}
	if got := cfg.Publishing.GitBook.MomentumStageSeeds; got != 5 {
		t.Fatalf("MomentumStageSeeds = %d, want 5", got)
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
	if !cfg.MomentumPool.Enabled {
		t.Fatalf("momentum pool should default to enabled")
	}
	if got := cfg.MomentumPool.SelectionCount; got != 3 {
		t.Fatalf("momentum pool selection count = %d, want 3", got)
	}
	weightTotal := cfg.MomentumPool.MomentumWeight20D + cfg.MomentumPool.MomentumWeight60D + cfg.MomentumPool.MomentumWeight120D + cfg.MomentumPool.MomentumWeight250D
	if weightTotal != 1 {
		t.Fatalf("momentum pool weight total = %.2f, want 1", weightTotal)
	}
	if cfg.MomentumPool.MomentumWeight20D != 0.15 || cfg.MomentumPool.MomentumWeight60D != 0.25 || cfg.MomentumPool.MomentumWeight120D != 0.40 || cfg.MomentumPool.MomentumWeight250D != 0.20 {
		t.Fatalf("unexpected momentum weights: %+v", cfg.MomentumPool)
	}
	if got := len(cfg.MomentumPool.AllowedCompanies); got != 30 {
		t.Fatalf("momentum pool allowed companies = %d, want 30", got)
	}
	if got := cfg.MomentumPool.AllowedCompanies; got[20] != "大成" || got[21] != "华商" || got[26] != "交银施罗德" || got[29] != "国寿安保" {
		t.Fatalf("unexpected momentum pool allowed companies: %v", got)
	}
	if cfg.MomentumPool.CandidateLimit != 0 || cfg.MomentumPool.MinFundSizeYi != 0 || cfg.MomentumPool.MinEstablishedYears != 0 || cfg.MomentumPool.MaxDrawdown120D != 1 {
		t.Fatalf("momentum visibility should not use candidate, product-size, or age caps: %+v", cfg.MomentumPool)
	}
	if got := cfg.MomentumPool.MaxDataAgeDays; got != 7 {
		t.Fatalf("momentum pool max data age days = %d, want 7", got)
	}
	if got := cfg.MomentumPool.ForwardValidationDays; got != 20 {
		t.Fatalf("momentum pool forward validation days = %d, want 20", got)
	}
}
