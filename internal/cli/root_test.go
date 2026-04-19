package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
	"github.com/derekdong-star/fund-advisor-cli/internal/service"
	"github.com/derekdong-star/fund-advisor-cli/internal/store"
)

func TestRootInitAndValidate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"validate", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("validate Execute() error = %v, stderr=%s", err, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Fatalf("expected validate output")
	}
}

func TestLatestAnalysisRoundTripIncludesCandidates(t *testing.T) {
	t.Parallel()
	st, err := store.Open(filepath.Join(t.TempDir(), "fundcli.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()
	now := time.Date(2026, 4, 15, 9, 0, 0, 0, time.UTC)
	report := model.AnalysisReport{
		Summary: model.AnalysisSummary{
			PortfolioName:        "test",
			RunDate:              now,
			PortfolioValue:       12345,
			WeightedDayChangePct: 0.012,
			ActionCounts: map[model.Action]int{
				model.ActionReduce: 1,
			},
			CandidateCount: 1,
			Notes:          []string{"candidate available"},
			GeneratedAt:    now,
		},
		Signals: []model.FundSignal{{
			FundCode:        "000001",
			FundName:        "Held Fund",
			Action:          model.ActionReduce,
			Score:           2,
			CurrentWeight:   0.2,
			TargetWeight:    0.1,
			Drift:           0.1,
			CurrentValue:    2000,
			Return20D:       -0.02,
			Return60D:       -0.05,
			Return120D:      -0.01,
			LatestTradeDate: now,
			Reason:          "overweight",
			CreatedAt:       now,
		}},
		Candidates: []model.CandidateSuggestion{{
			FundCode:        "000002",
			FundName:        "Candidate Fund",
			Category:        "active_qdii",
			Role:            "satellite",
			Score:           4,
			Return20D:       0.03,
			Return60D:       0.08,
			Return120D:      0.1,
			LatestTradeDate: now,
			ReplaceFor:      []string{"Held Fund"},
			Reason:          "better momentum",
		}},
	}
	runID, err := st.SaveAnalysis(report)
	if err != nil {
		t.Fatalf("SaveAnalysis() error = %v", err)
	}
	loaded, err := st.LatestAnalysis()
	if err != nil {
		t.Fatalf("LatestAnalysis() error = %v", err)
	}
	if loaded.RunID != runID {
		t.Fatalf("RunID = %d, want %d", loaded.RunID, runID)
	}
	if got := len(loaded.Candidates); got != 1 {
		t.Fatalf("candidate count = %d, want 1", got)
	}
	if loaded.DCAPlan != nil {
		t.Fatalf("legacy roundtrip should not require dca plan, got %+v", loaded.DCAPlan)
	}
	if got := loaded.Candidates[0].FundCode; got != "000002" {
		t.Fatalf("candidate code = %s, want 000002", got)
	}
	if got := loaded.Candidates[0].ReplaceFor[0]; got != "Held Fund" {
		t.Fatalf("replace target = %s, want Held Fund", got)
	}
}

func TestBacktestCommandRendersReport(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"backtest", "--config", configPath, "--days", "5"})
	if err := runner.Execute(); err == nil {
		t.Fatalf("expected backtest to fail without data")
	}
}

func TestDCAPlanCommandRendersWithoutFetchedData(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"dca-plan", "--config", configPath, "--format", "markdown"})
	if err := runner.Execute(); err != nil {
		t.Fatalf("dca-plan Execute() error = %v, stderr=%s", err, stderr.String())
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("DCA Plan")) {
		t.Fatalf("expected dca plan output, got %s", got)
	}
}

func TestDocsPublishCommandSupportsRefreshFlag(t *testing.T) {
	t.Parallel()
	cmd := NewRootCmd()
	docsPublishCmd, _, err := cmd.Find([]string{"docs", "publish"})
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if docsPublishCmd == nil {
		t.Fatalf("expected docs publish command")
	}
	refreshFlag := docsPublishCmd.Flags().Lookup("refresh")
	if refreshFlag == nil {
		t.Fatalf("expected refresh flag")
	}
	if got := refreshFlag.DefValue; got != "false" {
		t.Fatalf("refresh default = %s, want false", got)
	}
	dayFlag := docsPublishCmd.Flags().Lookup("days")
	if dayFlag == nil {
		t.Fatalf("expected days flag")
	}
	if got := dayFlag.DefValue; got != "180" {
		t.Fatalf("days default = %s, want 180", got)
	}
}

func TestDocsPublishCommandGeneratesGitBookTree(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	archiveDayDir := latestArchiveDayDir(t, docsRoot)
	for _, rel := range []string{
		".gitbook.yaml",
		"README.md",
		"SUMMARY.md",
		"latest/daily.md",
		"latest/dca-plan.md",
		"archive/README.md",
		filepath.Join("archive", archiveDayDir, "README.md"),
		filepath.Join("strategy", "overview.md"),
		filepath.Join("about", "risk.md"),
	} {
		if _, err := os.Stat(filepath.Join(docsRoot, rel)); err != nil {
			t.Fatalf("expected generated docs file %s: %v", rel, err)
		}
	}
}

func TestDocsPublishRespectsGenerateHomepageAndSummaryFlags(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	updated := bytes.Replace(buf, []byte("generate_homepage: true"), []byte("generate_homepage: false"), 1)
	updated = bytes.Replace(updated, []byte("generate_summary: true"), []byte("generate_summary: false"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "export", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs export Execute() error = %v, stderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "index", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs index Execute() error = %v, stderr=%s", err, stderr.String())
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	for _, rel := range []string{"README.md", "SUMMARY.md"} {
		if _, err := os.Stat(filepath.Join(docsRoot, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed when generation disabled, got err=%v", rel, err)
		}
	}
}

func TestDocsPublishRespectsArchiveByRunDateFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	updated := bytes.Replace(buf, []byte("archive_by_run_date: true"), []byte("archive_by_run_date: false"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	if _, err := os.Stat(filepath.Join(docsRoot, "archive", "README.md")); !os.IsNotExist(err) {
		t.Fatalf("expected archive index to be absent when archive_by_run_date disabled, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(docsRoot, "archive")); !os.IsNotExist(err) {
		t.Fatalf("expected archive directory to be absent when archive_by_run_date disabled, got err=%v", err)
	}
}

func TestDocsPublishRespectsOverwriteLatestFlag(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	updated := bytes.Replace(buf, []byte("overwrite_latest: true"), []byte("overwrite_latest: false"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	latestDaily := filepath.Join(docsRoot, "latest", "daily.md")
	if err := os.MkdirAll(filepath.Dir(latestDaily), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	const sentinel = "# preserved latest\n"
	if err := os.WriteFile(latestDaily, []byte(sentinel), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	content, err := os.ReadFile(latestDaily)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(content) != sentinel {
		t.Fatalf("expected latest daily page to remain unchanged when overwrite disabled, got %s", string(content))
	}
}

func TestDocsPublishGeneratesBacktestPageWhenEnabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	updated := bytes.Replace(buf, []byte("include_backtest: false"), []byte("include_backtest: true"), 1)
	updated = bytes.Replace(updated, []byte("hide_backtest_when_unavailable: true"), []byte("hide_backtest_when_unavailable: false"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	archiveDayDir := latestArchiveDayDir(t, docsRoot)
	if _, err := os.Stat(filepath.Join(docsRoot, "latest", "backtest.md")); err != nil {
		t.Fatalf("expected generated backtest page: %v", err)
	}
	if _, err := os.Stat(filepath.Join(docsRoot, "archive", archiveDayDir, "backtest.md")); err != nil {
		t.Fatalf("expected archived backtest page: %v", err)
	}
}

func TestDocsPublishPrunesExpiredArchiveSnapshots(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	updated := bytes.Replace(buf, []byte("retain_days: 0"), []byte("retain_days: 5"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	currentRunDate := time.Now().UTC()
	keptDate := currentRunDate.AddDate(0, 0, -4)
	keptDayDir := filepath.Join(docsRoot, "archive", keptDate.Format("2006"), keptDate.Format("01"), keptDate.Format("02"))
	if err := os.MkdirAll(keptDayDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(keptDayDir, "daily.md"), []byte("# keep\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	staleDate := currentRunDate.AddDate(0, 0, -5)
	staleDayDir := filepath.Join(docsRoot, "archive", staleDate.Format("2006"), staleDate.Format("01"), staleDate.Format("02"))
	if err := os.MkdirAll(staleDayDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(staleDayDir, "daily.md"), []byte("# stale\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	if _, err := os.Stat(staleDayDir); !os.IsNotExist(err) {
		t.Fatalf("expected stale archive snapshot to be removed, got err=%v", err)
	}
	if _, err := os.Stat(keptDayDir); err != nil {
		t.Fatalf("expected current archive snapshot directory to remain, got err=%v", err)
	}
}

func TestDocsPublishRemovesStaleBacktestArtifactsWhenUnavailable(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	updated := bytes.Replace(buf, []byte("include_backtest: false"), []byte("include_backtest: true"), 1)
	updated = bytes.Replace(updated, []byte("hide_backtest_when_unavailable: true"), []byte("hide_backtest_when_unavailable: false"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("initial docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	updated = bytes.Replace(updated, []byte("hide_backtest_when_unavailable: false"), []byte("hide_backtest_when_unavailable: true"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("second docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	archiveDayDir := latestArchiveDayDir(t, docsRoot)
	for _, rel := range []string{
		"latest/backtest.md",
		filepath.Join("archive", archiveDayDir, "backtest.md"),
	} {
		if _, err := os.Stat(filepath.Join(docsRoot, rel)); !os.IsNotExist(err) {
			t.Fatalf("expected stale backtest artifact %s to be removed, got err=%v", rel, err)
		}
	}
	for _, rel := range []string{"README.md", "SUMMARY.md"} {
		buf, err := os.ReadFile(filepath.Join(docsRoot, rel))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", rel, err)
		}
		if bytes.Contains(buf, []byte("backtest.md")) {
			t.Fatalf("expected %s to drop backtest link, got %s", rel, string(buf))
		}
	}
}

func TestDocsPublishRefreshBuildsMarketPool(t *testing.T) {
	t.Parallel()
	originalFetch := fetchForDocsPublish
	originalAnalyze := analyzeForDocsPublish
	originalBuild := buildMarketPoolForDocsPublish
	defer func() {
		fetchForDocsPublish = originalFetch
		analyzeForDocsPublish = originalAnalyze
		buildMarketPoolForDocsPublish = originalBuild
	}()

	fetchForDocsPublish = func(ctx context.Context, svc *service.Service, days int) error {
		return nil
	}
	analyzeForDocsPublish = func(svc *service.Service) (*model.AnalysisReport, error) {
		now := time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC)
		return &model.AnalysisReport{
			Summary: model.AnalysisSummary{
				PortfolioName:  "test",
				RunDate:        now,
				PortfolioValue: 100000,
				ActionCounts:   map[model.Action]int{},
			},
			DCAPlan: &model.DCAPlanReport{
				Summary: model.DCAPlanSummary{
					PortfolioName: "test",
					PlanDate:      now,
					Frequency:     "monthly",
				},
			},
		}, nil
	}
	buildMarketPoolForDocsPublish = func(ctx context.Context, svc *service.Service, days int) (*model.MarketPoolReport, error) {
		now := time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC)
		return &model.MarketPoolReport{
			Summary: model.MarketPoolSummary{
				RunDate:       now,
				UniverseCount: 10000,
				MatchedCount:  120,
				EligibleCount: 6,
				SelectedCount: 1,
				RetainedCount: 1,
			},
			Items: []model.MarketPoolItem{{
				Rank:            1,
				ThemeKey:        "cn_broad",
				ThemeLabel:      "A股宽基",
				FundCode:        "022459",
				FundName:        "易方达中证A500ETF联接A",
				Score:           8,
				Return120D:      0.12,
				Return250D:      0.21,
				MaxDrawdown120D: 0.09,
				FundSizeYi:      32,
				Retained:        true,
				Reason:          "250日收益 21.00%；指数工具更稳定",
			}},
		}, nil
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath, "--refresh", "--days", "10"})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs publish --refresh Execute() error = %v, stderr=%s", err, stderr.String())
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	if _, err := os.Stat(filepath.Join(docsRoot, "latest", "market-pool.md")); err != nil {
		t.Fatalf("expected refreshed market pool page: %v", err)
	}
}

func TestDocsPublishIncludesMarketPoolLinksInNavigation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	st, err := store.Open(filepath.Join(dir, "data", "fundcli.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer st.Close()
	now := time.Date(2026, 4, 19, 9, 0, 0, 0, time.UTC)
	if _, err := st.SaveMarketPool(model.MarketPoolReport{
		Summary: model.MarketPoolSummary{
			RunDate:       now,
			UniverseCount: 10000,
			MatchedCount:  120,
			EligibleCount: 6,
			SelectedCount: 1,
			RetainedCount: 1,
		},
		Items: []model.MarketPoolItem{{
			Rank:            1,
			ThemeKey:        "cn_broad",
			ThemeLabel:      "A股宽基",
			FundCode:        "022459",
			FundName:        "易方达中证A500ETF联接A",
			Score:           8,
			Return120D:      0.12,
			Return250D:      0.21,
			MaxDrawdown120D: 0.09,
			FundSizeYi:      32,
			Retained:        true,
			Reason:          "250日收益 21.00%；指数工具更稳定",
		}},
	}); err != nil {
		t.Fatalf("SaveMarketPool() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	for _, rel := range []string{"latest/market-pool.md", "README.md", "SUMMARY.md"} {
		buf, err := os.ReadFile(filepath.Join(docsRoot, rel))
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", rel, err)
		}
		if rel == "latest/market-pool.md" && !bytes.Contains(buf, []byte("Stable Market Pool")) {
			t.Fatalf("expected market pool page content, got %s", string(buf))
		}
		if rel != "latest/market-pool.md" && !bytes.Contains(buf, []byte("market-pool.md")) {
			t.Fatalf("expected navigation to include market-pool link in %s, got %s", rel, string(buf))
		}
	}
	homepage, err := os.ReadFile(filepath.Join(docsRoot, "README.md"))
	if err != nil {
		t.Fatalf("ReadFile(README.md) error = %v", err)
	}
	if !bytes.Contains(homepage, []byte("## Dashboard")) {
		t.Fatalf("expected homepage dashboard section, got %s", string(homepage))
	}
	if !bytes.Contains(homepage, []byte("## Theme Snapshot")) {
		t.Fatalf("expected homepage theme snapshot section, got %s", string(homepage))
	}
	if !bytes.Contains(homepage, []byte("A股宽基")) {
		t.Fatalf("expected homepage to include market theme summary, got %s", string(homepage))
	}
}

func TestDocsPublishHidesBacktestWhenUnavailableByDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	updated := bytes.Replace(buf, []byte("include_backtest: false"), []byte("include_backtest: true"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"docs", "publish", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("docs publish Execute() error = %v, stderr=%s", err, stderr.String())
	}

	docsRoot := filepath.Join(dir, "docs", "gitbook")
	if _, err := os.Stat(filepath.Join(docsRoot, "latest", "backtest.md")); !os.IsNotExist(err) {
		t.Fatalf("expected backtest page to be hidden when unavailable, got err=%v", err)
	}
}

func latestArchiveDayDir(t *testing.T, docsRoot string) string {
	t.Helper()
	archiveRoot := filepath.Join(docsRoot, "archive")
	years, err := listChildDirs(archiveRoot)
	if err != nil {
		t.Fatalf("listChildDirs(%s) error = %v", archiveRoot, err)
	}
	if len(years) == 0 {
		t.Fatalf("expected at least one archive year directory")
	}
	months, err := listChildDirs(filepath.Join(archiveRoot, years[0]))
	if err != nil {
		t.Fatalf("listChildDirs(months) error = %v", err)
	}
	if len(months) == 0 {
		t.Fatalf("expected at least one archive month directory")
	}
	days, err := listChildDirs(filepath.Join(archiveRoot, years[0], months[0]))
	if err != nil {
		t.Fatalf("listChildDirs(days) error = %v", err)
	}
	if len(days) == 0 {
		t.Fatalf("expected at least one archive day directory")
	}
	return filepath.Join(years[0], months[0], days[0])
}

func listChildDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	items := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			items = append(items, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(items)))
	return items, nil
}

func TestLLMPingCommandUsesConfiguredProvider(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "configs", "portfolio.yaml")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	runner := NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"init", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("init Execute() error = %v, stderr=%s", err, stderr.String())
	}

	buf, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	updated := bytes.Replace(buf, []byte("enabled: false"), []byte("enabled: true"), 1)
	updated = bytes.Replace(updated, []byte("provider: openai"), []byte("provider: mock"), 1)
	if err := os.WriteFile(configPath, updated, 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	runner = NewRootCmd()
	runner.SetOut(&stdout)
	runner.SetErr(&stderr)
	runner.SetArgs([]string{"llm", "ping", "--config", configPath})
	if err := runner.Execute(); err != nil {
		t.Fatalf("llm ping Execute() error = %v, stderr=%s", err, stderr.String())
	}
	if got := stdout.String(); !bytes.Contains([]byte(got), []byte("llm ping ok")) {
		t.Fatalf("expected llm ping success output, got %s", got)
	} else {
		if !bytes.Contains([]byte(got), []byte("base_url=")) {
			t.Fatalf("expected llm ping output to include base_url, got %s", got)
		}
		if !bytes.Contains([]byte(got), []byte("top_reason=")) {
			t.Fatalf("expected llm ping output to include top_reason, got %s", got)
		}
	}
}

func TestMarketPoolCommandIsRegistered(t *testing.T) {
	t.Parallel()
	cmd := NewRootCmd()
	marketPoolCmd, _, err := cmd.Find([]string{"market-pool"})
	if err != nil {
		t.Fatalf("Find() error = %v", err)
	}
	if marketPoolCmd == nil {
		t.Fatalf("expected market-pool command")
	}
	formatFlag := marketPoolCmd.Flags().Lookup("format")
	if formatFlag == nil || formatFlag.DefValue != "table" {
		t.Fatalf("expected format flag with default table")
	}
	daysFlag := marketPoolCmd.Flags().Lookup("days")
	if daysFlag == nil || daysFlag.DefValue != "300" {
		t.Fatalf("expected days flag with default 300")
	}
	outputFlag := marketPoolCmd.Flags().Lookup("output")
	if outputFlag == nil {
		t.Fatalf("expected output flag")
	}
}
