package cli

import (
	"bytes"
	"path/filepath"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
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
