package model

import "time"

type Action string

const (
	ActionBuy          Action = "BUY"
	ActionHold         Action = "HOLD"
	ActionPauseBuy     Action = "PAUSE_BUY"
	ActionReduce       Action = "REDUCE"
	ActionReplaceWatch Action = "REPLACE_WATCH"
)

type FundSnapshot struct {
	FundCode          string    `json:"fund_code"`
	FundName          string    `json:"fund_name"`
	TradeDate         time.Time `json:"trade_date"`
	NAV               float64   `json:"nav"`
	AccNAV            float64   `json:"acc_nav"`
	DayChangePct      float64   `json:"day_change_pct"`
	EstimateNAV       float64   `json:"estimate_nav"`
	EstimateChangePct float64   `json:"estimate_change_pct"`
	Source            string    `json:"source"`
	CreatedAt         time.Time `json:"created_at"`
}

type Position struct {
	FundCode       string  `json:"fund_code"`
	FundName       string  `json:"fund_name"`
	Category       string  `json:"category"`
	Benchmark      string  `json:"benchmark"`
	Role           string  `json:"role"`
	Status         string  `json:"status"`
	Protected      bool    `json:"protected,omitempty"`
	DCAEnabled     bool    `json:"dca_enabled,omitempty"`
	AccountValue   float64 `json:"account_value"`
	TargetWeight   float64 `json:"target_weight"`
	EstimatedUnits float64 `json:"estimated_units"`
}

type Candidate struct {
	FundCode         string   `json:"fund_code"`
	FundName         string   `json:"fund_name"`
	Category         string   `json:"category"`
	Benchmark        string   `json:"benchmark"`
	Role             string   `json:"role"`
	Protected        bool     `json:"protected,omitempty"`
	DCAEnabled       bool     `json:"dca_enabled,omitempty"`
	ExpenseRatio     float64  `json:"expense_ratio,omitempty"`
	FundSizeYi       float64  `json:"fund_size_yi,omitempty"`
	EstablishedYears float64  `json:"established_years,omitempty"`
	IsIndex          bool     `json:"is_index,omitempty"`
	Tags             []string `json:"tags,omitempty"`
}

type PositionState struct {
	Position          Position       `json:"position"`
	Latest            *FundSnapshot  `json:"latest"`
	History           []FundSnapshot `json:"history"`
	CurrentValue      float64        `json:"current_value"`
	CurrentWeight     float64        `json:"current_weight"`
	Return20D         float64        `json:"return_20d"`
	Return60D         float64        `json:"return_60d"`
	Return120D        float64        `json:"return_120d"`
	HoldingCost       float64        `json:"holding_cost,omitempty"`
	UnrealizedPnL     float64        `json:"unrealized_pnl,omitempty"`
	UnrealizedPnLPct  float64        `json:"unrealized_pnl_pct,omitempty"`
	LastLedgerTradeAt time.Time      `json:"last_ledger_trade_at,omitempty"`
	LedgerTradeCount  int            `json:"ledger_trade_count,omitempty"`
	LedgerApplied     bool           `json:"ledger_applied,omitempty"`
	Reasons           []string       `json:"reasons,omitempty"`
	Action            Action         `json:"action"`
	HealthScore       int            `json:"health_score"`
	Drift             float64        `json:"drift"`
}

type CandidateState struct {
	Candidate  Candidate      `json:"candidate"`
	Latest     *FundSnapshot  `json:"latest"`
	History    []FundSnapshot `json:"history"`
	Return20D  float64        `json:"return_20d"`
	Return60D  float64        `json:"return_60d"`
	Return120D float64        `json:"return_120d"`
	Score      int            `json:"score"`
	Reasons    []string       `json:"reasons,omitempty"`
	ReplaceFor []string       `json:"replace_for,omitempty"`
}

type FundSignal struct {
	FundCode        string    `json:"fund_code"`
	FundName        string    `json:"fund_name"`
	Action          Action    `json:"action"`
	Score           int       `json:"score"`
	CurrentWeight   float64   `json:"current_weight"`
	TargetWeight    float64   `json:"target_weight"`
	Drift           float64   `json:"drift"`
	CurrentValue    float64   `json:"current_value"`
	Return20D       float64   `json:"return_20d"`
	Return60D       float64   `json:"return_60d"`
	Return120D      float64   `json:"return_120d"`
	LatestTradeDate time.Time `json:"latest_trade_date"`
	Reason          string    `json:"reason"`
	CreatedAt       time.Time `json:"created_at"`
}

type CandidateSuggestion struct {
	FundCode         string    `json:"fund_code"`
	FundName         string    `json:"fund_name"`
	Category         string    `json:"category"`
	Benchmark        string    `json:"benchmark"`
	Role             string    `json:"role"`
	Score            int       `json:"score"`
	Return20D        float64   `json:"return_20d"`
	Return60D        float64   `json:"return_60d"`
	Return120D       float64   `json:"return_120d"`
	ExpenseRatio     float64   `json:"expense_ratio,omitempty"`
	FundSizeYi       float64   `json:"fund_size_yi,omitempty"`
	EstablishedYears float64   `json:"established_years,omitempty"`
	IsIndex          bool      `json:"is_index,omitempty"`
	LatestTradeDate  time.Time `json:"latest_trade_date"`
	ReplaceFor       []string  `json:"replace_for,omitempty"`
	Reason           string    `json:"reason"`
	EnhancedReason   string    `json:"enhanced_reason,omitempty"`
}

type TradeRecommendation struct {
	Kind            string    `json:"kind"`
	SourceFund      string    `json:"source_fund,omitempty"`
	TargetFund      string    `json:"target_fund,omitempty"`
	SuggestedWeight float64   `json:"suggested_weight"`
	SuggestedAmount float64   `json:"suggested_amount"`
	Reason          string    `json:"reason"`
	EnhancedReason  string    `json:"enhanced_reason,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type ExecutionStep struct {
	Order         int     `json:"order"`
	Action        string  `json:"action"`
	Fund          string  `json:"fund"`
	RelatedFund   string  `json:"related_fund,omitempty"`
	Amount        float64 `json:"amount"`
	Weight        float64 `json:"weight"`
	FundingSource string  `json:"funding_source,omitempty"`
	Reason        string  `json:"reason"`
}

type ExecutionPlan struct {
	GrossSellAmount float64         `json:"gross_sell_amount"`
	GrossBuyAmount  float64         `json:"gross_buy_amount"`
	SwapAmount      float64         `json:"swap_amount"`
	ReduceAmount    float64         `json:"reduce_amount"`
	BuyAmount       float64         `json:"buy_amount"`
	NetCashChange   float64         `json:"net_cash_change"`
	Steps           []ExecutionStep `json:"steps,omitempty"`
}

type AnalysisSummary struct {
	PortfolioName        string         `json:"portfolio_name"`
	RunDate              time.Time      `json:"run_date"`
	PortfolioValue       float64        `json:"portfolio_value"`
	WeightedDayChangePct float64        `json:"weighted_day_change_pct"`
	ActionCounts         map[Action]int `json:"action_counts"`
	CandidateCount       int            `json:"candidate_count"`
	Notes                []string       `json:"notes,omitempty"`
	GeneratedAt          time.Time      `json:"generated_at"`
}

type AnalysisReport struct {
	RunID           int64                 `json:"run_id"`
	Summary         AnalysisSummary       `json:"summary"`
	Signals         []FundSignal          `json:"signals"`
	Candidates      []CandidateSuggestion `json:"candidates,omitempty"`
	Recommendations []TradeRecommendation `json:"recommendations,omitempty"`
	ExecutionPlan   *ExecutionPlan        `json:"execution_plan,omitempty"`
	DCAPlan         *DCAPlanReport        `json:"dca_plan,omitempty"`
	Position        []PositionState       `json:"position,omitempty"`
}

type DCAPlanSummary struct {
	PortfolioName      string    `json:"portfolio_name"`
	PlanDate           time.Time `json:"plan_date"`
	Frequency          string    `json:"frequency"`
	Budget             float64   `json:"budget"`
	PlannedAmount      float64   `json:"planned_amount"`
	ReserveAmount      float64   `json:"reserve_amount"`
	EligibleFundCount  int       `json:"eligible_fund_count"`
	SelectedFundCount  int       `json:"selected_fund_count"`
	PauseOnRiskEnabled bool      `json:"pause_on_risk_enabled"`
	Notes              []string  `json:"notes,omitempty"`
	GeneratedAt        time.Time `json:"generated_at"`
}

type DCAPlanItem struct {
	FundCode      string  `json:"fund_code"`
	FundName      string  `json:"fund_name"`
	Role          string  `json:"role"`
	Action        Action  `json:"action"`
	CurrentWeight float64 `json:"current_weight"`
	TargetWeight  float64 `json:"target_weight"`
	GapWeight     float64 `json:"gap_weight"`
	PlannedAmount float64 `json:"planned_amount"`
	Priority      int     `json:"priority"`
	Reason        string  `json:"reason"`
}

type DCASkippedFund struct {
	FundCode string `json:"fund_code"`
	FundName string `json:"fund_name"`
	Action   Action `json:"action"`
	Reason   string `json:"reason"`
}

type DCAPlanReport struct {
	Summary DCAPlanSummary   `json:"summary"`
	Items   []DCAPlanItem    `json:"items,omitempty"`
	Skipped []DCASkippedFund `json:"skipped,omitempty"`
}

type BacktestSummary struct {
	PortfolioName             string    `json:"portfolio_name"`
	StartDate                 time.Time `json:"start_date"`
	EndDate                   time.Time `json:"end_date"`
	TradingDays               int       `json:"trading_days"`
	RebalanceEvery            int       `json:"rebalance_every"`
	RebalanceCount            int       `json:"rebalance_count"`
	TradeCount                int       `json:"trade_count"`
	InitialValue              float64   `json:"initial_value"`
	FinalValue                float64   `json:"final_value"`
	BenchmarkInitialValue     float64   `json:"benchmark_initial_value"`
	BenchmarkFinalValue       float64   `json:"benchmark_final_value"`
	CashFinal                 float64   `json:"cash_final"`
	TotalReturn               float64   `json:"total_return"`
	BenchmarkReturn           float64   `json:"benchmark_return"`
	ExcessReturn              float64   `json:"excess_return"`
	AnnualizedReturn          float64   `json:"annualized_return"`
	BenchmarkAnnualizedReturn float64   `json:"benchmark_annualized_return"`
	MaxDrawdown               float64   `json:"max_drawdown"`
	BenchmarkMaxDrawdown      float64   `json:"benchmark_max_drawdown"`
	Notes                     []string  `json:"notes,omitempty"`
}

type BacktestPoint struct {
	Date           time.Time `json:"date"`
	StrategyValue  float64   `json:"strategy_value"`
	BenchmarkValue float64   `json:"benchmark_value"`
	Cash           float64   `json:"cash"`
}

type BacktestTrade struct {
	Date        time.Time `json:"date"`
	Action      string    `json:"action"`
	Fund        string    `json:"fund"`
	RelatedFund string    `json:"related_fund,omitempty"`
	Amount      float64   `json:"amount"`
	Price       float64   `json:"price"`
	Units       float64   `json:"units"`
	Reason      string    `json:"reason"`
}

type BacktestReport struct {
	Summary BacktestSummary `json:"summary"`
	Points  []BacktestPoint `json:"points,omitempty"`
	Trades  []BacktestTrade `json:"trades,omitempty"`
}

type MarketPoolSummary struct {
	RunDate       time.Time `json:"run_date"`
	UniverseCount int       `json:"universe_count"`
	MatchedCount  int       `json:"matched_count"`
	EligibleCount int       `json:"eligible_count"`
	SelectedCount int       `json:"selected_count"`
	RetainedCount int       `json:"retained_count"`
	GeneratedAt   time.Time `json:"generated_at"`
	Notes         []string  `json:"notes,omitempty"`
}

type MarketPoolItem struct {
	Rank             int       `json:"rank"`
	ThemeKey         string    `json:"theme_key"`
	ThemeLabel       string    `json:"theme_label"`
	FundCode         string    `json:"fund_code"`
	FundName         string    `json:"fund_name"`
	FundType         string    `json:"fund_type"`
	Score            int       `json:"score"`
	Retained         bool      `json:"retained"`
	Return20D        float64   `json:"return_20d"`
	Return60D        float64   `json:"return_60d"`
	Return120D       float64   `json:"return_120d"`
	Return250D       float64   `json:"return_250d"`
	MaxDrawdown120D  float64   `json:"max_drawdown_120d"`
	FundSizeYi       float64   `json:"fund_size_yi"`
	EstablishedYears float64   `json:"established_years"`
	LatestTradeDate  time.Time `json:"latest_trade_date"`
	Reason           string    `json:"reason"`
}

type MarketPoolReport struct {
	RunID   int64             `json:"run_id"`
	Summary MarketPoolSummary `json:"summary"`
	Items   []MarketPoolItem  `json:"items,omitempty"`
}

type FetchResult struct {
	Code      string         `json:"code"`
	Name      string         `json:"name"`
	Snapshots []FundSnapshot `json:"snapshots"`
}

type MarketSearchFund struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	FundType string `json:"fund_type"`
	Spell    string `json:"spell"`
}

type MarketFundProfile struct {
	Fund             MarketSearchFund `json:"fund"`
	Latest           *FundSnapshot    `json:"latest,omitempty"`
	History          []FundSnapshot   `json:"history,omitempty"`
	FundSizeYi       float64          `json:"fund_size_yi"`
	EstablishedYears float64          `json:"established_years"`
	IsIndex          bool             `json:"is_index"`
}
