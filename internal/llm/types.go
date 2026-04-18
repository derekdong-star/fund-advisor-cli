package llm

import (
	"context"
	"time"
)

type Client interface {
	RerankCandidates(ctx context.Context, req CandidateRerankRequest) (*CandidateRerankResponse, error)
	Ping(ctx context.Context, req CandidateRerankRequest) (*CandidateRerankResponse, error)
}

type CandidateRerankRequest struct {
	PortfolioName string                 `json:"portfolio_name"`
	RunDate       time.Time              `json:"run_date"`
	Candidates    []CandidateRerankInput `json:"candidates"`
}

type CandidateRerankInput struct {
	FundCode         string   `json:"fund_code"`
	FundName         string   `json:"fund_name"`
	Category         string   `json:"category"`
	Benchmark        string   `json:"benchmark"`
	Role             string   `json:"role"`
	Score            int      `json:"score"`
	Return20D        float64  `json:"return_20d"`
	Return60D        float64  `json:"return_60d"`
	Return120D       float64  `json:"return_120d"`
	ExpenseRatio     float64  `json:"expense_ratio,omitempty"`
	FundSizeYi       float64  `json:"fund_size_yi,omitempty"`
	EstablishedYears float64  `json:"established_years,omitempty"`
	IsIndex          bool     `json:"is_index,omitempty"`
	ReplaceFor       []string `json:"replace_for,omitempty"`
	RuleReason       string   `json:"rule_reason"`
}

type CandidateRerankResponse struct {
	Rankings []CandidateRanking `json:"rankings"`
}

type CandidateRanking struct {
	FundCode string  `json:"fund_code"`
	Rank     int     `json:"rank"`
	Score    float64 `json:"score"`
	Reason   string  `json:"reason"`
}
