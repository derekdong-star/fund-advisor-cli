package strategy

import (
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

func TestAnalyzeProducesExpectedActions(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.Turnover.Mode = "active"
	cfg.Strategy.CandidatePool.MaxExpenseRatio = 0.01
	cfg.Strategy.CandidatePool.MinFundSizeYi = 20
	cfg.Strategy.CandidatePool.MinEstablishedYears = 2
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	makeHistory := func(nav float64, returns []float64) []model.FundSnapshot {
		history := make([]model.FundSnapshot, 0, len(returns)+1)
		base := nav
		for idx, ret := range returns {
			history = append(history, model.FundSnapshot{TradeDate: now.AddDate(0, 0, -(len(returns) - idx)), NAV: base})
			base = base * (1 + ret)
		}
		history = append(history, model.FundSnapshot{TradeDate: now, NAV: base, DayChangePct: 0.01})
		return history
	}
	states := []model.PositionState{
		{
			Position: model.Position{FundCode: "A", FundName: "Underweight", Category: "core", Benchmark: "sp500", Role: "core", AccountValue: 1000, TargetWeight: 0.50, EstimatedUnits: 40},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory(8, []float64{0.01, 0.01, 0.01, 0.01}),
		},
		{
			Position: model.Position{FundCode: "B", FundName: "Overweight", Category: "satellite", Benchmark: "china_growth", Role: "satellite", AccountValue: 1000, TargetWeight: 0.20, EstimatedUnits: 300},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory(10, []float64{0.00, 0.00, 0.00, 0.00}),
		},
		{
			Position: model.Position{FundCode: "C", FundName: "Weak", Category: "satellite", Benchmark: "nasdaq100", Role: "satellite", AccountValue: 1000, TargetWeight: 0.30, EstimatedUnits: 120},
			Latest:   &model.FundSnapshot{TradeDate: now.AddDate(0, 0, -7), NAV: 5, DayChangePct: -0.03},
			History:  makeHistory(10, []float64{-0.1, -0.1, -0.1, -0.1}),
		},
	}
	candidates := []model.CandidateState{
		{
			Candidate: model.Candidate{FundCode: "D", FundName: "Candidate", Category: "satellite", Benchmark: "nasdaq100", Role: "satellite", ExpenseRatio: 0.005, FundSizeYi: 30, EstablishedYears: 3, IsIndex: true},
			Latest:    &model.FundSnapshot{TradeDate: now, NAV: 10},
			History:   makeHistory(10, []float64{0.03, 0.02, 0.01, 0.01}),
		},
		{
			Candidate: model.Candidate{FundCode: "E", FundName: "TooSmall", Category: "satellite", Benchmark: "nasdaq100", Role: "satellite", ExpenseRatio: 0.005, FundSizeYi: 5, EstablishedYears: 3, IsIndex: true},
			Latest:    &model.FundSnapshot{TradeDate: now, NAV: 10},
			History:   makeHistory(10, []float64{0.03, 0.02, 0.01, 0.01}),
		},
	}
	report := engine.Analyze("test", states, candidates)
	if len(report.Signals) != 3 {
		t.Fatalf("signal count = %d, want 3", len(report.Signals))
	}
	actions := map[string]model.Action{}
	for _, signal := range report.Signals {
		actions[signal.FundCode] = signal.Action
	}
	if got := actions["A"]; got != model.ActionBuy {
		t.Fatalf("fund A action = %s, want %s", got, model.ActionBuy)
	}
	if got := actions["B"]; got != model.ActionReduce {
		t.Fatalf("fund B action = %s, want %s", got, model.ActionReduce)
	}
	if got := actions["C"]; got != model.ActionReplaceWatch {
		t.Fatalf("fund C action = %s, want %s", got, model.ActionReplaceWatch)
	}
	if len(report.Candidates) != 1 {
		t.Fatalf("candidate count = %d, want 1", len(report.Candidates))
	}
	if got := report.Candidates[0].ReplaceFor[0]; got != "Weak" {
		t.Fatalf("candidate replace target = %s, want Weak", got)
	}
	if got := report.Candidates[0].Benchmark; got != "nasdaq100" {
		t.Fatalf("candidate benchmark = %s, want nasdaq100", got)
	}
	if len(report.Recommendations) == 0 {
		t.Fatalf("recommendation count = 0, want > 0")
	}
	if got := report.Recommendations[0].Kind; got != "SWAP" {
		t.Fatalf("first recommendation kind = %s, want SWAP", got)
	}
	for _, recommendation := range report.Recommendations {
		if recommendation.Kind == "SWAP" && recommendation.SourceFund == "Weak" && recommendation.TargetFund != "Candidate" {
			t.Fatalf("expected best candidate chosen for Weak, got %s", recommendation.TargetFund)
		}
	}
}

func TestAnalyzePrefersHigherPrioritySwapSource(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.CandidatePool.MaxExpenseRatio = 0.01
	cfg.Strategy.CandidatePool.MinFundSizeYi = 10
	cfg.Strategy.CandidatePool.MinEstablishedYears = 1
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	makeHistory := func(values []float64, latestDate time.Time) []model.FundSnapshot {
		history := make([]model.FundSnapshot, 0, len(values))
		for idx, nav := range values {
			history = append(history, model.FundSnapshot{TradeDate: now.AddDate(0, 0, -(len(values) - 1 - idx)), NAV: nav})
		}
		history[len(history)-1].TradeDate = latestDate
		history[len(history)-1].DayChangePct = 0.01
		return history
	}
	states := []model.PositionState{
		{
			Position: model.Position{FundCode: "A", FundName: "Big Reduce", Category: "shared", Benchmark: "hs300", Role: "satellite", AccountValue: 1000, TargetWeight: 0.20, EstimatedUnits: 55},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{10, 10.1, 10.0, 9.6}, now),
		},
		{
			Position: model.Position{FundCode: "B", FundName: "Pause Buy", Category: "shared", Benchmark: "hs300", Role: "satellite", AccountValue: 1000, TargetWeight: 0.25, EstimatedUnits: 25},
			Latest:   &model.FundSnapshot{TradeDate: now.AddDate(0, 0, -7), NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{10, 10.2, 10.3, 10.4}, now.AddDate(0, 0, -7)),
		},
		{
			Position: model.Position{FundCode: "C", FundName: "Stable", Category: "core", Benchmark: "sp500", Role: "core", AccountValue: 1000, TargetWeight: 0.55, EstimatedUnits: 20},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{10, 10.1, 10.2, 10.3}, now),
		},
	}
	candidates := []model.CandidateState{
		{
			Candidate: model.Candidate{FundCode: "D", FundName: "Shared Candidate", Category: "shared", Benchmark: "a500", Role: "satellite", ExpenseRatio: 0.005, FundSizeYi: 30, EstablishedYears: 3, IsIndex: true},
			Latest:    &model.FundSnapshot{TradeDate: now, NAV: 10},
			History:   makeHistory([]float64{9.5, 9.7, 9.9, 10.1}, now),
		},
	}

	cfg.Strategy.Turnover.Mode = "active"
	engine = NewEngine(cfg.Strategy)
	report := engine.Analyze("test", states, candidates)
	var swaps []model.TradeRecommendation
	for _, recommendation := range report.Recommendations {
		if recommendation.Kind == "SWAP" {
			swaps = append(swaps, recommendation)
		}
	}
	if len(swaps) != 1 {
		t.Fatalf("swap count = %d, want 1", len(swaps))
	}
	if got := swaps[0].SourceFund; got != "Big Reduce" {
		t.Fatalf("swap source = %s, want Big Reduce", got)
	}
}

func TestAnalyzeBuildsExecutionPlan(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.Turnover.Mode = "active"
	cfg.Strategy.CandidatePool.MaxExpenseRatio = 0.01
	cfg.Strategy.CandidatePool.MinFundSizeYi = 20
	cfg.Strategy.CandidatePool.MinEstablishedYears = 2
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	makeHistory := func(nav float64, returns []float64) []model.FundSnapshot {
		history := make([]model.FundSnapshot, 0, len(returns)+1)
		base := nav
		for idx, ret := range returns {
			history = append(history, model.FundSnapshot{TradeDate: now.AddDate(0, 0, -(len(returns) - idx)), NAV: base})
			base = base * (1 + ret)
		}
		history = append(history, model.FundSnapshot{TradeDate: now, NAV: base, DayChangePct: 0.01})
		return history
	}
	states := []model.PositionState{
		{
			Position: model.Position{FundCode: "A", FundName: "Underweight", Category: "core", Benchmark: "sp500", Role: "core", AccountValue: 1000, TargetWeight: 0.50, EstimatedUnits: 40},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory(8, []float64{0.01, 0.01, 0.01, 0.01}),
		},
		{
			Position: model.Position{FundCode: "B", FundName: "Overweight", Category: "satellite", Benchmark: "china_growth", Role: "satellite", AccountValue: 1000, TargetWeight: 0.20, EstimatedUnits: 300},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory(10, []float64{0.00, 0.00, 0.00, 0.00}),
		},
		{
			Position: model.Position{FundCode: "C", FundName: "Weak", Category: "satellite", Benchmark: "nasdaq100", Role: "satellite", AccountValue: 1000, TargetWeight: 0.30, EstimatedUnits: 120},
			Latest:   &model.FundSnapshot{TradeDate: now.AddDate(0, 0, -7), NAV: 5, DayChangePct: -0.03},
			History:  makeHistory(10, []float64{-0.1, -0.1, -0.1, -0.1}),
		},
	}
	candidates := []model.CandidateState{
		{
			Candidate: model.Candidate{FundCode: "D", FundName: "Candidate", Category: "satellite", Benchmark: "nasdaq100", Role: "satellite", ExpenseRatio: 0.005, FundSizeYi: 30, EstablishedYears: 3, IsIndex: true},
			Latest:    &model.FundSnapshot{TradeDate: now, NAV: 10},
			History:   makeHistory(10, []float64{0.03, 0.02, 0.01, 0.01}),
		},
	}

	report := engine.Analyze("test", states, candidates)
	if report.ExecutionPlan == nil {
		t.Fatalf("expected execution plan")
	}
	if got := len(report.ExecutionPlan.Steps); got != 4 {
		t.Fatalf("execution step count = %d, want 4", got)
	}
	if got := report.ExecutionPlan.Steps[0].Action; got != "SELL" {
		t.Fatalf("first step action = %s, want SELL", got)
	}
	if got := report.ExecutionPlan.Steps[1].FundingSource; got != "卖出 Weak" {
		t.Fatalf("second step funding source = %s, want 卖出 Weak", got)
	}
	if report.ExecutionPlan.GrossSellAmount <= 0 || report.ExecutionPlan.GrossBuyAmount <= 0 {
		t.Fatalf("expected positive gross buy/sell amounts: %+v", report.ExecutionPlan)
	}
	if report.ExecutionPlan.SwapAmount <= 0 || report.ExecutionPlan.BuyAmount <= 0 || report.ExecutionPlan.ReduceAmount <= 0 {
		t.Fatalf("expected swap/buy/reduce amounts to be positive: %+v", report.ExecutionPlan)
	}
}

func TestAnalyzeProtectsLongTermConvictionHoldings(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.Turnover.MaxProtectedReduceWeight = 0.22
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	makeHistory := func(values []float64) []model.FundSnapshot {
		history := make([]model.FundSnapshot, 0, len(values))
		for idx, nav := range values {
			history = append(history, model.FundSnapshot{TradeDate: now.AddDate(0, 0, -(len(values) - 1 - idx)), NAV: nav, DayChangePct: 0.01})
		}
		return history
	}
	states := []model.PositionState{
		{
			Position: model.Position{FundCode: "A", FundName: "Conviction", Category: "active_cn_equity", Benchmark: "hs300_hk_mix", Role: "satellite", Protected: true, AccountValue: 1000, TargetWeight: 0.13, EstimatedUnits: 40},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{8.8, 9.1, 9.4, 9.8, 10}),
		},
		{
			Position: model.Position{FundCode: "B", FundName: "Core", Category: "sp500", Benchmark: "sp500", Role: "core", AccountValue: 1000, TargetWeight: 0.87, EstimatedUnits: 120, DCAEnabled: true},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{9.5, 9.6, 9.8, 9.9, 10}),
		},
	}
	report := engine.Analyze("test", states, nil)
	for _, signal := range report.Signals {
		if signal.FundCode == "A" && signal.Action == model.ActionReduce {
			t.Fatalf("protected fund should not be reduced below protection threshold")
		}
	}
}


func TestAnalyzeLowTurnoverSuppressesRoutineReduce(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.BuySignal.MaxSinglePositionWeight = 0.60
	cfg.Strategy.SellSignal.OverweightRelativeThreshold = 0.10
	cfg.Strategy.SellSignal.OverweightAbsoluteThreshold = 0.05
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	makeHistory := func(values []float64) []model.FundSnapshot {
		history := make([]model.FundSnapshot, 0, len(values))
		for idx, nav := range values {
			history = append(history, model.FundSnapshot{TradeDate: now.AddDate(0, 0, -(len(values)-1-idx)), NAV: nav, DayChangePct: 0.01})
		}
		return history
	}
	states := []model.PositionState{
		{
			Position: model.Position{FundCode: "A", FundName: "Routine Overweight", Category: "active_cn_equity", Benchmark: "hs300", Role: "satellite", AccountValue: 1000, TargetWeight: 0.40, EstimatedUnits: 44},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{9.7, 9.8, 10, 10.1, 10.2}),
		},
		{
			Position: model.Position{FundCode: "B", FundName: "Counterweight", Category: "sp500", Benchmark: "sp500", Role: "core", DCAEnabled: true, AccountValue: 1000, TargetWeight: 0.60, EstimatedUnits: 44},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{9.6, 9.7, 9.8, 10, 10.1}),
		},
	}
	report := engine.Analyze("test", states, nil)
	for _, signal := range report.Signals {
		if signal.FundCode == "A" && signal.Action != model.ActionPauseBuy {
			t.Fatalf("routine overweight action = %s, want %s", signal.Action, model.ActionPauseBuy)
		}
	}
	for _, recommendation := range report.Recommendations {
		if recommendation.Kind == "REDUCE" && recommendation.SourceFund == "Routine Overweight" {
			t.Fatalf("routine overweight fund should not be reduced in low turnover mode")
		}
	}
}

func TestAnalyzePrefersDCAFundsForBuyRecommendations(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.Turnover.MonthlyDCAAmount = 3000
	cfg.Strategy.Turnover.PreferDCA = true
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	makeHistory := func(values []float64) []model.FundSnapshot {
		history := make([]model.FundSnapshot, 0, len(values))
		for idx, nav := range values {
			history = append(history, model.FundSnapshot{TradeDate: now.AddDate(0, 0, -(len(values) - 1 - idx)), NAV: nav, DayChangePct: 0.01})
		}
		return history
	}
	states := []model.PositionState{
		{
			Position: model.Position{FundCode: "A", FundName: "DCA Fund", Category: "sp500", Benchmark: "sp500", Role: "core", DCAEnabled: true, AccountValue: 1000, TargetWeight: 0.6, EstimatedUnits: 20},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{9.2, 9.4, 9.6, 9.8, 10}),
		},
		{
			Position: model.Position{FundCode: "B", FundName: "Non DCA Fund", Category: "hk_dividend", Benchmark: "hsi_div_lowvol", Role: "core", AccountValue: 1000, TargetWeight: 0.4, EstimatedUnits: 25},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{9.2, 9.4, 9.6, 9.8, 10}),
		},
		{
			Position: model.Position{FundCode: "C", FundName: "Large Existing", Category: "active_cn_equity", Benchmark: "hs300", Role: "satellite", AccountValue: 1000, TargetWeight: 0.0, EstimatedUnits: 200},
			Latest:   &model.FundSnapshot{TradeDate: now, NAV: 10, DayChangePct: 0.01},
			History:  makeHistory([]float64{10, 10, 10, 10, 10}),
		},
	}
	report := engine.Analyze("test", states, nil)
	if len(report.Recommendations) == 0 {
		t.Fatalf("expected buy recommendations")
	}
	firstBuy := ""
	for _, recommendation := range report.Recommendations {
		if recommendation.Kind != "BUY" {
			continue
		}
		if firstBuy == "" {
			firstBuy = recommendation.TargetFund
		}
		if recommendation.TargetFund == "DCA Fund" && recommendation.SuggestedAmount > 3000 {
			t.Fatalf("dca amount = %.2f, want <= 3000", recommendation.SuggestedAmount)
		}
		if recommendation.TargetFund == "Non DCA Fund" {
			t.Fatalf("non-dca fund should not receive buy recommendation in low turnover mode")
		}
	}
	if firstBuy != "DCA Fund" {
		t.Fatalf("first buy target = %s, want DCA Fund", firstBuy)
	}
	for _, signal := range report.Signals {
		if signal.FundCode == "B" && signal.Action == model.ActionBuy {
			t.Fatalf("non-dca underweight fund should not be marked buy in low turnover mode")
		}
	}
}

func TestBuildDCAPlanSkipsRiskFundsAndCapsSelection(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.Turnover.MonthlyDCAAmount = 6000
	cfg.Strategy.Turnover.MaxDCAFunds = 2
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	states := []model.PositionState{
		{
			Position:      model.Position{FundCode: "A", FundName: "Core DCA", Role: "core", DCAEnabled: true, TargetWeight: 0.40, Protected: true},
			CurrentWeight: 0.10,
			Action:        model.ActionBuy,
			Reasons:       []string{"继续按计划定投"},
		},
		{
			Position:      model.Position{FundCode: "B", FundName: "Dividend DCA", Role: "core", DCAEnabled: true, TargetWeight: 0.30},
			CurrentWeight: 0.15,
			Action:        model.ActionBuy,
			Reasons:       []string{"当前权重低于目标"},
		},
		{
			Position:      model.Position{FundCode: "C", FundName: "Satellite DCA", Role: "satellite", DCAEnabled: true, TargetWeight: 0.15},
			CurrentWeight: 0.05,
			Action:        model.ActionHold,
			Reasons:       []string{"继续观察"},
		},
		{
			Position:      model.Position{FundCode: "D", FundName: "Paused DCA", Role: "core", DCAEnabled: true, TargetWeight: 0.15},
			CurrentWeight: 0.05,
			Action:        model.ActionPauseBuy,
			Reasons:       []string{"短期风险偏高"},
		},
	}

	plan := engine.BuildDCAPlan(now, "test", states, 100000)
	if got := len(plan.Items); got != 2 {
		t.Fatalf("selected items = %d, want 2", got)
	}
	if got := plan.Items[0].FundName; got != "Core DCA" {
		t.Fatalf("first selected fund = %s, want Core DCA", got)
	}
	if got := plan.Items[1].FundName; got != "Dividend DCA" {
		t.Fatalf("second selected fund = %s, want Dividend DCA", got)
	}
	if plan.Summary.PlannedAmount != 6000 {
		t.Fatalf("planned amount = %.2f, want 6000", plan.Summary.PlannedAmount)
	}
	if plan.Summary.ReserveAmount != 0 {
		t.Fatalf("reserve amount = %.2f, want 0", plan.Summary.ReserveAmount)
	}
	skipped := map[string]string{}
	for _, item := range plan.Skipped {
		skipped[item.FundName] = item.Reason
	}
	if got := skipped["Paused DCA"]; got != "短期风险偏高" {
		t.Fatalf("paused skip reason = %q, want 短期风险偏高", got)
	}
	if got := skipped["Satellite DCA"]; got != "优先级低于本期入选基金，本月暂缓定投" {
		t.Fatalf("satellite skip reason = %q, want priority skip", got)
	}
	if !plan.Summary.PauseOnRiskEnabled {
		t.Fatalf("pause on risk should be enabled")
	}
}

func TestBuildDCAPlanAllowsPauseBuyWhenRiskPauseDisabled(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	disabled := false
	cfg.Strategy.Turnover.PauseDCAOnRisk = &disabled
	cfg.Strategy.Turnover.MonthlyDCAAmount = 5000
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	states := []model.PositionState{
		{
			Position:      model.Position{FundCode: "A", FundName: "Paused But Allowed", Role: "core", DCAEnabled: true, TargetWeight: 0.30},
			CurrentWeight: 0.10,
			Action:        model.ActionPauseBuy,
			Reasons:       []string{"估值偏高，观察"},
		},
	}

	plan := engine.BuildDCAPlan(now, "test", states, 100000)
	if got := len(plan.Items); got != 1 {
		t.Fatalf("selected items = %d, want 1", got)
	}
	if plan.Items[0].FundName != "Paused But Allowed" {
		t.Fatalf("selected fund = %s, want Paused But Allowed", plan.Items[0].FundName)
	}
	if plan.Summary.PauseOnRiskEnabled {
		t.Fatalf("pause on risk should be disabled")
	}
}

func TestBuildDCAPlanSkipsTinyAllocationsBelowMinimum(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.Turnover.MonthlyDCAAmount = 5000
	cfg.Strategy.Turnover.MinDCAFundAmount = 1000
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	states := []model.PositionState{
		{
			Position:      model.Position{FundCode: "A", FundName: "Primary One", Role: "core", DCAEnabled: true, TargetWeight: 0.15},
			CurrentWeight: 0.0534,
			Action:        model.ActionBuy,
			Reasons:       []string{"继续定投"},
		},
		{
			Position:      model.Position{FundCode: "B", FundName: "Primary Two", Role: "core", DCAEnabled: true, TargetWeight: 0.15},
			CurrentWeight: 0.0534,
			Action:        model.ActionBuy,
			Reasons:       []string{"继续定投"},
		},
		{
			Position:      model.Position{FundCode: "C", FundName: "Tiny Residual", Role: "core", DCAEnabled: true, TargetWeight: 0.10},
			CurrentWeight: 0.0992,
			Action:        model.ActionHold,
			Reasons:       []string{"继续观察"},
		},
	}

	plan := engine.BuildDCAPlan(now, "test", states, 262100)
	if got := len(plan.Items); got != 2 {
		t.Fatalf("selected items = %d, want 2", got)
	}
	for _, item := range plan.Items {
		if item.PlannedAmount < 1000 {
			t.Fatalf("planned amount = %.2f, want >= 1000", item.PlannedAmount)
		}
	}
	skipped := map[string]string{}
	for _, item := range plan.Skipped {
		skipped[item.FundName] = item.Reason
	}
	if got := skipped["Tiny Residual"]; got != "按当前预算分配后低于单基金最低定投金额 1000，本月暂缓定投" {
		t.Fatalf("tiny residual skip reason = %q, want minimum-amount skip", got)
	}
}

func TestBuildDCAPlanReservesBudgetWhenBelowMinimumAmount(t *testing.T) {
	t.Parallel()
	cfg := config.Default()
	cfg.Strategy.Turnover.MonthlyDCAAmount = 500
	cfg.Strategy.Turnover.MinDCAFundAmount = 1000
	engine := NewEngine(cfg.Strategy)
	now := time.Now().UTC()
	plan := engine.BuildDCAPlan(now, "test", nil, 100000)
	if got := len(plan.Items); got != 0 {
		t.Fatalf("selected items = %d, want 0", got)
	}
	if plan.Summary.ReserveAmount != 500 {
		t.Fatalf("reserve amount = %.2f, want 500", plan.Summary.ReserveAmount)
	}
}
