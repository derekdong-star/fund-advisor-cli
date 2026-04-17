package strategy

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type Engine struct {
	strategy config.StrategyConfig
}

type dcaPlanCandidate struct {
	state     model.PositionState
	gapWeight float64
	gapAmount float64
	priority  int
	reason    string
}

func NewEngine(strategy config.StrategyConfig) *Engine {
	return &Engine{strategy: strategy}
}

func (e *Engine) Analyze(portfolioName string, states []model.PositionState, candidates []model.CandidateState) model.AnalysisReport {
	return e.AnalyzeAt(time.Now().UTC(), portfolioName, states, candidates)
}

func (e *Engine) AnalyzeAt(asOf time.Time, portfolioName string, states []model.PositionState, candidates []model.CandidateState) model.AnalysisReport {
	var totalValue float64
	var weightedDayChange float64
	for idx := range states {
		state := &states[idx]
		if state.Latest == nil {
			state.CurrentValue = state.Position.AccountValue
		} else if state.Position.EstimatedUnits > 0 {
			state.CurrentValue = state.Position.EstimatedUnits * state.Latest.NAV
		} else {
			state.CurrentValue = state.Position.AccountValue
		}
		totalValue += state.CurrentValue
	}
	for idx := range states {
		state := &states[idx]
		if totalValue > 0 {
			state.CurrentWeight = state.CurrentValue / totalValue
		}
		state.Drift = state.CurrentWeight - state.Position.TargetWeight
		state.Return20D = rollingReturn(state.History, 20)
		state.Return60D = rollingReturn(state.History, 60)
		state.Return120D = rollingReturn(state.History, 120)
		if state.Latest != nil {
			weightedDayChange += state.CurrentValue * state.Latest.DayChangePct
		}
	}
	peerAverage60D := peerAverage(states, 60)
	categoryCounts := categoryCounts(states)
	lowTurnoverMode := strings.EqualFold(e.strategy.Turnover.Mode, "low_turnover")
	replaceThreshold := e.strategy.HoldingHealth.ReplaceScoreThreshold
	if lowTurnoverMode {
		replaceThreshold++
	}
	signals := make([]model.FundSignal, 0, len(states))
	actionCounts := make(map[model.Action]int)
	for idx := range states {
		state := &states[idx]
		reasons := make([]string, 0, 4)
		score := 0
		if state.Return20D <= -0.08 {
			score++
			reasons = append(reasons, fmt.Sprintf("20日收益偏弱 %.2f%%", state.Return20D*100))
		}
		if state.Return60D <= e.strategy.HoldingHealth.Underperform60DThreshold {
			score++
			reasons = append(reasons, fmt.Sprintf("60日收益低于阈值 %.2f%%", state.Return60D*100))
		}
		if avg, ok := peerAverage60D[state.Position.Category]; ok && categoryCounts[state.Position.Category] > 1 && state.Return60D < avg-0.03 {
			score++
			reasons = append(reasons, "同类别基金中相对偏弱")
		}
		if categoryCounts[state.Position.Category] > 1 && strings.Contains(state.Position.Role, "satellite") && !state.Position.Protected {
			score++
			reasons = append(reasons, "组合内存在同类重复暴露")
		}
		if state.Latest != nil && asOf.Sub(state.Latest.TradeDate) > 96*time.Hour {
			score++
			reasons = append(reasons, "最新净值数据较旧")
		}
		action := model.ActionHold
		overweightRelative := state.Position.TargetWeight > 0 && state.CurrentWeight >= state.Position.TargetWeight*(1+e.strategy.SellSignal.OverweightRelativeThreshold)
		overweightAbsolute := state.CurrentWeight-state.Position.TargetWeight >= e.strategy.SellSignal.OverweightAbsoluteThreshold
		underTarget := state.Position.TargetWeight > 0 && state.CurrentWeight <= state.Position.TargetWeight*(1-e.strategy.BuySignal.MinGapToTarget)
		singleTooLarge := state.CurrentWeight >= e.strategy.BuySignal.MaxSinglePositionWeight
		extremeOverweightAbsolute := state.CurrentWeight-state.Position.TargetWeight >= e.strategy.SellSignal.OverweightAbsoluteThreshold+0.10
		extremeSingleTooLarge := state.CurrentWeight >= maxFloat64(e.strategy.BuySignal.MaxSinglePositionWeight+0.15, 0.35)
		routineOverweight := overweightRelative || overweightAbsolute || singleTooLarge
		protectedReduce := false
		severeReduce := (!state.Position.DCAEnabled && score >= replaceThreshold) || extremeOverweightAbsolute || extremeSingleTooLarge || state.Position.TargetWeight == 0
		if state.Position.Protected {
			overweightRelative = false
			overweightAbsolute = false
			singleTooLarge = false
			routineOverweight = false
			protectedReduce = state.CurrentWeight > e.strategy.Turnover.MaxProtectedReduceWeight && score >= replaceThreshold+1
		}
		switch {
		case score >= replaceThreshold && !state.Position.Protected:
			action = model.ActionReplaceWatch
		case protectedReduce:
			action = model.ActionReduce
			reasons = append(reasons, fmt.Sprintf("当前权重 %.2f%% 高于保护上限 %.2f%%", state.CurrentWeight*100, e.strategy.Turnover.MaxProtectedReduceWeight*100))
		case lowTurnoverMode && routineOverweight && severeReduce:
			action = model.ActionReduce
			reasons = append(reasons, fmt.Sprintf("当前权重 %.2f%% 明显高于目标 %.2f%%", state.CurrentWeight*100, state.Position.TargetWeight*100))
		case !lowTurnoverMode && routineOverweight:
			action = model.ActionReduce
			reasons = append(reasons, fmt.Sprintf("当前权重 %.2f%% 高于目标 %.2f%%", state.CurrentWeight*100, state.Position.TargetWeight*100))
		case lowTurnoverMode && routineOverweight:
			action = model.ActionPauseBuy
			reasons = append(reasons, "低换手模式下先暂停加仓，暂不主动减仓")
		case score >= e.strategy.HoldingHealth.ReviewScoreThreshold:
			action = model.ActionPauseBuy
		case underTarget && score < e.strategy.HoldingHealth.ReviewScoreThreshold && (!lowTurnoverMode || state.Position.DCAEnabled):
			action = model.ActionBuy
			reasons = append(reasons, fmt.Sprintf("当前权重 %.2f%% 低于目标 %.2f%%", state.CurrentWeight*100, state.Position.TargetWeight*100))
			if e.strategy.Turnover.PreferDCA && state.Position.DCAEnabled {
				reasons = append(reasons, "符合长期定投优先规则")
			}
		case underTarget && lowTurnoverMode:
			reasons = append(reasons, "低换手模式下非定投基金仅观察，不主动补仓")
		default:
			reasons = append(reasons, "权重和健康度均在可接受区间")
		}
		sort.Strings(reasons)
		state.HealthScore = score
		state.Action = action
		state.Reasons = reasons
		signal := model.FundSignal{
			FundCode:      state.Position.FundCode,
			FundName:      state.Position.FundName,
			Action:        action,
			Score:         score,
			CurrentWeight: state.CurrentWeight,
			TargetWeight:  state.Position.TargetWeight,
			Drift:         state.Drift,
			CurrentValue:  state.CurrentValue,
			Return20D:     state.Return20D,
			Return60D:     state.Return60D,
			Return120D:    state.Return120D,
			Reason:        strings.Join(reasons, "；"),
			CreatedAt:     asOf,
		}
		if state.Latest != nil {
			signal.LatestTradeDate = state.Latest.TradeDate
		}
		signals = append(signals, signal)
		actionCounts[action]++
	}
	candidateSuggestions := e.evaluateCandidates(asOf, states, candidates)
	recommendations := e.buildRecommendations(asOf, states, candidateSuggestions, totalValue)
	executionPlan := buildExecutionPlan(recommendations)
	summary := model.AnalysisSummary{
		PortfolioName:        portfolioName,
		RunDate:              asOf,
		PortfolioValue:       totalValue,
		WeightedDayChangePct: safeWeightedChange(totalValue, weightedDayChange),
		ActionCounts:         actionCounts,
		CandidateCount:       len(candidateSuggestions),
		GeneratedAt:          asOf,
	}
	if len(candidateSuggestions) > 0 {
		summary.Notes = append(summary.Notes, fmt.Sprintf("候选池中有 %d 只基金满足替换观察条件", len(candidateSuggestions)))
	}
	return model.AnalysisReport{Summary: summary, Signals: signals, Candidates: candidateSuggestions, Recommendations: recommendations, ExecutionPlan: executionPlan, Position: states}
}

func (e *Engine) evaluateCandidates(asOf time.Time, states []model.PositionState, candidates []model.CandidateState) []model.CandidateSuggestion {
	if len(candidates) == 0 {
		return nil
	}
	targetsByCategory := make(map[string][]string)
	targetsByRole := make(map[string][]string)
	for _, state := range states {
		shouldEvaluate := !state.Position.Protected && state.Action == model.ActionReplaceWatch
		if !shouldEvaluate && !strings.EqualFold(e.strategy.Turnover.Mode, "low_turnover") && state.Action == model.ActionPauseBuy {
			shouldEvaluate = true
		}
		if !shouldEvaluate && !strings.EqualFold(e.strategy.Turnover.Mode, "low_turnover") && state.Action == model.ActionReduce && state.HealthScore >= e.strategy.HoldingHealth.ReviewScoreThreshold {
			shouldEvaluate = true
		}
		if !shouldEvaluate {
			continue
		}
		targetsByCategory[state.Position.Category] = append(targetsByCategory[state.Position.Category], state.Position.FundName)
		targetsByRole[state.Position.Role] = append(targetsByRole[state.Position.Role], state.Position.FundName)
	}
	if len(targetsByCategory) == 0 && len(targetsByRole) == 0 {
		return nil
	}
	suggestions := make([]model.CandidateSuggestion, 0, len(candidates))
	for idx := range candidates {
		candidate := &candidates[idx]
		candidate.Return20D = rollingReturn(candidate.History, 20)
		candidate.Return60D = rollingReturn(candidate.History, 60)
		candidate.Return120D = rollingReturn(candidate.History, 120)
		var reasons []string
		score := 0
		replaceFor := append([]string(nil), targetsByCategory[candidate.Candidate.Category]...)
		if len(replaceFor) > 0 {
			score += 2
			reasons = append(reasons, "类别可直接覆盖当前观察名单")
		} else if roleTargets := targetsByRole[candidate.Candidate.Role]; len(roleTargets) > 0 {
			replaceFor = append(replaceFor, roleTargets...)
			score++
			reasons = append(reasons, "角色上可替换当前观察名单")
		}
		if len(replaceFor) == 0 {
			continue
		}
		if e.strategy.CandidatePool.PreferBenchmarkMatch {
			matched := false
			for _, state := range states {
				if !containsTarget(replaceFor, state.Position.FundName) {
					continue
				}
				if candidate.Candidate.Benchmark != "" && candidate.Candidate.Benchmark == state.Position.Benchmark {
					matched = true
					break
				}
			}
			if matched {
				score++
				reasons = append(reasons, "基准与待替换持仓一致")
			}
		}
		if e.strategy.CandidatePool.CoreRequireIndex && candidate.Candidate.Role == "core" {
			if candidate.Candidate.IsIndex {
				score++
				reasons = append(reasons, "核心仓优先指数工具")
			} else {
				reasons = append(reasons, "核心仓缺少指数属性")
				continue
			}
		}
		if e.strategy.CandidatePool.MaxExpenseRatio > 0 && candidate.Candidate.ExpenseRatio > 0 {
			if candidate.Candidate.ExpenseRatio <= e.strategy.CandidatePool.MaxExpenseRatio {
				score++
				reasons = append(reasons, fmt.Sprintf("费率可控 %.2f%%", candidate.Candidate.ExpenseRatio*100))
			} else {
				reasons = append(reasons, fmt.Sprintf("费率偏高 %.2f%%", candidate.Candidate.ExpenseRatio*100))
				continue
			}
		}
		if e.strategy.CandidatePool.MinFundSizeYi > 0 && candidate.Candidate.FundSizeYi > 0 {
			if candidate.Candidate.FundSizeYi >= e.strategy.CandidatePool.MinFundSizeYi {
				score++
				reasons = append(reasons, fmt.Sprintf("规模达标 %.1f亿", candidate.Candidate.FundSizeYi))
			} else {
				reasons = append(reasons, fmt.Sprintf("规模偏小 %.1f亿", candidate.Candidate.FundSizeYi))
				continue
			}
		}
		if e.strategy.CandidatePool.MinEstablishedYears > 0 && candidate.Candidate.EstablishedYears > 0 {
			if candidate.Candidate.EstablishedYears >= e.strategy.CandidatePool.MinEstablishedYears {
				score++
				reasons = append(reasons, fmt.Sprintf("成立年限达标 %.1f年", candidate.Candidate.EstablishedYears))
			} else {
				reasons = append(reasons, fmt.Sprintf("成立年限偏短 %.1f年", candidate.Candidate.EstablishedYears))
				continue
			}
		}
		if candidate.Return20D > 0 {
			score++
			reasons = append(reasons, fmt.Sprintf("20日收益较强 %.2f%%", candidate.Return20D*100))
		}
		if candidate.Return60D > 0 {
			score += 2
			reasons = append(reasons, fmt.Sprintf("60日收益较强 %.2f%%", candidate.Return60D*100))
		} else if candidate.Return60D > e.strategy.HoldingHealth.Underperform60DThreshold {
			score++
			reasons = append(reasons, "60日表现不弱于当前阈值")
		}
		if candidate.Latest != nil && asOf.Sub(candidate.Latest.TradeDate) <= 96*time.Hour {
			score++
			reasons = append(reasons, "净值数据较新")
		}
		candidate.Score = score
		candidate.Reasons = reasons
		candidate.ReplaceFor = replaceFor
		minCandidateScore := 4
		if strings.EqualFold(e.strategy.Turnover.Mode, "low_turnover") && e.strategy.Turnover.MinSwapScore > minCandidateScore {
			minCandidateScore = e.strategy.Turnover.MinSwapScore
		}
		if score < minCandidateScore {
			continue
		}
		suggestion := model.CandidateSuggestion{
			FundCode:         candidate.Candidate.FundCode,
			FundName:         candidate.Candidate.FundName,
			Category:         candidate.Candidate.Category,
			Benchmark:        candidate.Candidate.Benchmark,
			Role:             candidate.Candidate.Role,
			Score:            score,
			Return20D:        candidate.Return20D,
			Return60D:        candidate.Return60D,
			Return120D:       candidate.Return120D,
			ExpenseRatio:     candidate.Candidate.ExpenseRatio,
			FundSizeYi:       candidate.Candidate.FundSizeYi,
			EstablishedYears: candidate.Candidate.EstablishedYears,
			IsIndex:          candidate.Candidate.IsIndex,
			ReplaceFor:       replaceFor,
			Reason:           strings.Join(reasons, "；"),
		}
		if candidate.Latest != nil {
			suggestion.LatestTradeDate = candidate.Latest.TradeDate
		}
		suggestions = append(suggestions, suggestion)
	}
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Score == suggestions[j].Score {
			return suggestions[i].FundCode < suggestions[j].FundCode
		}
		return suggestions[i].Score > suggestions[j].Score
	})
	return suggestions
}

func (e *Engine) buildRecommendations(asOf time.Time, states []model.PositionState, candidates []model.CandidateSuggestion, totalValue float64) []model.TradeRecommendation {
	if totalValue <= 0 {
		return nil
	}
	lowTurnoverMode := strings.EqualFold(e.strategy.Turnover.Mode, "low_turnover")
	stateByName := make(map[string]model.PositionState, len(states))
	for _, state := range states {
		stateByName[state.Position.FundName] = state
	}
	recommendations := make([]model.TradeRecommendation, 0, len(states)+len(candidates))
	usedSources := make(map[string]struct{})
	usedTargets := make(map[string]struct{})
	for _, candidate := range candidates {
		if _, exists := usedTargets[candidate.FundName]; exists {
			continue
		}
		bestSource := ""
		var bestState model.PositionState
		var bestAmount float64
		for _, sourceName := range candidate.ReplaceFor {
			if _, exists := usedSources[sourceName]; exists {
				continue
			}
			state, ok := stateByName[sourceName]
			if !ok {
				continue
			}
			amount := replacementAmount(state, totalValue)
			if amount <= 0 {
				continue
			}
			if bestSource == "" || preferSwapSource(state, amount, bestState, bestAmount) {
				bestSource = sourceName
				bestState = state
				bestAmount = amount
			}
		}
		if bestSource == "" {
			continue
		}
		usedSources[bestSource] = struct{}{}
		usedTargets[candidate.FundName] = struct{}{}
		recommendations = append(recommendations, model.TradeRecommendation{
			Kind:            "SWAP",
			SourceFund:      bestSource,
			TargetFund:      candidate.FundName,
			SuggestedWeight: bestAmount / totalValue,
			SuggestedAmount: bestAmount,
			Reason:          fmt.Sprintf("用最优候选替换弱势持仓：%s", candidate.Reason),
			CreatedAt:       asOf,
		})
	}
	var availableCapital float64
	for _, recommendation := range recommendations {
		if recommendation.Kind == "SWAP" || recommendation.Kind == "REDUCE" {
			availableCapital += recommendation.SuggestedAmount
		}
	}
	for _, state := range states {
		if _, swapped := usedSources[state.Position.FundName]; swapped {
			continue
		}
		if state.Action != model.ActionReduce {
			continue
		}
		if strings.EqualFold(e.strategy.Turnover.Mode, "low_turnover") && state.Position.DCAEnabled {
			continue
		}
		amount := maxFloat64(0, (state.CurrentWeight-state.Position.TargetWeight)*totalValue)
		if amount <= 0 {
			continue
		}
		availableCapital += amount
		recommendations = append(recommendations, model.TradeRecommendation{
			Kind:            "REDUCE",
			SourceFund:      state.Position.FundName,
			SuggestedWeight: amount / totalValue,
			SuggestedAmount: amount,
			Reason:          stateSignalReason(state),
			CreatedAt:       asOf,
		})
	}
	buyRecommendations := make([]model.TradeRecommendation, 0)
	for _, state := range states {
		if state.Action != model.ActionBuy {
			continue
		}
		if lowTurnoverMode && !state.Position.DCAEnabled {
			continue
		}
		amount := maxFloat64(0, (state.Position.TargetWeight-state.CurrentWeight)*totalValue)
		if e.strategy.Turnover.PreferDCA && state.Position.DCAEnabled && e.strategy.Turnover.MonthlyDCAAmount > 0 {
			amount = minFloat64(amount, e.strategy.Turnover.MonthlyDCAAmount)
		}
		if amount <= 0 {
			continue
		}
		buyRecommendations = append(buyRecommendations, model.TradeRecommendation{
			Kind:            "BUY",
			TargetFund:      state.Position.FundName,
			SuggestedWeight: amount / totalValue,
			SuggestedAmount: amount,
			Reason:          stateSignalReason(state),
			CreatedAt:       asOf,
		})
	}
	sort.Slice(buyRecommendations, func(i, j int) bool {
		leftPreferred := isPreferredDCAState(states, buyRecommendations[i].TargetFund)
		rightPreferred := isPreferredDCAState(states, buyRecommendations[j].TargetFund)
		if leftPreferred != rightPreferred {
			return leftPreferred
		}
		return buyRecommendations[i].SuggestedAmount > buyRecommendations[j].SuggestedAmount
	})
	var buyDemand float64
	for _, recommendation := range buyRecommendations {
		buyDemand += recommendation.SuggestedAmount
	}
	if buyDemand > 0 && availableCapital > 0 {
		scale := minFloat64(1, availableCapital/buyDemand)
		for idx := range buyRecommendations {
			buyRecommendations[idx].SuggestedAmount *= scale
			buyRecommendations[idx].SuggestedWeight *= scale
		}
	}
	recommendations = append(recommendations, buyRecommendations...)
	sort.Slice(recommendations, func(i, j int) bool {
		order := map[string]int{"SWAP": 1, "REDUCE": 2, "BUY": 3}
		if order[recommendations[i].Kind] == order[recommendations[j].Kind] {
			return recommendations[i].SuggestedAmount > recommendations[j].SuggestedAmount
		}
		return order[recommendations[i].Kind] < order[recommendations[j].Kind]
	})
	return recommendations
}

func preferSwapSource(candidate model.PositionState, candidateAmount float64, current model.PositionState, currentAmount float64) bool {
	candidatePriority := swapSourcePriority(candidate)
	currentPriority := swapSourcePriority(current)
	if candidatePriority != currentPriority {
		return candidatePriority < currentPriority
	}
	if candidateAmount != currentAmount {
		return candidateAmount > currentAmount
	}
	if candidate.HealthScore != current.HealthScore {
		return candidate.HealthScore > current.HealthScore
	}
	return candidate.Position.FundName < current.Position.FundName
}

func swapSourcePriority(state model.PositionState) int {
	switch state.Action {
	case model.ActionReplaceWatch:
		return 1
	case model.ActionReduce:
		return 2
	case model.ActionPauseBuy:
		return 3
	default:
		return 4
	}
}

func replacementAmount(state model.PositionState, totalValue float64) float64 {
	excess := maxFloat64(0, (state.CurrentWeight-state.Position.TargetWeight)*totalValue)
	if excess > 0 {
		return excess
	}
	switch state.Action {
	case model.ActionReplaceWatch:
		return minFloat64(state.CurrentValue*0.30, totalValue*0.05)
	case model.ActionPauseBuy:
		return minFloat64(state.CurrentValue*0.20, totalValue*0.03)
	case model.ActionReduce:
		return minFloat64(state.CurrentValue*0.15, totalValue*0.02)
	default:
		return 0
	}
}

func stateSignalReason(state model.PositionState) string {
	if len(state.Reasons) == 0 {
		return "按目标权重执行"
	}
	return strings.Join(state.Reasons, "；")
}

func (e *Engine) BuildDCAPlan(asOf time.Time, portfolioName string, states []model.PositionState, totalValue float64) model.DCAPlanReport {
	frequency := strings.TrimSpace(e.strategy.Turnover.DCAFrequency)
	if frequency == "" {
		frequency = "monthly"
	}
	budget := maxFloat64(0, e.strategy.Turnover.MonthlyDCAAmount)
	minFundAmount := maxFloat64(0, e.strategy.Turnover.MinDCAFundAmount)
	maxFunds := e.strategy.Turnover.MaxDCAFunds
	if maxFunds <= 0 {
		maxFunds = 3
	}
	pauseOnRisk := e.pauseDCAOnRiskEnabled()
	plan := model.DCAPlanReport{
		Summary: model.DCAPlanSummary{
			PortfolioName:      portfolioName,
			PlanDate:           asOf,
			Frequency:          frequency,
			Budget:             budget,
			PauseOnRiskEnabled: pauseOnRisk,
			GeneratedAt:        asOf,
		},
	}
	if budget <= 0 {
		plan.Summary.Notes = append(plan.Summary.Notes, "未设置月度定投预算，本期不生成定投动作")
		return plan
	}
	if minFundAmount > 0 && budget < minFundAmount {
		plan.Summary.ReserveAmount = budget
		plan.Summary.Notes = append(plan.Summary.Notes, fmt.Sprintf("本期预算 %.0f 低于单基金最低定投金额 %.0f，暂不执行", budget, minFundAmount))
		return plan
	}
	if totalValue <= 0 {
		plan.Summary.ReserveAmount = budget
		plan.Summary.Notes = append(plan.Summary.Notes, "组合市值不可用，月度预算暂全部保留")
		return plan
	}

	candidates := make([]dcaPlanCandidate, 0)
	for _, state := range states {
		if !state.Position.DCAEnabled {
			continue
		}
		plan.Summary.EligibleFundCount++
		gapWeight := maxFloat64(0, state.Position.TargetWeight-state.CurrentWeight)
		gapAmount := gapWeight * totalValue
		reason := stateSignalReason(state)
		skipReason := ""
		switch state.Action {
		case model.ActionReplaceWatch, model.ActionReduce:
			skipReason = reason
		case model.ActionPauseBuy:
			if pauseOnRisk {
				skipReason = reason
			}
		}
		if skipReason == "" && gapAmount <= 0 {
			skipReason = "当前权重已接近或高于目标，本月预算暂不追加"
		}
		if skipReason != "" {
			plan.Skipped = append(plan.Skipped, model.DCASkippedFund{
				FundCode: state.Position.FundCode,
				FundName: state.Position.FundName,
				Action:   state.Action,
				Reason:   skipReason,
			})
			continue
		}
		candidates = append(candidates, dcaPlanCandidate{
			state:     state,
			gapWeight: gapWeight,
			gapAmount: gapAmount,
			priority:  dcaPriority(state, gapWeight),
			reason:    reason,
		})
	}

	if len(candidates) == 0 {
		plan.Summary.ReserveAmount = budget
		plan.Summary.Notes = append(plan.Summary.Notes, "当前没有适合继续定投的基金，本月预算暂保留")
		return plan
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].priority != candidates[j].priority {
			return candidates[i].priority > candidates[j].priority
		}
		if candidates[i].gapAmount != candidates[j].gapAmount {
			return candidates[i].gapAmount > candidates[j].gapAmount
		}
		if candidates[i].state.CurrentWeight != candidates[j].state.CurrentWeight {
			return candidates[i].state.CurrentWeight < candidates[j].state.CurrentWeight
		}
		return candidates[i].state.Position.FundCode < candidates[j].state.Position.FundCode
	})

	if len(candidates) > maxFunds {
		plan.Summary.Notes = append(plan.Summary.Notes, fmt.Sprintf("候选定投基金 %d 只，仅保留优先级最高的 %d 只", len(candidates), maxFunds))
		for _, candidate := range candidates[maxFunds:] {
			plan.Skipped = append(plan.Skipped, model.DCASkippedFund{
				FundCode: candidate.state.Position.FundCode,
				FundName: candidate.state.Position.FundName,
				Action:   candidate.state.Action,
				Reason:   "优先级低于本期入选基金，本月暂缓定投",
			})
		}
		candidates = candidates[:maxFunds]
	}
	if minFundAmount > 0 {
		filtered, skipped := filterSmallDCACandidates(candidates, budget, minFundAmount)
		for _, item := range skipped {
			plan.Skipped = append(plan.Skipped, model.DCASkippedFund{
				FundCode: item.state.Position.FundCode,
				FundName: item.state.Position.FundName,
				Action:   item.state.Action,
				Reason:   fmt.Sprintf("按当前预算分配后低于单基金最低定投金额 %.0f，本月暂缓定投", minFundAmount),
			})
		}
		candidates = filtered
	}
	if len(candidates) == 0 {
		plan.Summary.ReserveAmount = budget
		plan.Summary.Notes = append(plan.Summary.Notes, fmt.Sprintf("当前候选基金分配后均低于单基金最低定投金额 %.0f，本月预算暂保留", minFundAmount))
		return plan
	}
	allocations := distributeDCABudget(budget, candidates)
	for idx, candidate := range candidates {
		amount := allocations[idx]
		if amount <= 0 {
			plan.Skipped = append(plan.Skipped, model.DCASkippedFund{
				FundCode: candidate.state.Position.FundCode,
				FundName: candidate.state.Position.FundName,
				Action:   candidate.state.Action,
				Reason:   "预算分配后本期金额为 0，暂不执行",
			})
			continue
		}
		plan.Items = append(plan.Items, model.DCAPlanItem{
			FundCode:      candidate.state.Position.FundCode,
			FundName:      candidate.state.Position.FundName,
			Role:          candidate.state.Position.Role,
			Action:        candidate.state.Action,
			CurrentWeight: candidate.state.CurrentWeight,
			TargetWeight:  candidate.state.Position.TargetWeight,
			GapWeight:     candidate.gapWeight,
			PlannedAmount: amount,
			Priority:      idx + 1,
			Reason:        candidate.reason,
		})
		plan.Summary.PlannedAmount += amount
	}
	plan.Summary.SelectedFundCount = len(plan.Items)
	plan.Summary.ReserveAmount = maxFloat64(0, budget-plan.Summary.PlannedAmount)
	if plan.Summary.SelectedFundCount == 0 {
		plan.Summary.Notes = append(plan.Summary.Notes, "候选基金存在，但本期预算未分配到任何基金")
	}
	if plan.Summary.ReserveAmount > 0 {
		plan.Summary.Notes = append(plan.Summary.Notes, fmt.Sprintf("有 %.0f 元预算未分配，保留为本月机动资金", plan.Summary.ReserveAmount))
	}
	if minFundAmount > 0 {
		plan.Summary.Notes = append(plan.Summary.Notes, fmt.Sprintf("已应用单基金最低定投金额 %.0f 元", minFundAmount))
	}
	if pauseOnRisk {
		plan.Summary.Notes = append(plan.Summary.Notes, "已启用风险暂停规则，`PAUSE_BUY/REDUCE/REPLACE_WATCH` 基金不进入本月定投")
	}
	return plan
}

func (e *Engine) pauseDCAOnRiskEnabled() bool {
	if e.strategy.Turnover.PauseDCAOnRisk == nil {
		return true
	}
	return *e.strategy.Turnover.PauseDCAOnRisk
}

func dcaPriority(state model.PositionState, gapWeight float64) int {
	priority := int(gapWeight * 10000)
	if state.Action == model.ActionBuy {
		priority += 3000
	}
	if state.Position.Role == "core" {
		priority += 2000
	}
	if state.Position.Protected {
		priority += 500
	}
	return priority
}

func distributeDCABudget(budget float64, candidates []dcaPlanCandidate) []float64 {
	allocations := make([]float64, len(candidates))
	if budget <= 0 || len(candidates) == 0 {
		return allocations
	}
	var totalGapAmount float64
	for _, candidate := range candidates {
		totalGapAmount += candidate.gapAmount
	}
	if totalGapAmount <= 0 {
		equalAmount := budget / float64(len(candidates))
		for idx := range allocations {
			allocations[idx] = equalAmount
		}
		return allocations
	}
	for idx, candidate := range candidates {
		share := budget * candidate.gapAmount / totalGapAmount
		allocations[idx] = minFloat64(share, candidate.gapAmount)
	}
	return allocations
}

func filterSmallDCACandidates(candidates []dcaPlanCandidate, budget, minFundAmount float64) ([]dcaPlanCandidate, []dcaPlanCandidate) {
	if minFundAmount <= 0 || len(candidates) == 0 {
		return candidates, nil
	}
	working := append([]dcaPlanCandidate(nil), candidates...)
	skipped := make([]dcaPlanCandidate, 0)
	for len(working) > 0 {
		allocations := distributeDCABudget(budget, working)
		filtered := make([]dcaPlanCandidate, 0, len(working))
		removed := make([]dcaPlanCandidate, 0)
		for idx, candidate := range working {
			if allocations[idx] < minFundAmount {
				removed = append(removed, candidate)
				continue
			}
			filtered = append(filtered, candidate)
		}
		if len(removed) == 0 {
			return working, skipped
		}
		skipped = append(skipped, removed...)
		working = filtered
	}
	return nil, skipped
}

func isPreferredDCAState(states []model.PositionState, fundName string) bool {
	for _, state := range states {
		if state.Position.FundName == fundName {
			return state.Position.DCAEnabled
		}
	}
	return false
}

func buildExecutionPlan(recommendations []model.TradeRecommendation) *model.ExecutionPlan {
	if len(recommendations) == 0 {
		return nil
	}
	plan := &model.ExecutionPlan{}
	steps := make([]model.ExecutionStep, 0, len(recommendations)*2)
	order := 1
	for _, recommendation := range recommendations {
		switch recommendation.Kind {
		case "SWAP":
			plan.GrossSellAmount += recommendation.SuggestedAmount
			plan.GrossBuyAmount += recommendation.SuggestedAmount
			plan.SwapAmount += recommendation.SuggestedAmount
			steps = append(steps,
				model.ExecutionStep{
					Order:       order,
					Action:      "SELL",
					Fund:        recommendation.SourceFund,
					RelatedFund: recommendation.TargetFund,
					Amount:      recommendation.SuggestedAmount,
					Weight:      recommendation.SuggestedWeight,
					Reason:      recommendation.Reason,
				},
				model.ExecutionStep{
					Order:         order + 1,
					Action:        "BUY",
					Fund:          recommendation.TargetFund,
					RelatedFund:   recommendation.SourceFund,
					Amount:        recommendation.SuggestedAmount,
					Weight:        recommendation.SuggestedWeight,
					FundingSource: fmt.Sprintf("卖出 %s", recommendation.SourceFund),
					Reason:        recommendation.Reason,
				},
			)
			order += 2
		case "REDUCE":
			plan.GrossSellAmount += recommendation.SuggestedAmount
			plan.ReduceAmount += recommendation.SuggestedAmount
			steps = append(steps, model.ExecutionStep{
				Order:  order,
				Action: "SELL",
				Fund:   recommendation.SourceFund,
				Amount: recommendation.SuggestedAmount,
				Weight: recommendation.SuggestedWeight,
				Reason: recommendation.Reason,
			})
			order++
		case "BUY":
			plan.GrossBuyAmount += recommendation.SuggestedAmount
			plan.BuyAmount += recommendation.SuggestedAmount
			steps = append(steps, model.ExecutionStep{
				Order:         order,
				Action:        "BUY",
				Fund:          recommendation.TargetFund,
				Amount:        recommendation.SuggestedAmount,
				Weight:        recommendation.SuggestedWeight,
				FundingSource: "组合卖出回笼资金",
				Reason:        recommendation.Reason,
			})
			order++
		}
	}
	plan.NetCashChange = plan.GrossSellAmount - plan.GrossBuyAmount
	plan.Steps = steps
	return plan
}

func minFloat64(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat64(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func rollingReturn(history []model.FundSnapshot, days int) float64 {
	if len(history) < 2 {
		return 0
	}
	end := history[len(history)-1].NAV
	idx := len(history) - 1 - days
	if idx < 0 {
		idx = 0
	}
	start := history[idx].NAV
	if start == 0 {
		return 0
	}
	return end/start - 1
}

func peerAverage(states []model.PositionState, days int) map[string]float64 {
	totals := make(map[string]float64)
	counts := make(map[string]int)
	for _, state := range states {
		var value float64
		switch days {
		case 60:
			value = state.Return60D
		case 120:
			value = state.Return120D
		default:
			value = state.Return20D
		}
		totals[state.Position.Category] += value
		counts[state.Position.Category]++
	}
	avg := make(map[string]float64, len(totals))
	for category, total := range totals {
		avg[category] = total / float64(counts[category])
	}
	return avg
}

func containsTarget(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func categoryCounts(states []model.PositionState) map[string]int {
	counts := make(map[string]int)
	for _, state := range states {
		counts[state.Position.Category]++
	}
	return counts
}

func safeWeightedChange(totalValue, weightedDayChange float64) float64 {
	if totalValue == 0 {
		return 0
	}
	return weightedDayChange / totalValue
}
