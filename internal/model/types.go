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
	Opportunity     *OpportunityReport    `json:"opportunity,omitempty"`
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

type MomentumPoolSummary struct {
	RunDate                       time.Time `json:"run_date"`
	StrategyFingerprint           string    `json:"strategy_fingerprint"`
	UniverseCount                 int       `json:"universe_count"`
	EquityCount                   int       `json:"equity_count"`
	EvaluatedCount                int       `json:"evaluated_count"`
	QualifiedCount                int       `json:"qualified_count"`
	SelectedCount                 int       `json:"selected_count"`
	ChallengerCount               int       `json:"challenger_count"`
	PositiveBreadth               float64   `json:"positive_breadth"`
	MarketMedianReturn120D        float64   `json:"market_median_return_120d"`
	AveragePairCorrelation        float64   `json:"average_pair_correlation"`
	MaxPairCorrelation            float64   `json:"max_pair_correlation"`
	MaxCorrelationPair            string    `json:"max_correlation_pair,omitempty"`
	Regime                        string    `json:"regime"`
	PreviousEvaluatedCount        int       `json:"previous_evaluated_count"`
	PreviousAverageReturn         float64   `json:"previous_average_return"`
	PreviousWinRate               float64   `json:"previous_win_rate"`
	PreviousChallengerCount       int       `json:"previous_challenger_evaluated_count"`
	PreviousChallengerReturn      float64   `json:"previous_challenger_average_return"`
	PreviousChallengerWinRate     float64   `json:"previous_challenger_win_rate"`
	ChallengerExcessReturn        float64   `json:"challenger_excess_vs_selected"`
	ChallengerTrackingStartDate   time.Time `json:"challenger_tracking_start_date"`
	ChallengerTrackingDays        int       `json:"challenger_tracking_days"`
	ChallengerValidatedCandidates int       `json:"challenger_validated_candidates"`
	ChallengerQualifiedCandidates int       `json:"challenger_qualified_candidates"`
	ForwardValidationBatches      int       `json:"forward_validation_batches"`
	ForwardValidationCandidates   int       `json:"forward_validation_candidates"`
	CurrentValidatedCandidates    int       `json:"current_validated_candidates"`
	CurrentQualifiedCandidates    int       `json:"current_qualified_candidates"`
	ForwardValidationDays         int       `json:"forward_validation_days"`
	TrialReturnThreshold          float64   `json:"trial_return_threshold"`
	PendingValidationBatches      int       `json:"pending_validation_batches"`
	ExpiredValidationBatches      int       `json:"expired_validation_batches"`
	UnverifiableValidationBatches int       `json:"unverifiable_validation_batches"`
	OldestPendingStartDate        time.Time `json:"oldest_pending_start_date"`
	OldestPendingTradingDays      int       `json:"oldest_pending_trading_days"`
	ForwardAverageReturn          float64   `json:"forward_average_return"`
	ForwardWinRate                float64   `json:"forward_win_rate"`
	ForwardBatchWinRate           float64   `json:"forward_batch_win_rate"`
	Readiness                     string    `json:"readiness"`
	SuggestedTotalWeight          float64   `json:"suggested_total_weight"`
	GeneratedAt                   time.Time `json:"generated_at"`
	Notes                         []string  `json:"notes,omitempty"`
}

type MomentumPoolItem struct {
	Rank               int       `json:"rank"`
	FundCode           string    `json:"fund_code"`
	FundName           string    `json:"fund_name"`
	FundType           string    `json:"fund_type"`
	MomentumScore      float64   `json:"momentum_score"`
	Score              float64   `json:"score"`
	Return20D          float64   `json:"return_20d"`
	Return60D          float64   `json:"return_60d"`
	Return120D         float64   `json:"return_120d"`
	Return250D         float64   `json:"return_250d"`
	ObservationDays    int       `json:"observation_days"`
	ObservationReturn  float64   `json:"observation_return"`
	MaxDrawdown120D    float64   `json:"max_drawdown_120d"`
	FundSizeYi         float64   `json:"fund_size_yi"`
	EstablishedYears   float64   `json:"established_years"`
	LatestTradeDate    time.Time `json:"latest_trade_date"`
	LatestNAV          float64   `json:"latest_nav"`
	PurchaseStatus     string    `json:"purchase_status"`
	MinimumPurchase    float64   `json:"minimum_purchase"`
	DailyLimit         float64   `json:"daily_limit"`
	MaxPeerCorrelation float64   `json:"max_peer_correlation"`
	MaxCorrelatedPeer  string    `json:"max_correlated_peer,omitempty"`
	SuggestedWeight    float64   `json:"suggested_weight"`
	Reason             string    `json:"reason"`
}

type MomentumPoolReport struct {
	Summary               MomentumPoolSummary       `json:"summary"`
	Items                 []MomentumPoolItem        `json:"items,omitempty"`
	ItemValidations       []MomentumFundValidation  `json:"item_validations,omitempty"`
	Challengers           []MomentumPoolItem        `json:"challengers,omitempty"`
	ChallengerValidations []MomentumFundValidation  `json:"challenger_validations,omitempty"`
	Opportunities         []MomentumOpportunityItem `json:"opportunities,omitempty"`
	ValidatedFundCodes    []string                  `json:"validated_fund_codes,omitempty"`
	QualifiedFundCodes    []string                  `json:"qualified_fund_codes,omitempty"`
}

type MomentumFundValidation struct {
	FundCode         string    `json:"fund_code"`
	CompletedBatches int       `json:"completed_batches"`
	AverageReturn    float64   `json:"average_return"`
	LatestReturn     float64   `json:"latest_return"`
	WinRate          float64   `json:"win_rate"`
	LatestEndDate    time.Time `json:"latest_end_date"`
	Qualified        bool      `json:"qualified"`
	Execution        string    `json:"execution"`
}

type MomentumOpportunityItem struct {
	Rank              int     `json:"rank"`
	FundCode          string  `json:"fund_code"`
	FundName          string  `json:"fund_name"`
	Source            string  `json:"source"`
	Evidence          string  `json:"evidence"`
	Execution         string  `json:"execution"`
	SuggestedWeight   float64 `json:"suggested_weight"`
	MomentumScore     float64 `json:"momentum_score"`
	Return20D         float64 `json:"return_20d"`
	Return60D         float64 `json:"return_60d"`
	Return120D        float64 `json:"return_120d"`
	Return250D        float64 `json:"return_250d"`
	ObservationDays   int     `json:"observation_days"`
	ObservationReturn float64 `json:"observation_return"`
	PurchaseStatus    string  `json:"purchase_status"`
}

type OpportunitySummary struct {
	RunDate              time.Time `json:"run_date"`
	Window               string    `json:"window"`
	Reason               string    `json:"reason"`
	HoldingOpportunities int       `json:"holding_opportunities"`
	CandidateCount       int       `json:"candidate_count"`
	GeneratedAt          time.Time `json:"generated_at"`
}

type OpportunityHolding struct {
	Priority      int       `json:"priority"`
	FundCode      string    `json:"fund_code"`
	FundName      string    `json:"fund_name"`
	CurrentWeight float64   `json:"current_weight"`
	TargetWeight  float64   `json:"target_weight"`
	PlannedAmount float64   `json:"planned_amount"`
	Return20D     float64   `json:"return_20d"`
	Return60D     float64   `json:"return_60d"`
	Return120D    float64   `json:"return_120d"`
	CreatedAt     time.Time `json:"created_at"`
	Reason        string    `json:"reason"`
}

type OpportunityCandidate struct {
	Rank       int       `json:"rank"`
	ThemeLabel string    `json:"theme_label"`
	FundCode   string    `json:"fund_code"`
	FundName   string    `json:"fund_name"`
	Score      int       `json:"score"`
	Retained   bool      `json:"retained"`
	Return20D  float64   `json:"return_20d"`
	Return60D  float64   `json:"return_60d"`
	Return120D float64   `json:"return_120d"`
	CreatedAt  time.Time `json:"created_at"`
	Reason     string    `json:"reason"`
}

type OpportunityReport struct {
	Summary    OpportunitySummary     `json:"summary"`
	Holdings   []OpportunityHolding   `json:"holdings,omitempty"`
	Candidates []OpportunityCandidate `json:"candidates,omitempty"`
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

type MomentumBacktestSummary struct {
	StartDate                 time.Time `json:"start_date"`
	EndDate                   time.Time `json:"end_date"`
	CandidateCount            int       `json:"candidate_count"`
	UniverseSeed              uint32    `json:"universe_seed"`
	SelectionCount            int       `json:"selection_count"`
	RebalanceEvery            int       `json:"rebalance_every"`
	PeriodCount               int       `json:"period_count"`
	InvestedPeriodCount       int       `json:"invested_period_count"`
	CashPeriodCount           int       `json:"cash_period_count"`
	TotalReturn               float64   `json:"total_return"`
	AnnualizedReturn          float64   `json:"annualized_return"`
	MaxDrawdown               float64   `json:"max_drawdown"`
	BenchmarkTotalReturn      float64   `json:"benchmark_total_return"`
	BenchmarkAnnualizedReturn float64   `json:"benchmark_annualized_return"`
	BenchmarkMaxDrawdown      float64   `json:"benchmark_max_drawdown"`
	ExcessReturn              float64   `json:"excess_return"`
	ExcessWinRate             float64   `json:"excess_win_rate"`
	ValidationSplitDate       time.Time `json:"validation_split_date"`
	EarlyExcessReturn         float64   `json:"early_excess_return"`
	RecentExcessReturn        float64   `json:"recent_excess_return"`
	WinRate                   float64   `json:"win_rate"`
	AveragePeriodReturn       float64   `json:"average_period_return"`
	AverageTurnover           float64   `json:"average_turnover"`
	EstimatedCost             float64   `json:"estimated_cost"`
	FutureTopDecileHitRate    float64   `json:"future_top_decile_hit_rate"`
	FutureTopDecileCapture    float64   `json:"future_top_decile_capture_rate"`
	FutureTopSelectionCapture float64   `json:"future_top_selection_capture_rate"`
	AverageFuturePercentile   float64   `json:"average_future_return_percentile"`
	RecentTopDecileHitRate    float64   `json:"recent_future_top_decile_hit_rate"`
	RecentTopDecileCapture    float64   `json:"recent_future_top_decile_capture_rate"`
	WinnerEvaluationPeriods   int       `json:"winner_evaluation_periods"`
	RecentWinnerPeriods       int       `json:"recent_winner_evaluation_periods"`
	Notes                     []string  `json:"notes,omitempty"`
}

type MomentumBacktestPeriod struct {
	StartDate        time.Time `json:"start_date"`
	EndDate          time.Time `json:"end_date"`
	Return           float64   `json:"return"`
	BenchmarkReturn  float64   `json:"benchmark_return"`
	ExcessReturn     float64   `json:"excess_return"`
	PositiveBreadth  float64   `json:"positive_breadth"`
	MedianReturn60D  float64   `json:"median_return_60d"`
	MedianReturn120D float64   `json:"median_return_120d"`
	Turnover         float64   `json:"turnover"`
	TopDecileHitRate float64   `json:"future_top_decile_hit_rate"`
	TopDecileCapture float64   `json:"future_top_decile_capture_rate"`
	TopCaptureRate   float64   `json:"future_top_selection_capture_rate"`
	FuturePercentile float64   `json:"average_future_return_percentile"`
	WinnerEvaluated  bool      `json:"winner_evaluated"`
	Funds            []string  `json:"funds"`
}

type MomentumBacktestReport struct {
	Summary MomentumBacktestSummary  `json:"summary"`
	Periods []MomentumBacktestPeriod `json:"periods,omitempty"`
}

type MomentumBacktestSensitivitySummary struct {
	GeneratedAt               time.Time `json:"generated_at"`
	RequestedSeeds            int       `json:"requested_seeds"`
	CompletedSeeds            int       `json:"completed_seeds"`
	FailedSeeds               int       `json:"failed_seeds"`
	Days                      int       `json:"days"`
	SelectionCount            int       `json:"selection_count"`
	RebalanceEvery            int       `json:"rebalance_every"`
	PositiveTotalReturnSeeds  int       `json:"positive_total_return_seeds"`
	PositiveExcessReturnSeeds int       `json:"positive_excess_return_seeds"`
	PositiveRecentExcessSeeds int       `json:"positive_recent_excess_seeds"`
	AverageTotalReturn        float64   `json:"average_total_return"`
	AverageExcessReturn       float64   `json:"average_excess_return"`
	WorstTotalReturn          float64   `json:"worst_total_return"`
	WorstExcessReturn         float64   `json:"worst_excess_return"`
	WorstRecentExcessReturn   float64   `json:"worst_recent_excess_return"`
	WorstMaxDrawdown          float64   `json:"worst_max_drawdown"`
	AverageTopDecileHitRate   float64   `json:"average_future_top_decile_hit_rate"`
	WorstTopDecileHitRate     float64   `json:"worst_future_top_decile_hit_rate"`
	AverageTopDecileCapture   float64   `json:"average_future_top_decile_capture_rate"`
	WorstTopDecileCapture     float64   `json:"worst_future_top_decile_capture_rate"`
	AverageTopCaptureRate     float64   `json:"average_future_top_selection_capture_rate"`
	WorstTopCaptureRate       float64   `json:"worst_future_top_selection_capture_rate"`
	AverageFuturePercentile   float64   `json:"average_future_return_percentile"`
	WorstFuturePercentile     float64   `json:"worst_future_return_percentile"`
	AverageRecentTopDecileHit float64   `json:"average_recent_future_top_decile_hit_rate"`
	WorstRecentTopDecileHit   float64   `json:"worst_recent_future_top_decile_hit_rate"`
	Notes                     []string  `json:"notes,omitempty"`
}

type MomentumBacktestSensitivityItem struct {
	UniverseSeed       uint32  `json:"universe_seed"`
	CandidateCount     int     `json:"candidate_count"`
	TotalReturn        float64 `json:"total_return"`
	ExcessReturn       float64 `json:"excess_return"`
	RecentExcessReturn float64 `json:"recent_excess_return"`
	MaxDrawdown        float64 `json:"max_drawdown"`
	TopDecileHitRate   float64 `json:"future_top_decile_hit_rate"`
	TopDecileCapture   float64 `json:"future_top_decile_capture_rate"`
	TopCaptureRate     float64 `json:"future_top_selection_capture_rate"`
	FuturePercentile   float64 `json:"average_future_return_percentile"`
	RecentTopDecileHit float64 `json:"recent_future_top_decile_hit_rate"`
	Error              string  `json:"error,omitempty"`
}

type MomentumBacktestSensitivityReport struct {
	Summary MomentumBacktestSensitivitySummary `json:"summary"`
	Items   []MomentumBacktestSensitivityItem  `json:"items,omitempty"`
}

type MomentumBacktestParameterGridSummary struct {
	GeneratedAt           time.Time `json:"generated_at"`
	Days                  int       `json:"days"`
	Seeds                 int       `json:"seeds"`
	CombinationCount      int       `json:"combination_count"`
	CurrentSelectionCount int       `json:"current_selection_count"`
	CurrentRebalanceEvery int       `json:"current_rebalance_every"`
	CurrentRank           int       `json:"current_rank"`
	LeadingSelectionCount int       `json:"leading_selection_count"`
	LeadingRebalanceEvery int       `json:"leading_rebalance_every"`
	SelectionCounts       []int     `json:"selection_counts,omitempty"`
	RebalanceEveryValues  []int     `json:"rebalance_every_values,omitempty"`
	Notes                 []string  `json:"notes,omitempty"`
}

type MomentumBacktestParameterGridItem struct {
	Rank                      int     `json:"rank"`
	SelectionCount            int     `json:"selection_count"`
	RebalanceEvery            int     `json:"rebalance_every"`
	CompletedSeeds            int     `json:"completed_seeds"`
	PositiveTotalReturnSeeds  int     `json:"positive_total_return_seeds"`
	PositiveExcessReturnSeeds int     `json:"positive_excess_return_seeds"`
	PositiveRecentExcessSeeds int     `json:"positive_recent_excess_seeds"`
	AverageTotalReturn        float64 `json:"average_total_return"`
	AverageExcessReturn       float64 `json:"average_excess_return"`
	WorstTotalReturn          float64 `json:"worst_total_return"`
	WorstExcessReturn         float64 `json:"worst_excess_return"`
	WorstRecentExcessReturn   float64 `json:"worst_recent_excess_return"`
	WorstMaxDrawdown          float64 `json:"worst_max_drawdown"`
	AverageTopDecileHitRate   float64 `json:"average_future_top_decile_hit_rate"`
	WorstTopDecileHitRate     float64 `json:"worst_future_top_decile_hit_rate"`
	AverageTopDecileCapture   float64 `json:"average_future_top_decile_capture_rate"`
	WorstTopDecileCapture     float64 `json:"worst_future_top_decile_capture_rate"`
	AverageTopCaptureRate     float64 `json:"average_future_top_selection_capture_rate"`
	WorstTopCaptureRate       float64 `json:"worst_future_top_selection_capture_rate"`
	AverageFuturePercentile   float64 `json:"average_future_return_percentile"`
	WorstFuturePercentile     float64 `json:"worst_future_return_percentile"`
	AverageRecentTopDecileHit float64 `json:"average_recent_future_top_decile_hit_rate"`
	WorstRecentTopDecileHit   float64 `json:"worst_recent_future_top_decile_hit_rate"`
	Error                     string  `json:"error,omitempty"`
}

type MomentumBacktestParameterGridReport struct {
	Summary MomentumBacktestParameterGridSummary `json:"summary"`
	Items   []MomentumBacktestParameterGridItem  `json:"items,omitempty"`
}

type MomentumBacktestStageSummary struct {
	GeneratedAt                 time.Time `json:"generated_at"`
	Days                        int       `json:"days"`
	Seeds                       int       `json:"seeds"`
	SelectionCount              int       `json:"selection_count"`
	RebalanceEvery              int       `json:"rebalance_every"`
	StageCount                  int       `json:"stage_count"`
	CompleteStageCount          int       `json:"complete_stage_count"`
	PositiveAverageExcessStages int       `json:"positive_average_excess_stages"`
	PositiveAllSeedExcessStages int       `json:"positive_all_seed_excess_stages"`
	WorstCompleteStage          string    `json:"worst_complete_stage"`
	WorstCompleteAverageExcess  float64   `json:"worst_complete_average_excess"`
	WorstCompleteSeedExcess     float64   `json:"worst_complete_seed_excess"`
	Notes                       []string  `json:"notes,omitempty"`
}

type MomentumBacktestStageItem struct {
	StageLabel              string    `json:"stage_label"`
	StartDate               time.Time `json:"start_date"`
	EndDate                 time.Time `json:"end_date"`
	Complete                bool      `json:"complete"`
	SeedCount               int       `json:"seed_count"`
	PeriodCount             int       `json:"period_count"`
	PositiveReturnSeeds     int       `json:"positive_return_seeds"`
	PositiveExcessSeeds     int       `json:"positive_excess_seeds"`
	AverageReturn           float64   `json:"average_return"`
	AverageExcessReturn     float64   `json:"average_excess_return"`
	WorstReturn             float64   `json:"worst_return"`
	WorstExcessReturn       float64   `json:"worst_excess_return"`
	BestExcessReturn        float64   `json:"best_excess_return"`
	AveragePositiveBreadth  float64   `json:"average_positive_breadth"`
	AverageMedianReturn60D  float64   `json:"average_median_return_60d"`
	AverageMedianReturn120D float64   `json:"average_median_return_120d"`
}

type MomentumBacktestStageReport struct {
	Summary MomentumBacktestStageSummary `json:"summary"`
	Items   []MomentumBacktestStageItem  `json:"items,omitempty"`
}

type MomentumBacktestEntryGridSummary struct {
	GeneratedAt               time.Time `json:"generated_at"`
	Days                      int       `json:"days"`
	Seeds                     int       `json:"seeds"`
	CombinationCount          int       `json:"combination_count"`
	CurrentRank               int       `json:"current_rank"`
	CurrentMinPositiveBreadth float64   `json:"current_min_positive_breadth"`
	CurrentMinReturn20D       float64   `json:"current_min_return_20d"`
	CurrentMinReturn60D       float64   `json:"current_min_return_60d"`
	LeadingMinPositiveBreadth float64   `json:"leading_min_positive_breadth"`
	LeadingMinReturn20D       float64   `json:"leading_min_return_20d"`
	LeadingMinReturn60D       float64   `json:"leading_min_return_60d"`
	Notes                     []string  `json:"notes,omitempty"`
}

type MomentumBacktestEntryGridItem struct {
	Rank                      int     `json:"rank"`
	MinPositiveBreadth        float64 `json:"min_positive_breadth"`
	MinReturn20D              float64 `json:"min_return_20d"`
	MinReturn60D              float64 `json:"min_return_60d"`
	CompletedSeeds            int     `json:"completed_seeds"`
	PositiveTotalReturnSeeds  int     `json:"positive_total_return_seeds"`
	PositiveExcessReturnSeeds int     `json:"positive_excess_return_seeds"`
	PositiveRecentExcessSeeds int     `json:"positive_recent_excess_seeds"`
	AverageTotalReturn        float64 `json:"average_total_return"`
	AverageExcessReturn       float64 `json:"average_excess_return"`
	WorstTotalReturn          float64 `json:"worst_total_return"`
	WorstExcessReturn         float64 `json:"worst_excess_return"`
	WorstRecentExcessReturn   float64 `json:"worst_recent_excess_return"`
	WorstMaxDrawdown          float64 `json:"worst_max_drawdown"`
	Error                     string  `json:"error,omitempty"`
}

type MomentumBacktestEntryGridReport struct {
	Summary MomentumBacktestEntryGridSummary `json:"summary"`
	Items   []MomentumBacktestEntryGridItem  `json:"items,omitempty"`
}

type MomentumBacktestCorrelationGridSummary struct {
	GeneratedAt           time.Time `json:"generated_at"`
	Days                  int       `json:"days"`
	Seeds                 int       `json:"seeds"`
	CombinationCount      int       `json:"combination_count"`
	CurrentRank           int       `json:"current_rank"`
	LeadingMaxCorrelation float64   `json:"leading_max_correlation"`
	CorrelationWindowDays int       `json:"correlation_window_days"`
	Notes                 []string  `json:"notes,omitempty"`
}

type MomentumBacktestCorrelationGridItem struct {
	Rank                      int     `json:"rank"`
	MaxCorrelation            float64 `json:"max_correlation"`
	CompletedSeeds            int     `json:"completed_seeds"`
	PositiveTotalReturnSeeds  int     `json:"positive_total_return_seeds"`
	PositiveExcessReturnSeeds int     `json:"positive_excess_return_seeds"`
	PositiveRecentExcessSeeds int     `json:"positive_recent_excess_seeds"`
	AverageTotalReturn        float64 `json:"average_total_return"`
	AverageExcessReturn       float64 `json:"average_excess_return"`
	WorstTotalReturn          float64 `json:"worst_total_return"`
	WorstExcessReturn         float64 `json:"worst_excess_return"`
	WorstRecentExcessReturn   float64 `json:"worst_recent_excess_return"`
	WorstMaxDrawdown          float64 `json:"worst_max_drawdown"`
	Stage2024Seeds            int     `json:"stage_2024_seeds"`
	Positive2024ExcessSeeds   int     `json:"positive_2024_excess_seeds"`
	Average2024ExcessReturn   float64 `json:"average_2024_excess_return"`
	Worst2024ExcessReturn     float64 `json:"worst_2024_excess_return"`
	Error                     string  `json:"error,omitempty"`
}

type MomentumBacktestCorrelationGridReport struct {
	Summary MomentumBacktestCorrelationGridSummary `json:"summary"`
	Items   []MomentumBacktestCorrelationGridItem  `json:"items,omitempty"`
}

type MomentumBacktestWeightGridSummary struct {
	GeneratedAt  time.Time `json:"generated_at"`
	Days         int       `json:"days"`
	Seeds        int       `json:"seeds"`
	ProfileCount int       `json:"profile_count"`
	CurrentRank  int       `json:"current_rank"`
	LeadingLabel string    `json:"leading_label"`
	Notes        []string  `json:"notes,omitempty"`
}

type MomentumBacktestWeightGridItem struct {
	Rank                          int     `json:"rank"`
	Label                         string  `json:"label"`
	Weight20D                     float64 `json:"weight_20d"`
	Weight60D                     float64 `json:"weight_60d"`
	Weight120D                    float64 `json:"weight_120d"`
	Weight250D                    float64 `json:"weight_250d"`
	CompletedSeeds                int     `json:"completed_seeds"`
	PositiveRecentExcessSeeds     int     `json:"positive_recent_excess_seeds"`
	AverageTotalReturn            float64 `json:"average_total_return"`
	WorstTotalReturn              float64 `json:"worst_total_return"`
	AverageExcessReturn           float64 `json:"average_excess_return"`
	WorstExcessReturn             float64 `json:"worst_excess_return"`
	WorstRecentExcessReturn       float64 `json:"worst_recent_excess_return"`
	WorstMaxDrawdown              float64 `json:"worst_max_drawdown"`
	AverageTopDecileHitRate       float64 `json:"average_future_top_decile_hit_rate"`
	WorstTopDecileHitRate         float64 `json:"worst_future_top_decile_hit_rate"`
	AverageRecentTopDecileHitRate float64 `json:"average_recent_future_top_decile_hit_rate"`
	WorstRecentTopDecileHitRate   float64 `json:"worst_recent_future_top_decile_hit_rate"`
	AverageFuturePercentile       float64 `json:"average_future_return_percentile"`
	Error                         string  `json:"error,omitempty"`
}

type MomentumBacktestWeightGridReport struct {
	Summary MomentumBacktestWeightGridSummary `json:"summary"`
	Items   []MomentumBacktestWeightGridItem  `json:"items,omitempty"`
}
