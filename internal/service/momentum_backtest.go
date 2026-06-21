package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/fetcher"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type momentumBacktestFund struct {
	Fund    model.MarketSearchFund
	History []model.FundSnapshot
}

type momentumBacktestCandidate struct {
	Fund    model.MarketSearchFund
	Score   float64
	Metrics marketPoolMetrics
	History []model.FundSnapshot
}

type momentumBacktestStageSeed struct {
	StartDate        time.Time
	EndDate          time.Time
	PeriodCount      int
	Return           float64
	ExcessReturn     float64
	Complete         bool
	PositiveBreadth  float64
	MedianReturn60D  float64
	MedianReturn120D float64
}

type momentumWeightProfile struct {
	Label      string
	Weight20D  float64
	Weight60D  float64
	Weight120D float64
	Weight250D float64
}

const momentumBacktestRoundTripCost = 0.004
const momentumBacktestUniverseLimit = 200
const momentumBacktestProfileWorkers = 12
const momentumCorrelationWindowDays = 60

func (s *Service) MomentumBacktest(ctx context.Context, days, rebalanceEvery, selectionCount int, universeSeed uint32) (*model.MomentumBacktestReport, error) {
	if days < 360 {
		days = 360
	}
	if rebalanceEvery <= 0 {
		rebalanceEvery = s.config.MomentumPool.ForwardValidationDays
	}
	if selectionCount <= 0 {
		selectionCount = s.config.MomentumPool.SelectionCount
	}
	funds, err := s.loadMarketFunds(ctx)
	if err != nil {
		return nil, err
	}
	rankings, err := s.loadMarketRankings(ctx)
	if err != nil {
		return nil, err
	}
	universe := buildFormalMomentumBacktestUniverse(funds, rankings, s.config.MomentumPool, maxInt(s.config.MomentumPool.CandidateLimit, momentumBacktestUniverseLimit), universeSeed)
	profiles := s.fetchMomentumBacktestProfiles(ctx, universe, days)
	report := runMomentumBacktest(profiles, s.config.MomentumPool, rebalanceEvery, selectionCount)
	report.Summary.UniverseSeed = universeSeed
	return report, nil
}

func (s *Service) MomentumBacktestSensitivity(ctx context.Context, days, rebalanceEvery, selectionCount, seeds int) (*model.MomentumBacktestSensitivityReport, error) {
	if seeds <= 0 {
		seeds = 5
	}
	if days < 360 {
		days = 360
	}
	if rebalanceEvery <= 0 {
		rebalanceEvery = s.config.MomentumPool.ForwardValidationDays
	}
	if selectionCount <= 0 {
		selectionCount = s.config.MomentumPool.SelectionCount
	}
	report := &model.MomentumBacktestSensitivityReport{Summary: model.MomentumBacktestSensitivitySummary{
		GeneratedAt:    time.Now().UTC(),
		RequestedSeeds: seeds,
		Days:           days,
		SelectionCount: selectionCount,
		RebalanceEvery: rebalanceEvery,
		Notes: []string{
			"多样本敏感性回测使用不同确定性候选宇宙，重点观察最差结果，避免只相信单一样本的漂亮回测。",
			"后半段超额为负的样本说明策略近期可能失效；真实执行仍必须通过前瞻验证。",
		},
	}}
	for seed := 0; seed < seeds; seed++ {
		result, err := s.MomentumBacktest(ctx, days, rebalanceEvery, selectionCount, uint32(seed))
		if err != nil {
			report.Items = append(report.Items, model.MomentumBacktestSensitivityItem{UniverseSeed: uint32(seed), Error: err.Error()})
			continue
		}
		summary := result.Summary
		if summary.PeriodCount == 0 {
			report.Items = append(report.Items, model.MomentumBacktestSensitivityItem{UniverseSeed: uint32(seed), CandidateCount: summary.CandidateCount, Error: "没有足够历史数据完成回测"})
			continue
		}
		report.Items = append(report.Items, model.MomentumBacktestSensitivityItem{
			UniverseSeed:       uint32(seed),
			CandidateCount:     summary.CandidateCount,
			TotalReturn:        summary.TotalReturn,
			ExcessReturn:       summary.ExcessReturn,
			RecentExcessReturn: summary.RecentExcessReturn,
			MaxDrawdown:        summary.MaxDrawdown,
			TopDecileHitRate:   summary.FutureTopDecileHitRate,
			TopDecileCapture:   summary.FutureTopDecileCapture,
			TopCaptureRate:     summary.FutureTopSelectionCapture,
			FuturePercentile:   summary.AverageFuturePercentile,
			RecentTopDecileHit: summary.RecentTopDecileHitRate,
		})
	}
	finalizeMomentumBacktestSensitivity(report)
	if report.Summary.CompletedSeeds == 0 {
		return report, fmt.Errorf("no momentum sensitivity samples completed")
	}
	return report, nil
}

func finalizeMomentumBacktestSensitivity(report *model.MomentumBacktestSensitivityReport) {
	completed := 0
	var totalReturnSum float64
	var excessReturnSum float64
	var topDecileHitRateSum float64
	var topDecileCaptureSum float64
	var topCaptureRateSum float64
	var futurePercentileSum float64
	var recentTopDecileHitSum float64
	for _, item := range report.Items {
		if item.Error != "" {
			continue
		}
		if completed == 0 {
			report.Summary.WorstTotalReturn = item.TotalReturn
			report.Summary.WorstExcessReturn = item.ExcessReturn
			report.Summary.WorstRecentExcessReturn = item.RecentExcessReturn
			report.Summary.WorstTopDecileHitRate = item.TopDecileHitRate
			report.Summary.WorstTopDecileCapture = item.TopDecileCapture
			report.Summary.WorstTopCaptureRate = item.TopCaptureRate
			report.Summary.WorstFuturePercentile = item.FuturePercentile
			report.Summary.WorstRecentTopDecileHit = item.RecentTopDecileHit
		}
		completed++
		totalReturnSum += item.TotalReturn
		excessReturnSum += item.ExcessReturn
		topDecileHitRateSum += item.TopDecileHitRate
		topDecileCaptureSum += item.TopDecileCapture
		topCaptureRateSum += item.TopCaptureRate
		futurePercentileSum += item.FuturePercentile
		recentTopDecileHitSum += item.RecentTopDecileHit
		report.Summary.WorstTotalReturn = math.Min(report.Summary.WorstTotalReturn, item.TotalReturn)
		report.Summary.WorstExcessReturn = math.Min(report.Summary.WorstExcessReturn, item.ExcessReturn)
		report.Summary.WorstRecentExcessReturn = math.Min(report.Summary.WorstRecentExcessReturn, item.RecentExcessReturn)
		report.Summary.WorstMaxDrawdown = math.Max(report.Summary.WorstMaxDrawdown, item.MaxDrawdown)
		report.Summary.WorstTopDecileHitRate = math.Min(report.Summary.WorstTopDecileHitRate, item.TopDecileHitRate)
		report.Summary.WorstTopDecileCapture = math.Min(report.Summary.WorstTopDecileCapture, item.TopDecileCapture)
		report.Summary.WorstTopCaptureRate = math.Min(report.Summary.WorstTopCaptureRate, item.TopCaptureRate)
		report.Summary.WorstFuturePercentile = math.Min(report.Summary.WorstFuturePercentile, item.FuturePercentile)
		report.Summary.WorstRecentTopDecileHit = math.Min(report.Summary.WorstRecentTopDecileHit, item.RecentTopDecileHit)
		if item.TotalReturn > 0 {
			report.Summary.PositiveTotalReturnSeeds++
		}
		if item.ExcessReturn > 0 {
			report.Summary.PositiveExcessReturnSeeds++
		}
		if item.RecentExcessReturn > 0 {
			report.Summary.PositiveRecentExcessSeeds++
		}
	}
	report.Summary.CompletedSeeds = completed
	report.Summary.FailedSeeds = report.Summary.RequestedSeeds - completed
	if completed > 0 {
		report.Summary.AverageTotalReturn = totalReturnSum / float64(completed)
		report.Summary.AverageExcessReturn = excessReturnSum / float64(completed)
		report.Summary.AverageTopDecileHitRate = topDecileHitRateSum / float64(completed)
		report.Summary.AverageTopDecileCapture = topDecileCaptureSum / float64(completed)
		report.Summary.AverageTopCaptureRate = topCaptureRateSum / float64(completed)
		report.Summary.AverageFuturePercentile = futurePercentileSum / float64(completed)
		report.Summary.AverageRecentTopDecileHit = recentTopDecileHitSum / float64(completed)
	}
}

func (s *Service) MomentumBacktestParameterGrid(ctx context.Context, days int, selectionCounts, rebalanceEveryValues []int, seeds int) (*model.MomentumBacktestParameterGridReport, error) {
	if days < 360 {
		days = 360
	}
	if seeds <= 0 {
		seeds = 5
	}
	currentSelectionCount := s.config.MomentumPool.SelectionCount
	currentRebalanceEvery := s.config.MomentumPool.ForwardValidationDays
	selectionCounts = normalizedParameterValues(selectionCounts, []int{currentSelectionCount, currentSelectionCount + 1, currentSelectionCount + 2}, 1)
	rebalanceEveryValues = normalizedParameterValues(rebalanceEveryValues, []int{currentRebalanceEvery - 5, currentRebalanceEvery, currentRebalanceEvery + 5}, 5)
	profilesBySeed, err := s.loadMomentumBacktestProfilesBySeed(ctx, days, seeds)
	if err != nil {
		return nil, err
	}
	report := &model.MomentumBacktestParameterGridReport{Summary: model.MomentumBacktestParameterGridSummary{
		GeneratedAt:           time.Now().UTC(),
		Days:                  days,
		Seeds:                 seeds,
		CurrentSelectionCount: currentSelectionCount,
		CurrentRebalanceEvery: currentRebalanceEvery,
		SelectionCounts:       selectionCounts,
		RebalanceEveryValues:  rebalanceEveryValues,
		Notes: []string{
			"参数排名优先比较近期前10%赢家命中率、前10%赢家覆盖率和未来收益百分位，再用执行收益稳健性约束。",
			"领先参数仅用于检查正式参数是否落在稳健区域，不应仅凭历史排名自动切换策略。",
		},
	}}
	for _, selectionCount := range selectionCounts {
		for _, rebalanceEvery := range rebalanceEveryValues {
			cfg := s.config.MomentumPool
			cfg.MinSelectionCount = selectionCount
			sensitivity := sensitivityFromMomentumProfiles(profilesBySeed, cfg, rebalanceEvery, selectionCount)
			item := model.MomentumBacktestParameterGridItem{SelectionCount: selectionCount, RebalanceEvery: rebalanceEvery}
			if sensitivity.Summary.CompletedSeeds == 0 {
				item.Error = "没有足够历史数据完成回测"
				report.Items = append(report.Items, item)
				continue
			}
			summary := sensitivity.Summary
			item.CompletedSeeds = summary.CompletedSeeds
			item.PositiveTotalReturnSeeds = summary.PositiveTotalReturnSeeds
			item.PositiveExcessReturnSeeds = summary.PositiveExcessReturnSeeds
			item.PositiveRecentExcessSeeds = summary.PositiveRecentExcessSeeds
			item.AverageTotalReturn = summary.AverageTotalReturn
			item.AverageExcessReturn = summary.AverageExcessReturn
			item.WorstTotalReturn = summary.WorstTotalReturn
			item.WorstExcessReturn = summary.WorstExcessReturn
			item.WorstRecentExcessReturn = summary.WorstRecentExcessReturn
			item.WorstMaxDrawdown = summary.WorstMaxDrawdown
			item.AverageTopDecileHitRate = summary.AverageTopDecileHitRate
			item.WorstTopDecileHitRate = summary.WorstTopDecileHitRate
			item.AverageTopDecileCapture = summary.AverageTopDecileCapture
			item.WorstTopDecileCapture = summary.WorstTopDecileCapture
			item.AverageTopCaptureRate = summary.AverageTopCaptureRate
			item.WorstTopCaptureRate = summary.WorstTopCaptureRate
			item.AverageFuturePercentile = summary.AverageFuturePercentile
			item.WorstFuturePercentile = summary.WorstFuturePercentile
			item.AverageRecentTopDecileHit = summary.AverageRecentTopDecileHit
			item.WorstRecentTopDecileHit = summary.WorstRecentTopDecileHit
			report.Items = append(report.Items, item)
		}
	}
	finalizeMomentumBacktestParameterGrid(report)
	if report.Summary.CombinationCount == 0 {
		return report, fmt.Errorf("no momentum parameter grid combinations completed")
	}
	return report, nil
}

func normalizedParameterValues(values, defaults []int, minimum int) []int {
	if len(values) == 0 {
		values = defaults
	}
	seen := make(map[int]struct{}, len(values))
	result := make([]int, 0, len(values))
	for _, value := range values {
		if value < minimum {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Ints(result)
	return result
}

func finalizeMomentumBacktestParameterGrid(report *model.MomentumBacktestParameterGridReport) {
	sort.SliceStable(report.Items, func(i, j int) bool {
		return preferMomentumParameterGridItem(report.Items[i], report.Items[j])
	})
	completed := 0
	for index := range report.Items {
		item := &report.Items[index]
		if item.Error != "" {
			continue
		}
		completed++
		item.Rank = completed
		if completed == 1 {
			report.Summary.LeadingSelectionCount = item.SelectionCount
			report.Summary.LeadingRebalanceEvery = item.RebalanceEvery
		}
		if item.SelectionCount == report.Summary.CurrentSelectionCount && item.RebalanceEvery == report.Summary.CurrentRebalanceEvery {
			report.Summary.CurrentRank = item.Rank
		}
	}
	report.Summary.CombinationCount = completed
}

func preferMomentumParameterGridItem(left, right model.MomentumBacktestParameterGridItem) bool {
	if left.Error != "" || right.Error != "" {
		return left.Error == "" && right.Error != ""
	}
	if left.AverageRecentTopDecileHit != right.AverageRecentTopDecileHit {
		return left.AverageRecentTopDecileHit > right.AverageRecentTopDecileHit
	}
	if left.WorstRecentTopDecileHit != right.WorstRecentTopDecileHit {
		return left.WorstRecentTopDecileHit > right.WorstRecentTopDecileHit
	}
	if left.AverageTopDecileCapture != right.AverageTopDecileCapture {
		return left.AverageTopDecileCapture > right.AverageTopDecileCapture
	}
	if left.WorstTopDecileCapture != right.WorstTopDecileCapture {
		return left.WorstTopDecileCapture > right.WorstTopDecileCapture
	}
	if left.AverageFuturePercentile != right.AverageFuturePercentile {
		return left.AverageFuturePercentile > right.AverageFuturePercentile
	}
	if left.PositiveRecentExcessSeeds != right.PositiveRecentExcessSeeds {
		return left.PositiveRecentExcessSeeds > right.PositiveRecentExcessSeeds
	}
	if left.WorstRecentExcessReturn != right.WorstRecentExcessReturn {
		return left.WorstRecentExcessReturn > right.WorstRecentExcessReturn
	}
	if left.WorstExcessReturn != right.WorstExcessReturn {
		return left.WorstExcessReturn > right.WorstExcessReturn
	}
	if left.WorstTotalReturn != right.WorstTotalReturn {
		return left.WorstTotalReturn > right.WorstTotalReturn
	}
	if left.WorstMaxDrawdown != right.WorstMaxDrawdown {
		return left.WorstMaxDrawdown < right.WorstMaxDrawdown
	}
	if left.SelectionCount != right.SelectionCount {
		return left.SelectionCount < right.SelectionCount
	}
	return left.RebalanceEvery < right.RebalanceEvery
}

func (s *Service) MomentumBacktestStageRobustness(ctx context.Context, days, rebalanceEvery, selectionCount, seeds int) (*model.MomentumBacktestStageReport, error) {
	if days < 360 {
		days = 360
	}
	if seeds <= 0 {
		seeds = 5
	}
	if rebalanceEvery <= 0 {
		rebalanceEvery = s.config.MomentumPool.ForwardValidationDays
	}
	if selectionCount <= 0 {
		selectionCount = s.config.MomentumPool.SelectionCount
	}
	stages := make(map[string][]momentumBacktestStageSeed)
	for seed := 0; seed < seeds; seed++ {
		backtest, err := s.MomentumBacktest(ctx, days, rebalanceEvery, selectionCount, uint32(seed))
		if err != nil || len(backtest.Periods) == 0 {
			continue
		}
		for label, stage := range momentumBacktestStagesByYear(backtest.Periods) {
			stages[label] = append(stages[label], stage)
		}
	}
	report := &model.MomentumBacktestStageReport{Summary: model.MomentumBacktestStageSummary{
		GeneratedAt:    time.Now().UTC(),
		Days:           days,
		Seeds:          seeds,
		SelectionCount: selectionCount,
		RebalanceEvery: rebalanceEvery,
		Notes: []string{
			"阶段稳健性按自然年汇总不同候选宇宙的复合收益；完整年度用于总体判断，首尾不足一年的阶段只展示不计入完整年度结论。",
			"历史阶段表现只用于识别策略失效区间，真实执行仍必须通过前瞻验证。",
		},
	}}
	labels := make([]string, 0, len(stages))
	for label := range stages {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		report.Items = append(report.Items, buildMomentumBacktestStageItem(label, stages[label], seeds))
	}
	finalizeMomentumBacktestStages(report)
	if len(report.Items) == 0 {
		return report, fmt.Errorf("no momentum stage samples completed")
	}
	return report, nil
}

func momentumBacktestStagesByYear(periods []model.MomentumBacktestPeriod) map[string]momentumBacktestStageSeed {
	type stageValues struct {
		startDate           time.Time
		endDate             time.Time
		periodCount         int
		strategyValue       float64
		benchmarkValue      float64
		positiveBreadthSum  float64
		medianReturn60DSum  float64
		medianReturn120DSum float64
	}
	values := make(map[string]stageValues)
	for _, period := range periods {
		label := period.StartDate.Format("2006")
		stage := values[label]
		if stage.periodCount == 0 {
			stage.startDate = period.StartDate
			stage.strategyValue = 1
			stage.benchmarkValue = 1
		}
		stage.endDate = period.EndDate
		stage.periodCount++
		stage.strategyValue *= 1 + period.Return
		stage.benchmarkValue *= 1 + period.BenchmarkReturn
		stage.positiveBreadthSum += period.PositiveBreadth
		stage.medianReturn60DSum += period.MedianReturn60D
		stage.medianReturn120DSum += period.MedianReturn120D
		values[label] = stage
	}
	result := make(map[string]momentumBacktestStageSeed, len(values))
	for label, stage := range values {
		result[label] = momentumBacktestStageSeed{
			StartDate:        stage.startDate,
			EndDate:          stage.endDate,
			PeriodCount:      stage.periodCount,
			Return:           stage.strategyValue - 1,
			ExcessReturn:     stage.strategyValue - stage.benchmarkValue,
			Complete:         stage.endDate.Sub(stage.startDate) >= 300*24*time.Hour,
			PositiveBreadth:  stage.positiveBreadthSum / float64(stage.periodCount),
			MedianReturn60D:  stage.medianReturn60DSum / float64(stage.periodCount),
			MedianReturn120D: stage.medianReturn120DSum / float64(stage.periodCount),
		}
	}
	return result
}

func buildMomentumBacktestStageItem(label string, stages []momentumBacktestStageSeed, requestedSeeds int) model.MomentumBacktestStageItem {
	item := model.MomentumBacktestStageItem{StageLabel: label, SeedCount: len(stages)}
	var returnSum float64
	var excessSum float64
	completeSeeds := 0
	for index, stage := range stages {
		if index == 0 || stage.StartDate.Before(item.StartDate) {
			item.StartDate = stage.StartDate
		}
		if stage.EndDate.After(item.EndDate) {
			item.EndDate = stage.EndDate
		}
		item.PeriodCount += stage.PeriodCount
		returnSum += stage.Return
		excessSum += stage.ExcessReturn
		item.AveragePositiveBreadth += stage.PositiveBreadth
		item.AverageMedianReturn60D += stage.MedianReturn60D
		item.AverageMedianReturn120D += stage.MedianReturn120D
		if index == 0 || stage.Return < item.WorstReturn {
			item.WorstReturn = stage.Return
		}
		if index == 0 || stage.ExcessReturn < item.WorstExcessReturn {
			item.WorstExcessReturn = stage.ExcessReturn
		}
		if index == 0 || stage.ExcessReturn > item.BestExcessReturn {
			item.BestExcessReturn = stage.ExcessReturn
		}
		if stage.Return > 0 {
			item.PositiveReturnSeeds++
		}
		if stage.ExcessReturn > 0 {
			item.PositiveExcessSeeds++
		}
		if stage.Complete {
			completeSeeds++
		}
	}
	if len(stages) > 0 {
		item.AverageReturn = returnSum / float64(len(stages))
		item.AverageExcessReturn = excessSum / float64(len(stages))
		item.AveragePositiveBreadth /= float64(len(stages))
		item.AverageMedianReturn60D /= float64(len(stages))
		item.AverageMedianReturn120D /= float64(len(stages))
	}
	item.Complete = len(stages) == requestedSeeds && completeSeeds == requestedSeeds
	return item
}

func finalizeMomentumBacktestStages(report *model.MomentumBacktestStageReport) {
	report.Summary.StageCount = len(report.Items)
	firstComplete := true
	for _, item := range report.Items {
		if !item.Complete {
			continue
		}
		report.Summary.CompleteStageCount++
		if item.AverageExcessReturn > 0 {
			report.Summary.PositiveAverageExcessStages++
		}
		if item.PositiveExcessSeeds == item.SeedCount {
			report.Summary.PositiveAllSeedExcessStages++
		}
		if firstComplete || item.AverageExcessReturn < report.Summary.WorstCompleteAverageExcess {
			report.Summary.WorstCompleteStage = item.StageLabel
			report.Summary.WorstCompleteAverageExcess = item.AverageExcessReturn
		}
		if firstComplete || item.WorstExcessReturn < report.Summary.WorstCompleteSeedExcess {
			report.Summary.WorstCompleteSeedExcess = item.WorstExcessReturn
		}
		firstComplete = false
	}
}

func (s *Service) MomentumBacktestEntryGrid(ctx context.Context, days, seeds int, breadthValues, minReturn20DValues, minReturn60DValues []float64) (*model.MomentumBacktestEntryGridReport, error) {
	if days < 360 {
		days = 360
	}
	if seeds <= 0 {
		seeds = 5
	}
	base := s.config.MomentumPool
	breadthValues = normalizedFloatValues(breadthValues, []float64{base.MinPositiveBreadth, 0.30})
	minReturn20DValues = normalizedFloatValues(minReturn20DValues, []float64{base.MinReturn20D, -0.05, 0})
	minReturn60DValues = normalizedFloatValues(minReturn60DValues, []float64{base.MinReturn60D, 0.03})
	profilesBySeed, err := s.loadMomentumBacktestProfilesBySeed(ctx, days, seeds)
	if err != nil {
		return nil, err
	}
	report := &model.MomentumBacktestEntryGridReport{Summary: model.MomentumBacktestEntryGridSummary{
		GeneratedAt:               time.Now().UTC(),
		Days:                      days,
		Seeds:                     seeds,
		CurrentMinPositiveBreadth: base.MinPositiveBreadth,
		CurrentMinReturn20D:       base.MinReturn20D,
		CurrentMinReturn60D:       base.MinReturn60D,
		Notes: []string{
			"入场门槛网格只提高最低趋势要求，不限制高涨幅基金；高涨幅 AI/科技基金仍可正常优先入选。",
			"排名优先比较近期超额为正样本、最差近期超额、最差整体超额和最差总收益。",
		},
	}}
	for _, breadth := range breadthValues {
		for _, minReturn20D := range minReturn20DValues {
			for _, minReturn60D := range minReturn60DValues {
				cfg := base
				cfg.MinPositiveBreadth = breadth
				cfg.MinReturn20D = minReturn20D
				cfg.MinReturn60D = minReturn60D
				sensitivity := sensitivityFromMomentumProfiles(profilesBySeed, cfg, cfg.ForwardValidationDays, cfg.SelectionCount)
				item := model.MomentumBacktestEntryGridItem{MinPositiveBreadth: breadth, MinReturn20D: minReturn20D, MinReturn60D: minReturn60D}
				if sensitivity.Summary.CompletedSeeds == 0 {
					item.Error = "没有足够历史数据完成回测"
				} else {
					summary := sensitivity.Summary
					item.CompletedSeeds = summary.CompletedSeeds
					item.PositiveTotalReturnSeeds = summary.PositiveTotalReturnSeeds
					item.PositiveExcessReturnSeeds = summary.PositiveExcessReturnSeeds
					item.PositiveRecentExcessSeeds = summary.PositiveRecentExcessSeeds
					item.AverageTotalReturn = summary.AverageTotalReturn
					item.AverageExcessReturn = summary.AverageExcessReturn
					item.WorstTotalReturn = summary.WorstTotalReturn
					item.WorstExcessReturn = summary.WorstExcessReturn
					item.WorstRecentExcessReturn = summary.WorstRecentExcessReturn
					item.WorstMaxDrawdown = summary.WorstMaxDrawdown
				}
				report.Items = append(report.Items, item)
			}
		}
	}
	finalizeMomentumBacktestEntryGrid(report)
	if report.Summary.CombinationCount == 0 {
		return report, fmt.Errorf("no momentum entry grid combinations completed")
	}
	return report, nil
}

func (s *Service) loadMomentumBacktestProfilesBySeed(ctx context.Context, days, seeds int) ([][]momentumBacktestFund, error) {
	funds, err := s.loadMarketFunds(ctx)
	if err != nil {
		return nil, err
	}
	rankings, err := s.loadMarketRankings(ctx)
	if err != nil {
		return nil, err
	}
	result := make([][]momentumBacktestFund, seeds)
	for seed := 0; seed < seeds; seed++ {
		universe := buildFormalMomentumBacktestUniverse(funds, rankings, s.config.MomentumPool, maxInt(s.config.MomentumPool.CandidateLimit, momentumBacktestUniverseLimit), uint32(seed))
		result[seed] = s.fetchMomentumBacktestProfiles(ctx, universe, days)
	}
	return result, nil
}

func sensitivityFromMomentumProfiles(profilesBySeed [][]momentumBacktestFund, cfg config.MomentumPoolConfig, rebalanceEvery, selectionCount int) *model.MomentumBacktestSensitivityReport {
	return sensitivityFromMomentumProfilesWithCorrelation(profilesBySeed, cfg, rebalanceEvery, selectionCount, 0)
}

func sensitivityFromMomentumProfilesWithCorrelation(profilesBySeed [][]momentumBacktestFund, cfg config.MomentumPoolConfig, rebalanceEvery, selectionCount int, maxCorrelation float64) *model.MomentumBacktestSensitivityReport {
	report := &model.MomentumBacktestSensitivityReport{Summary: model.MomentumBacktestSensitivitySummary{RequestedSeeds: len(profilesBySeed), SelectionCount: selectionCount, RebalanceEvery: rebalanceEvery}}
	for seed, profiles := range profilesBySeed {
		backtest := runMomentumBacktestWithCorrelation(profiles, cfg, rebalanceEvery, selectionCount, maxCorrelation)
		if backtest.Summary.PeriodCount == 0 {
			report.Items = append(report.Items, model.MomentumBacktestSensitivityItem{UniverseSeed: uint32(seed), Error: "没有足够历史数据完成回测"})
			continue
		}
		summary := backtest.Summary
		report.Items = append(report.Items, model.MomentumBacktestSensitivityItem{UniverseSeed: uint32(seed), CandidateCount: summary.CandidateCount, TotalReturn: summary.TotalReturn, ExcessReturn: summary.ExcessReturn, RecentExcessReturn: summary.RecentExcessReturn, MaxDrawdown: summary.MaxDrawdown, TopDecileHitRate: summary.FutureTopDecileHitRate, TopDecileCapture: summary.FutureTopDecileCapture, TopCaptureRate: summary.FutureTopSelectionCapture, FuturePercentile: summary.AverageFuturePercentile, RecentTopDecileHit: summary.RecentTopDecileHitRate})
	}
	finalizeMomentumBacktestSensitivity(report)
	return report
}

func (s *Service) MomentumBacktestCorrelationGrid(ctx context.Context, days, seeds int, correlationValues []float64) (*model.MomentumBacktestCorrelationGridReport, error) {
	if days < 360 {
		days = 360
	}
	if seeds <= 0 {
		seeds = 5
	}
	correlationValues = normalizedFloatValues(correlationValues, []float64{0, 0.90, 0.95, 0.98})
	profilesBySeed, err := s.loadMomentumBacktestProfilesBySeed(ctx, days, seeds)
	if err != nil {
		return nil, err
	}
	base := s.config.MomentumPool
	report := &model.MomentumBacktestCorrelationGridReport{Summary: model.MomentumBacktestCorrelationGridSummary{
		GeneratedAt:           time.Now().UTC(),
		Days:                  days,
		Seeds:                 seeds,
		CorrelationWindowDays: momentumCorrelationWindowDays,
		Notes: []string{
			"相关性阈值 0 表示正式策略：不自动去重；其他阈值只用于历史诊断，不会写入正式配置。",
			"每个历史调仓日只使用当时之前最近60个共同交易日收益计算相关性，避免未来数据泄漏。",
			"相关性去重仍允许高涨幅基金入选，只在已选基金高度同涨同跌时尝试下一个强势候选。",
		},
	}}
	for _, maxCorrelation := range correlationValues {
		sensitivity := sensitivityFromMomentumProfilesWithCorrelation(profilesBySeed, base, base.ForwardValidationDays, base.SelectionCount, maxCorrelation)
		item := correlationGridItem(maxCorrelation, sensitivity.Summary)
		applyCorrelationGrid2024Stage(&item, profilesBySeed, base)
		report.Items = append(report.Items, item)
	}
	finalizeMomentumBacktestCorrelationGrid(report)
	if report.Summary.CombinationCount == 0 {
		return report, fmt.Errorf("no momentum correlation grid combinations completed")
	}
	return report, nil
}

func (s *Service) MomentumBacktestWeightGrid(ctx context.Context, days, seeds int) (*model.MomentumBacktestWeightGridReport, error) {
	if days < 360 {
		days = 360
	}
	if seeds <= 0 {
		seeds = 5
	}
	profilesBySeed, err := s.loadMomentumBacktestProfilesBySeed(ctx, days, seeds)
	if err != nil {
		return nil, err
	}
	base := s.config.MomentumPool
	profiles := momentumWeightProfiles(base)
	report := &model.MomentumBacktestWeightGridReport{Summary: model.MomentumBacktestWeightGridSummary{
		GeneratedAt: time.Now().UTC(), Days: days, Seeds: seeds, ProfileCount: len(profiles),
		Notes: []string{
			"未来赢家捕获率以每个历史调仓日之后的真实收益计算，只用于评价排序能力，不参与当时选基，避免未来数据泄漏。",
			"未来前10%命中率衡量入选基金中有多少进入下一周期候选宇宙前10%；未来收益百分位越高越接近真正赢家。",
			"权重网格不会惩罚高涨幅基金，也不会自动修改正式策略；必须同时兼顾赢家捕获、近期超额和最差收益。",
		},
	}}
	for _, profile := range profiles {
		cfg := base
		cfg.MomentumWeight20D = profile.Weight20D
		cfg.MomentumWeight60D = profile.Weight60D
		cfg.MomentumWeight120D = profile.Weight120D
		cfg.MomentumWeight250D = profile.Weight250D
		report.Items = append(report.Items, buildMomentumWeightGridItem(profile, profilesBySeed, cfg))
	}
	finalizeMomentumBacktestWeightGrid(report, base)
	return report, nil
}

func momentumWeightProfiles(base config.MomentumPoolConfig) []momentumWeightProfile {
	profiles := []momentumWeightProfile{
		{Label: "正式权重", Weight20D: base.MomentumWeight20D, Weight60D: base.MomentumWeight60D, Weight120D: base.MomentumWeight120D, Weight250D: base.MomentumWeight250D},
		{Label: "短中期加速", Weight20D: 0.25, Weight60D: 0.40, Weight120D: 0.25, Weight250D: 0.10},
		{Label: "60日主导", Weight20D: 0.15, Weight60D: 0.50, Weight120D: 0.25, Weight250D: 0.10},
		{Label: "120日主导", Weight20D: 0.10, Weight60D: 0.25, Weight120D: 0.50, Weight250D: 0.15},
		{Label: "轻度120日倾斜", Weight20D: 0.15, Weight60D: 0.25, Weight120D: 0.40, Weight250D: 0.20},
		{Label: "中度120日倾斜", Weight20D: 0.10, Weight60D: 0.30, Weight120D: 0.45, Weight250D: 0.15},
		{Label: "120日倾斜保留长趋势", Weight20D: 0.10, Weight60D: 0.25, Weight120D: 0.45, Weight250D: 0.20},
		{Label: "120日倾斜保留短趋势", Weight20D: 0.15, Weight60D: 0.25, Weight120D: 0.45, Weight250D: 0.15},
		{Label: "均衡动量", Weight20D: 0.25, Weight60D: 0.25, Weight120D: 0.25, Weight250D: 0.25},
		{Label: "长周期趋势", Weight20D: 0.05, Weight60D: 0.20, Weight120D: 0.35, Weight250D: 0.40},
	}
	result := make([]momentumWeightProfile, 0, len(profiles))
	for _, profile := range profiles {
		duplicate := false
		for _, existing := range result {
			if almostEqual(profile.Weight20D, existing.Weight20D) && almostEqual(profile.Weight60D, existing.Weight60D) && almostEqual(profile.Weight120D, existing.Weight120D) && almostEqual(profile.Weight250D, existing.Weight250D) {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, profile)
		}
	}
	return result
}

func buildMomentumWeightGridItem(profile momentumWeightProfile, profilesBySeed [][]momentumBacktestFund, cfg config.MomentumPoolConfig) model.MomentumBacktestWeightGridItem {
	item := model.MomentumBacktestWeightGridItem{Label: profile.Label, Weight20D: profile.Weight20D, Weight60D: profile.Weight60D, Weight120D: profile.Weight120D, Weight250D: profile.Weight250D}
	for _, funds := range profilesBySeed {
		backtest := runMomentumBacktest(funds, cfg, cfg.ForwardValidationDays, cfg.SelectionCount)
		if backtest.Summary.PeriodCount == 0 {
			continue
		}
		summary := backtest.Summary
		if item.CompletedSeeds == 0 {
			item.WorstTotalReturn = summary.TotalReturn
			item.WorstExcessReturn = summary.ExcessReturn
			item.WorstRecentExcessReturn = summary.RecentExcessReturn
			item.WorstTopDecileHitRate = summary.FutureTopDecileHitRate
			item.WorstRecentTopDecileHitRate = summary.RecentTopDecileHitRate
		}
		item.CompletedSeeds++
		item.AverageTotalReturn += summary.TotalReturn
		item.AverageExcessReturn += summary.ExcessReturn
		item.AverageTopDecileHitRate += summary.FutureTopDecileHitRate
		item.AverageRecentTopDecileHitRate += summary.RecentTopDecileHitRate
		item.AverageFuturePercentile += summary.AverageFuturePercentile
		item.WorstTotalReturn = math.Min(item.WorstTotalReturn, summary.TotalReturn)
		item.WorstExcessReturn = math.Min(item.WorstExcessReturn, summary.ExcessReturn)
		item.WorstRecentExcessReturn = math.Min(item.WorstRecentExcessReturn, summary.RecentExcessReturn)
		item.WorstMaxDrawdown = math.Max(item.WorstMaxDrawdown, summary.MaxDrawdown)
		item.WorstTopDecileHitRate = math.Min(item.WorstTopDecileHitRate, summary.FutureTopDecileHitRate)
		item.WorstRecentTopDecileHitRate = math.Min(item.WorstRecentTopDecileHitRate, summary.RecentTopDecileHitRate)
		if summary.RecentExcessReturn > 0 {
			item.PositiveRecentExcessSeeds++
		}
	}
	if item.CompletedSeeds > 0 {
		denominator := float64(item.CompletedSeeds)
		item.AverageTotalReturn /= denominator
		item.AverageExcessReturn /= denominator
		item.AverageTopDecileHitRate /= denominator
		item.AverageRecentTopDecileHitRate /= denominator
		item.AverageFuturePercentile /= denominator
	} else {
		item.Error = "没有足够历史数据完成回测"
	}
	return item
}

func finalizeMomentumBacktestWeightGrid(report *model.MomentumBacktestWeightGridReport, base config.MomentumPoolConfig) {
	sort.SliceStable(report.Items, func(i, j int) bool { return preferMomentumWeightGridItem(report.Items[i], report.Items[j]) })
	for index := range report.Items {
		item := &report.Items[index]
		item.Rank = index + 1
		if index == 0 {
			report.Summary.LeadingLabel = item.Label
		}
		if almostEqual(item.Weight20D, base.MomentumWeight20D) && almostEqual(item.Weight60D, base.MomentumWeight60D) && almostEqual(item.Weight120D, base.MomentumWeight120D) && almostEqual(item.Weight250D, base.MomentumWeight250D) {
			report.Summary.CurrentRank = item.Rank
		}
	}
}

func preferMomentumWeightGridItem(left, right model.MomentumBacktestWeightGridItem) bool {
	if left.Error != "" || right.Error != "" {
		return left.Error == "" && right.Error != ""
	}
	if left.PositiveRecentExcessSeeds != right.PositiveRecentExcessSeeds {
		return left.PositiveRecentExcessSeeds > right.PositiveRecentExcessSeeds
	}
	if left.WorstRecentExcessReturn != right.WorstRecentExcessReturn {
		return left.WorstRecentExcessReturn > right.WorstRecentExcessReturn
	}
	if left.AverageRecentTopDecileHitRate != right.AverageRecentTopDecileHitRate {
		return left.AverageRecentTopDecileHitRate > right.AverageRecentTopDecileHitRate
	}
	if left.WorstRecentTopDecileHitRate != right.WorstRecentTopDecileHitRate {
		return left.WorstRecentTopDecileHitRate > right.WorstRecentTopDecileHitRate
	}
	if left.WorstExcessReturn != right.WorstExcessReturn {
		return left.WorstExcessReturn > right.WorstExcessReturn
	}
	return left.WorstTotalReturn > right.WorstTotalReturn
}

func correlationGridItem(maxCorrelation float64, summary model.MomentumBacktestSensitivitySummary) model.MomentumBacktestCorrelationGridItem {
	return model.MomentumBacktestCorrelationGridItem{
		MaxCorrelation: maxCorrelation, CompletedSeeds: summary.CompletedSeeds, PositiveTotalReturnSeeds: summary.PositiveTotalReturnSeeds,
		PositiveExcessReturnSeeds: summary.PositiveExcessReturnSeeds, PositiveRecentExcessSeeds: summary.PositiveRecentExcessSeeds,
		AverageTotalReturn: summary.AverageTotalReturn, AverageExcessReturn: summary.AverageExcessReturn, WorstTotalReturn: summary.WorstTotalReturn,
		WorstExcessReturn: summary.WorstExcessReturn, WorstRecentExcessReturn: summary.WorstRecentExcessReturn, WorstMaxDrawdown: summary.WorstMaxDrawdown,
	}
}

func applyCorrelationGrid2024Stage(item *model.MomentumBacktestCorrelationGridItem, profilesBySeed [][]momentumBacktestFund, cfg config.MomentumPoolConfig) {
	for _, profiles := range profilesBySeed {
		backtest := runMomentumBacktestWithCorrelation(profiles, cfg, cfg.ForwardValidationDays, cfg.SelectionCount, item.MaxCorrelation)
		stage, ok := momentumBacktestStagesByYear(backtest.Periods)["2024"]
		if !ok || !stage.Complete {
			continue
		}
		if item.Stage2024Seeds == 0 {
			item.Worst2024ExcessReturn = stage.ExcessReturn
		}
		item.Stage2024Seeds++
		item.Average2024ExcessReturn += stage.ExcessReturn
		item.Worst2024ExcessReturn = math.Min(item.Worst2024ExcessReturn, stage.ExcessReturn)
		if stage.ExcessReturn > 0 {
			item.Positive2024ExcessSeeds++
		}
	}
	if item.Stage2024Seeds > 0 {
		item.Average2024ExcessReturn /= float64(item.Stage2024Seeds)
	}
}

func finalizeMomentumBacktestCorrelationGrid(report *model.MomentumBacktestCorrelationGridReport) {
	sort.SliceStable(report.Items, func(i, j int) bool { return preferMomentumCorrelationGridItem(report.Items[i], report.Items[j]) })
	for index := range report.Items {
		item := &report.Items[index]
		if item.CompletedSeeds == 0 {
			continue
		}
		report.Summary.CombinationCount++
		item.Rank = report.Summary.CombinationCount
		if item.Rank == 1 {
			report.Summary.LeadingMaxCorrelation = item.MaxCorrelation
		}
		if item.MaxCorrelation == 0 {
			report.Summary.CurrentRank = item.Rank
		}
	}
}

func preferMomentumCorrelationGridItem(left, right model.MomentumBacktestCorrelationGridItem) bool {
	if left.CompletedSeeds == 0 || right.CompletedSeeds == 0 {
		return left.CompletedSeeds > right.CompletedSeeds
	}
	if left.PositiveRecentExcessSeeds != right.PositiveRecentExcessSeeds {
		return left.PositiveRecentExcessSeeds > right.PositiveRecentExcessSeeds
	}
	if left.WorstRecentExcessReturn != right.WorstRecentExcessReturn {
		return left.WorstRecentExcessReturn > right.WorstRecentExcessReturn
	}
	if left.WorstExcessReturn != right.WorstExcessReturn {
		return left.WorstExcessReturn > right.WorstExcessReturn
	}
	if left.WorstTotalReturn != right.WorstTotalReturn {
		return left.WorstTotalReturn > right.WorstTotalReturn
	}
	if left.WorstMaxDrawdown != right.WorstMaxDrawdown {
		return left.WorstMaxDrawdown < right.WorstMaxDrawdown
	}
	return left.MaxCorrelation < right.MaxCorrelation
}

func normalizedFloatValues(values, defaults []float64) []float64 {
	if len(values) == 0 {
		values = defaults
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]float64, 0, len(values))
	for _, value := range values {
		key := fmt.Sprintf("%.8f", value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, value)
	}
	sort.Float64s(result)
	return result
}

func finalizeMomentumBacktestEntryGrid(report *model.MomentumBacktestEntryGridReport) {
	sort.SliceStable(report.Items, func(i, j int) bool {
		left := report.Items[i]
		right := report.Items[j]
		if left.Error != "" || right.Error != "" {
			return left.Error == "" && right.Error != ""
		}
		if left.PositiveRecentExcessSeeds != right.PositiveRecentExcessSeeds {
			return left.PositiveRecentExcessSeeds > right.PositiveRecentExcessSeeds
		}
		if left.WorstRecentExcessReturn != right.WorstRecentExcessReturn {
			return left.WorstRecentExcessReturn > right.WorstRecentExcessReturn
		}
		if left.WorstExcessReturn != right.WorstExcessReturn {
			return left.WorstExcessReturn > right.WorstExcessReturn
		}
		if left.WorstTotalReturn != right.WorstTotalReturn {
			return left.WorstTotalReturn > right.WorstTotalReturn
		}
		return left.WorstMaxDrawdown < right.WorstMaxDrawdown
	})
	completed := 0
	for index := range report.Items {
		item := &report.Items[index]
		if item.Error != "" {
			continue
		}
		completed++
		item.Rank = completed
		if completed == 1 {
			report.Summary.LeadingMinPositiveBreadth = item.MinPositiveBreadth
			report.Summary.LeadingMinReturn20D = item.MinReturn20D
			report.Summary.LeadingMinReturn60D = item.MinReturn60D
		}
		if almostEqual(item.MinPositiveBreadth, report.Summary.CurrentMinPositiveBreadth) && almostEqual(item.MinReturn20D, report.Summary.CurrentMinReturn20D) && almostEqual(item.MinReturn60D, report.Summary.CurrentMinReturn60D) {
			report.Summary.CurrentRank = item.Rank
		}
	}
	report.Summary.CombinationCount = completed
}

func almostEqual(left, right float64) bool {
	return math.Abs(left-right) < 0.0000001
}

func buildMomentumBacktestUniverse(funds []model.MarketSearchFund, rankings []fetcher.MarketRankEntry, limit int, universeSeed uint32) []momentumRankCandidate {
	establishedByCode := make(map[string]time.Time, len(rankings))
	for _, entry := range rankings {
		establishedByCode[entry.Code] = entry.EstablishedDate
	}
	candidates := make([]momentumRankCandidate, 0, len(funds))
	seen := make(map[string]struct{})
	for _, fund := range funds {
		name := canonicalFundName(fund.Name)
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		candidates = append(candidates, momentumRankCandidate{Fund: fund, EstablishedDate: establishedByCode[fund.Code]})
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := momentumUniverseHash(universeSeed, candidates[i].Fund.Code)
		right := momentumUniverseHash(universeSeed, candidates[j].Fund.Code)
		if left != right {
			return left < right
		}
		return candidates[i].Fund.Code < candidates[j].Fund.Code
	})
	if limit > 0 && len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates
}

func buildFormalMomentumBacktestUniverse(funds []model.MarketSearchFund, rankings []fetcher.MarketRankEntry, cfg config.MomentumPoolConfig, limit int, universeSeed uint32) []momentumRankCandidate {
	sampled := buildMomentumBacktestUniverse(filterMomentumFunds(funds, cfg.AllowedCompanies), rankings, limit, universeSeed)
	formal := make([]momentumRankCandidate, 0, len(sampled))
	for _, candidate := range sampled {
		if isAllowedFundCompany(candidate.Fund.Name, cfg.AllowedCompanies) {
			formal = append(formal, candidate)
		}
	}
	return formal
}

func momentumUniverseHash(seed uint32, code string) uint64 {
	if seed == 0 {
		return uint64(crc32.ChecksumIEEE([]byte(code)))
	}
	payload := make([]byte, 4+len(code))
	binary.BigEndian.PutUint32(payload, seed)
	copy(payload[4:], code)
	digest := sha256.Sum256(payload)
	return binary.BigEndian.Uint64(digest[:8])
}

func (s *Service) fetchMomentumBacktestProfiles(ctx context.Context, candidates []momentumRankCandidate, days int) []momentumBacktestFund {
	type profileResult struct {
		index   int
		profile momentumBacktestFund
		valid   bool
	}
	jobs := make(chan int)
	results := make(chan profileResult, len(candidates))
	minimumHistory := minimumMomentumBacktestHistory(days)
	workerCount := minInt(momentumBacktestProfileWorkers, len(candidates))
	var workers sync.WaitGroup
	for range workerCount {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for index := range jobs {
				candidate := candidates[index]
				profile, err := s.loadMarketProfile(ctx, candidate.Fund, candidate.EstablishedDate, days)
				if err != nil || len(profile.History) < minimumHistory {
					results <- profileResult{index: index}
					continue
				}
				results <- profileResult{index: index, profile: momentumBacktestFund{Fund: candidate.Fund, History: profile.History}, valid: true}
			}
		}()
	}
	go func() {
		for index := range candidates {
			jobs <- index
		}
		close(jobs)
		workers.Wait()
		close(results)
	}()
	ordered := make([]profileResult, len(candidates))
	for result := range results {
		ordered[result.index] = result
	}
	profiles := make([]momentumBacktestFund, 0, len(candidates))
	for _, result := range ordered {
		if result.valid {
			profiles = append(profiles, result.profile)
		}
	}
	return profiles
}

func minimumMomentumBacktestHistory(days int) int {
	minimum := days * 3 / 4
	if minimum < 300 {
		return 300
	}
	return minimum
}

func runMomentumBacktest(funds []momentumBacktestFund, cfg config.MomentumPoolConfig, rebalanceEvery, selectionCount int) *model.MomentumBacktestReport {
	return runMomentumBacktestWithCorrelation(funds, cfg, rebalanceEvery, selectionCount, 0)
}

func runMomentumBacktestWithCorrelation(funds []momentumBacktestFund, cfg config.MomentumPoolConfig, rebalanceEvery, selectionCount int, maxCorrelation float64) *model.MomentumBacktestReport {
	report := &model.MomentumBacktestReport{Summary: model.MomentumBacktestSummary{
		CandidateCount: len(funds), SelectionCount: selectionCount, RebalanceEvery: rebalanceEvery,
		Notes: []string{
			"收益与回撤按累计净值计算，避免基金分红或拆分造成虚假涨跌。",
			"诊断回测从当前存续的知名基金公司产品中使用稳定样本回放历史，不按当前收益挑选样本；仍存在存续基金偏差，应使用多个 universe_seed 检查样本敏感性，只用于参数比较，不代表未来收益。",
			"诊断回测未还原历史日期的申购暂停和单日限额；实时候选会校验当前申购状态，但历史回测结果仍可能高估可成交性。",
		},
	}}
	if len(funds) == 0 {
		return report
	}
	dates := commonMomentumDates(funds)
	if len(dates) <= 250+rebalanceEvery {
		return report
	}
	value := 1.0
	peak := 1.0
	benchmarkValue := 1.0
	benchmarkPeak := 1.0
	previousCodes := []string(nil)
	for index := 250; index+rebalanceEvery < len(dates); index += rebalanceEvery {
		startDate := dates[index]
		endDate := dates[index+rebalanceEvery]
		positiveBreadth, medianReturn60D, medianReturn120D := momentumMarketStateAtDate(funds, startDate)
		selected := selectMomentumAtDateWithCorrelation(funds, cfg, startDate, selectionCount, maxCorrelation)
		visible := selectVisibleMomentumAtDate(funds, cfg, startDate, selectionCount)
		periodReturn, names := momentumPeriodReturn(selected, startDate, endDate)
		topDecileHitRate, topDecileCapture, topCaptureRate, futurePercentile := momentumFutureWinnerMetrics(funds, visible, startDate, endDate)
		codes := momentumCandidateCodes(selected)
		benchmarkReturn := momentumUniverseReturn(funds, startDate, endDate)
		turnover := momentumSelectionTurnover(previousCodes, codes)
		cost := turnover * momentumBacktestRoundTripCost
		periodReturn -= cost
		report.Summary.EstimatedCost += cost
		value *= 1 + periodReturn
		benchmarkValue *= 1 + benchmarkReturn
		if value > peak {
			peak = value
		}
		drawdown := 1 - value/peak
		if drawdown > report.Summary.MaxDrawdown {
			report.Summary.MaxDrawdown = drawdown
		}
		if benchmarkValue > benchmarkPeak {
			benchmarkPeak = benchmarkValue
		}
		benchmarkDrawdown := 1 - benchmarkValue/benchmarkPeak
		if benchmarkDrawdown > report.Summary.BenchmarkMaxDrawdown {
			report.Summary.BenchmarkMaxDrawdown = benchmarkDrawdown
		}
		report.Periods = append(report.Periods, model.MomentumBacktestPeriod{StartDate: startDate, EndDate: endDate, Return: periodReturn, BenchmarkReturn: benchmarkReturn, ExcessReturn: periodReturn - benchmarkReturn, PositiveBreadth: positiveBreadth, MedianReturn60D: medianReturn60D, MedianReturn120D: medianReturn120D, Turnover: turnover, TopDecileHitRate: topDecileHitRate, TopDecileCapture: topDecileCapture, TopCaptureRate: topCaptureRate, FuturePercentile: futurePercentile, WinnerEvaluated: len(visible) > 0, Funds: names})
		previousCodes = codes
	}
	finalizeMomentumBacktest(report, value, benchmarkValue)
	return report
}

func momentumMarketStateAtDate(funds []momentumBacktestFund, date time.Time) (float64, float64, float64) {
	metrics := make([]momentumMetric, 0, len(funds))
	returns60D := make([]float64, 0, len(funds))
	returns120D := make([]float64, 0, len(funds))
	for _, fund := range funds {
		history := historyThrough(fund.History, date)
		if len(history) < 121 {
			continue
		}
		value := buildMarketPoolMetrics(history)
		metrics = append(metrics, momentumMetric{Return3M: value.Return60D, Return6M: value.Return120D})
		returns60D = append(returns60D, value.Return60D)
		returns120D = append(returns120D, value.Return120D)
	}
	return momentumMetricBreadth(metrics), medianFloat(returns60D), medianFloat(returns120D)
}

func medianFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	middle := len(values) / 2
	if len(values)%2 == 1 {
		return values[middle]
	}
	return (values[middle-1] + values[middle]) / 2
}

func commonMomentumDates(funds []momentumBacktestFund) []time.Time {
	counts := make(map[string]int)
	dateByKey := make(map[string]time.Time)
	for _, fund := range funds {
		seen := make(map[string]struct{})
		for _, snapshot := range fund.History {
			key := snapshot.TradeDate.Format("2006-01-02")
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			counts[key]++
			dateByKey[key] = snapshot.TradeDate
		}
	}
	minimumCoverage := int(math.Ceil(float64(len(funds)) * 0.8))
	dates := make([]time.Time, 0)
	for key, count := range counts {
		if count >= minimumCoverage {
			dates = append(dates, dateByKey[key])
		}
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	return dates
}

func selectVisibleMomentumAtDate(funds []momentumBacktestFund, cfg config.MomentumPoolConfig, date time.Time, count int) []momentumBacktestCandidate {
	candidates := momentumCandidatesAtDate(funds, cfg, date)
	if len(candidates) > count {
		candidates = candidates[:count]
	}
	return candidates
}

func selectMomentumAtDateWithCorrelation(funds []momentumBacktestFund, cfg config.MomentumPoolConfig, date time.Time, count int, maxCorrelation float64) []momentumBacktestCandidate {
	candidates := momentumCandidatesAtDate(funds, cfg, date)
	metrics := make([]momentumMetric, 0, len(candidates))
	for _, candidate := range candidates {
		metrics = append(metrics, momentumMetric{Return3M: candidate.Metrics.Return60D, Return6M: candidate.Metrics.Return120D})
	}
	if momentumMetricBreadth(metrics) < cfg.MinPositiveBreadth {
		return nil
	}
	executable := candidates[:0]
	for _, candidate := range candidates {
		if passesMomentumEntry(cfg, candidate.Score, candidate.Metrics) {
			executable = append(executable, candidate)
		}
	}
	candidates = executable
	if maxCorrelation > 0 {
		candidates = retainDiversifiedMomentumCandidates(candidates, count, maxCorrelation, date)
	} else {
		candidates = topMomentumCandidates(candidates, count)
	}
	if len(candidates) < cfg.MinSelectionCount {
		return nil
	}
	return candidates
}

func momentumCandidatesAtDate(funds []momentumBacktestFund, cfg config.MomentumPoolConfig, date time.Time) []momentumBacktestCandidate {
	candidates := make([]momentumBacktestCandidate, 0, len(funds))
	metrics := make([]momentumMetric, 0, len(funds))
	metricByCode := make(map[string]marketPoolMetrics)
	fundByCode := make(map[string]model.MarketSearchFund)
	historyByCode := make(map[string][]model.FundSnapshot)
	for _, fund := range funds {
		history := historyThrough(fund.History, date)
		if len(history) < 251 {
			continue
		}
		value := buildMarketPoolMetrics(history)
		metricByCode[fund.Fund.Code] = value
		fundByCode[fund.Fund.Code] = fund.Fund
		historyByCode[fund.Fund.Code] = fund.History
		metrics = append(metrics, momentumMetric{Code: fund.Fund.Code, Return1M: value.Return20D, Return3M: value.Return60D, Return6M: value.Return120D, Return1Y: value.Return250D})
	}
	for code, momentumScore := range momentumPercentileScores(metrics, cfg) {
		value := metricByCode[code]
		candidates = append(candidates, momentumBacktestCandidate{Fund: fundByCode[code], Score: momentumScore, Metrics: value, History: historyByCode[code]})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		if candidates[i].Metrics.Return250D != candidates[j].Metrics.Return250D {
			return candidates[i].Metrics.Return250D > candidates[j].Metrics.Return250D
		}
		if candidates[i].Metrics.Return120D != candidates[j].Metrics.Return120D {
			return candidates[i].Metrics.Return120D > candidates[j].Metrics.Return120D
		}
		return candidates[i].Fund.Code < candidates[j].Fund.Code
	})
	return candidates
}

func retainDiversifiedMomentumCandidates(candidates []momentumBacktestCandidate, count int, maxCorrelation float64, date time.Time) []momentumBacktestCandidate {
	selected := make([]momentumBacktestCandidate, 0, minInt(count, len(candidates)))
	for _, candidate := range candidates {
		if len(selected) >= count {
			break
		}
		if exceedsMomentumCorrelation(candidate, selected, maxCorrelation, date) {
			continue
		}
		selected = append(selected, candidate)
	}
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].Score != selected[j].Score {
			return selected[i].Score > selected[j].Score
		}
		return selected[i].Fund.Code < selected[j].Fund.Code
	})
	return selected
}

func exceedsMomentumCorrelation(candidate momentumBacktestCandidate, selected []momentumBacktestCandidate, maxCorrelation float64, date time.Time) bool {
	candidateHistory := historyThrough(candidate.History, date)
	for _, current := range selected {
		correlation, ok := fundReturnCorrelation(candidateHistory, historyThrough(current.History, date), momentumCorrelationWindowDays)
		if ok && correlation > maxCorrelation {
			return true
		}
	}
	return false
}

func topMomentumCandidates(candidates []momentumBacktestCandidate, count int) []momentumBacktestCandidate {
	selected := append([]momentumBacktestCandidate(nil), candidates[:minInt(count, len(candidates))]...)
	sort.Slice(selected, func(i, j int) bool {
		if selected[i].Score != selected[j].Score {
			return selected[i].Score > selected[j].Score
		}
		return selected[i].Fund.Code < selected[j].Fund.Code
	})
	return selected
}

func momentumCandidateCodes(candidates []momentumBacktestCandidate) []string {
	codes := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		codes = append(codes, candidate.Fund.Code)
	}
	return codes
}

func momentumMetricBreadth(metrics []momentumMetric) float64 {
	if len(metrics) == 0 {
		return 0
	}
	positive := 0
	for _, metric := range metrics {
		if metric.Return3M > 0 && metric.Return6M > 0 {
			positive++
		}
	}
	return float64(positive) / float64(len(metrics))
}

func historyThrough(history []model.FundSnapshot, date time.Time) []model.FundSnapshot {
	index := sort.Search(len(history), func(index int) bool { return history[index].TradeDate.After(date) })
	return history[:index]
}

func passesMomentumEntry(cfg config.MomentumPoolConfig, momentumScore float64, metrics marketPoolMetrics) bool {
	return momentumScore >= cfg.MinMomentumScore && metrics.Return20D >= cfg.MinReturn20D && metrics.Return60D >= cfg.MinReturn60D && metrics.Return120D >= cfg.MinReturn120D && metrics.MaxDrawdown120D <= cfg.MaxDrawdown120D
}

func momentumPeriodReturn(selected []momentumBacktestCandidate, startDate, endDate time.Time) (float64, []string) {
	var total float64
	names := make([]string, 0, len(selected))
	valid := 0
	for _, candidate := range selected {
		prices := snapshotsByDate(candidate.History)
		startNAV, startOK := returnAdjustedNAVOnOrBefore(prices, startDate)
		endNAV, endOK := returnAdjustedNAVOnOrBefore(prices, endDate)
		if !startOK || !endOK || startNAV <= 0 {
			continue
		}
		total += endNAV/startNAV - 1
		names = append(names, candidate.Fund.Name)
		valid++
	}
	if valid == 0 {
		return 0, names
	}
	return total / float64(valid), names
}

func momentumUniverseReturn(funds []momentumBacktestFund, startDate, endDate time.Time) float64 {
	var total float64
	valid := 0
	for _, fund := range funds {
		prices := snapshotsByDate(fund.History)
		startNAV, startOK := returnAdjustedNAVOnOrBefore(prices, startDate)
		endNAV, endOK := returnAdjustedNAVOnOrBefore(prices, endDate)
		if !startOK || !endOK || startNAV <= 0 {
			continue
		}
		total += endNAV/startNAV - 1
		valid++
	}
	if valid == 0 {
		return 0
	}
	return total / float64(valid)
}

func momentumFutureWinnerMetrics(funds []momentumBacktestFund, selected []momentumBacktestCandidate, startDate, endDate time.Time) (float64, float64, float64, float64) {
	returns := make(map[string]float64, len(funds))
	values := make([]float64, 0, len(funds))
	for _, fund := range funds {
		value, ok := momentumFundPeriodReturn(fund.History, startDate, endDate)
		if !ok {
			continue
		}
		returns[fund.Fund.Code] = value
		values = append(values, value)
	}
	if len(values) == 0 || len(selected) == 0 {
		return 0, 0, 0, 0
	}
	sort.Float64s(values)
	topDecileThreshold := values[maxInt(0, len(values)-maxInt(1, int(math.Ceil(float64(len(values))*0.10))))]
	topDecileCount := maxInt(1, int(math.Ceil(float64(len(values))*0.10)))
	topSelectionThreshold := values[maxInt(0, len(values)-minInt(len(selected), len(values)))]
	var topDecileHits, topSelectionHits int
	var percentileSum float64
	valid := 0
	for _, candidate := range selected {
		value, ok := returns[candidate.Fund.Code]
		if !ok {
			continue
		}
		valid++
		percentileSum += percentileRank(values, value)
		if value >= topDecileThreshold {
			topDecileHits++
		}
		if value >= topSelectionThreshold {
			topSelectionHits++
		}
	}
	if valid == 0 {
		return 0, 0, 0, 0
	}
	return float64(topDecileHits) / float64(valid), float64(topDecileHits) / float64(topDecileCount), float64(topSelectionHits) / float64(minInt(len(selected), len(values))), percentileSum / float64(valid)
}

func momentumFundPeriodReturn(history []model.FundSnapshot, startDate, endDate time.Time) (float64, bool) {
	prices := snapshotsByDate(history)
	startNAV, startOK := returnAdjustedNAVOnOrBefore(prices, startDate)
	endNAV, endOK := returnAdjustedNAVOnOrBefore(prices, endDate)
	if !startOK || !endOK || startNAV <= 0 {
		return 0, false
	}
	return endNAV/startNAV - 1, true
}

func snapshotsByDate(history []model.FundSnapshot) map[string]model.FundSnapshot {
	index := make(map[string]model.FundSnapshot, len(history))
	for _, snapshot := range history {
		index[snapshot.TradeDate.Format("2006-01-02")] = snapshot
	}
	return index
}

func returnAdjustedNAVOnOrBefore(index map[string]model.FundSnapshot, date time.Time) (float64, bool) {
	if len(index) == 0 {
		return 0, false
	}
	for current := date; !current.Before(date.AddDate(0, 0, -14)); current = current.AddDate(0, 0, -1) {
		if snapshot, ok := index[current.Format("2006-01-02")]; ok {
			return returnAdjustedNAV(snapshot), true
		}
	}
	return 0, false
}

func momentumSelectionTurnover(previous, current []string) float64 {
	if len(previous) == 0 && len(current) == 0 {
		return 0
	}
	if len(previous) == 0 || len(current) == 0 {
		return 1
	}
	seen := make(map[string]struct{}, len(previous))
	for _, name := range previous {
		seen[name] = struct{}{}
	}
	kept := 0
	for _, name := range current {
		if _, ok := seen[name]; ok {
			kept++
		}
	}
	return 1 - float64(kept)/float64(len(current))
}

func finalizeMomentumBacktest(report *model.MomentumBacktestReport, value, benchmarkValue float64) {
	if len(report.Periods) == 0 {
		return
	}
	report.Summary.StartDate = report.Periods[0].StartDate
	report.Summary.EndDate = report.Periods[len(report.Periods)-1].EndDate
	report.Summary.PeriodCount = len(report.Periods)
	report.Summary.TotalReturn = value - 1
	report.Summary.BenchmarkTotalReturn = benchmarkValue - 1
	report.Summary.ExcessReturn = report.Summary.TotalReturn - report.Summary.BenchmarkTotalReturn
	var totalReturn float64
	var totalTurnover float64
	var topDecileHitRateSum float64
	var topDecileCaptureSum float64
	var topCaptureRateSum float64
	var futurePercentileSum float64
	var recentTopDecileHitRateSum float64
	var recentTopDecileCaptureSum float64
	wins := 0
	excessWins := 0
	splitIndex := len(report.Periods) / 2
	earlyStrategyValue := 1.0
	earlyBenchmarkValue := 1.0
	recentStrategyValue := 1.0
	recentBenchmarkValue := 1.0
	for index, period := range report.Periods {
		totalReturn += period.Return
		totalTurnover += period.Turnover
		if len(period.Funds) == 0 {
			report.Summary.CashPeriodCount++
		} else {
			report.Summary.InvestedPeriodCount++
			if period.Return > 0 {
				wins++
			}
		}
		if period.WinnerEvaluated {
			report.Summary.WinnerEvaluationPeriods++
			topDecileHitRateSum += period.TopDecileHitRate
			topDecileCaptureSum += period.TopDecileCapture
			topCaptureRateSum += period.TopCaptureRate
			futurePercentileSum += period.FuturePercentile
		}
		if period.ExcessReturn > 0 {
			excessWins++
		}
		if index < splitIndex {
			earlyStrategyValue *= 1 + period.Return
			earlyBenchmarkValue *= 1 + period.BenchmarkReturn
		} else {
			recentStrategyValue *= 1 + period.Return
			recentBenchmarkValue *= 1 + period.BenchmarkReturn
			if period.WinnerEvaluated {
				report.Summary.RecentWinnerPeriods++
				recentTopDecileHitRateSum += period.TopDecileHitRate
				recentTopDecileCaptureSum += period.TopDecileCapture
			}
		}
	}
	report.Summary.AveragePeriodReturn = totalReturn / float64(len(report.Periods))
	report.Summary.AverageTurnover = totalTurnover / float64(len(report.Periods))
	if report.Summary.WinnerEvaluationPeriods > 0 {
		denominator := float64(report.Summary.WinnerEvaluationPeriods)
		report.Summary.FutureTopDecileHitRate = topDecileHitRateSum / denominator
		report.Summary.FutureTopDecileCapture = topDecileCaptureSum / denominator
		report.Summary.FutureTopSelectionCapture = topCaptureRateSum / denominator
		report.Summary.AverageFuturePercentile = futurePercentileSum / denominator
	}
	if report.Summary.RecentWinnerPeriods > 0 {
		report.Summary.RecentTopDecileHitRate = recentTopDecileHitRateSum / float64(report.Summary.RecentWinnerPeriods)
		report.Summary.RecentTopDecileCapture = recentTopDecileCaptureSum / float64(report.Summary.RecentWinnerPeriods)
	}
	if report.Summary.InvestedPeriodCount > 0 {
		report.Summary.WinRate = float64(wins) / float64(report.Summary.InvestedPeriodCount)
	}
	report.Summary.ExcessWinRate = float64(excessWins) / float64(len(report.Periods))
	report.Summary.ValidationSplitDate = report.Periods[splitIndex].StartDate
	report.Summary.EarlyExcessReturn = earlyStrategyValue - earlyBenchmarkValue
	report.Summary.RecentExcessReturn = recentStrategyValue - recentBenchmarkValue
	years := report.Summary.EndDate.Sub(report.Summary.StartDate).Hours() / 24 / 365
	if years > 0 && value > 0 {
		report.Summary.AnnualizedReturn = math.Pow(value, 1/years) - 1
	}
	if years > 0 && benchmarkValue > 0 {
		report.Summary.BenchmarkAnnualizedReturn = math.Pow(benchmarkValue, 1/years) - 1
	}
}
