package service

import (
	"context"
	"fmt"
	"io"
	"math"
	"sort"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/fetcher"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
	"github.com/derekdong-star/fund-advisor-cli/internal/report"
	"github.com/derekdong-star/fund-advisor-cli/internal/store"
	"github.com/derekdong-star/fund-advisor-cli/internal/strategy"
)

type Service struct {
	config  *config.Config
	store   *store.Store
	fetcher *fetcher.EastmoneyFetcher
	engine  *strategy.Engine
}

func New(configPath string) (*Service, error) {
	cfg, err := config.Load(configPath)
	if err != nil {
		return nil, err
	}
	st, err := store.Open(cfg.ResolveStorageDSN())
	if err != nil {
		return nil, err
	}
	timeout := time.Duration(cfg.DataSource.RequestTimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &Service{
		config:  cfg,
		store:   st,
		fetcher: fetcher.NewEastmoneyFetcher(timeout),
		engine:  strategy.NewEngine(cfg.Strategy),
	}, nil
}

func (s *Service) Close() error { return s.store.Close() }

func (s *Service) Validate() error { return s.config.Validate() }

func (s *Service) SyncPositions() error {
	for _, fund := range s.config.Funds {
		position := model.Position{
			FundCode:     fund.Code,
			FundName:     fund.Name,
			Category:     fund.Category,
			Benchmark:    fund.Benchmark,
			Role:         fund.Role,
			Status:       fund.Status,
			Protected:    fund.Protected,
			DCAEnabled:   fund.DCAEnabled,
			AccountValue: fund.AccountValue,
			TargetWeight: fund.TargetWeight,
		}
		if err := s.store.UpsertPosition(position); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) Fetch(ctx context.Context, days int) error {
	if err := s.SyncPositions(); err != nil {
		return err
	}
	codeSeen := make(map[string]struct{})
	positions, err := s.store.ListPositions()
	if err != nil {
		return err
	}
	for _, position := range positions {
		codeSeen[position.FundCode] = struct{}{}
		if err := s.fetchAndStore(ctx, position.FundCode, days, position.AccountValue, position.EstimatedUnits); err != nil {
			return err
		}
	}
	for _, candidate := range s.config.Candidates {
		if _, exists := codeSeen[candidate.Code]; exists {
			continue
		}
		if err := s.fetchAndStore(ctx, candidate.Code, days, 0, 0); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) fetchAndStore(ctx context.Context, code string, days int, accountValue, estimatedUnits float64) error {
	result, err := s.fetcher.FetchHistory(ctx, code, days)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", code, err)
	}
	if err := s.store.SaveSnapshots(result.Snapshots); err != nil {
		return fmt.Errorf("save snapshots %s: %w", code, err)
	}
	if estimatedUnits <= 0 && len(result.Snapshots) > 0 && accountValue > 0 {
		latest := result.Snapshots[len(result.Snapshots)-1]
		if latest.NAV > 0 {
			if err := s.store.SetEstimatedUnits(code, accountValue/latest.NAV); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) Analyze() (*model.AnalysisReport, error) {
	return s.buildAnalysis(true)
}

func (s *Service) CurrentReport() (*model.AnalysisReport, error) {
	return s.buildAnalysis(false)
}

func (s *Service) buildAnalysis(save bool) (*model.AnalysisReport, error) {
	if err := s.SyncPositions(); err != nil {
		return nil, err
	}
	positions, err := s.store.ListPositions()
	if err != nil {
		return nil, err
	}
	states := make([]model.PositionState, 0, len(positions))
	for _, position := range positions {
		latest, err := s.store.LatestSnapshot(position.FundCode)
		if err != nil {
			latest = nil
		}
		history, err := s.store.SnapshotHistory(position.FundCode, 180)
		if err != nil {
			return nil, err
		}
		states = append(states, model.PositionState{Position: position, Latest: latest, History: history})
	}
	heldCodes := make(map[string]struct{}, len(positions))
	for _, position := range positions {
		heldCodes[position.FundCode] = struct{}{}
	}
	candidates := make([]model.CandidateState, 0, len(s.config.Candidates))
	for _, candidate := range s.config.Candidates {
		if _, exists := heldCodes[candidate.Code]; exists {
			continue
		}
		latest, err := s.store.LatestSnapshot(candidate.Code)
		if err != nil {
			latest = nil
		}
		history, err := s.store.SnapshotHistory(candidate.Code, 180)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, model.CandidateState{
			Candidate: model.Candidate{
				FundCode:         candidate.Code,
				FundName:         candidate.Name,
				Category:         candidate.Category,
				Benchmark:        candidate.Benchmark,
				Role:             candidate.Role,
				Protected:        candidate.Protected,
				DCAEnabled:       candidate.DCAEnabled,
				ExpenseRatio:     candidate.ExpenseRatio,
				FundSizeYi:       candidate.FundSizeYi,
				EstablishedYears: candidate.EstablishedYears,
				IsIndex:          candidate.IsIndex,
				Tags:             candidate.Tags,
			},
			Latest:  latest,
			History: history,
		})
	}
	reportData := s.engine.Analyze(s.config.Portfolio.Name, states, candidates)
	if save {
		runID, err := s.store.SaveAnalysis(reportData)
		if err != nil {
			return nil, err
		}
		reportData.RunID = runID
	}
	return &reportData, nil
}

func (s *Service) LatestReport() (*model.AnalysisReport, error) {
	return s.store.LatestAnalysis()
}

func (s *Service) RenderCurrent(format string) (string, error) {
	reportData, err := s.CurrentReport()
	if err != nil {
		return "", err
	}
	return report.Render(*reportData, format)
}

func (s *Service) Run(ctx context.Context, days int, format string, out io.Writer) error {
	if err := s.Fetch(ctx, days); err != nil {
		return err
	}
	reportData, err := s.Analyze()
	if err != nil {
		return err
	}
	rendered, err := report.Render(*reportData, format)
	if err != nil {
		return err
	}
	_, err = io.WriteString(out, rendered)
	return err
}

func (s *Service) Backtest(days, rebalanceEvery int) (*model.BacktestReport, error) {
	if days <= 0 {
		return nil, fmt.Errorf("days must be positive")
	}
	if rebalanceEvery <= 0 {
		rebalanceEvery = 20
	}
	if err := s.SyncPositions(); err != nil {
		return nil, err
	}
	basePositions, err := s.store.ListPositions()
	if err != nil {
		return nil, err
	}
	if len(basePositions) == 0 {
		return nil, fmt.Errorf("no positions configured")
	}

	positionHistories := make(map[string][]model.FundSnapshot, len(basePositions))
	for _, position := range basePositions {
		history, err := s.store.SnapshotHistory(position.FundCode, days+200)
		if err != nil {
			return nil, err
		}
		if len(history) < days {
			return nil, fmt.Errorf("insufficient history for %s: need at least %d points, got %d", position.FundName, days, len(history))
		}
		positionHistories[position.FundCode] = history
	}

	candidateHistories := make(map[string][]model.FundSnapshot, len(s.config.Candidates))
	for _, candidate := range s.config.Candidates {
		history, err := s.store.SnapshotHistory(candidate.Code, days+200)
		if err != nil {
			return nil, err
		}
		candidateHistories[candidate.Code] = history
	}

	tradingDates := trailingCommonDates(positionHistories, days)
	if len(tradingDates) < 2 {
		return nil, fmt.Errorf("not enough overlapping trading dates")
	}
	startDate := tradingDates[0]
	endDate := tradingDates[len(tradingDates)-1]
	priceByCode := buildPriceLookup(positionHistories, candidateHistories)
	benchmarkValue, benchmarkUnits := initializeBenchmark(basePositions, startDate, priceByCode)
	if benchmarkValue <= 0 {
		return nil, fmt.Errorf("failed to initialize benchmark")
	}

	currentPositions := clonePositions(basePositions)
	fundMeta := fundMetadata(basePositions, s.config.Candidates)
	units := make(map[string]float64, len(fundMeta))
	var initialValue float64
	for _, position := range basePositions {
		price, ok := navOnOrBefore(priceByCode[position.FundCode], startDate)
		if !ok || price <= 0 {
			return nil, fmt.Errorf("missing start price for %s", position.FundName)
		}
		positionValue := position.AccountValue
		if positionValue <= 0 {
			positionValue = position.TargetWeight * benchmarkValue
		}
		units[position.FundCode] = positionValue / price
		initialValue += positionValue
	}
	if initialValue <= 0 {
		return nil, fmt.Errorf("initial portfolio value must be positive")
	}

	points := make([]model.BacktestPoint, 0, len(tradingDates))
	trades := make([]model.BacktestTrade, 0)
	cash := 0.0
	rebalanceCount := 0
	maxValue := 0.0
	maxBenchmark := 0.0
	maxDrawdown := 0.0
	maxBenchmarkDrawdown := 0.0

	for idx, currentDate := range tradingDates {
		activePositions := snapshotPositions(currentPositions, units)
		strategyValue := portfolioValueOnDate(units, cash, currentDate, priceByCode)
		benchmarkSnapshot := benchmarkValueOnDate(basePositions, benchmarkUnits, currentDate, priceByCode)
		if strategyValue > maxValue {
			maxValue = strategyValue
		}
		if benchmarkSnapshot > maxBenchmark {
			maxBenchmark = benchmarkSnapshot
		}
		if maxValue > 0 {
			drawdown := 1 - strategyValue/maxValue
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
		if maxBenchmark > 0 {
			drawdown := 1 - benchmarkSnapshot/maxBenchmark
			if drawdown > maxBenchmarkDrawdown {
				maxBenchmarkDrawdown = drawdown
			}
		}
		points = append(points, model.BacktestPoint{
			Date:           currentDate,
			StrategyValue:  strategyValue,
			BenchmarkValue: benchmarkSnapshot,
			Cash:           cash,
		})
		if idx == 0 || idx == len(tradingDates)-1 || idx%rebalanceEvery != 0 {
			continue
		}

		reportSnapshot, currentValues, err := s.backtestSnapshot(activePositions, units, currentDate, priceByCode, candidateHistories)
		if err != nil {
			return nil, err
		}
		if len(reportSnapshot.Recommendations) == 0 {
			continue
		}

		availableCash := cash
		for _, recommendation := range reportSnapshot.Recommendations {
			switch recommendation.Kind {
			case "SWAP":
				sellCode, ok := positionCodeByName(activePositions, recommendation.SourceFund)
				if !ok {
					continue
				}
				sellPrice, ok := navOnOrBefore(priceByCode[sellCode], currentDate)
				if !ok || sellPrice <= 0 {
					continue
				}
				sellAmount := minFloat(recommendation.SuggestedAmount, currentValues[sellCode])
				if sellAmount <= 0 {
					continue
				}
				sellUnits := sellAmount / sellPrice
				if sellUnits > units[sellCode] {
					sellUnits = units[sellCode]
					sellAmount = sellUnits * sellPrice
				}
				units[sellCode] -= sellUnits
				availableCash += sellAmount
				trades = append(trades, model.BacktestTrade{Date: currentDate, Action: "SELL", Fund: recommendation.SourceFund, RelatedFund: recommendation.TargetFund, Amount: sellAmount, Price: sellPrice, Units: sellUnits, Reason: recommendation.Reason})

				buyCode, ok := fundCodeByName(fundMeta, recommendation.TargetFund)
				if !ok {
					continue
				}
				buyPrice, ok := navOnOrBefore(priceByCode[buyCode], currentDate)
				if !ok || buyPrice <= 0 {
					continue
				}
				buyAmount := minFloat(sellAmount, availableCash)
				if buyAmount <= 0 {
					continue
				}
				buyUnits := buyAmount / buyPrice
				units[buyCode] += buyUnits
				availableCash -= buyAmount
				trades = append(trades, model.BacktestTrade{Date: currentDate, Action: "BUY", Fund: recommendation.TargetFund, RelatedFund: recommendation.SourceFund, Amount: buyAmount, Price: buyPrice, Units: buyUnits, Reason: recommendation.Reason})

				sellPosition := currentPositions[sellCode]
				targetShift := transferredTargetWeight(sellPosition.TargetWeight, currentValues[sellCode], sellAmount)
				sellPosition.TargetWeight -= targetShift
				if sellPosition.TargetWeight < 0 {
					sellPosition.TargetWeight = 0
				}
				currentPositions[sellCode] = sellPosition

				buyPosition, ok := currentPositions[buyCode]
				if !ok {
					buyPosition = fundMeta[buyCode]
				}
				buyPosition.TargetWeight += targetShift
				currentPositions[buyCode] = buyPosition
			case "REDUCE":
				sellCode, ok := positionCodeByName(activePositions, recommendation.SourceFund)
				if !ok {
					continue
				}
				sellPrice, ok := navOnOrBefore(priceByCode[sellCode], currentDate)
				if !ok || sellPrice <= 0 {
					continue
				}
				sellAmount := minFloat(recommendation.SuggestedAmount, currentValues[sellCode])
				if sellAmount <= 0 {
					continue
				}
				sellUnits := sellAmount / sellPrice
				if sellUnits > units[sellCode] {
					sellUnits = units[sellCode]
					sellAmount = sellUnits * sellPrice
				}
				units[sellCode] -= sellUnits
				availableCash += sellAmount
				trades = append(trades, model.BacktestTrade{Date: currentDate, Action: "SELL", Fund: recommendation.SourceFund, Amount: sellAmount, Price: sellPrice, Units: sellUnits, Reason: recommendation.Reason})
			case "BUY":
				buyCode, ok := positionCodeByName(activePositions, recommendation.TargetFund)
				if !ok {
					continue
				}
				buyPrice, ok := navOnOrBefore(priceByCode[buyCode], currentDate)
				if !ok || buyPrice <= 0 {
					continue
				}
				buyAmount := minFloat(recommendation.SuggestedAmount, availableCash)
				if buyAmount <= 0 {
					continue
				}
				buyUnits := buyAmount / buyPrice
				units[buyCode] += buyUnits
				availableCash -= buyAmount
				trades = append(trades, model.BacktestTrade{Date: currentDate, Action: "BUY", Fund: recommendation.TargetFund, Amount: buyAmount, Price: buyPrice, Units: buyUnits, Reason: recommendation.Reason})
			}
		}
		cash = availableCash
		rebalanceCount++
	}

	finalStrategy := portfolioValueOnDate(units, cash, endDate, priceByCode)
	finalBenchmark := benchmarkValueOnDate(basePositions, benchmarkUnits, endDate, priceByCode)
	years := float64(len(tradingDates)) / 252.0
	strategyReturn := safeReturn(initialValue, finalStrategy)
	benchmarkReturn := safeReturn(benchmarkValue, finalBenchmark)
	reportData := &model.BacktestReport{
		Summary: model.BacktestSummary{
			PortfolioName:             s.config.Portfolio.Name,
			StartDate:                 startDate,
			EndDate:                   endDate,
			TradingDays:               len(tradingDates),
			RebalanceEvery:            rebalanceEvery,
			RebalanceCount:            rebalanceCount,
			TradeCount:                len(trades),
			InitialValue:              initialValue,
			FinalValue:                finalStrategy,
			BenchmarkInitialValue:     benchmarkValue,
			BenchmarkFinalValue:       finalBenchmark,
			CashFinal:                 cash,
			TotalReturn:               strategyReturn,
			BenchmarkReturn:           benchmarkReturn,
			ExcessReturn:              strategyReturn - benchmarkReturn,
			AnnualizedReturn:          annualizedReturn(strategyReturn, years),
			BenchmarkAnnualizedReturn: annualizedReturn(benchmarkReturn, years),
			MaxDrawdown:               maxDrawdown,
			BenchmarkMaxDrawdown:      maxBenchmarkDrawdown,
			Notes: []string{
				"回测按固定周期重跑当前规则，不含申赎费、滑点、税费",
				"基准为初始持仓按买入并持有计算",
				"买入仅使用卖出回笼现金，不额外注资",
			},
		},
		Points: points,
		Trades: trades,
	}
	return reportData, nil
}

func (s *Service) backtestSnapshot(positions []model.Position, units map[string]float64, currentDate time.Time, priceByCode map[string]map[string]model.FundSnapshot, candidateHistories map[string][]model.FundSnapshot) (model.AnalysisReport, map[string]float64, error) {
	states := make([]model.PositionState, 0, len(positions))
	currentValues := make(map[string]float64, len(positions))
	heldCodes := make(map[string]struct{}, len(positions))
	for _, position := range positions {
		heldCodes[position.FundCode] = struct{}{}
		history := sliceHistoryUntil(priceByCode[position.FundCode], currentDate, 180)
		if len(history) == 0 {
			return model.AnalysisReport{}, nil, fmt.Errorf("missing history for %s on %s", position.FundName, currentDate.Format("2006-01-02"))
		}
		latest := history[len(history)-1]
		state := model.PositionState{
			Position: model.Position{
				FundCode:       position.FundCode,
				FundName:       position.FundName,
				Category:       position.Category,
				Benchmark:      position.Benchmark,
				Role:           position.Role,
				Status:         position.Status,
				Protected:      position.Protected,
				DCAEnabled:     position.DCAEnabled,
				AccountValue:   position.AccountValue,
				TargetWeight:   position.TargetWeight,
				EstimatedUnits: units[position.FundCode],
			},
			Latest:  &latest,
			History: history,
		}
		currentValues[position.FundCode] = units[position.FundCode] * latest.NAV
		states = append(states, state)
	}

	candidates := make([]model.CandidateState, 0, len(s.config.Candidates))
	for _, candidate := range s.config.Candidates {
		if _, held := heldCodes[candidate.Code]; held {
			continue
		}
		history := sliceRawHistoryUntil(candidateHistories[candidate.Code], currentDate, 180)
		var latest *model.FundSnapshot
		if len(history) > 0 {
			copyLatest := history[len(history)-1]
			latest = &copyLatest
		}
		candidates = append(candidates, model.CandidateState{
			Candidate: model.Candidate{
				FundCode:         candidate.Code,
				FundName:         candidate.Name,
				Category:         candidate.Category,
				Benchmark:        candidate.Benchmark,
				Role:             candidate.Role,
				Protected:        candidate.Protected,
				DCAEnabled:       candidate.DCAEnabled,
				ExpenseRatio:     candidate.ExpenseRatio,
				FundSizeYi:       candidate.FundSizeYi,
				EstablishedYears: candidate.EstablishedYears,
				IsIndex:          candidate.IsIndex,
				Tags:             candidate.Tags,
			},
			Latest:  latest,
			History: history,
		})
	}
	return s.engine.AnalyzeAt(currentDate, s.config.Portfolio.Name, states, candidates), currentValues, nil
}

func fundMetadata(positions []model.Position, candidates []config.FundConfig) map[string]model.Position {
	meta := make(map[string]model.Position, len(positions)+len(candidates))
	for _, position := range positions {
		meta[position.FundCode] = position
	}
	for _, candidate := range candidates {
		if _, exists := meta[candidate.Code]; exists {
			continue
		}
		meta[candidate.Code] = model.Position{
			FundCode:   candidate.Code,
			FundName:   candidate.Name,
			Category:   candidate.Category,
			Benchmark:  candidate.Benchmark,
			Role:       candidate.Role,
			Status:     candidate.Status,
			Protected:  candidate.Protected,
			DCAEnabled: candidate.DCAEnabled,
		}
	}
	return meta
}

func clonePositions(positions []model.Position) map[string]model.Position {
	cloned := make(map[string]model.Position, len(positions))
	for _, position := range positions {
		cloned[position.FundCode] = position
	}
	return cloned
}

func snapshotPositions(currentPositions map[string]model.Position, units map[string]float64) []model.Position {
	positions := make([]model.Position, 0, len(currentPositions))
	for code, position := range currentPositions {
		if position.TargetWeight <= 0 && units[code] <= 0 {
			continue
		}
		positions = append(positions, position)
	}
	sort.Slice(positions, func(i, j int) bool {
		return positions[i].FundCode < positions[j].FundCode
	})
	return positions
}

func buildPriceLookup(positionHistories map[string][]model.FundSnapshot, candidateHistories map[string][]model.FundSnapshot) map[string]map[string]model.FundSnapshot {
	lookup := make(map[string]map[string]model.FundSnapshot, len(positionHistories)+len(candidateHistories))
	for code, history := range positionHistories {
		lookup[code] = historyIndex(history)
	}
	for code, history := range candidateHistories {
		lookup[code] = historyIndex(history)
	}
	return lookup
}

func historyIndex(history []model.FundSnapshot) map[string]model.FundSnapshot {
	indexed := make(map[string]model.FundSnapshot, len(history))
	for _, snapshot := range history {
		indexed[snapshot.TradeDate.Format("2006-01-02")] = snapshot
	}
	return indexed
}

func trailingCommonDates(histories map[string][]model.FundSnapshot, days int) []time.Time {
	dateCounts := make(map[string]int)
	required := len(histories)
	for _, history := range histories {
		for _, snapshot := range history {
			dateCounts[snapshot.TradeDate.Format("2006-01-02")]++
		}
	}
	dates := make([]time.Time, 0)
	for key, count := range dateCounts {
		if count != required {
			continue
		}
		date, err := time.Parse("2006-01-02", key)
		if err != nil {
			continue
		}
		dates = append(dates, date)
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	if len(dates) > days {
		return dates[len(dates)-days:]
	}
	return dates
}

func initializeBenchmark(positions []model.Position, startDate time.Time, priceByCode map[string]map[string]model.FundSnapshot) (float64, map[string]float64) {
	weights := make(map[string]float64, len(positions))
	var totalValue float64
	for _, position := range positions {
		price, ok := navOnOrBefore(priceByCode[position.FundCode], startDate)
		if !ok || price <= 0 {
			return 0, nil
		}
		weights[position.FundCode] = position.AccountValue / price
		totalValue += position.AccountValue
	}
	return totalValue, weights
}

func portfolioValueOnDate(units map[string]float64, cash float64, currentDate time.Time, priceByCode map[string]map[string]model.FundSnapshot) float64 {
	total := cash
	for code, heldUnits := range units {
		if heldUnits <= 0 {
			continue
		}
		price, ok := navOnOrBefore(priceByCode[code], currentDate)
		if !ok {
			continue
		}
		total += heldUnits * price
	}
	return total
}

func benchmarkValueOnDate(positions []model.Position, units map[string]float64, currentDate time.Time, priceByCode map[string]map[string]model.FundSnapshot) float64 {
	var total float64
	for _, position := range positions {
		price, ok := navOnOrBefore(priceByCode[position.FundCode], currentDate)
		if !ok {
			continue
		}
		total += units[position.FundCode] * price
	}
	return total
}

func navOnOrBefore(index map[string]model.FundSnapshot, date time.Time) (float64, bool) {
	if len(index) == 0 {
		return 0, false
	}
	for current := date; !current.Before(date.AddDate(0, 0, -14)); current = current.AddDate(0, 0, -1) {
		if snapshot, ok := index[current.Format("2006-01-02")]; ok {
			return snapshot.NAV, true
		}
	}
	return 0, false
}

func sliceHistoryUntil(index map[string]model.FundSnapshot, currentDate time.Time, limit int) []model.FundSnapshot {
	history := make([]model.FundSnapshot, 0)
	for key, snapshot := range index {
		date, err := time.Parse("2006-01-02", key)
		if err != nil || date.After(currentDate) {
			continue
		}
		history = append(history, snapshot)
	}
	sort.Slice(history, func(i, j int) bool { return history[i].TradeDate.Before(history[j].TradeDate) })
	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:]
	}
	return history
}

func sliceRawHistoryUntil(history []model.FundSnapshot, currentDate time.Time, limit int) []model.FundSnapshot {
	filtered := make([]model.FundSnapshot, 0, len(history))
	for _, snapshot := range history {
		if snapshot.TradeDate.After(currentDate) {
			continue
		}
		filtered = append(filtered, snapshot)
	}
	if limit > 0 && len(filtered) > limit {
		return filtered[len(filtered)-limit:]
	}
	return filtered
}

func positionCodeByName(positions []model.Position, fundName string) (string, bool) {
	for _, position := range positions {
		if position.FundName == fundName {
			return position.FundCode, true
		}
	}
	return "", false
}

func fundCodeByName(funds map[string]model.Position, fundName string) (string, bool) {
	for code, fund := range funds {
		if fund.FundName == fundName {
			return code, true
		}
	}
	return "", false
}

func candidateCodeByName(candidates []config.FundConfig, fundName string) (string, bool) {
	for _, candidate := range candidates {
		if candidate.Name == fundName {
			return candidate.Code, true
		}
	}
	return "", false
}

func transferredTargetWeight(targetWeight, currentValue, sellAmount float64) float64 {
	if targetWeight <= 0 || currentValue <= 0 || sellAmount <= 0 {
		return 0
	}
	ratio := sellAmount / currentValue
	if ratio > 1 {
		ratio = 1
	}
	return targetWeight * ratio
}

func safeReturn(start, end float64) float64 {
	if start <= 0 {
		return 0
	}
	return end/start - 1
}

func annualizedReturn(totalReturn, years float64) float64 {
	if years <= 0 {
		return 0
	}
	return pow(1+totalReturn, 1/years) - 1
}

func pow(base, exp float64) float64 {
	return math.Pow(base, exp)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
