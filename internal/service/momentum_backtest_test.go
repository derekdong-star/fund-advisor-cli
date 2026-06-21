package service

import (
	"fmt"
	"math"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/fetcher"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

func TestBuildMomentumBacktestUniverseIgnoresCurrentReturns(t *testing.T) {
	funds := []model.MarketSearchFund{{Code: "001", Name: "易方达科技混合A"}, {Code: "002", Name: "华夏成长混合A"}, {Code: "003", Name: "富国创新混合A"}}
	first := []fetcher.MarketRankEntry{{Code: "001", Return1Y: 10}, {Code: "002", Return1Y: -1}, {Code: "003", Return1Y: 2}}
	second := []fetcher.MarketRankEntry{{Code: "001", Return1Y: -10}, {Code: "002", Return1Y: 20}, {Code: "003", Return1Y: -2}}

	firstUniverse := buildMomentumBacktestUniverse(funds, first, 2, 0)
	secondUniverse := buildMomentumBacktestUniverse(funds, second, 2, 0)

	if !reflect.DeepEqual(firstUniverse, secondUniverse) {
		t.Fatalf("backtest universe changed with current returns: %+v != %+v", firstUniverse, secondUniverse)
	}
}

func TestBuildMomentumBacktestUniverseChangesWithSeed(t *testing.T) {
	funds := []model.MarketSearchFund{{Code: "001", Name: "易方达科技混合A"}, {Code: "002", Name: "华夏成长混合A"}, {Code: "003", Name: "富国创新混合A"}, {Code: "004", Name: "南方科技混合A"}}

	first := buildMomentumBacktestUniverse(funds, nil, 2, 0)
	second := buildMomentumBacktestUniverse(funds, nil, 2, 1)

	if reflect.DeepEqual(first, second) {
		t.Fatalf("expected universe sample to change with seed: %+v", first)
	}
}

func TestBuildFormalMomentumBacktestUniverseUsesOnlyAllowedCompanies(t *testing.T) {
	funds := []model.MarketSearchFund{
		{Code: "001", Name: "易方达科技混合A", FundType: "混合型"},
		{Code: "002", Name: "大成科技创新混合A", FundType: "混合型"},
		{Code: "003", Name: "交银科技创新混合A", FundType: "混合型"},
	}
	cfg := config.Default().MomentumPool
	cfg.AllowedCompanies = []string{"易方达", "大成"}

	universe := buildFormalMomentumBacktestUniverse(funds, nil, cfg, 3, 0)

	if len(universe) != 2 || universe[0].Fund.Code == "003" || universe[1].Fund.Code == "003" {
		t.Fatalf("unexpected formal universe: %+v", universe)
	}
}

func TestMinimumMomentumBacktestHistoryRejectsShortRecentFunds(t *testing.T) {
	if got := minimumMomentumBacktestHistory(1200); got != 900 {
		t.Fatalf("minimumMomentumBacktestHistory(1200) = %d, want 900", got)
	}
	if got := minimumMomentumBacktestHistory(360); got != 300 {
		t.Fatalf("minimumMomentumBacktestHistory(360) = %d, want 300", got)
	}
}

func TestTopMomentumCandidatesUsesCurrentRankOnly(t *testing.T) {
	candidates := []momentumBacktestCandidate{
		{Fund: model.MarketSearchFund{Code: "new-1"}, Score: 100},
		{Fund: model.MarketSearchFund{Code: "new-2"}, Score: 99},
		{Fund: model.MarketSearchFund{Code: "old"}, Score: 98},
	}

	selected := topMomentumCandidates(candidates, 2)

	if len(selected) != 2 || selected[0].Fund.Code != "new-1" || selected[1].Fund.Code != "new-2" {
		t.Fatalf("topMomentumCandidates() = %+v", selected)
	}
}

func TestRetainDiversifiedMomentumCandidatesSkipsHighlyCorrelatedFund(t *testing.T) {
	leaderHistory := varyingMomentumHistory("leader", 100)
	duplicateHistory := append([]model.FundSnapshot(nil), leaderHistory...)
	for index := range duplicateHistory {
		duplicateHistory[index].FundCode = "duplicate"
	}
	alternativeHistory := momentumHistory("alternative", 100, 0.002)
	candidates := []momentumBacktestCandidate{
		{Fund: model.MarketSearchFund{Code: "leader"}, Score: 100, History: leaderHistory},
		{Fund: model.MarketSearchFund{Code: "duplicate"}, Score: 99, History: duplicateHistory},
		{Fund: model.MarketSearchFund{Code: "alternative"}, Score: 98, History: alternativeHistory},
	}

	selected := retainDiversifiedMomentumCandidates(candidates, 2, 0.95, leaderHistory[len(leaderHistory)-1].TradeDate)

	if len(selected) != 2 || selected[0].Fund.Code != "leader" || selected[1].Fund.Code != "alternative" {
		t.Fatalf("selected codes = %v", momentumCandidateCodes(selected))
	}
}

func TestRetainDiversifiedMomentumCandidatesDoesNotUseFutureReturns(t *testing.T) {
	leaderHistory := varyingMomentumHistory("leader", 100)
	duplicateHistory := append([]model.FundSnapshot(nil), leaderHistory...)
	for index := range duplicateHistory {
		duplicateHistory[index].FundCode = "duplicate"
		if index >= 80 {
			duplicateHistory[index].NAV *= 1 + float64(index-79)*0.10
		}
	}
	candidates := []momentumBacktestCandidate{
		{Fund: model.MarketSearchFund{Code: "leader"}, Score: 100, History: leaderHistory},
		{Fund: model.MarketSearchFund{Code: "duplicate"}, Score: 99, History: duplicateHistory},
	}

	selected := retainDiversifiedMomentumCandidates(candidates, 2, 0.95, leaderHistory[79].TradeDate)

	if len(selected) != 1 || selected[0].Fund.Code != "leader" {
		t.Fatalf("future returns affected correlation selection: %v", momentumCandidateCodes(selected))
	}
}

func varyingMomentumHistory(code string, days int) []model.FundSnapshot {
	history := momentumHistory(code, days, 0)
	nav := 1.0
	for index := range history {
		if index > 0 {
			nav *= 1 + 0.005 + 0.003*math.Sin(float64(index))
		}
		history[index].NAV = nav
	}
	return history
}

func TestRunMomentumBacktestProducesPositiveReturnForPersistentLeader(t *testing.T) {
	funds := []momentumBacktestFund{
		{Fund: model.MarketSearchFund{Code: "leader", Name: "Leader"}, History: momentumHistory("leader", 400, 0.003)},
		{Fund: model.MarketSearchFund{Code: "weak", Name: "Weak"}, History: momentumHistory("weak", 400, 0.0002)},
	}
	cfg := config.Default().MomentumPool
	cfg.MinMomentumScore = 0
	cfg.MinPositiveBreadth = 0
	cfg.MinSelectionCount = 1
	cfg.MaxDrawdown120D = 1

	report := runMomentumBacktest(funds, cfg, 20, 1)

	if report.Summary.PeriodCount == 0 || report.Summary.TotalReturn <= 0 || report.Summary.WinRate != 1 {
		t.Fatalf("unexpected momentum backtest: %+v", report.Summary)
	}
	for _, period := range report.Periods {
		if len(period.Funds) != 1 || period.Funds[0] != "Leader" {
			t.Fatalf("unexpected selected funds: %+v", period.Funds)
		}
	}
}

func TestRunMomentumBacktestEvaluatesVisibleWinnersWhileExecutionStaysInCash(t *testing.T) {
	funds := []momentumBacktestFund{
		{Fund: model.MarketSearchFund{Code: "leader", Name: "Leader"}, History: momentumHistory("leader", 400, 0.003)},
		{Fund: model.MarketSearchFund{Code: "weak", Name: "Weak"}, History: momentumHistory("weak", 400, -0.0002)},
	}
	cfg := config.Default().MomentumPool
	cfg.MinPositiveBreadth = 1
	cfg.MinMomentumScore = 100
	cfg.MinSelectionCount = 1

	report := runMomentumBacktest(funds, cfg, 20, 1)

	if report.Summary.InvestedPeriodCount != 0 || report.Summary.CashPeriodCount == 0 || report.Summary.WinnerEvaluationPeriods == 0 || report.Summary.AverageFuturePercentile <= 0.5 {
		t.Fatalf("visible ranking should still be evaluated while execution stays in cash: %+v", report.Summary)
	}
}

func TestMomentumWeightProfilesDeduplicatesFormalWeights(t *testing.T) {
	profiles := momentumWeightProfiles(config.Default().MomentumPool)

	formalCount := 0
	for _, profile := range profiles {
		if almostEqual(profile.Weight20D, 0.15) && almostEqual(profile.Weight60D, 0.25) && almostEqual(profile.Weight120D, 0.40) && almostEqual(profile.Weight250D, 0.20) {
			formalCount++
		}
	}
	if formalCount != 1 {
		t.Fatalf("expected one formal weight profile, got %d: %+v", formalCount, profiles)
	}
}

func TestHistoryThroughDoesNotUseFutureSnapshots(t *testing.T) {
	history := momentumHistory("fund", 10, 0.01)
	cutoff := history[4].TradeDate

	selected := historyThrough(history, cutoff)

	if len(selected) != 5 || selected[len(selected)-1].TradeDate.After(cutoff) {
		t.Fatalf("historyThrough() used future data: %+v", selected)
	}
}

func TestMomentumPeriodReturnHoldsUntilRebalance(t *testing.T) {
	history := momentumHistory("trend", 30, 0)
	for index := range history {
		if index <= 10 {
			history[index].NAV = 1 + float64(index)*0.05
		} else {
			history[index].NAV = 1.5 - float64(index-10)*0.03
		}
	}
	selected := []momentumBacktestCandidate{{Fund: model.MarketSearchFund{Name: "Trend"}, History: history}}

	result, names := momentumPeriodReturn(selected, history[0].TradeDate, history[29].TradeDate)

	if len(names) != 1 || result >= 0 {
		t.Fatalf("expected full-period loss without early stop: %.4f names=%v", result, names)
	}
}

func TestMomentumFutureWinnerMetricsMeasuresNextPeriodWinners(t *testing.T) {
	funds := []momentumBacktestFund{
		{Fund: model.MarketSearchFund{Code: "winner"}, History: momentumHistory("winner", 30, 0.02)},
		{Fund: model.MarketSearchFund{Code: "middle"}, History: momentumHistory("middle", 30, 0.01)},
		{Fund: model.MarketSearchFund{Code: "weak"}, History: momentumHistory("weak", 30, 0)},
	}
	selected := []momentumBacktestCandidate{{Fund: funds[0].Fund, History: funds[0].History}}

	topDecileHitRate, topDecileCapture, topCaptureRate, futurePercentile := momentumFutureWinnerMetrics(funds, selected, funds[0].History[9].TradeDate, funds[0].History[19].TradeDate)

	if topDecileHitRate != 1 || topDecileCapture != 1 || topCaptureRate != 1 || futurePercentile != 1 {
		t.Fatalf("winner metrics = %.2f, %.2f, %.2f, %.2f", topDecileHitRate, topDecileCapture, topCaptureRate, futurePercentile)
	}
}

func TestMomentumFutureWinnerMetricsPenalizesMissedWinner(t *testing.T) {
	funds := []momentumBacktestFund{
		{Fund: model.MarketSearchFund{Code: "winner"}, History: momentumHistory("winner", 30, 0.02)},
		{Fund: model.MarketSearchFund{Code: "middle"}, History: momentumHistory("middle", 30, 0.01)},
		{Fund: model.MarketSearchFund{Code: "weak"}, History: momentumHistory("weak", 30, 0)},
	}
	selected := []momentumBacktestCandidate{{Fund: funds[2].Fund, History: funds[2].History}}

	topDecileHitRate, topDecileCapture, topCaptureRate, futurePercentile := momentumFutureWinnerMetrics(funds, selected, funds[0].History[9].TradeDate, funds[0].History[19].TradeDate)

	if topDecileHitRate != 0 || topDecileCapture != 0 || topCaptureRate != 0 || math.Abs(futurePercentile-1.0/3.0) > 0.000001 {
		t.Fatalf("missed-winner metrics = %.2f, %.2f, %.2f, %.2f", topDecileHitRate, topDecileCapture, topCaptureRate, futurePercentile)
	}
}

func TestMomentumFutureWinnerMetricsSeparatesPrecisionAndCoverage(t *testing.T) {
	funds := make([]momentumBacktestFund, 20)
	for index := range funds {
		code := fmt.Sprintf("%02d", index)
		funds[index] = momentumBacktestFund{Fund: model.MarketSearchFund{Code: code}, History: momentumHistory(code, 30, float64(20-index)/1000)}
	}
	selected := []momentumBacktestCandidate{{Fund: funds[0].Fund}, {Fund: funds[5].Fund}}

	hitRate, captureRate, _, _ := momentumFutureWinnerMetrics(funds, selected, funds[0].History[9].TradeDate, funds[0].History[19].TradeDate)

	if hitRate != 0.5 || captureRate != 0.5 {
		t.Fatalf("precision/capture = %.2f/%.2f, want 0.5/0.5", hitRate, captureRate)
	}
}

func TestMomentumPeriodReturnUsesAccumulatedNAVAcrossDistribution(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	history := []model.FundSnapshot{
		{TradeDate: start, NAV: 2, AccNAV: 2},
		{TradeDate: start.AddDate(0, 0, 1), NAV: 1.1, AccNAV: 2.1},
	}
	selected := []momentumBacktestCandidate{{Fund: model.MarketSearchFund{Name: "Dividend Fund"}, History: history}}

	result, _ := momentumPeriodReturn(selected, history[0].TradeDate, history[1].TradeDate)

	if math.Abs(result-0.05) > 0.000001 {
		t.Fatalf("momentumPeriodReturn() = %.4f, want 0.0500", result)
	}
}

func TestFinalizeMomentumBacktestWinRateExcludesCashPeriods(t *testing.T) {
	report := &model.MomentumBacktestReport{Periods: []model.MomentumBacktestPeriod{
		{StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), Return: 0.10, Funds: []string{"Winner"}},
		{StartDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)},
		{StartDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC), Return: -0.05, Funds: []string{"Loser"}},
	}}

	finalizeMomentumBacktest(report, 1.045, 1.0)

	if report.Summary.InvestedPeriodCount != 2 || report.Summary.CashPeriodCount != 1 || report.Summary.WinRate != 0.5 {
		t.Fatalf("unexpected period counts or win rate: %+v", report.Summary)
	}
}

func TestFinalizeMomentumBacktestCalculatesBenchmarkAndSplitExcess(t *testing.T) {
	report := &model.MomentumBacktestReport{Periods: []model.MomentumBacktestPeriod{
		{StartDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), Return: 0.10, BenchmarkReturn: 0.02, ExcessReturn: 0.08, Funds: []string{"Winner"}},
		{StartDate: time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), Return: 0, BenchmarkReturn: -0.03, ExcessReturn: 0.03},
	}}

	finalizeMomentumBacktest(report, 1.10, 0.9894)

	if report.Summary.ExcessReturn <= 0 || report.Summary.ExcessWinRate != 1 || report.Summary.EarlyExcessReturn <= 0 || report.Summary.RecentExcessReturn <= 0 {
		t.Fatalf("unexpected benchmark comparison: %+v", report.Summary)
	}
}

func TestFinalizeMomentumBacktestSensitivityUsesWorstCompletedSamples(t *testing.T) {
	report := &model.MomentumBacktestSensitivityReport{
		Summary: model.MomentumBacktestSensitivitySummary{RequestedSeeds: 3},
		Items: []model.MomentumBacktestSensitivityItem{
			{UniverseSeed: 0, TotalReturn: 0.40, ExcessReturn: 0.20, RecentExcessReturn: 0.10, MaxDrawdown: 0.12},
			{UniverseSeed: 1, TotalReturn: 0.25, ExcessReturn: 0.05, RecentExcessReturn: -0.08, MaxDrawdown: 0.18},
			{UniverseSeed: 2, Error: "failed"},
		},
	}

	finalizeMomentumBacktestSensitivity(report)

	if report.Summary.CompletedSeeds != 2 || report.Summary.FailedSeeds != 1 {
		t.Fatalf("unexpected sample counts: %+v", report.Summary)
	}
	if report.Summary.WorstTotalReturn != 0.25 || report.Summary.WorstExcessReturn != 0.05 || report.Summary.WorstRecentExcessReturn != -0.08 || report.Summary.WorstMaxDrawdown != 0.18 {
		t.Fatalf("unexpected worst results: %+v", report.Summary)
	}
	if report.Summary.PositiveRecentExcessSeeds != 1 {
		t.Fatalf("positive recent excess seeds = %d, want 1", report.Summary.PositiveRecentExcessSeeds)
	}
}

func TestFinalizeMomentumBacktestParameterGridPrioritizesRecentRobustness(t *testing.T) {
	report := &model.MomentumBacktestParameterGridReport{
		Summary: model.MomentumBacktestParameterGridSummary{CurrentSelectionCount: 6, CurrentRebalanceEvery: 30},
		Items: []model.MomentumBacktestParameterGridItem{
			{SelectionCount: 6, RebalanceEvery: 30, CompletedSeeds: 5, PositiveRecentExcessSeeds: 3, WorstRecentExcessReturn: -0.12, WorstExcessReturn: 0.11, WorstTotalReturn: 0.39, WorstMaxDrawdown: 0.16},
			{SelectionCount: 8, RebalanceEvery: 30, CompletedSeeds: 5, PositiveRecentExcessSeeds: 4, WorstRecentExcessReturn: -0.04, WorstExcessReturn: 0.08, WorstTotalReturn: 0.35, WorstMaxDrawdown: 0.18},
			{SelectionCount: 4, RebalanceEvery: 20, Error: "failed"},
		},
	}

	finalizeMomentumBacktestParameterGrid(report)

	if report.Summary.CombinationCount != 2 || report.Summary.CurrentRank != 2 {
		t.Fatalf("unexpected grid summary: %+v", report.Summary)
	}
	if report.Items[0].SelectionCount != 8 || report.Items[0].Rank != 1 || report.Items[2].Rank != 0 {
		t.Fatalf("unexpected grid order: %+v", report.Items)
	}
}

func TestNormalizedParameterValuesUsesSortedUniquePositiveValues(t *testing.T) {
	got := normalizedParameterValues([]int{30, 10, 30, 0, -1}, nil, 1)
	want := []int{10, 30}
	if !slices.Equal(got, want) {
		t.Fatalf("normalizedParameterValues() = %v, want %v", got, want)
	}
}

func TestNormalizedParameterValuesAllowsSmallFocusCounts(t *testing.T) {
	got := normalizedParameterValues([]int{2, 3, 4}, nil, 1)
	if !slices.Equal(got, []int{2, 3, 4}) {
		t.Fatalf("normalizedParameterValues() = %v", got)
	}
}

func TestMomentumBacktestStagesByYearCompoundsPeriods(t *testing.T) {
	periods := []model.MomentumBacktestPeriod{
		{StartDate: time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), Return: 0.10, BenchmarkReturn: 0.02},
		{StartDate: time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC), Return: -0.05, BenchmarkReturn: 0.01},
		{StartDate: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC), EndDate: time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC), Return: 0.02, BenchmarkReturn: 0.01},
	}

	stages := momentumBacktestStagesByYear(periods)

	if math.Abs(stages["2024"].Return-0.045) > 0.000001 || !stages["2024"].Complete {
		t.Fatalf("unexpected 2024 stage: %+v", stages["2024"])
	}
	if stages["2025"].Complete {
		t.Fatalf("expected partial 2025 stage: %+v", stages["2025"])
	}
}

func TestFinalizeMomentumBacktestStagesCountsOnlyCompleteYears(t *testing.T) {
	report := &model.MomentumBacktestStageReport{Items: []model.MomentumBacktestStageItem{
		{StageLabel: "2024", Complete: true, SeedCount: 5, PositiveExcessSeeds: 5, AverageExcessReturn: 0.10, WorstExcessReturn: 0.02},
		{StageLabel: "2025", Complete: true, SeedCount: 5, PositiveExcessSeeds: 3, AverageExcessReturn: -0.01, WorstExcessReturn: -0.05},
		{StageLabel: "2026", Complete: false, SeedCount: 5, PositiveExcessSeeds: 0, AverageExcessReturn: -0.20, WorstExcessReturn: -0.30},
	}}

	finalizeMomentumBacktestStages(report)

	if report.Summary.CompleteStageCount != 2 || report.Summary.PositiveAverageExcessStages != 1 || report.Summary.PositiveAllSeedExcessStages != 1 || report.Summary.WorstCompleteStage != "2025" {
		t.Fatalf("unexpected stage summary: %+v", report.Summary)
	}
}

func momentumHistory(code string, days int, dailyReturn float64) []model.FundSnapshot {
	history := make([]model.FundSnapshot, 0, days)
	nav := 1.0
	start := time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC)
	for index := 0; index < days; index++ {
		if index > 0 {
			nav *= 1 + dailyReturn
		}
		history = append(history, model.FundSnapshot{FundCode: code, TradeDate: start.AddDate(0, 0, index), NAV: nav})
	}
	return history
}
