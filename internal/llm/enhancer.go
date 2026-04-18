package llm

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type Enhancer struct {
	cfg    config.LLMConfig
	client Client
}

func NewEnhancer(cfg config.LLMConfig) *Enhancer {
	return &Enhancer{cfg: cfg, client: NewClient(cfg)}
}

func (e *Enhancer) EnhanceAnalysis(ctx context.Context, report *model.AnalysisReport) error {
	if !e.cfg.Enabled || report == nil {
		return nil
	}
	if len(report.Candidates) == 0 {
		report.Summary.Notes = append(report.Summary.Notes, "LLM enhancement not triggered: no replacement candidates available")
		return nil
	}
	req, untouched := buildCandidateRerankRequest(e.cfg, report)
	if len(req.Candidates) == 0 {
		return nil
	}
	resp, err := e.client.RerankCandidates(ctx, req)
	if err != nil {
		return err
	}
	if err := validateCandidateRerankResponse(req, resp); err != nil {
		return err
	}
	applyCandidateRerank(report, req, resp, untouched)
	applyRecommendationEnhancements(report)
	report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("LLM enhancement applied in %s mode for %d candidate replacements", e.cfg.Mode, len(req.Candidates)))
	return nil
}

func buildCandidateRerankRequest(cfg config.LLMConfig, report *model.AnalysisReport) (CandidateRerankRequest, []model.CandidateSuggestion) {
	limit := cfg.MaxCandidatesPerCall
	if limit <= 0 || limit > len(report.Candidates) {
		limit = len(report.Candidates)
	}
	selected := report.Candidates[:limit]
	req := CandidateRerankRequest{
		PortfolioName: report.Summary.PortfolioName,
		RunDate:       report.Summary.RunDate,
		Candidates:    make([]CandidateRerankInput, 0, len(selected)),
	}
	for _, candidate := range selected {
		req.Candidates = append(req.Candidates, CandidateRerankInput{
			FundCode:         candidate.FundCode,
			FundName:         candidate.FundName,
			Category:         candidate.Category,
			Benchmark:        candidate.Benchmark,
			Role:             candidate.Role,
			Score:            candidate.Score,
			Return20D:        candidate.Return20D,
			Return60D:        candidate.Return60D,
			Return120D:       candidate.Return120D,
			ExpenseRatio:     candidate.ExpenseRatio,
			FundSizeYi:       candidate.FundSizeYi,
			EstablishedYears: candidate.EstablishedYears,
			IsIndex:          candidate.IsIndex,
			ReplaceFor:       append([]string(nil), candidate.ReplaceFor...),
			RuleReason:       candidate.Reason,
		})
	}
	return req, append([]model.CandidateSuggestion(nil), report.Candidates[limit:]...)
}

func applyCandidateRerank(report *model.AnalysisReport, req CandidateRerankRequest, resp *CandidateRerankResponse, untouched []model.CandidateSuggestion) {
	selected := make(map[string]model.CandidateSuggestion, len(report.Candidates))
	for _, candidate := range report.Candidates {
		selected[candidate.FundCode] = candidate
	}
	rankings := append([]CandidateRanking(nil), resp.Rankings...)
	sort.Slice(rankings, func(i, j int) bool {
		if rankings[i].Rank != rankings[j].Rank {
			return rankings[i].Rank < rankings[j].Rank
		}
		if rankings[i].Score != rankings[j].Score {
			return rankings[i].Score > rankings[j].Score
		}
		return rankings[i].FundCode < rankings[j].FundCode
	})
	reordered := make([]model.CandidateSuggestion, 0, len(rankings)+len(untouched))
	for _, ranking := range rankings {
		candidate := selected[ranking.FundCode]
		candidate.EnhancedReason = strings.TrimSpace(ranking.Reason)
		reordered = append(reordered, candidate)
	}
	reordered = append(reordered, untouched...)
	report.Candidates = reordered
}

func applyRecommendationEnhancements(report *model.AnalysisReport) {
	byFundName := make(map[string]model.CandidateSuggestion, len(report.Candidates))
	for _, candidate := range report.Candidates {
		byFundName[candidate.FundName] = candidate
	}
	for idx := range report.Recommendations {
		recommendation := &report.Recommendations[idx]
		switch recommendation.Kind {
		case "SWAP":
			if candidate, ok := byFundName[recommendation.TargetFund]; ok && strings.TrimSpace(candidate.EnhancedReason) != "" {
				recommendation.EnhancedReason = candidate.EnhancedReason
				continue
			}
			recommendation.EnhancedReason = fmt.Sprintf("LLM rerank keeps the replacement focused on %s while preserving the rule-based swap constraint.", recommendation.TargetFund)
		}
	}
}
