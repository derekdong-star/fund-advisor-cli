package service

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/fetcher"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
	"github.com/derekdong-star/fund-advisor-cli/internal/store"
)

func TestMomentumPercentileScoresRanksPersistentLeaderFirst(t *testing.T) {
	metrics := []momentumMetric{
		{Code: "leader", Return1M: 0.10, Return3M: 0.30, Return6M: 0.50, Return1Y: 0.80},
		{Code: "middle", Return1M: 0.05, Return3M: 0.15, Return6M: 0.20, Return1Y: 0.30},
		{Code: "weak", Return1M: -0.02, Return3M: 0.01, Return6M: 0.03, Return1Y: 0.05},
	}

	scores := momentumPercentileScores(metrics, config.Default().MomentumPool)

	if scores["leader"] <= scores["middle"] || scores["middle"] <= scores["weak"] {
		t.Fatalf("momentum scores not ordered: %+v", scores)
	}
}

func TestMomentumStrategyFingerprintChangesWithSelectionRules(t *testing.T) {
	first := config.Default().MomentumPool
	second := first
	second.MinReturn20D = first.MinReturn20D - 0.01

	if momentumStrategyFingerprint(first) == momentumStrategyFingerprint(second) {
		t.Fatal("momentum strategy fingerprint should change with selection rules")
	}
}

func TestMomentumStrategyFingerprintChangesWithValidationRules(t *testing.T) {
	first := config.Default().MomentumPool
	second := first
	second.MinForwardAverageReturn = first.MinForwardAverageReturn + 0.01

	if momentumStrategyFingerprint(first) == momentumStrategyFingerprint(second) {
		t.Fatal("momentum strategy fingerprint should change with validation rules")
	}
}

func TestDedupeMomentumCandidatesKeepsBestShareClass(t *testing.T) {
	candidates := []momentumRankCandidate{
		{Fund: model.MarketSearchFund{Code: "A", Name: "示例成长混合A"}, Score: 95},
		{Fund: model.MarketSearchFund{Code: "C", Name: "示例成长混合C"}, Score: 94},
		{Fund: model.MarketSearchFund{Code: "X", Name: "另一只基金A"}, Score: 90},
	}

	result := dedupeMomentumCandidates(candidates, 10)

	if len(result) != 2 || result[0].Fund.Code != "A" || result[1].Fund.Code != "X" {
		t.Fatalf("dedupeMomentumCandidates() = %+v", result)
	}
}

func TestBuildMomentumPoolItemAcceptsHighReturnFund(t *testing.T) {
	profile := momentumProfile("hot", 1, 1.10, 300)
	for index := len(profile.History) - 21; index < len(profile.History); index++ {
		profile.History[index].NAV = 1.10 + 0.35*float64(index-(len(profile.History)-21))/20
	}
	latest := profile.History[len(profile.History)-1]
	profile.Latest = &latest
	cfg := config.Default().MomentumPool

	item, qualified := buildMomentumPoolItem(cfg, profile, 95)

	if !qualified || item.Return20D <= 0.20 {
		t.Fatalf("high-return fund should remain eligible: qualified=%v item=%+v", qualified, item)
	}
}

func TestBuildMomentumPoolItemAcceptsPersistentTrend(t *testing.T) {
	profile := momentumProfile("leader", 1, 1.22, 300)
	cfg := config.Default().MomentumPool

	item, qualified := buildMomentumPoolItem(cfg, profile, 95)

	if !qualified {
		t.Fatalf("qualified=%v", qualified)
	}
	if item.Return20D <= 0 || item.Return60D <= 0 || item.Return120D <= 0 {
		t.Fatalf("expected positive trend metrics: %+v", item)
	}
}

func TestRankMomentumCandidatesUsesFullRankingOrderWithoutEarlyShareClassDedupe(t *testing.T) {
	funds := []model.MarketSearchFund{
		{Code: "A", Name: "成长基金A", FundType: "混合型"},
		{Code: "C", Name: "成长基金C", FundType: "混合型"},
		{Code: "B", Name: "价值基金A", FundType: "股票型"},
	}
	rankings := []fetcher.MarketRankEntry{
		{Code: "A", Return1M: 0.2, Return3M: 0.4, Return6M: 0.6, Return1Y: 1.0},
		{Code: "C", Return1M: 0.19, Return3M: 0.39, Return6M: 0.59, Return1Y: 0.99},
		{Code: "B", Return1M: 0.1, Return3M: 0.2, Return6M: 0.3, Return1Y: 0.5},
	}

	cfg := config.Default().MomentumPool
	cfg.CandidateLimit = 1
	cfg.MinMomentumScore = 0
	candidates := rankMomentumCandidates(funds, rankings, cfg)

	if len(candidates) != 3 || candidates[0].Fund.Code != "A" || candidates[1].Fund.Code != "C" || candidates[2].Fund.Code != "B" {
		t.Fatalf("rankMomentumCandidates() = %+v", candidates)
	}
}

func TestRankMomentumCandidatesKeepsBestAvailableWithoutHardEntryThresholds(t *testing.T) {
	funds := []model.MarketSearchFund{{Code: "best", Name: "易方达相对最强基金A", FundType: "混合型"}, {Code: "weak", Name: "华夏较弱基金A", FundType: "混合型"}}
	rankings := []fetcher.MarketRankEntry{{Code: "best", Return1M: -0.01, Return3M: -0.02, Return6M: -0.03, Return1Y: -0.04}, {Code: "weak", Return1M: -0.10, Return3M: -0.20, Return6M: -0.30, Return1Y: -0.40}}
	cfg := config.Default().MomentumPool
	cfg.MinMomentumScore = 100
	cfg.MinReturn20D = 1
	cfg.MinReturn60D = 1
	cfg.MinReturn120D = 1

	result := rankMomentumCandidates(funds, rankings, cfg)

	if len(result) != 2 || result[0].Fund.Code != "best" {
		t.Fatalf("relative ranking should keep the best available funds: %+v", result)
	}
}

func TestBuildMomentumPoolItemKeepsStrongVisibilityDespiteLegacyRiskThresholds(t *testing.T) {
	profile := momentumProfile("visible", 1, 1.5, 300)
	profile.FundSizeYi = 0.1
	profile.EstablishedYears = 0.5
	cfg := config.Default().MomentumPool
	cfg.MinFundSizeYi = 100
	cfg.MinEstablishedYears = 10
	cfg.MinMomentumScore = 100
	cfg.MaxDrawdown120D = 0

	item, qualified := buildMomentumPoolItem(cfg, profile, 50)

	if !qualified || item.FundCode != "visible" {
		t.Fatalf("visibility ranking should not apply legacy execution thresholds: qualified=%v item=%+v", qualified, item)
	}
}

func TestEvaluateMomentumCandidatesContinuesPastProfileFailuresUntilFilled(t *testing.T) {
	cfg := config.Default()
	cfg.MomentumPool.MinMomentumScore = 0
	ranked := []momentumRankCandidate{
		{Fund: model.MarketSearchFund{Code: "failed", Name: "易方达失败基金A"}, Score: 100},
		{Fund: model.MarketSearchFund{Code: "first", Name: "华夏强势基金A"}, Score: 99},
		{Fund: model.MarketSearchFund{Code: "second", Name: "富国强势基金A"}, Score: 98},
		{Fund: model.MarketSearchFund{Code: "unused", Name: "大成强势基金A"}, Score: 97},
	}
	fetched := make([]string, 0, len(ranked))
	service := &Service{config: cfg, marketProfileCache: map[string]*model.MarketFundProfile{}, fetchMarketProfile: func(_ context.Context, fund model.MarketSearchFund, _ time.Time, _ int) (*model.MarketFundProfile, error) {
		fetched = append(fetched, fund.Code)
		if fund.Code == "failed" {
			return nil, errors.New("temporary profile failure")
		}
		profile := momentumProfile(fund.Code, 1, 1.5, 300)
		profile.Fund = fund
		profile.FundSizeYi = 0.1
		profile.EstablishedYears = 0.5
		return profile, nil
	}}

	items, qualified, failures := service.evaluateMomentumCandidates(context.Background(), ranked, 300, 2)

	if len(items) != 2 || qualified != 2 || items[0].FundCode != "first" || items[1].FundCode != "second" || len(failures) != 1 || failures[0] != "failed" {
		t.Fatalf("unexpected evaluated candidates: items=%+v qualified=%d failures=%v", items, qualified, failures)
	}
	if strings.Join(fetched, ",") != "failed,failed,failed,first,second" {
		t.Fatalf("expected scan to stop after filling ranking, fetched=%v", fetched)
	}
}

func TestEvaluateMomentumCandidatesFallsBackToAnotherShareClassAfterFailure(t *testing.T) {
	cfg := config.Default()
	ranked := []momentumRankCandidate{
		{Fund: model.MarketSearchFund{Code: "A", Name: "易方达科技成长混合A"}, Score: 100},
		{Fund: model.MarketSearchFund{Code: "C", Name: "易方达科技成长混合C"}, Score: 99},
		{Fund: model.MarketSearchFund{Code: "other", Name: "华夏数字产业混合A"}, Score: 98},
	}
	service := &Service{config: cfg, marketProfileCache: map[string]*model.MarketFundProfile{}, fetchMarketProfile: func(_ context.Context, fund model.MarketSearchFund, _ time.Time, _ int) (*model.MarketFundProfile, error) {
		if fund.Code == "A" {
			return nil, errors.New("A share unavailable")
		}
		profile := momentumProfile(fund.Code, 1, 1.5, 300)
		profile.Fund = fund
		return profile, nil
	}}

	items, _, failures := service.evaluateMomentumCandidates(context.Background(), ranked, 300, 2)

	if len(items) != 2 || items[0].FundCode != "C" || items[1].FundCode != "other" || !slices.Equal(failures, []string{"A"}) {
		t.Fatalf("expected successful share-class fallback: items=%+v failures=%v", items, failures)
	}
}

func momentumProfile(code string, startNAV, endNAV float64, days int) *model.MarketFundProfile {
	history := make([]model.FundSnapshot, 0, days)
	now := time.Now().UTC()
	for index := 0; index < days; index++ {
		nav := startNAV + (endNAV-startNAV)*float64(index)/float64(days-1)
		history = append(history, model.FundSnapshot{TradeDate: now.AddDate(0, 0, index-days+1), NAV: nav})
	}
	latest := history[len(history)-1]
	return &model.MarketFundProfile{
		Fund:             model.MarketSearchFund{Code: code, Name: code + "基金A", FundType: "股票型"},
		Latest:           &latest,
		History:          history,
		FundSizeYi:       20,
		EstablishedYears: 3,
	}
}

func TestFilterMomentumFundsKeepsOnlyAllowedCompaniesAndAllowsMultipleFromSameCompany(t *testing.T) {
	funds := []model.MarketSearchFund{
		{Name: "易方达人工智能ETF联接A", FundType: "指数型"},
		{Name: "易方达科技创新混合A", FundType: "混合型"},
		{Name: "易方达科技创新一年持有混合A", FundType: "混合型"},
		{Name: "华夏科技定开混合A", FundType: "混合型"},
		{Name: "富国科技封闭运作混合A", FundType: "混合型"},
		{Name: "工银瑞信科技创新混合A", FundType: "混合型"},
		{Name: "兴全合润混合", FundType: "混合型"},
		{Name: "交银科锐科技创新混合A", FundType: "混合型"},
		{Name: "大成科技创新混合A", FundType: "混合型"},
		{Name: "华商均衡成长混合A", FundType: "混合型"},
		{Name: "国泰君安科技创新混合A", FundType: "混合型"},
		{Name: "中银证券科技混合A", FundType: "混合型"},
		{Name: "小公司人工智能混合A", FundType: "混合型"},
	}

	result := filterMomentumFunds(funds, []string{"易方达", "华夏", "富国", "工银瑞信", "兴证全球", "交银施罗德", "大成", "华商", "国泰", "中银"})

	if len(result) != 10 || result[0].Name != "易方达人工智能ETF联接A" || result[2].Name != "易方达科技创新一年持有混合A" || result[3].Name != "华夏科技定开混合A" || result[4].Name != "富国科技封闭运作混合A" || result[9].Name != "华商均衡成长混合A" {
		t.Fatalf("filterMomentumFunds() = %+v", result)
	}
}

func TestBuildMomentumChallengersFillsRemainingRankingSlots(t *testing.T) {
	ranked := momentumPoolItems("A", "B", "C", "D", "E")
	selected := momentumPoolItems("A", "B")

	result := buildMomentumChallengers(ranked, selected, momentumOpportunityCount-len(selected))

	if len(result) != 3 || result[0].FundCode != "C" || result[2].FundCode != "E" {
		t.Fatalf("buildMomentumChallengers() = %+v", result)
	}
}

func TestBuildMomentumChallengersExcludesSelectedFunds(t *testing.T) {
	ranked := []model.MomentumPoolItem{{FundCode: "A", Score: 100}, {FundCode: "B", Score: 99}, {FundCode: "C", Score: 98}, {FundCode: "D", Score: 97}}
	selected := []model.MomentumPoolItem{{FundCode: "A"}, {FundCode: "C"}}

	challengers := buildMomentumChallengers(ranked, selected, 2)

	if len(challengers) != 2 || challengers[0].FundCode != "B" || challengers[0].Rank != 2 || challengers[1].FundCode != "D" || challengers[1].Rank != 4 {
		t.Fatalf("buildMomentumChallengers() = %+v", challengers)
	}
}

func TestBuildMomentumOpportunitiesDoesNotClaimProductLevelBacktestProof(t *testing.T) {
	report := &model.MomentumPoolReport{Items: []model.MomentumPoolItem{{FundCode: "A", FundName: "强势基金A", MomentumScore: 99}}}

	opportunities := buildMomentumOpportunities(report, 0.05)

	if len(opportunities) != 1 || !strings.Contains(opportunities[0].Evidence, "当前强势重点关注") || strings.Contains(opportunities[0].Evidence, "10样本") {
		t.Fatalf("unexpected formal opportunity evidence: %+v", opportunities)
	}
}

func TestBuildMomentumOpportunitiesNeverAllowsTrialOutsideTopThree(t *testing.T) {
	report := &model.MomentumPoolReport{
		Challengers:           []model.MomentumPoolItem{{FundCode: "challenger", FundName: "其他强势基金", MomentumScore: 99}},
		ChallengerValidations: []model.MomentumFundValidation{{FundCode: "challenger", CompletedBatches: 1, LatestReturn: 0.20, Qualified: true, Execution: "允许小额试投"}},
	}

	opportunities := buildMomentumOpportunities(report, 0.05)

	if len(opportunities) != 1 || opportunities[0].Execution != "验证达标，等待进入前3" || opportunities[0].SuggestedWeight != 0 {
		t.Fatalf("non-top-three fund must not receive trial signal: %+v", opportunities)
	}
}

func TestAppendMomentumProfileFailureNoteListsSkippedFunds(t *testing.T) {
	report := &model.MomentumPoolReport{}

	appendMomentumProfileFailureNote(report, "正式候选", []string{"A", "B"})

	if len(report.Summary.Notes) != 1 || !strings.Contains(report.Summary.Notes[0], "2 只") || !strings.Contains(report.Summary.Notes[0], "A、B") {
		t.Fatalf("unexpected profile failure note: %v", report.Summary.Notes)
	}
}

func TestMomentumItemCodeSetsIgnoreOrder(t *testing.T) {
	left := momentumItemCodeSet([]model.MomentumPoolItem{{FundCode: "A"}, {FundCode: "B"}})
	right := momentumItemCodeSet([]model.MomentumPoolItem{{FundCode: "B"}, {FundCode: "A"}})
	different := momentumItemCodeSet([]model.MomentumPoolItem{{FundCode: "A"}, {FundCode: "C"}})

	if !equalStringSets(left, right) || equalStringSets(left, different) {
		t.Fatalf("unexpected code-set equality")
	}
}

func TestContinuousChallengerBaselineUsesStartOfCurrentCombination(t *testing.T) {
	service, reports := momentumBaselineTestService(t, func(codes ...string) model.MomentumPoolReport {
		return model.MomentumPoolReport{Challengers: momentumPoolItems(codes...)}
	})

	baseline := service.continuousChallengerBaseline(&reports[len(reports)-1])

	if !baseline.Summary.RunDate.Equal(reports[2].Summary.RunDate) {
		t.Fatalf("continuousChallengerBaseline() run date = %s, want %s", baseline.Summary.RunDate, reports[2].Summary.RunDate)
	}
}

func momentumBaselineTestService(t *testing.T, buildReport func(...string) model.MomentumPoolReport, combinations ...[]string) (*Service, []model.MomentumPoolReport) {
	t.Helper()
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	t.Cleanup(func() { store.Close() })

	if len(combinations) == 0 {
		combinations = [][]string{{"old-a", "old-b"}, {"old-b", "old-a"}, {"new-a", "new-b"}, {"new-b", "new-a"}}
	}
	reports := make([]model.MomentumPoolReport, 0, len(combinations))
	for _, combination := range combinations {
		reports = append(reports, buildReport(combination...))
	}
	for index := range reports {
		reports[index].Summary.RunDate = time.Date(2026, time.June, 1+index, 0, 0, 0, 0, time.UTC)
		reports[index].Summary.StrategyFingerprint = "formal-strategy"
		if _, err := store.SaveMomentumPool(reports[index]); err != nil {
			t.Fatalf("SaveMomentumPool() error = %v", err)
		}
	}
	return &Service{store: store}, reports
}

func momentumPoolItems(codes ...string) []model.MomentumPoolItem {
	items := make([]model.MomentumPoolItem, 0, len(codes))
	for _, code := range codes {
		items = append(items, model.MomentumPoolItem{FundCode: code})
	}
	return items
}

func TestMomentumStrategyFingerprintIncludesCurrentFormalStrategy(t *testing.T) {
	cfg := config.Default().MomentumPool

	if got := momentumStrategyFingerprint(cfg); got != "0b17c8f77e87d5c4" {
		t.Fatalf("momentumStrategyFingerprint() = %s, want current formal strategy fingerprint", got)
	}
}

func TestApplyUnifiedOpportunityScoresUsesAllowedUniverse(t *testing.T) {
	cfg := config.Default().MomentumPool
	formal := []model.MarketSearchFund{{Code: "formal", Name: "易方达科技混合A", FundType: "混合型"}}
	rankings := []fetcher.MarketRankEntry{{Code: "formal", Return1M: 1, Return3M: 1, Return6M: 1, Return1Y: 1}}
	report := &model.MomentumPoolReport{Items: []model.MomentumPoolItem{{FundCode: "formal"}}}

	applyUnifiedOpportunityScores(report, formal, rankings, cfg)

	if report.Items[0].MomentumScore <= 0 {
		t.Fatalf("expected allowed fund to receive a score: %+v", report)
	}
}

func TestPearsonCorrelationDetectsMatchingAndOppositeReturns(t *testing.T) {
	matching, ok := pearsonCorrelation([]float64{1, 2, 3, 4}, []float64{2, 4, 6, 8})
	if !ok || math.Abs(matching-1) > 0.000001 {
		t.Fatalf("matching correlation = %.4f, ok=%v", matching, ok)
	}
	opposite, ok := pearsonCorrelation([]float64{1, 2, 3, 4}, []float64{8, 6, 4, 2})
	if !ok || math.Abs(opposite+1) > 0.000001 {
		t.Fatalf("opposite correlation = %.4f, ok=%v", opposite, ok)
	}
}

func TestFundReturnCorrelationUsesSharedRecentDates(t *testing.T) {
	left := momentumHistory("left", 80, 0.01)
	right := momentumHistory("right", 80, 0.01)
	for index := range left {
		factor := 1 + float64(index%5)*0.001
		left[index].NAV *= factor
		right[index].NAV *= factor
	}

	correlation, ok := fundReturnCorrelation(left, right, 60)

	if !ok || correlation < 0.999 {
		t.Fatalf("fundReturnCorrelation() = %.4f, ok=%v", correlation, ok)
	}
}

func TestFixedHorizonReturnUsesTradingDays(t *testing.T) {
	history := momentumHistory("fund", 30, 0.01)

	forwardReturn, endDate, ok := fixedHorizonReturn(history, history[2].TradeDate, 20)

	if !ok || forwardReturn <= 0 || !endDate.Equal(history[22].TradeDate) {
		t.Fatalf("fixedHorizonReturn() = %.4f %s %v", forwardReturn, endDate, ok)
	}
}

func TestFixedHorizonReturnUsesAccumulatedNAVAcrossDistribution(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	history := []model.FundSnapshot{
		{TradeDate: start, NAV: 2, AccNAV: 2},
		{TradeDate: start.AddDate(0, 0, 1), NAV: 1.1, AccNAV: 2.1},
	}

	forwardReturn, _, ok := fixedHorizonReturn(history, start, 1)

	if !ok || math.Abs(forwardReturn-0.05) > 0.000001 {
		t.Fatalf("fixedHorizonReturn() = %.4f, want 0.0500", forwardReturn)
	}
}

func TestFixedHorizonReturnRequiresExactRecordedStartDate(t *testing.T) {
	history := momentumHistory("leader", 30, 0.001)
	missingDate := history[5].TradeDate.Add(12 * time.Hour)

	if _, _, ok := fixedHorizonReturn(history, missingDate, 5); ok {
		t.Fatal("fixedHorizonReturn() should reject a missing exact start date")
	}
}

func TestFixedHorizonReturnFromRecordedNAVFreezesSelectionPrice(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	history := []model.FundSnapshot{
		{TradeDate: start, NAV: 2},
		{TradeDate: start.AddDate(0, 0, 1), NAV: 2.2},
	}

	forwardReturn, _, ok := fixedHorizonReturnFromRecordedNAV(history, start, 1, 1)

	if !ok || math.Abs(forwardReturn-1.2) > 0.000001 {
		t.Fatalf("fixedHorizonReturnFromRecordedNAV() = %.4f, want 1.2000", forwardReturn)
	}
}

func TestMomentumBatchStartDateUsesLatestCandidateDate(t *testing.T) {
	earlier := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	later := earlier.AddDate(0, 0, 2)

	start := momentumBatchStartDate([]model.MomentumPoolItem{{LatestTradeDate: earlier}, {LatestTradeDate: later}})

	if !start.Equal(later) {
		t.Fatalf("momentumBatchStartDate() = %s, want %s", start, later)
	}
}

func TestApproximateTradingDays(t *testing.T) {
	start := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	days := approximateTradingDays(start, start.AddDate(0, 0, 14))

	if days != 10 {
		t.Fatalf("approximateTradingDays() = %d, want 10", days)
	}
}

func TestMomentumObservationProgressCountsActualNAVTradingDays(t *testing.T) {
	start := time.Date(2026, time.June, 17, 0, 0, 0, 0, time.UTC)
	history := []model.FundSnapshot{
		{TradeDate: start, NAV: 1, AccNAV: 1},
		{TradeDate: start.AddDate(0, 0, 5), NAV: 1.02, AccNAV: 1.02},
		{TradeDate: start.AddDate(0, 0, 6), NAV: 1.03, AccNAV: 1.03},
	}

	days, observedReturn := momentumObservationProgress(history, history[0].TradeDate, 1, history[2].TradeDate, 1.03)

	if days != 2 || math.Abs(observedReturn-0.03) > 0.000001 {
		t.Fatalf("momentumObservationProgress() = days %d return %.4f", days, observedReturn)
	}
}

func TestMomentumObservationProgressIgnoresSameDayNAVRevision(t *testing.T) {
	tradeDate := time.Date(2026, time.June, 17, 0, 0, 0, 0, time.UTC)
	history := []model.FundSnapshot{{TradeDate: tradeDate, NAV: 1.02, AccNAV: 1.02}}

	days, observedReturn := momentumObservationProgress(history, tradeDate, 1, tradeDate, 1.02)

	if days != 0 || observedReturn != 0 {
		t.Fatalf("same-day revision must not count as observation return: days %d return %.4f", days, observedReturn)
	}
}

func TestExpiredMomentumBatchUsesAvailableHistoryWindow(t *testing.T) {
	now := time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC)

	if !isExpiredMomentumBatch(now.AddDate(-2, 0, 0), now, 300, 20) {
		t.Fatal("expected old batch outside history window to expire")
	}
	if isExpiredMomentumBatch(now.AddDate(0, 0, -20), now, 300, 20) {
		t.Fatal("did not expect recent pending batch to expire")
	}
}

func TestMomentumBatchHasRecordedStarts(t *testing.T) {
	start := time.Now().UTC()
	valid := []model.MomentumPoolItem{{LatestTradeDate: start, LatestNAV: 1}, {LatestTradeDate: start, LatestNAV: 2}}
	missingNAV := []model.MomentumPoolItem{{LatestTradeDate: start}}
	missingDate := []model.MomentumPoolItem{{LatestNAV: 1}}
	unalignedDates := []model.MomentumPoolItem{{LatestTradeDate: start, LatestNAV: 1}, {LatestTradeDate: start.AddDate(0, 0, -1), LatestNAV: 2}}

	if !momentumBatchHasRecordedStarts(valid) || momentumBatchHasRecordedStarts(missingNAV) || momentumBatchHasRecordedStarts(missingDate) || momentumBatchHasRecordedStarts(unalignedDates) {
		t.Fatal("unexpected recorded-start classification")
	}
}

func TestMarkCurrentMomentumBatchPendingStartsFirstValidationBatch(t *testing.T) {
	start := time.Now().UTC().AddDate(0, 0, -2)
	report := &model.MomentumPoolReport{Items: []model.MomentumPoolItem{
		{FundCode: "A", LatestTradeDate: start, LatestNAV: 1},
		{FundCode: "B", LatestTradeDate: start, LatestNAV: 2},
	}}

	markCurrentMomentumBatchPending(report)

	if report.Summary.PendingValidationBatches != 1 || !report.Summary.OldestPendingStartDate.Equal(start) || report.Summary.OldestPendingTradingDays <= 0 {
		t.Fatalf("unexpected pending validation summary: %+v", report.Summary)
	}
}

func TestMarkCurrentMomentumBatchPendingKeepsOlderPendingBatch(t *testing.T) {
	older := time.Now().UTC().AddDate(0, 0, -10)
	current := time.Now().UTC().AddDate(0, 0, -2)
	report := &model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{PendingValidationBatches: 1, OldestPendingStartDate: older},
		Items:   []model.MomentumPoolItem{{FundCode: "A", LatestTradeDate: current, LatestNAV: 1}},
	}

	markCurrentMomentumBatchPending(report)

	if !report.Summary.OldestPendingStartDate.Equal(older) {
		t.Fatalf("older pending batch was replaced: %+v", report.Summary)
	}
}

func TestQualifiedMomentumFundCodesRequiresCompletedProfitableCycle(t *testing.T) {
	cfg := config.Default().MomentumPool
	now := time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC)
	validations := map[string]momentumFundValidation{
		"qualified":          {Count: 1, Wins: 1, TotalReturn: 0.12, LatestReturn: 0.12, LatestEnd: now.AddDate(0, 0, -10)},
		"too-few":            {Count: 0, Wins: 0, TotalReturn: 0, LatestEnd: now.AddDate(0, 0, -10)},
		"low-return":         {Count: 1, Wins: 1, TotalReturn: 0.005, LatestReturn: 0.005, LatestEnd: now.AddDate(0, 0, -10)},
		"profitable-average": {Count: 2, Wins: 1, TotalReturn: 0.24, LatestReturn: 0.12, LatestEnd: now.AddDate(0, 0, -10)},
		"recently-weak":      {Count: 2, Wins: 1, TotalReturn: 0.30, LatestReturn: -0.02, LatestEnd: now.AddDate(0, 0, -10)},
		"stale":              {Count: 1, Wins: 1, TotalReturn: 0.12, LatestReturn: 0.12, LatestEnd: now.AddDate(0, 0, -60)},
	}

	qualified := qualifiedMomentumFundCodes(validations, cfg, now)

	if !slices.Equal(qualified, []string{"profitable-average", "qualified"}) {
		t.Fatalf("qualifiedMomentumFundCodes() = %v", qualified)
	}
}

func TestQualifiedMomentumFundCodesRejectsStrongHistoryWhenLatestCycleWeakens(t *testing.T) {
	cfg := config.Default().MomentumPool
	now := time.Now().UTC()
	validations := map[string]momentumFundValidation{
		"faded": {Count: 3, Wins: 2, TotalReturn: 0.50, LatestReturn: -0.02, LatestEnd: now.AddDate(0, 0, -1)},
	}

	qualified := qualifiedMomentumFundCodes(validations, cfg, now)

	if len(qualified) != 0 {
		t.Fatalf("latest weak cycle must block trial despite strong history: %v", qualified)
	}
}

func TestForwardMomentumValidationSkipsUnverifiableBatchWithoutBlocking(t *testing.T) {
	cfg := config.Default()
	strategyFingerprint := momentumStrategyFingerprint(cfg.MomentumPool)
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()
	history := momentumHistory("leader", 100, 0.001)
	reports := []model.MomentumPoolReport{
		{Summary: model.MomentumPoolSummary{StrategyFingerprint: strategyFingerprint}, Items: []model.MomentumPoolItem{{FundCode: "leader", LatestTradeDate: history[0].TradeDate}}},
		{Summary: model.MomentumPoolSummary{StrategyFingerprint: strategyFingerprint}, Items: []model.MomentumPoolItem{{FundCode: "leader", LatestTradeDate: history[31].TradeDate, LatestNAV: returnAdjustedNAV(history[31])}}},
	}
	for _, report := range reports {
		if _, err := store.SaveMomentumPool(report); err != nil {
			t.Fatalf("SaveMomentumPool() error = %v", err)
		}
	}
	service := &Service{config: cfg, store: store, marketProfileCache: map[string]*model.MarketFundProfile{"leader:300": {History: history}}}
	current := &model.MomentumPoolReport{Summary: model.MomentumPoolSummary{StrategyFingerprint: strategyFingerprint}}

	service.evaluateForwardMomentumValidation(context.Background(), 300, current)

	if current.Summary.UnverifiableValidationBatches != 1 || current.Summary.ForwardValidationBatches != 1 || current.Summary.PendingValidationBatches != 0 {
		t.Fatalf("unexpected validation summary: %+v", current.Summary)
	}
}

func TestApplyMomentumReadinessRequiresForwardValidation(t *testing.T) {
	cfg := config.Default()
	service := &Service{config: cfg}
	report := &model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{Regime: "risk-on"},
		Items:   []model.MomentumPoolItem{{FundName: "A"}, {FundName: "B"}},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "watch-only" || report.Summary.SuggestedTotalWeight != 0 || report.Items[0].SuggestedWeight != 0 {
		t.Fatalf("unexpected unvalidated readiness: %+v %+v", report.Summary, report.Items)
	}
}

func TestApplyMomentumReadinessCapsTrialWeight(t *testing.T) {
	cfg := config.Default()
	cfg.MomentumPool.MinSelectionCount = 2
	service := &Service{config: cfg}
	recent := time.Now().UTC().AddDate(0, 0, -1)
	report := &model.MomentumPoolReport{
		Summary:            model.MomentumPoolSummary{Regime: "risk-on", ForwardValidationBatches: 3, ForwardAverageReturn: 0.02, ForwardWinRate: 0.60, ForwardBatchWinRate: 1},
		Items:              []model.MomentumPoolItem{{FundCode: "A", FundName: "A", MomentumScore: 99, Return20D: 0.10, Return60D: 0.20, Return120D: 0.30, LatestTradeDate: recent, PurchaseStatus: "开放申购"}, {FundCode: "B", FundName: "B", MomentumScore: 98, Return20D: 0.08, Return60D: 0.18, Return120D: 0.28, LatestTradeDate: recent, PurchaseStatus: "限大额"}},
		ValidatedFundCodes: []string{"A", "B"},
		QualifiedFundCodes: []string{"A", "B"},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "small-trial" || report.Summary.SuggestedTotalWeight != 0.05 || report.Items[0].SuggestedWeight != 0.025 || report.Items[1].SuggestedWeight != 0.025 {
		t.Fatalf("unexpected validated readiness: %+v %+v", report.Summary, report.Items)
	}
}

func TestApplyMomentumReadinessKeepsVisibleButRejectsWeakCurrentTrend(t *testing.T) {
	cfg := config.Default()
	cfg.MomentumPool.MinSelectionCount = 1
	service := &Service{config: cfg}
	recent := time.Now().UTC().AddDate(0, 0, -1)
	report := &model.MomentumPoolReport{
		Summary:            model.MomentumPoolSummary{Regime: "risk-on", ForwardValidationBatches: 3, ForwardAverageReturn: 0.02, ForwardWinRate: 0.60, ForwardBatchWinRate: 1},
		Items:              []model.MomentumPoolItem{{FundCode: "A", FundName: "当前转弱但仍相对领先", MomentumScore: 99, Return20D: -0.20, Return60D: -0.10, Return120D: -0.05, LatestTradeDate: recent, PurchaseStatus: "开放申购"}},
		ValidatedFundCodes: []string{"A"},
		QualifiedFundCodes: []string{"A"},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "watch-only" || len(report.Items) != 1 || report.Summary.SuggestedTotalWeight != 0 {
		t.Fatalf("weak current trend should stay visible without trial: %+v %+v", report.Summary, report.Items)
	}
}

func TestApplyMomentumReadinessRejectsStaleMarketData(t *testing.T) {
	cfg := config.Default()
	service := &Service{config: cfg}
	report := &model.MomentumPoolReport{
		Summary:            model.MomentumPoolSummary{Regime: "risk-on", ForwardValidationBatches: 3, ForwardAverageReturn: 0.02, ForwardWinRate: 0.60, ForwardBatchWinRate: 1},
		Items:              []model.MomentumPoolItem{{FundCode: "A", FundName: "A", LatestTradeDate: time.Now().UTC().AddDate(0, 0, -8), PurchaseStatus: "开放申购"}},
		ValidatedFundCodes: []string{"A"},
		QualifiedFundCodes: []string{"A"},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "watch-only" || report.Summary.SuggestedTotalWeight != 0 || report.Items[0].SuggestedWeight != 0 {
		t.Fatalf("unexpected stale-data readiness: %+v %+v", report.Summary, report.Items)
	}
}

func TestApplyMomentumReadinessRejectsUnknownPurchaseStatus(t *testing.T) {
	cfg := config.Default()
	service := &Service{config: cfg}
	report := &model.MomentumPoolReport{
		Summary:            model.MomentumPoolSummary{Regime: "risk-on", ForwardValidationBatches: 3, ForwardAverageReturn: 0.02, ForwardWinRate: 0.60, ForwardBatchWinRate: 1},
		Items:              []model.MomentumPoolItem{{FundCode: "A", FundName: "A", LatestTradeDate: time.Now().UTC()}},
		ValidatedFundCodes: []string{"A"},
		QualifiedFundCodes: []string{"A"},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "watch-only" || report.Summary.SuggestedTotalWeight != 0 {
		t.Fatalf("unexpected unknown-purchase readiness: %+v", report.Summary)
	}
}

func TestApplyMomentumReadinessRejectsWeakBatchConsistency(t *testing.T) {
	cfg := config.Default()
	service := &Service{config: cfg}
	report := &model.MomentumPoolReport{
		Summary:            model.MomentumPoolSummary{Regime: "risk-on", ForwardValidationBatches: 3, ForwardAverageReturn: 0.02, ForwardWinRate: 0.60, ForwardBatchWinRate: 0.33},
		Items:              []model.MomentumPoolItem{{FundCode: "A", FundName: "A", LatestTradeDate: time.Now().UTC(), PurchaseStatus: "开放申购"}},
		ValidatedFundCodes: []string{"A"},
		QualifiedFundCodes: []string{"A"},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "watch-only" || report.Summary.SuggestedTotalWeight != 0 {
		t.Fatalf("unexpected weak-batch readiness: %+v", report.Summary)
	}
}

func TestApplyMomentumReadinessRejectsNewUnvalidatedCandidate(t *testing.T) {
	cfg := config.Default()
	service := &Service{config: cfg}
	recent := time.Now().UTC().AddDate(0, 0, -1)
	report := &model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{Regime: "risk-on", ForwardValidationBatches: 3, ForwardAverageReturn: 0.02, ForwardWinRate: 0.60, ForwardBatchWinRate: 1},
		Items: []model.MomentumPoolItem{
			{FundCode: "validated", LatestTradeDate: recent, PurchaseStatus: "开放申购"},
			{FundCode: "new", LatestTradeDate: recent, PurchaseStatus: "开放申购"},
		},
		ValidatedFundCodes: []string{"validated"},
		QualifiedFundCodes: []string{"validated"},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "watch-only" || report.Summary.CurrentValidatedCandidates != 1 || report.Summary.SuggestedTotalWeight != 0 {
		t.Fatalf("new candidate should remain watch-only: %+v", report.Summary)
	}
}

func TestApplyMomentumReadinessAllowsQualifiedFundWithoutWaitingForNewCandidate(t *testing.T) {
	cfg := config.Default()
	cfg.MomentumPool.MinSelectionCount = 2
	service := &Service{config: cfg}
	recent := time.Now().UTC().AddDate(0, 0, -1)
	report := &model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{Regime: "risk-on"},
		Items: []model.MomentumPoolItem{
			{FundCode: "qualified", MomentumScore: 99, Return20D: 0.10, Return60D: 0.20, Return120D: 0.30, LatestTradeDate: recent, PurchaseStatus: "开放申购"},
			{FundCode: "new", MomentumScore: 98, Return20D: 0.08, Return60D: 0.18, Return120D: 0.28, LatestTradeDate: recent, PurchaseStatus: "开放申购"},
		},
		ValidatedFundCodes: []string{"qualified"},
		QualifiedFundCodes: []string{"qualified"},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "small-trial" || report.Summary.SuggestedTotalWeight != 0.05 || report.Items[0].SuggestedWeight != 0.05 || report.Items[1].SuggestedWeight != 0 {
		t.Fatalf("qualified fund should unlock trial without waiting for new candidate: %+v %+v", report.Summary, report.Items)
	}
}

func TestApplyMomentumReadinessRejectsCoveredButIndividuallyWeakCandidate(t *testing.T) {
	cfg := config.Default()
	service := &Service{config: cfg}
	recent := time.Now().UTC().AddDate(0, 0, -1)
	report := &model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{Regime: "risk-on", ForwardValidationBatches: 3, ForwardAverageReturn: 0.02, ForwardWinRate: 0.60, ForwardBatchWinRate: 1},
		Items: []model.MomentumPoolItem{
			{FundCode: "strong", LatestTradeDate: recent, PurchaseStatus: "开放申购"},
			{FundCode: "weak", LatestTradeDate: recent, PurchaseStatus: "开放申购"},
		},
		ValidatedFundCodes: []string{"strong", "weak"},
		QualifiedFundCodes: []string{"strong"},
	}

	service.applyMomentumReadiness(report)

	if report.Summary.Readiness != "watch-only" || report.Summary.CurrentValidatedCandidates != 2 || report.Summary.CurrentQualifiedCandidates != 1 || report.Summary.SuggestedTotalWeight != 0 {
		t.Fatalf("individually weak candidate should remain watch-only: %+v", report.Summary)
	}
}

func TestPurchasableStatus(t *testing.T) {
	if !isPurchasableStatus("开放申购") || !isPurchasableStatus("限大额") || isPurchasableStatus("暂停申购") || isPurchasableStatus("") {
		t.Fatal("unexpected purchase status classification")
	}
}

func TestAnnotateMomentumPurchaseStatusesKeepsUnavailableFundsVisible(t *testing.T) {
	items := []model.MomentumPoolItem{{FundCode: "open"}, {FundCode: "limited"}, {FundCode: "paused"}, {FundCode: "unknown"}}
	statuses := map[string]fetcher.FundPurchaseStatus{
		"open":    {PurchaseStatus: "开放申购", DailyLimit: 100000000000},
		"limited": {PurchaseStatus: "限大额", DailyLimit: 10000},
		"paused":  {PurchaseStatus: "暂停申购"},
	}

	unknown, unavailable := annotateMomentumPurchaseStatuses(items, statuses)

	if len(items) != 4 || unknown != 1 || unavailable != 1 || items[1].DailyLimit != 10000 || items[2].PurchaseStatus != "暂停申购" || items[3].PurchaseStatus != "未知" {
		t.Fatalf("unexpected purchase status annotations: %+v unknown=%d unavailable=%d", items, unknown, unavailable)
	}
}

func TestEvaluatePreviousMomentumPoolIgnoresDifferentStrategyFingerprint(t *testing.T) {
	cfg := config.Default()
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()
	if _, err := store.SaveMomentumPool(model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{StrategyFingerprint: "old-strategy"},
		Items:   []model.MomentumPoolItem{{FundCode: "leader", LatestNAV: 1, LatestTradeDate: time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)}},
	}); err != nil {
		t.Fatalf("SaveMomentumPool() error = %v", err)
	}

	service := &Service{config: cfg, store: store, marketProfileCache: map[string]*model.MarketFundProfile{}}
	report := &model.MomentumPoolReport{Summary: model.MomentumPoolSummary{StrategyFingerprint: "new-strategy"}}

	service.evaluatePreviousMomentumPool(context.Background(), 300, report)

	if report.Summary.PreviousEvaluatedCount != 0 {
		t.Fatalf("unexpected previous evaluation: %+v", report.Summary)
	}
}

func TestForwardMomentumValidationUnlocksSmallTrialAfterThreeStoredBatches(t *testing.T) {
	cfg := config.Default()
	cfg.MomentumPool.MinSelectionCount = 1
	strategyFingerprint := momentumStrategyFingerprint(cfg.MomentumPool)
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()

	history := momentumHistory("leader", 130, 0.006)
	latestTradeDate := time.Now().UTC()
	firstTradeDate := latestTradeDate.AddDate(0, 0, -len(history)+1)
	for historyIndex := range history {
		history[historyIndex].TradeDate = firstTradeDate.AddDate(0, 0, historyIndex)
	}
	for _, historyIndex := range []int{30, 61, 92} {
		report := model.MomentumPoolReport{
			Summary: model.MomentumPoolSummary{RunDate: history[historyIndex].TradeDate, StrategyFingerprint: strategyFingerprint, Regime: "risk-on"},
			Items:   []model.MomentumPoolItem{{FundCode: "leader", FundName: "易方达人工智能基金A", FundType: "混合型", LatestTradeDate: history[historyIndex].TradeDate, LatestNAV: returnAdjustedNAV(history[historyIndex])}},
		}
		if _, err := store.SaveMomentumPool(report); err != nil {
			t.Fatalf("SaveMomentumPool() error = %v", err)
		}
	}

	service := &Service{
		config: cfg,
		store:  store,
		marketProfileCache: map[string]*model.MarketFundProfile{
			"leader:300": {History: history},
		},
	}
	current := &model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{StrategyFingerprint: strategyFingerprint, Regime: "risk-on"},
		Items:   []model.MomentumPoolItem{{FundCode: "leader", FundName: "易方达人工智能基金A", MomentumScore: 99, Return20D: 0.10, Return60D: 0.20, Return120D: 0.30, LatestTradeDate: latestTradeDate, PurchaseStatus: "开放申购"}},
	}

	service.evaluateForwardMomentumValidation(context.Background(), 300, current)
	service.applyMomentumReadiness(current)

	if current.Summary.ForwardValidationBatches != 3 || current.Summary.ForwardValidationCandidates != 3 {
		t.Fatalf("unexpected validation counts: %+v", current.Summary)
	}
	if !slices.Equal(current.ValidatedFundCodes, []string{"leader"}) || !slices.Equal(current.QualifiedFundCodes, []string{"leader"}) || current.Summary.CurrentValidatedCandidates != 1 || current.Summary.CurrentQualifiedCandidates != 1 {
		t.Fatalf("unexpected current validation coverage: %+v validated=%v qualified=%v", current.Summary, current.ValidatedFundCodes, current.QualifiedFundCodes)
	}
	if current.Summary.ForwardAverageReturn < cfg.MomentumPool.MinForwardAverageReturn || current.Summary.ForwardWinRate != 1 {
		t.Fatalf("unexpected validation results: %+v", current.Summary)
	}
	if current.Summary.Readiness != "small-trial" || current.Summary.SuggestedTotalWeight != cfg.MomentumPool.MaxTrialPortfolioWeight || current.Items[0].SuggestedWeight != cfg.MomentumPool.MaxTrialPortfolioWeight {
		t.Fatalf("unexpected readiness: %+v %+v", current.Summary, current.Items)
	}
}

func TestChallengerForwardMomentumValidationQualifiesOnlyProvenProduct(t *testing.T) {
	cfg := config.Default()
	strategyFingerprint := momentumStrategyFingerprint(cfg.MomentumPool)
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()

	history := momentumHistory("challenger", 130, 0.006)
	latestTradeDate := time.Now().UTC()
	firstTradeDate := latestTradeDate.AddDate(0, 0, -len(history)+1)
	for historyIndex := range history {
		history[historyIndex].TradeDate = firstTradeDate.AddDate(0, 0, historyIndex)
	}
	for _, historyIndex := range []int{30, 61, 92} {
		report := model.MomentumPoolReport{
			Summary:     model.MomentumPoolSummary{RunDate: history[historyIndex].TradeDate, StrategyFingerprint: strategyFingerprint, ForwardValidationDays: cfg.MomentumPool.ForwardValidationDays, Regime: "risk-on"},
			Challengers: []model.MomentumPoolItem{{FundCode: "challenger", FundName: "易方达信息产业混合A", FundType: "混合型", LatestTradeDate: history[historyIndex].TradeDate, LatestNAV: returnAdjustedNAV(history[historyIndex])}},
		}
		if _, err := store.SaveMomentumPool(report); err != nil {
			t.Fatalf("SaveMomentumPool() error = %v", err)
		}
	}

	service := &Service{
		config: cfg,
		store:  store,
		marketProfileCache: map[string]*model.MarketFundProfile{
			"challenger:300": {History: history},
		},
	}
	current := &model.MomentumPoolReport{
		Summary:     model.MomentumPoolSummary{StrategyFingerprint: strategyFingerprint, Regime: "risk-on"},
		Challengers: []model.MomentumPoolItem{{FundCode: "challenger", FundName: "易方达信息产业混合A", LatestTradeDate: latestTradeDate, PurchaseStatus: "开放申购"}, {FundCode: "unproven", FundName: "中欧信息科技混合发起A", LatestTradeDate: latestTradeDate, PurchaseStatus: "开放申购"}},
	}

	service.evaluateChallengerForwardMomentumValidation(context.Background(), 300, current)

	if current.Summary.ChallengerValidatedCandidates != 1 || current.Summary.ChallengerQualifiedCandidates != 1 || len(current.ChallengerValidations) != 2 || !current.ChallengerValidations[0].Qualified || current.ChallengerValidations[1].Qualified {
		t.Fatalf("unexpected challenger validations: summary=%+v validations=%+v", current.Summary, current.ChallengerValidations)
	}
}

func TestProductForwardMomentumValidationReusesEvidenceAcrossSelectionChanges(t *testing.T) {
	cfg := config.Default()
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()

	history := momentumHistory("leader", 130, 0.006)
	latestTradeDate := time.Now().UTC()
	firstTradeDate := latestTradeDate.AddDate(0, 0, -len(history)+1)
	for historyIndex := range history {
		history[historyIndex].TradeDate = firstTradeDate.AddDate(0, 0, historyIndex)
	}
	for _, historyIndex := range []int{30, 61, 92} {
		report := model.MomentumPoolReport{
			Summary: model.MomentumPoolSummary{StrategyFingerprint: "older-selection-rules", ForwardValidationDays: cfg.MomentumPool.ForwardValidationDays},
			Items:   []model.MomentumPoolItem{{FundCode: "leader", LatestTradeDate: history[historyIndex].TradeDate, LatestNAV: returnAdjustedNAV(history[historyIndex])}},
		}
		if _, err := store.SaveMomentumPool(report); err != nil {
			t.Fatalf("SaveMomentumPool() error = %v", err)
		}
	}
	service := &Service{config: cfg, store: store, marketProfileCache: map[string]*model.MarketFundProfile{"leader:300": {History: history}}}
	current := &model.MomentumPoolReport{Summary: model.MomentumPoolSummary{StrategyFingerprint: "current-selection-rules", ForwardValidationDays: cfg.MomentumPool.ForwardValidationDays, Regime: "risk-on"}}

	validations, validated, qualified := service.evaluateProductForwardMomentumValidation(context.Background(), 300, current, []model.MomentumPoolItem{{FundCode: "leader", LatestTradeDate: time.Now().UTC(), PurchaseStatus: "开放申购"}})

	if validated != 1 || qualified != 1 || len(validations) != 1 || validations[0].CompletedBatches != 3 || !validations[0].Qualified {
		t.Fatalf("expected reusable product evidence: validated=%d qualified=%d validations=%+v", validated, qualified, validations)
	}
}

func TestProductForwardMomentumValidationRejectsDifferentValidationHorizon(t *testing.T) {
	cfg := config.Default()
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()

	history := momentumHistory("leader", 80, 0.001)
	report := model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{StrategyFingerprint: "older-selection-rules", ForwardValidationDays: cfg.MomentumPool.ForwardValidationDays + 10},
		Items:   []model.MomentumPoolItem{{FundCode: "leader", LatestTradeDate: history[30].TradeDate, LatestNAV: returnAdjustedNAV(history[30])}},
	}
	if _, err := store.SaveMomentumPool(report); err != nil {
		t.Fatalf("SaveMomentumPool() error = %v", err)
	}
	service := &Service{config: cfg, store: store, marketProfileCache: map[string]*model.MarketFundProfile{"leader:300": {History: history}}}
	current := &model.MomentumPoolReport{Summary: model.MomentumPoolSummary{ForwardValidationDays: cfg.MomentumPool.ForwardValidationDays}}

	validations, validated, qualified := service.evaluateProductForwardMomentumValidation(context.Background(), 300, current, []model.MomentumPoolItem{{FundCode: "leader"}})

	if validated != 0 || qualified != 0 || len(validations) != 1 || validations[0].CompletedBatches != 0 {
		t.Fatalf("different validation horizons must not share evidence: validated=%d qualified=%d validations=%+v", validated, qualified, validations)
	}
}

func TestProductForwardMomentumValidationFollowsFundAcrossRankingSections(t *testing.T) {
	cfg := config.Default()
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()

	history := momentumHistory("rising", 80, 0.006)
	latestTradeDate := time.Now().UTC()
	firstTradeDate := latestTradeDate.AddDate(0, 0, -len(history)+1)
	for index := range history {
		history[index].TradeDate = firstTradeDate.AddDate(0, 0, index)
	}
	previous := model.MomentumPoolReport{
		Summary:     model.MomentumPoolSummary{ForwardValidationDays: cfg.MomentumPool.ForwardValidationDays},
		Challengers: []model.MomentumPoolItem{{FundCode: "rising", LatestTradeDate: history[30].TradeDate, LatestNAV: returnAdjustedNAV(history[30])}},
	}
	if _, err := store.SaveMomentumPool(previous); err != nil {
		t.Fatalf("SaveMomentumPool() error = %v", err)
	}
	service := &Service{config: cfg, store: store, marketProfileCache: map[string]*model.MarketFundProfile{"rising:300": {History: history}}}
	current := &model.MomentumPoolReport{Summary: model.MomentumPoolSummary{ForwardValidationDays: cfg.MomentumPool.ForwardValidationDays, Regime: "risk-on"}}

	validations, validated, qualified := service.evaluateProductForwardMomentumValidation(context.Background(), 300, current, []model.MomentumPoolItem{{FundCode: "rising", LatestTradeDate: latestTradeDate, PurchaseStatus: "开放申购"}})

	if validated != 1 || qualified != 1 || len(validations) != 1 || validations[0].CompletedBatches != 1 || !validations[0].Qualified {
		t.Fatalf("promoted fund should retain product evidence: validated=%d qualified=%d validations=%+v", validated, qualified, validations)
	}
}
