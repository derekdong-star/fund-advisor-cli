package service

import (
	"math"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/fetcher"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

func TestMarketPoolMetricsUseAccumulatedNAVAcrossDistribution(t *testing.T) {
	history := []model.FundSnapshot{
		{NAV: 2, AccNAV: 2},
		{NAV: 1.1, AccNAV: 2.1},
	}

	metrics := buildMarketPoolMetrics(history)

	if math.Abs(metrics.Return20D-0.05) > 0.000001 || metrics.MaxDrawdown120D != 0 {
		t.Fatalf("unexpected accumulated-nav metrics: %+v", metrics)
	}
}

func TestSelectThemeRankedCandidatesFindsFundsOutsideOldTop200Window(t *testing.T) {
	matched := []model.MarketSearchFund{
		{Code: "510300", Name: "沪深300ETF"},
		{Code: "513500", Name: "标普500ETF"},
	}
	rankings := make([]fetcher.MarketRankEntry, 0, 252)
	for index := 0; index < 250; index++ {
		rankings = append(rankings, fetcher.MarketRankEntry{Code: "other"})
	}
	establishedDate := time.Date(2013, time.January, 1, 0, 0, 0, 0, time.UTC)
	rankings = append(rankings,
		fetcher.MarketRankEntry{Code: "510300", EstablishedDate: establishedDate},
		fetcher.MarketRankEntry{Code: "513500", EstablishedDate: establishedDate},
	)

	candidates := selectThemeRankedCandidates(matched, rankings, 2)

	if len(candidates) != 2 {
		t.Fatalf("selectThemeRankedCandidates() returned %d candidates, want 2", len(candidates))
	}
	if candidates[0].Fund.Code != "510300" || candidates[1].Fund.Code != "513500" {
		t.Fatalf("selectThemeRankedCandidates() codes = %s, %s", candidates[0].Fund.Code, candidates[1].Fund.Code)
	}
}

func TestSelectThemeRankedCandidatesRespectsRankingOrderAndLimit(t *testing.T) {
	matched := []model.MarketSearchFund{
		{Code: "first", Name: "First"},
		{Code: "second", Name: "Second"},
	}
	rankings := []fetcher.MarketRankEntry{
		{Code: "unmatched"},
		{Code: "second"},
		{Code: "first"},
	}

	candidates := selectThemeRankedCandidates(matched, rankings, 1)

	if len(candidates) != 1 || candidates[0].Fund.Code != "second" {
		t.Fatalf("selectThemeRankedCandidates() = %+v, want second only", candidates)
	}
}

func TestSelectThemeRankedCandidatesReturnsEmptyWhenRankingsDoNotContainTheme(t *testing.T) {
	matched := []model.MarketSearchFund{{Code: "510300", Name: "沪深300ETF"}}
	rankings := []fetcher.MarketRankEntry{{Code: "unmatched"}}

	candidates := selectThemeRankedCandidates(matched, rankings, 2)

	if len(candidates) != 0 {
		t.Fatalf("selectThemeRankedCandidates() = %+v, want no candidates", candidates)
	}
}

func TestFilterThemeFundsExcludesReformDividendFundFromDividendTheme(t *testing.T) {
	cfg := config.Default()
	theme := defaultMarketThemes(cfg)[1]
	funds := []model.MarketSearchFund{
		{Code: "001076", Name: "易方达改革红利混合"},
		{Code: "009051", Name: "易方达中证红利ETF联接发起式A"},
	}

	matched := filterThemeFunds(funds, theme)

	if len(matched) != 1 || matched[0].Code != "009051" {
		t.Fatalf("filterThemeFunds() = %+v, want dividend index fund only", matched)
	}
}
