package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/fetcher"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type momentumRankCandidate struct {
	Fund            model.MarketSearchFund
	EstablishedDate time.Time
	Score           float64
	Return1M        float64
	Return3M        float64
	Return6M        float64
	Return1Y        float64
}

type momentumMetric struct {
	Code     string
	Return1M float64
	Return3M float64
	Return6M float64
	Return1Y float64
}

var shareClassSuffixPattern = regexp.MustCompile(`(?i)(?:ETF)?联接(?:发起式)?[A-Z](?:\(人民币\))?$|[A-Z](?:\(人民币\))?$`)

const momentumOpportunityCount = 12

func (s *Service) BuildMomentumPool(ctx context.Context, days int) (*model.MomentumPoolReport, error) {
	if !s.config.MomentumPool.Enabled {
		return nil, fmt.Errorf("momentum pool is disabled")
	}
	if days < 300 {
		days = 300
	}
	allFunds, err := s.loadMarketFunds(ctx)
	if err != nil {
		return nil, err
	}
	rankings, err := s.loadMarketRankings(ctx)
	if err != nil {
		return nil, err
	}
	eligibleFunds := filterMomentumFunds(allFunds, s.config.MomentumPool.AllowedCompanies)
	report := newMomentumPoolReport(len(allFunds), len(eligibleFunds))
	report.Summary.StrategyFingerprint = momentumStrategyFingerprint(s.config.MomentumPool)
	report.Summary.ForwardValidationDays = s.config.MomentumPool.ForwardValidationDays
	report.Summary.TrialReturnThreshold = s.config.MomentumPool.MinForwardAverageReturn
	report.Summary.Notes = append(report.Summary.Notes, "仅筛选配置中的知名基金公司产品，同一知名公司可入选多只基金。")
	report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("重点关注始终采用实时前 %d 名，不保留已跌出排名的旧候选。", s.config.MomentumPool.SelectionCount))
	report.Summary.Notes = append(report.Summary.Notes, "高涨幅和高波动不会阻止基金进入视野；回撤仅影响是否试投。")
	report.Summary.Notes = append(report.Summary.Notes, "暂停申购、封闭、终止交易和申购状态未知的强势基金仍会展示，但不会解锁试投。")
	report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("候选池每日更新用于观察；真实执行按至少 %d 个交易日持有周期评估，不把日报变化当作每日换仓指令。", s.config.MomentumPool.ForwardValidationDays))
	report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("只有拟试投基金净值在最近 %d 天内更新时，才允许解锁小额试投。", s.config.MomentumPool.MaxDataAgeDays))
	report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("当前策略版本：%s；筛选参数变化不清空同一验证周期下的产品级历史证据。", report.Summary.StrategyFingerprint))
	report.Summary.Notes = append(report.Summary.Notes, "产品级验证跟随基金本身，基金在重点关注与其他强势之间升降不会清空证据。")
	s.evaluatePreviousMomentumPool(ctx, days, report)
	s.evaluateForwardMomentumValidation(ctx, days, report)
	report.Summary.PositiveBreadth = rankingPositiveBreadth(eligibleFunds, rankings)
	report.Summary.MarketMedianReturn120D = rankingMedianReturn120D(eligibleFunds, rankings)
	if report.Summary.PositiveBreadth < s.config.MomentumPool.MinPositiveBreadth {
		report.Summary.Regime = "cash"
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("市场广度 %.1f%% 低于 %.1f%%，本期保持现金观察；局部强势基金仍继续展示。", report.Summary.PositiveBreadth*100, s.config.MomentumPool.MinPositiveBreadth*100))
	} else {
		report.Summary.Regime = "risk-on"
	}
	ranked := rankMomentumCandidates(eligibleFunds, rankings, s.config.MomentumPool)
	report.Summary.EvaluatedCount = len(ranked)
	items, qualifiedCount, profileFailures := s.evaluateMomentumCandidates(ctx, ranked, days, momentumOpportunityCount)
	report.Items = items
	report.Summary.QualifiedCount = qualifiedCount
	appendMomentumProfileFailureNote(report, "正式候选", profileFailures)
	s.applyMomentumPurchaseStatuses(ctx, report)
	rawCandidates := append([]model.MomentumPoolItem(nil), report.Items...)
	if len(report.Items) < s.config.MomentumPool.MinSelectionCount {
		report.Summary.Regime = "cash"
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("仅 %d 只基金满足入场条件，少于最低 %d 只，本期保持现金观察；已有强势基金仍继续展示。", len(report.Items), s.config.MomentumPool.MinSelectionCount))
	}
	if len(report.Items) > s.config.MomentumPool.SelectionCount {
		report.Items = report.Items[:s.config.MomentumPool.SelectionCount]
	}
	for index := range report.Items {
		report.Items[index].Rank = index + 1
	}
	report.ItemValidations, report.Summary.CurrentValidatedCandidates, report.Summary.CurrentQualifiedCandidates = s.evaluateProductForwardMomentumValidation(ctx, days, report, report.Items)
	report.ValidatedFundCodes, report.QualifiedFundCodes = momentumValidationCodes(report.ItemValidations)
	report.Challengers = buildMomentumChallengers(rawCandidates, report.Items, momentumOpportunityCount-len(report.Items))
	report.Summary.ChallengerCount = len(report.Challengers)
	s.evaluateChallengerForwardMomentumValidation(ctx, days, report)
	if len(report.Challengers) > 0 {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("强势榜额外展示 %d 只当前排名靠前的基金，帮助及时发现新的赚钱机会。", len(report.Challengers)))
	}
	applyUnifiedOpportunityScores(report, eligibleFunds, rankings, s.config.MomentumPool)
	if len(report.Challengers) > 0 {
		report.Summary.Notes = append(report.Summary.Notes, "其他强势基金继续积累产品级验证证据，但只有进入当前前 3 后才允许试投。")
	}
	s.applyMomentumCorrelationDiagnostics(ctx, report, minInt(days, 120))
	report.Summary.SelectedCount = len(report.Items)
	markCurrentMomentumBatchPending(report)
	s.applyMomentumObservationProgress(ctx, days, report)
	s.applyMomentumReadiness(report)
	report.Opportunities = buildMomentumOpportunities(report, s.config.MomentumPool.MaxTrialPortfolioWeight)
	if _, err := s.store.SaveMomentumPool(*report); err != nil {
		return nil, err
	}
	return report, nil
}

func applyUnifiedOpportunityScores(report *model.MomentumPoolReport, eligibleFunds []model.MarketSearchFund, rankings []fetcher.MarketRankEntry, cfg config.MomentumPoolConfig) {
	ranked := rankMomentumCandidates(eligibleFunds, rankings, cfg)
	scores := make(map[string]float64, len(ranked))
	for _, candidate := range ranked {
		scores[candidate.Fund.Code] = candidate.Score
	}
	applyScores := func(items []model.MomentumPoolItem) {
		for index := range items {
			if score, ok := scores[items[index].FundCode]; ok {
				items[index].MomentumScore = score
				items[index].Score = score
			}
		}
	}
	applyScores(report.Items)
	applyScores(report.Challengers)
}

func buildMomentumOpportunities(report *model.MomentumPoolReport, _ float64) []model.MomentumOpportunityItem {
	result := make([]model.MomentumOpportunityItem, 0, len(report.Items)+len(report.Challengers))
	appendValidatedProducts := func(items []model.MomentumPoolItem, validations []model.MomentumFundValidation, source, defaultEvidence string) {
		validationByCode := make(map[string]model.MomentumFundValidation, len(validations))
		for _, validation := range validations {
			validationByCode[validation.FundCode] = validation
		}
		for _, item := range items {
			validation := validationByCode[item.FundCode]
			evidence := fmt.Sprintf("%s；产品级前瞻验证 %d 批", defaultEvidence, validation.CompletedBatches)
			execution := "持续跟踪，不买入"
			if validation.Qualified {
				evidence = fmt.Sprintf("产品级前瞻验证已达标：最近周期收益 %.2f%%，历史平均 %.2f%%", validation.LatestReturn*100, validation.AverageReturn*100)
				execution = validation.Execution
			}
			if source == "重点关注" {
				execution = "持续跟踪，不买入"
				if item.SuggestedWeight > 0 {
					execution = "允许小额试投"
				}
			} else if validation.Qualified {
				execution = "验证达标，等待进入前3"
			}
			result = append(result, model.MomentumOpportunityItem{FundCode: item.FundCode, FundName: item.FundName, Source: source, Evidence: evidence, Execution: execution, SuggestedWeight: item.SuggestedWeight, MomentumScore: item.MomentumScore, Return20D: item.Return20D, Return60D: item.Return60D, Return120D: item.Return120D, Return250D: item.Return250D, ObservationDays: item.ObservationDays, ObservationReturn: item.ObservationReturn, PurchaseStatus: item.PurchaseStatus})
		}
	}
	appendValidatedProducts(report.Items, report.ItemValidations, "重点关注", "当前强势重点关注")
	appendValidatedProducts(report.Challengers, report.ChallengerValidations, "其他强势基金", "当前强势排名靠前，进入视野")
	sort.Slice(result, func(i, j int) bool {
		if result[i].MomentumScore != result[j].MomentumScore {
			return result[i].MomentumScore > result[j].MomentumScore
		}
		if result[i].Return120D != result[j].Return120D {
			return result[i].Return120D > result[j].Return120D
		}
		if result[i].Return60D != result[j].Return60D {
			return result[i].Return60D > result[j].Return60D
		}
		if result[i].Return20D != result[j].Return20D {
			return result[i].Return20D > result[j].Return20D
		}
		return result[i].FundCode < result[j].FundCode
	})
	for index := range result {
		result[index].Rank = index + 1
	}
	if len(result) > momentumOpportunityCount {
		result = result[:momentumOpportunityCount]
	}
	return result
}

func productValidationExecution(regime string, item model.MomentumPoolItem, now time.Time, maxAgeDays int) string {
	if regime != "risk-on" {
		return "验证达标，等待市场顺势"
	}
	if !momentumItemsFresh([]model.MomentumPoolItem{item}, now, maxAgeDays) {
		return "验证达标，等待净值更新"
	}
	if !isPurchasableStatus(item.PurchaseStatus) {
		return "验证达标，等待恢复申购"
	}
	return "允许小额试投"
}

func momentumReadinessExecution(readiness string) string {
	if readiness == "small-trial" {
		return "允许小额试投"
	}
	return "仅观察，等待前瞻验证"
}

func buildMomentumChallengers(ranked, selected []model.MomentumPoolItem, limit int) []model.MomentumPoolItem {
	if limit <= 0 || len(ranked) == 0 {
		return nil
	}
	selectedCodes := make(map[string]struct{}, len(selected))
	for _, item := range selected {
		selectedCodes[item.FundCode] = struct{}{}
	}
	challengers := make([]model.MomentumPoolItem, 0, minInt(limit, len(ranked)))
	for rawIndex, item := range ranked {
		if _, ok := selectedCodes[item.FundCode]; ok {
			continue
		}
		item.Rank = rawIndex + 1
		item.SuggestedWeight = 0
		item.Reason = fmt.Sprintf("原始动量排名第 %d，满足全部条件但暂未进入正式候选。", item.Rank)
		challengers = append(challengers, item)
		if len(challengers) >= limit {
			break
		}
	}
	return challengers
}

func (s *Service) applyMomentumCorrelationDiagnostics(ctx context.Context, report *model.MomentumPoolReport, days int) {
	if len(report.Items) < 2 {
		return
	}
	histories := make(map[string][]model.FundSnapshot, len(report.Items))
	for _, item := range report.Items {
		profile, err := s.loadMarketProfile(ctx, model.MarketSearchFund{Code: item.FundCode, Name: item.FundName, FundType: item.FundType}, time.Time{}, days)
		if err != nil {
			continue
		}
		histories[item.FundCode] = profile.History
	}
	var correlationSum float64
	pairCount := 0
	for leftIndex := 0; leftIndex < len(report.Items); leftIndex++ {
		for rightIndex := leftIndex + 1; rightIndex < len(report.Items); rightIndex++ {
			left := &report.Items[leftIndex]
			right := &report.Items[rightIndex]
			correlation, ok := fundReturnCorrelation(histories[left.FundCode], histories[right.FundCode], 60)
			if !ok {
				continue
			}
			correlationSum += correlation
			pairCount++
			if correlation > left.MaxPeerCorrelation {
				left.MaxPeerCorrelation = correlation
				left.MaxCorrelatedPeer = right.FundName
			}
			if correlation > right.MaxPeerCorrelation {
				right.MaxPeerCorrelation = correlation
				right.MaxCorrelatedPeer = left.FundName
			}
			if report.Summary.MaxCorrelationPair == "" || correlation > report.Summary.MaxPairCorrelation {
				report.Summary.MaxPairCorrelation = correlation
				report.Summary.MaxCorrelationPair = left.FundName + " / " + right.FundName
			}
		}
	}
	if pairCount > 0 {
		report.Summary.AveragePairCorrelation = correlationSum / float64(pairCount)
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("候选池最近60个交易日平均相关性 %.2f，最高相关性 %.2f（%s）；仅用于暴露重复风险，不会自动剔除高涨幅基金。", report.Summary.AveragePairCorrelation, report.Summary.MaxPairCorrelation, report.Summary.MaxCorrelationPair))
	}
}

func fundReturnCorrelation(leftHistory, rightHistory []model.FundSnapshot, window int) (float64, bool) {
	left := dailyReturnsByDate(leftHistory)
	right := dailyReturnsByDate(rightHistory)
	dates := make([]string, 0, minInt(len(left), len(right)))
	for date := range left {
		if _, ok := right[date]; ok {
			dates = append(dates, date)
		}
	}
	sort.Strings(dates)
	if window > 0 && len(dates) > window {
		dates = dates[len(dates)-window:]
	}
	if len(dates) < 20 {
		return 0, false
	}
	leftValues := make([]float64, 0, len(dates))
	rightValues := make([]float64, 0, len(dates))
	for _, date := range dates {
		leftValues = append(leftValues, left[date])
		rightValues = append(rightValues, right[date])
	}
	return pearsonCorrelation(leftValues, rightValues)
}

func dailyReturnsByDate(history []model.FundSnapshot) map[string]float64 {
	result := make(map[string]float64, len(history))
	for index := 1; index < len(history); index++ {
		previous := returnAdjustedNAV(history[index-1])
		current := returnAdjustedNAV(history[index])
		if previous <= 0 || current <= 0 {
			continue
		}
		result[history[index].TradeDate.Format("2006-01-02")] = current/previous - 1
	}
	return result
}

func pearsonCorrelation(left, right []float64) (float64, bool) {
	if len(left) != len(right) || len(left) < 2 {
		return 0, false
	}
	var leftSum, rightSum float64
	for index := range left {
		leftSum += left[index]
		rightSum += right[index]
	}
	leftMean := leftSum / float64(len(left))
	rightMean := rightSum / float64(len(right))
	var covariance, leftVariance, rightVariance float64
	for index := range left {
		leftDelta := left[index] - leftMean
		rightDelta := right[index] - rightMean
		covariance += leftDelta * rightDelta
		leftVariance += leftDelta * leftDelta
		rightVariance += rightDelta * rightDelta
	}
	if leftVariance == 0 || rightVariance == 0 {
		return 0, false
	}
	return covariance / math.Sqrt(leftVariance*rightVariance), true
}

func (s *Service) evaluateForwardMomentumValidation(ctx context.Context, days int, report *model.MomentumPoolReport) {
	history, err := s.store.MomentumPoolHistory()
	if err != nil {
		return
	}
	var lastBatchEnd time.Time
	var totalReturn float64
	wins := 0
	batchWins := 0
	validatedCodes := make(map[string]struct{})
	validationByCode := make(map[string]momentumFundValidation)
	for _, previous := range history {
		if previous.Summary.StrategyFingerprint != report.Summary.StrategyFingerprint {
			continue
		}
		batchStart := momentumBatchStartDate(previous.Items)
		if len(previous.Items) == 0 || batchStart.IsZero() || (!lastBatchEnd.IsZero() && !batchStart.After(lastBatchEnd)) {
			continue
		}
		if !momentumBatchHasRecordedStarts(previous.Items) {
			report.Summary.UnverifiableValidationBatches++
			continue
		}
		batchReturn, candidateWins, evaluated, batchEnd, fundReturns := s.evaluateMomentumBatch(ctx, days, previous)
		if evaluated != len(previous.Items) {
			if isExpiredMomentumBatch(batchStart, time.Now().UTC(), days, s.config.MomentumPool.ForwardValidationDays) {
				report.Summary.ExpiredValidationBatches++
				continue
			}
			report.Summary.PendingValidationBatches = 1
			report.Summary.OldestPendingStartDate = batchStart
			report.Summary.OldestPendingTradingDays = approximateTradingDays(batchStart, time.Now().UTC())
			break
		}
		report.Summary.ForwardValidationBatches++
		report.Summary.ForwardValidationCandidates += evaluated
		for _, item := range previous.Items {
			validatedCodes[item.FundCode] = struct{}{}
		}
		for code, fundReturn := range fundReturns {
			validation := validationByCode[code]
			validation.Count++
			validation.TotalReturn += fundReturn
			if fundReturn > 0 {
				validation.Wins++
			}
			if batchEnd.After(validation.LatestEnd) {
				validation.LatestReturn = fundReturn
				validation.LatestEnd = batchEnd
			}
			validationByCode[code] = validation
		}
		totalReturn += batchReturn * float64(evaluated)
		wins += candidateWins
		if batchReturn > 0 {
			batchWins++
		}
		lastBatchEnd = batchEnd
	}
	if report.Summary.ForwardValidationCandidates == 0 {
		report.ValidatedFundCodes = sortedStringSet(validatedCodes)
		report.QualifiedFundCodes = qualifiedMomentumFundCodes(validationByCode, s.config.MomentumPool, time.Now().UTC())
		appendSkippedValidationNotes(report)
		return
	}
	report.Summary.ForwardAverageReturn = totalReturn / float64(report.Summary.ForwardValidationCandidates)
	report.Summary.ForwardWinRate = float64(wins) / float64(report.Summary.ForwardValidationCandidates)
	report.Summary.ForwardBatchWinRate = float64(batchWins) / float64(report.Summary.ForwardValidationBatches)
	report.ValidatedFundCodes = sortedStringSet(validatedCodes)
	report.QualifiedFundCodes = qualifiedMomentumFundCodes(validationByCode, s.config.MomentumPool, time.Now().UTC())
	appendSkippedValidationNotes(report)
}

func markCurrentMomentumBatchPending(report *model.MomentumPoolReport) {
	if report.Summary.PendingValidationBatches > 0 || !momentumBatchHasRecordedStarts(report.Items) {
		return
	}
	start := momentumBatchStartDate(report.Items)
	report.Summary.PendingValidationBatches = 1
	report.Summary.OldestPendingStartDate = start
	report.Summary.OldestPendingTradingDays = approximateTradingDays(start, time.Now().UTC())
}

func (s *Service) applyMomentumObservationProgress(ctx context.Context, days int, report *model.MomentumPoolReport) {
	history, err := s.store.MomentumPoolHistory()
	if err != nil {
		return
	}
	earliest := make(map[string]model.MomentumPoolItem)
	for _, previous := range history {
		for _, item := range append(previous.Items, previous.Challengers...) {
			known, ok := earliest[item.FundCode]
			if item.LatestNAV > 0 && !item.LatestTradeDate.IsZero() && (!ok || item.LatestTradeDate.Before(known.LatestTradeDate)) {
				earliest[item.FundCode] = item
			}
		}
	}
	annotate := func(items []model.MomentumPoolItem) {
		for index := range items {
			start, ok := earliest[items[index].FundCode]
			if !ok {
				start = items[index]
			}
			profile, err := s.loadMarketProfile(ctx, model.MarketSearchFund{Code: items[index].FundCode, Name: items[index].FundName, FundType: items[index].FundType}, time.Time{}, days)
			if err != nil {
				continue
			}
			items[index].ObservationDays, items[index].ObservationReturn = momentumObservationProgress(profile.History, start.LatestTradeDate, start.LatestNAV, items[index].LatestTradeDate, items[index].LatestNAV)
		}
	}
	annotate(report.Items)
	annotate(report.Challengers)
	report.Summary.OldestPendingTradingDays = 0
	for _, item := range report.Items {
		if item.ObservationDays > report.Summary.OldestPendingTradingDays {
			report.Summary.OldestPendingTradingDays = item.ObservationDays
		}
	}
}

func momentumObservationProgress(history []model.FundSnapshot, startDate time.Time, startNAV float64, endDate time.Time, endNAV float64) (int, float64) {
	if startNAV <= 0 || endNAV <= 0 || startDate.IsZero() || endDate.Before(startDate) {
		return 0, 0
	}
	startIndex := sort.Search(len(history), func(index int) bool { return !history[index].TradeDate.Before(startDate) })
	endIndex := sort.Search(len(history), func(index int) bool { return !history[index].TradeDate.Before(endDate) })
	if startIndex >= len(history) || endIndex >= len(history) || !history[startIndex].TradeDate.Equal(startDate) || !history[endIndex].TradeDate.Equal(endDate) {
		return 0, 0
	}
	observationDays := endIndex - startIndex
	if observationDays == 0 {
		return 0, 0
	}
	return observationDays, endNAV/startNAV - 1
}

func (s *Service) evaluateChallengerForwardMomentumValidation(ctx context.Context, days int, report *model.MomentumPoolReport) {
	report.ChallengerValidations, report.Summary.ChallengerValidatedCandidates, report.Summary.ChallengerQualifiedCandidates = s.evaluateProductForwardMomentumValidation(ctx, days, report, report.Challengers)
}

func (s *Service) evaluateProductForwardMomentumValidation(ctx context.Context, days int, report *model.MomentumPoolReport, currentItems []model.MomentumPoolItem) ([]model.MomentumFundValidation, int, int) {
	history, err := s.store.MomentumPoolHistory()
	if err != nil {
		return nil, 0, 0
	}
	lastBatchEndByCode := make(map[string]time.Time)
	validationByCode := make(map[string]momentumFundValidation)
	for _, previous := range history {
		if previous.Summary.ForwardValidationDays != s.config.MomentumPool.ForwardValidationDays {
			continue
		}
		for _, item := range append(previous.Items, previous.Challengers...) {
			if item.FundCode == "" || item.LatestTradeDate.IsZero() || item.LatestNAV <= 0 || !item.LatestTradeDate.After(lastBatchEndByCode[item.FundCode]) {
				continue
			}
			fundReturn, batchEnd, ok := s.evaluateMomentumItem(ctx, days, item)
			if !ok {
				continue
			}
			validation := validationByCode[item.FundCode]
			validation.Count++
			validation.TotalReturn += fundReturn
			if fundReturn > 0 {
				validation.Wins++
			}
			if batchEnd.After(validation.LatestEnd) {
				validation.LatestReturn = fundReturn
				validation.LatestEnd = batchEnd
			}
			validationByCode[item.FundCode] = validation
			lastBatchEndByCode[item.FundCode] = batchEnd
		}
	}
	qualifiedCodes := qualifiedMomentumFundCodes(validationByCode, s.config.MomentumPool, time.Now().UTC())
	qualified := make(map[string]struct{}, len(qualifiedCodes))
	for _, code := range qualifiedCodes {
		qualified[code] = struct{}{}
	}
	results := make([]model.MomentumFundValidation, 0, len(currentItems))
	validatedCount := 0
	qualifiedCount := 0
	for _, item := range currentItems {
		validation := validationByCode[item.FundCode]
		result := model.MomentumFundValidation{FundCode: item.FundCode, CompletedBatches: validation.Count, LatestReturn: validation.LatestReturn, LatestEndDate: validation.LatestEnd}
		if validation.Count > 0 {
			result.AverageReturn = validation.TotalReturn / float64(validation.Count)
			result.WinRate = float64(validation.Wins) / float64(validation.Count)
			validatedCount++
		}
		if _, ok := qualified[item.FundCode]; ok {
			result.Qualified = true
			result.Execution = productValidationExecution(report.Summary.Regime, item, time.Now().UTC(), s.config.MomentumPool.MaxDataAgeDays)
			qualifiedCount++
		}
		results = append(results, result)
	}
	return results, validatedCount, qualifiedCount
}

type momentumFundValidation struct {
	Count        int
	Wins         int
	TotalReturn  float64
	LatestReturn float64
	LatestEnd    time.Time
}

func qualifiedMomentumFundCodes(validations map[string]momentumFundValidation, cfg config.MomentumPoolConfig, now time.Time) []string {
	qualified := make(map[string]struct{})
	for code, validation := range validations {
		if validation.Count < 1 || approximateTradingDays(validation.LatestEnd, now) > cfg.ForwardValidationDays {
			continue
		}
		if validation.LatestReturn >= cfg.MinForwardAverageReturn {
			qualified[code] = struct{}{}
		}
	}
	return sortedStringSet(qualified)
}

func sortedStringSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func momentumBatchHasRecordedStarts(items []model.MomentumPoolItem) bool {
	if len(items) == 0 {
		return false
	}
	startDate := items[0].LatestTradeDate
	for _, item := range items {
		if item.LatestTradeDate.IsZero() || !item.LatestTradeDate.Equal(startDate) || item.LatestNAV <= 0 {
			return false
		}
	}
	return true
}

func appendSkippedValidationNotes(report *model.MomentumPoolReport) {
	if report.Summary.ExpiredValidationBatches > 0 {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("已跳过 %d 个超出历史拉取窗口、无法完整复算的旧验证批次。", report.Summary.ExpiredValidationBatches))
	}
	if report.Summary.UnverifiableValidationBatches > 0 {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("已跳过 %d 个缺少冻结入选净值或精确起始日期、不可验证的旧批次。", report.Summary.UnverifiableValidationBatches))
	}
}

func isExpiredMomentumBatch(start, now time.Time, historyDays, validationDays int) bool {
	if start.IsZero() || historyDays <= validationDays {
		return false
	}
	return approximateTradingDays(start, now) > historyDays-validationDays
}

func momentumStrategyFingerprint(cfg config.MomentumPoolConfig) string {
	payload, _ := json.Marshal(struct {
		Version string               `json:"version"`
		Config  momentumFormalConfig `json:"config"`
	}{Version: "momentum-v5-visibility-first", Config: newMomentumFormalConfig(cfg)})
	digest := sha256.Sum256(payload)
	return fmt.Sprintf("%x", digest[:8])
}

type momentumFormalConfig struct {
	Enabled                 bool
	SelectionCount          int
	MinSelectionCount       int
	AllowedCompanies        []string
	MinMomentumScore        float64
	MomentumWeight20D       float64
	MomentumWeight60D       float64
	MomentumWeight120D      float64
	MomentumWeight250D      float64
	MinReturn20D            float64
	MinReturn60D            float64
	MinReturn120D           float64
	MaxDrawdown120D         float64
	MinPositiveBreadth      float64
	ForwardValidationDays   int
	MinForwardAverageReturn float64
	MaxTrialPortfolioWeight float64
	MaxDataAgeDays          int
}

func newMomentumFormalConfig(cfg config.MomentumPoolConfig) momentumFormalConfig {
	return momentumFormalConfig{
		Enabled: cfg.Enabled, SelectionCount: cfg.SelectionCount, MinSelectionCount: cfg.MinSelectionCount,
		AllowedCompanies: cfg.AllowedCompanies,
		MinMomentumScore: cfg.MinMomentumScore, MomentumWeight20D: cfg.MomentumWeight20D, MomentumWeight60D: cfg.MomentumWeight60D, MomentumWeight120D: cfg.MomentumWeight120D,
		MomentumWeight250D: cfg.MomentumWeight250D, MinReturn20D: cfg.MinReturn20D, MinReturn60D: cfg.MinReturn60D, MinReturn120D: cfg.MinReturn120D,
		MaxDrawdown120D: cfg.MaxDrawdown120D, MinPositiveBreadth: cfg.MinPositiveBreadth, ForwardValidationDays: cfg.ForwardValidationDays,
		MinForwardAverageReturn: cfg.MinForwardAverageReturn,
		MaxTrialPortfolioWeight: cfg.MaxTrialPortfolioWeight, MaxDataAgeDays: cfg.MaxDataAgeDays,
	}
}

func approximateTradingDays(start, end time.Time) int {
	if start.IsZero() || !end.After(start) {
		return 0
	}
	return int(end.Sub(start).Hours() / 24 * 5 / 7)
}

func momentumBatchStartDate(items []model.MomentumPoolItem) time.Time {
	var start time.Time
	for _, item := range items {
		if item.LatestTradeDate.After(start) {
			start = item.LatestTradeDate
		}
	}
	return start
}

func (s *Service) evaluateMomentumBatch(ctx context.Context, days int, previous model.MomentumPoolReport) (float64, int, int, time.Time, map[string]float64) {
	var totalReturn float64
	wins := 0
	evaluated := 0
	var batchEnd time.Time
	fundReturns := make(map[string]float64, len(previous.Items))
	for _, item := range previous.Items {
		forwardReturn, endDate, ok := s.evaluateMomentumItem(ctx, days, item)
		if !ok {
			continue
		}
		totalReturn += forwardReturn
		fundReturns[item.FundCode] = forwardReturn
		if forwardReturn > 0 {
			wins++
		}
		evaluated++
		if endDate.After(batchEnd) {
			batchEnd = endDate
		}
	}
	if evaluated == 0 {
		return 0, 0, 0, time.Time{}, nil
	}
	return totalReturn / float64(evaluated), wins, evaluated, batchEnd, fundReturns
}

func (s *Service) evaluateMomentumItem(ctx context.Context, days int, item model.MomentumPoolItem) (float64, time.Time, bool) {
	profile, err := s.loadMarketProfile(ctx, model.MarketSearchFund{Code: item.FundCode, Name: item.FundName, FundType: item.FundType}, time.Time{}, days)
	if err != nil {
		return 0, time.Time{}, false
	}
	return fixedHorizonReturnFromRecordedNAV(profile.History, item.LatestTradeDate, item.LatestNAV, s.config.MomentumPool.ForwardValidationDays)
}

func fixedHorizonReturn(history []model.FundSnapshot, startDate time.Time, tradingDays int) (float64, time.Time, bool) {
	startIndex := sort.Search(len(history), func(index int) bool { return !history[index].TradeDate.Before(startDate) })
	endIndex := startIndex + tradingDays
	if startIndex >= len(history) || !history[startIndex].TradeDate.Equal(startDate) || endIndex >= len(history) {
		return 0, time.Time{}, false
	}
	startNAV := returnAdjustedNAV(history[startIndex])
	endNAV := returnAdjustedNAV(history[endIndex])
	if startNAV <= 0 {
		return 0, time.Time{}, false
	}
	return endNAV/startNAV - 1, history[endIndex].TradeDate, true
}

func fixedHorizonReturnFromRecordedNAV(history []model.FundSnapshot, startDate time.Time, startNAV float64, tradingDays int) (float64, time.Time, bool) {
	startIndex := sort.Search(len(history), func(index int) bool { return !history[index].TradeDate.Before(startDate) })
	endIndex := startIndex + tradingDays
	if startNAV <= 0 || startIndex >= len(history) || !history[startIndex].TradeDate.Equal(startDate) || endIndex >= len(history) {
		return 0, time.Time{}, false
	}
	endNAV := returnAdjustedNAV(history[endIndex])
	if endNAV <= 0 {
		return 0, time.Time{}, false
	}
	return endNAV/startNAV - 1, history[endIndex].TradeDate, true
}

func (s *Service) applyMomentumReadiness(report *model.MomentumPoolReport) {
	report.Summary.CurrentValidatedCandidates = countValidatedMomentumItems(report.Items, report.ValidatedFundCodes)
	report.Summary.CurrentQualifiedCandidates = countValidatedMomentumItems(report.Items, report.QualifiedFundCodes)
	for index := range report.Items {
		report.Items[index].SuggestedWeight = 0
	}
	qualified := make(map[string]struct{}, len(report.QualifiedFundCodes))
	for _, code := range report.QualifiedFundCodes {
		qualified[code] = struct{}{}
	}
	trialIndexes := make([]int, 0, len(report.Items))
	for index, item := range report.Items {
		if _, ok := qualified[item.FundCode]; !ok || report.Summary.Regime != "risk-on" || !momentumItemsFresh([]model.MomentumPoolItem{item}, time.Now().UTC(), s.config.MomentumPool.MaxDataAgeDays) || !isPurchasableStatus(item.PurchaseStatus) || !momentumItemsPassExecutionTrend([]model.MomentumPoolItem{item}, singleMomentumSelectionConfig(s.config.MomentumPool)) {
			continue
		}
		trialIndexes = append(trialIndexes, index)
	}
	if len(trialIndexes) > 0 {
		report.Summary.Readiness = "small-trial"
		report.Summary.SuggestedTotalWeight = s.config.MomentumPool.MaxTrialPortfolioWeight
		weight := report.Summary.SuggestedTotalWeight / float64(len(trialIndexes))
		for _, index := range trialIndexes {
			report.Items[index].SuggestedWeight = weight
		}
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("%d 只当前强势基金已逐只通过前瞻验证，仅允许总仓位不超过 %.1f%% 的小额试投并继续验证。", len(trialIndexes), report.Summary.SuggestedTotalWeight*100))
		return
	}
	report.Summary.Readiness = "watch-only"
	report.Summary.SuggestedTotalWeight = 0
	report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("当前尚无基金完成一个真实周期且收益达到 %.1f%%，同时满足实时执行条件，当前仅观察。", s.config.MomentumPool.MinForwardAverageReturn*100))
}

func singleMomentumSelectionConfig(cfg config.MomentumPoolConfig) config.MomentumPoolConfig {
	cfg.MinSelectionCount = 1
	return cfg
}

func momentumValidationCodes(validations []model.MomentumFundValidation) ([]string, []string) {
	validated := make(map[string]struct{})
	qualified := make(map[string]struct{})
	for _, validation := range validations {
		if validation.CompletedBatches > 0 {
			validated[validation.FundCode] = struct{}{}
		}
		if validation.Qualified {
			qualified[validation.FundCode] = struct{}{}
		}
	}
	return sortedStringSet(validated), sortedStringSet(qualified)
}

func momentumItemsPassExecutionTrend(items []model.MomentumPoolItem, cfg config.MomentumPoolConfig) bool {
	if len(items) < cfg.MinSelectionCount {
		return false
	}
	for _, item := range items {
		metrics := marketPoolMetrics{Return20D: item.Return20D, Return60D: item.Return60D, Return120D: item.Return120D, Return250D: item.Return250D, MaxDrawdown120D: item.MaxDrawdown120D}
		if !passesMomentumEntry(cfg, item.MomentumScore, metrics) {
			return false
		}
	}
	return true
}

func countValidatedMomentumItems(items []model.MomentumPoolItem, validatedCodes []string) int {
	validated := make(map[string]struct{}, len(validatedCodes))
	for _, code := range validatedCodes {
		validated[code] = struct{}{}
	}
	count := 0
	for _, item := range items {
		if _, ok := validated[item.FundCode]; ok {
			count++
		}
	}
	return count
}

func (s *Service) applyMomentumPurchaseStatuses(ctx context.Context, report *model.MomentumPoolReport) {
	statuses, err := s.loadFundPurchaseStatuses(ctx)
	if err != nil {
		report.Summary.Notes = append(report.Summary.Notes, "申购状态数据暂不可用；候选继续展示，但不会解锁试投。")
		return
	}
	unknown, unavailable := annotateMomentumPurchaseStatuses(report.Items, statuses)
	if unavailable > 0 {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("有 %d 只强势基金当前不可申购；继续展示用于发现机会，但不会解锁试投。", unavailable))
	}
	if unknown > 0 {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("有 %d 只强势基金申购状态未知；继续展示用于发现机会，但不会解锁试投。", unknown))
	}
}

func annotateMomentumPurchaseStatuses(items []model.MomentumPoolItem, statuses map[string]fetcher.FundPurchaseStatus) (int, int) {
	unknown := 0
	unavailable := 0
	for index := range items {
		status, ok := statuses[items[index].FundCode]
		if !ok {
			items[index].PurchaseStatus = "未知"
			unknown++
			continue
		}
		items[index].PurchaseStatus = status.PurchaseStatus
		items[index].MinimumPurchase = status.MinimumPurchase
		items[index].DailyLimit = status.DailyLimit
		if !isPurchasableStatus(status.PurchaseStatus) {
			unavailable++
		}
	}
	return unknown, unavailable
}

func (s *Service) loadFundPurchaseStatuses(ctx context.Context) (map[string]fetcher.FundPurchaseStatus, error) {
	if s.purchaseStatusReady {
		return s.purchaseStatusCache, s.purchaseStatusErr
	}
	s.purchaseStatusCache, s.purchaseStatusErr = s.fetcher.FetchFundPurchaseStatuses(ctx)
	s.purchaseStatusReady = true
	return s.purchaseStatusCache, s.purchaseStatusErr
}

func isPurchasableStatus(status string) bool {
	return status == "开放申购" || status == "限大额"
}

func momentumItemsPurchasable(items []model.MomentumPoolItem) bool {
	if len(items) == 0 {
		return false
	}
	for _, item := range items {
		if !isPurchasableStatus(item.PurchaseStatus) {
			return false
		}
	}
	return true
}

func momentumItemsFresh(items []model.MomentumPoolItem, now time.Time, maxAgeDays int) bool {
	if len(items) == 0 || maxAgeDays <= 0 {
		return false
	}
	maxAge := time.Duration(maxAgeDays) * 24 * time.Hour
	for _, item := range items {
		if item.LatestTradeDate.IsZero() || now.Sub(item.LatestTradeDate) > maxAge {
			return false
		}
	}
	return true
}

func (s *Service) evaluatePreviousMomentumPool(ctx context.Context, days int, report *model.MomentumPoolReport) {
	previous, err := s.store.LatestSelectedMomentumPool()
	if err != nil || previous == nil || len(previous.Items) == 0 || previous.Summary.StrategyFingerprint != report.Summary.StrategyFingerprint {
		return
	}
	challengerBaseline := s.continuousChallengerBaseline(previous)
	selectedBaseline := previous.Items
	if len(challengerBaseline.Challengers) > 0 {
		selectedBaseline = challengerBaseline.Items
	}
	report.Summary.PreviousAverageReturn, report.Summary.PreviousWinRate, report.Summary.PreviousEvaluatedCount = s.evaluateMomentumItemsToLatest(ctx, days, selectedBaseline)
	report.Summary.PreviousChallengerReturn, report.Summary.PreviousChallengerWinRate, report.Summary.PreviousChallengerCount = s.evaluateMomentumItemsToLatest(ctx, days, challengerBaseline.Challengers)
	report.Summary.ChallengerTrackingStartDate = challengerBaseline.Summary.RunDate
	report.Summary.ChallengerTrackingDays = approximateTradingDays(report.Summary.ChallengerTrackingStartDate, time.Now().UTC())
	if report.Summary.PreviousEvaluatedCount > 0 {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("影子批次正式候选截至当前平均收益 %.2f%%，胜率 %.1f%%；该数据仅为实时跟踪，不计入固定周期前瞻验证。", report.Summary.PreviousAverageReturn*100, report.Summary.PreviousWinRate*100))
	}
	if report.Summary.PreviousChallengerCount > 0 {
		report.Summary.ChallengerExcessReturn = report.Summary.PreviousChallengerReturn - report.Summary.PreviousAverageReturn
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("最近一期挑战者截至当前平均收益 %.2f%%，相对正式候选 %+.2f%%；仅作为影子组合跟踪，不自动替换。", report.Summary.PreviousChallengerReturn*100, report.Summary.ChallengerExcessReturn*100))
	}
}

func (s *Service) continuousChallengerBaseline(latest *model.MomentumPoolReport) model.MomentumPoolReport {
	if latest == nil || len(latest.Challengers) == 0 {
		return model.MomentumPoolReport{}
	}
	baseline := *latest
	history, err := s.store.MomentumPoolHistory()
	if err != nil {
		return baseline
	}
	wanted := momentumItemCodeSet(latest.Challengers)
	for index := len(history) - 1; index >= 0; index-- {
		candidate := history[index]
		if candidate.Summary.StrategyFingerprint != latest.Summary.StrategyFingerprint || !equalStringSets(wanted, momentumItemCodeSet(candidate.Challengers)) {
			break
		}
		baseline = candidate
	}
	return baseline
}

func momentumItemCodeSet(items []model.MomentumPoolItem) map[string]struct{} {
	result := make(map[string]struct{}, len(items))
	for _, item := range items {
		result[item.FundCode] = struct{}{}
	}
	return result
}

func equalStringSets(left, right map[string]struct{}) bool {
	if len(left) != len(right) {
		return false
	}
	for value := range left {
		if _, ok := right[value]; !ok {
			return false
		}
	}
	return true
}

func (s *Service) evaluateMomentumItemsToLatest(ctx context.Context, days int, items []model.MomentumPoolItem) (float64, float64, int) {
	var totalReturn float64
	wins := 0
	evaluated := 0
	for _, item := range items {
		if item.LatestNAV <= 0 {
			continue
		}
		profile, err := s.loadMarketProfile(ctx, model.MarketSearchFund{Code: item.FundCode, Name: item.FundName, FundType: item.FundType}, time.Time{}, days)
		if err != nil || profile.Latest == nil || returnAdjustedNAV(*profile.Latest) <= 0 || !profile.Latest.TradeDate.After(item.LatestTradeDate) {
			continue
		}
		forwardReturn := returnAdjustedNAV(*profile.Latest)/item.LatestNAV - 1
		totalReturn += forwardReturn
		if forwardReturn > 0 {
			wins++
		}
		evaluated++
	}
	if evaluated == 0 {
		return 0, 0, 0
	}
	return totalReturn / float64(evaluated), float64(wins) / float64(evaluated), evaluated
}

func newMomentumPoolReport(universeCount, equityCount int) *model.MomentumPoolReport {
	now := time.Now().UTC()
	return &model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{
			RunDate:       now,
			UniverseCount: universeCount,
			EquityCount:   equityCount,
			GeneratedAt:   now,
			Notes: []string{
				"动量池用于发现当前强势基金，不代表未来收益保证。",
				"收益、回撤和固定周期验证按累计净值计算，避免基金分红或拆分造成虚假涨跌。",
				"同一基金不同份额只保留一只；高涨幅不会被惩罚或剔除，强趋势基金可正常入选。",
			},
		},
	}
}

func filterMomentumFunds(funds []model.MarketSearchFund, allowedCompanies []string) []model.MarketSearchFund {
	result := make([]model.MarketSearchFund, 0, len(funds))
	for _, fund := range funds {
		if isMomentumEligibleFund(fund) && isAllowedFundCompany(fund.Name, allowedCompanies) {
			result = append(result, fund)
		}
	}
	return result
}

func isAllowedFundCompany(name string, allowedCompanies []string) bool {
	name = strings.TrimSpace(name)
	excludedPrefixes := []string{"广发资管", "国泰君安", "华安证券", "招商资管", "中银证券"}
	for _, prefix := range excludedPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	aliases := map[string][]string{"工银瑞信": {"工银"}, "兴证全球": {"兴全"}, "交银施罗德": {"交银"}}
	for _, company := range allowedCompanies {
		company = strings.TrimSpace(company)
		if company != "" && strings.HasPrefix(name, company) {
			return true
		}
		for _, alias := range aliases[company] {
			if strings.HasPrefix(name, alias) {
				return true
			}
		}
	}
	return false
}

func isMomentumEligibleFund(fund model.MarketSearchFund) bool {
	name := strings.TrimSpace(fund.Name)
	fundType := strings.TrimSpace(fund.FundType)
	if name == "" || strings.Contains(name, "后端") {
		return false
	}
	excluded := []string{"货币", "债券", "短债", "纯债", "同业存单", "现金", "理财", "FOF", "养老", "美元", "现汇", "现钞"}
	for _, keyword := range excluded {
		if strings.Contains(name, keyword) || strings.Contains(fundType, keyword) {
			return false
		}
	}
	return strings.Contains(fundType, "股票") || strings.Contains(fundType, "混合") || strings.Contains(fundType, "指数") || strings.Contains(fundType, "QDII") || strings.Contains(strings.ToUpper(name), "ETF")
}

func rankingPositiveBreadth(funds []model.MarketSearchFund, rankings []fetcher.MarketRankEntry) float64 {
	allowed := make(map[string]struct{}, len(funds))
	for _, fund := range funds {
		allowed[fund.Code] = struct{}{}
	}
	positive := 0
	total := 0
	for _, entry := range rankings {
		if _, ok := allowed[entry.Code]; !ok {
			continue
		}
		total++
		if entry.Return3M > 0 && entry.Return6M > 0 {
			positive++
		}
	}
	if total == 0 {
		return 0
	}
	return float64(positive) / float64(total)
}

func rankingMedianReturn120D(funds []model.MarketSearchFund, rankings []fetcher.MarketRankEntry) float64 {
	allowed := make(map[string]struct{}, len(funds))
	for _, fund := range funds {
		allowed[fund.Code] = struct{}{}
	}
	values := make([]float64, 0, len(funds))
	for _, entry := range rankings {
		if _, ok := allowed[entry.Code]; ok {
			values = append(values, entry.Return6M)
		}
	}
	return medianFloat(values)
}

func rankMomentumCandidates(funds []model.MarketSearchFund, rankings []fetcher.MarketRankEntry, cfg config.MomentumPoolConfig) []momentumRankCandidate {
	fundByCode := make(map[string]model.MarketSearchFund, len(funds))
	metrics := make([]momentumMetric, 0, len(funds))
	entryByCode := make(map[string]fetcher.MarketRankEntry, len(rankings))
	for _, fund := range funds {
		fundByCode[fund.Code] = fund
	}
	for _, entry := range rankings {
		if _, ok := fundByCode[entry.Code]; !ok {
			continue
		}
		entryByCode[entry.Code] = entry
		metrics = append(metrics, momentumMetric{Code: entry.Code, Return1M: entry.Return1M, Return3M: entry.Return3M, Return6M: entry.Return6M, Return1Y: entry.Return1Y})
	}
	scores := momentumPercentileScores(metrics, cfg)
	candidates := make([]momentumRankCandidate, 0, len(scores))
	for code, score := range scores {
		entry := entryByCode[code]
		candidates = append(candidates, momentumRankCandidate{Fund: fundByCode[code], EstablishedDate: entry.EstablishedDate, Score: score, Return1M: entry.Return1M, Return3M: entry.Return3M, Return6M: entry.Return6M, Return1Y: entry.Return1Y})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		if candidates[i].Return1Y != candidates[j].Return1Y {
			return candidates[i].Return1Y > candidates[j].Return1Y
		}
		if candidates[i].Return6M != candidates[j].Return6M {
			return candidates[i].Return6M > candidates[j].Return6M
		}
		if candidates[i].Return3M != candidates[j].Return3M {
			return candidates[i].Return3M > candidates[j].Return3M
		}
		if candidates[i].Return1M != candidates[j].Return1M {
			return candidates[i].Return1M > candidates[j].Return1M
		}
		return candidates[i].Fund.Code < candidates[j].Fund.Code
	})
	return candidates
}

func momentumPercentileScores(metrics []momentumMetric, cfg config.MomentumPoolConfig) map[string]float64 {
	oneMonth := sortedMomentumValues(metrics, func(metric momentumMetric) float64 { return metric.Return1M })
	threeMonth := sortedMomentumValues(metrics, func(metric momentumMetric) float64 { return metric.Return3M })
	sixMonth := sortedMomentumValues(metrics, func(metric momentumMetric) float64 { return metric.Return6M })
	oneYear := sortedMomentumValues(metrics, func(metric momentumMetric) float64 { return metric.Return1Y })
	scores := make(map[string]float64, len(metrics))
	for _, metric := range metrics {
		scores[metric.Code] = 100 * (cfg.MomentumWeight20D*percentileRank(oneMonth, metric.Return1M) + cfg.MomentumWeight60D*percentileRank(threeMonth, metric.Return3M) + cfg.MomentumWeight120D*percentileRank(sixMonth, metric.Return6M) + cfg.MomentumWeight250D*percentileRank(oneYear, metric.Return1Y))
	}
	return scores
}

func sortedMomentumValues(metrics []momentumMetric, value func(momentumMetric) float64) []float64 {
	values := make([]float64, 0, len(metrics))
	for _, metric := range metrics {
		values = append(values, value(metric))
	}
	sort.Float64s(values)
	return values
}

func percentileRank(sortedValues []float64, value float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	index := sort.Search(len(sortedValues), func(index int) bool { return sortedValues[index] > value })
	return float64(index) / float64(len(sortedValues))
}

func dedupeMomentumCandidates(candidates []momentumRankCandidate, limit int) []momentumRankCandidate {
	capacity := len(candidates)
	if limit > 0 && limit < capacity {
		capacity = limit
	}
	result := make([]momentumRankCandidate, 0, capacity)
	seen := make(map[string]struct{})
	for _, candidate := range candidates {
		key := canonicalFundName(candidate.Fund.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
		if limit > 0 && len(result) >= limit {
			break
		}
	}
	return result
}

func canonicalFundName(name string) string {
	trimmed := strings.ReplaceAll(strings.TrimSpace(name), " ", "")
	return shareClassSuffixPattern.ReplaceAllString(trimmed, "")
}

func (s *Service) evaluateMomentumCandidates(ctx context.Context, ranked []momentumRankCandidate, days, limit int) ([]model.MomentumPoolItem, int, []string) {
	items := make([]model.MomentumPoolItem, 0, limit)
	qualifiedCount := 0
	profileFailures := make([]string, 0)
	seenFunds := make(map[string]struct{})
	for _, candidate := range ranked {
		canonicalName := canonicalFundName(candidate.Fund.Name)
		if _, ok := seenFunds[canonicalName]; ok {
			continue
		}
		profile, err := s.loadMarketProfile(ctx, candidate.Fund, candidate.EstablishedDate, days)
		if err != nil {
			profileFailures = append(profileFailures, candidate.Fund.Code)
			continue
		}
		item, qualified := buildMomentumPoolItem(s.config.MomentumPool, profile, candidate.Score)
		if !qualified {
			continue
		}
		qualifiedCount++
		seenFunds[canonicalName] = struct{}{}
		items = append(items, item)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		if items[i].Return250D != items[j].Return250D {
			return items[i].Return250D > items[j].Return250D
		}
		if items[i].Return120D != items[j].Return120D {
			return items[i].Return120D > items[j].Return120D
		}
		if items[i].Return60D != items[j].Return60D {
			return items[i].Return60D > items[j].Return60D
		}
		return items[i].FundCode < items[j].FundCode
	})
	return items, qualifiedCount, profileFailures
}

func appendMomentumProfileFailureNote(report *model.MomentumPoolReport, scope string, codes []string) {
	if len(codes) == 0 {
		return
	}
	report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("%s有 %d 只基金画像重试后仍抓取失败，已暂时跳过：%s。", scope, len(codes), strings.Join(codes, "、")))
}

func buildMomentumPoolItem(cfg config.MomentumPoolConfig, profile *model.MarketFundProfile, score float64) (model.MomentumPoolItem, bool) {
	if profile == nil || profile.Latest == nil || len(profile.History) < 121 {
		return model.MomentumPoolItem{}, false
	}
	metrics := buildMarketPoolMetrics(profile.History)
	reason := fmt.Sprintf("相对动量 %.1f 分；20日 %.2f%%；60日 %.2f%%", score, metrics.Return20D*100, metrics.Return60D*100)
	return model.MomentumPoolItem{
		FundCode:         profile.Fund.Code,
		FundName:         profile.Fund.Name,
		FundType:         profile.Fund.FundType,
		MomentumScore:    score,
		Score:            score,
		Return20D:        metrics.Return20D,
		Return60D:        metrics.Return60D,
		Return120D:       metrics.Return120D,
		Return250D:       metrics.Return250D,
		MaxDrawdown120D:  metrics.MaxDrawdown120D,
		FundSizeYi:       profile.FundSizeYi,
		EstablishedYears: profile.EstablishedYears,
		LatestTradeDate:  profile.Latest.TradeDate,
		LatestNAV:        returnAdjustedNAV(*profile.Latest),
		Reason:           reason,
	}, true
}
