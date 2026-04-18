package llm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

func TestEnhancerAddsNoteWhenNoCandidates(t *testing.T) {
	t.Parallel()
	report := &model.AnalysisReport{Summary: model.AnalysisSummary{PortfolioName: "test"}}
	enhancer := NewEnhancer(config.LLMConfig{Enabled: true, Provider: "mock", Model: "mock-1", APIKeyEnv: "OPENAI_API_KEY", TimeoutSeconds: 20, Mode: "rerank_only", MaxCandidatesPerCall: 8})
	if err := enhancer.EnhanceAnalysis(context.Background(), report); err != nil {
		t.Fatalf("EnhanceAnalysis() error = %v", err)
	}
	if len(report.Summary.Notes) == 0 || !strings.Contains(report.Summary.Notes[0], "not triggered") {
		t.Fatalf("expected no-candidate note, got %+v", report.Summary.Notes)
	}
}

func TestEnhancerReordersCandidatesAndAddsReasons(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 18, 9, 0, 0, 0, time.UTC)
	report := &model.AnalysisReport{
		Summary: model.AnalysisSummary{PortfolioName: "test", RunDate: now},
		Candidates: []model.CandidateSuggestion{
			{FundCode: "A", FundName: "Alpha", Score: 7, Return20D: 0.02, Return60D: 0.04, ReplaceFor: []string{"Weak Fund"}, Reason: "rule alpha"},
			{FundCode: "B", FundName: "Beta", Score: 7, Return20D: 0.03, Return60D: 0.08, ReplaceFor: []string{"Weak Fund"}, Reason: "rule beta"},
		},
		Recommendations: []model.TradeRecommendation{{Kind: "SWAP", SourceFund: "Weak Fund", TargetFund: "Beta", Reason: "swap to beta", CreatedAt: now}},
	}
	enhancer := NewEnhancer(config.LLMConfig{Enabled: true, Provider: "mock", Model: "mock-1", APIKeyEnv: "OPENAI_API_KEY", TimeoutSeconds: 20, Mode: "rerank_only", MaxCandidatesPerCall: 8})
	if err := enhancer.EnhanceAnalysis(context.Background(), report); err != nil {
		t.Fatalf("EnhanceAnalysis() error = %v", err)
	}
	if got := report.Candidates[0].FundCode; got != "B" {
		t.Fatalf("top candidate = %s, want B", got)
	}
	if strings.TrimSpace(report.Candidates[0].EnhancedReason) == "" {
		t.Fatalf("expected enhanced reason on reranked candidate")
	}
	if strings.TrimSpace(report.Recommendations[0].EnhancedReason) == "" {
		t.Fatalf("expected enhanced reason on swap recommendation")
	}
	if len(report.Summary.Notes) == 0 {
		t.Fatalf("expected summary note for llm enhancement")
	}
}
