package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Portfolio  PortfolioConfig  `yaml:"portfolio"`
	Storage    StorageConfig    `yaml:"storage"`
	DataSource DataSourceConfig `yaml:"data_source"`
	Strategy   StrategyConfig   `yaml:"strategy"`
	Funds      []FundConfig     `yaml:"funds"`
	Candidates []FundConfig     `yaml:"candidates"`

	configPath string
}

type PortfolioConfig struct {
	Name      string `yaml:"name"`
	Currency  string `yaml:"currency"`
	Benchmark string `yaml:"benchmark"`
}

type StorageConfig struct {
	Driver string `yaml:"driver"`
	DSN    string `yaml:"dsn"`
}

type DataSourceConfig struct {
	Provider              string `yaml:"provider"`
	TushareTokenEnv       string `yaml:"tushare_token_env"`
	RequestTimeoutSeconds int    `yaml:"request_timeout_seconds"`
}

type StrategyConfig struct {
	Rebalance     RebalanceConfig     `yaml:"rebalance"`
	HoldingHealth HoldingHealthConfig `yaml:"holding_health"`
	BuySignal     BuySignalConfig     `yaml:"buy_signal"`
	SellSignal    SellSignalConfig    `yaml:"sell_signal"`
	CandidatePool CandidatePoolConfig `yaml:"candidate_pool"`
	Turnover      TurnoverConfig      `yaml:"turnover"`
}

type RebalanceConfig struct {
	RelativeDriftThreshold float64 `yaml:"relative_drift_threshold"`
	AbsoluteDriftThreshold float64 `yaml:"absolute_drift_threshold"`
}

type HoldingHealthConfig struct {
	Underperform60DThreshold float64 `yaml:"underperform_60d_threshold"`
	ReviewScoreThreshold     int     `yaml:"review_score_threshold"`
	ReplaceScoreThreshold    int     `yaml:"replace_score_threshold"`
}

type BuySignalConfig struct {
	MaxSinglePositionWeight float64 `yaml:"max_single_position_weight"`
	MinGapToTarget          float64 `yaml:"min_gap_to_target"`
}

type SellSignalConfig struct {
	OverweightRelativeThreshold float64 `yaml:"overweight_relative_threshold"`
	OverweightAbsoluteThreshold float64 `yaml:"overweight_absolute_threshold"`
}

type CandidatePoolConfig struct {
	MinFundSizeYi        float64 `yaml:"min_fund_size_yi"`
	MinEstablishedYears  float64 `yaml:"min_established_years"`
	MaxExpenseRatio      float64 `yaml:"max_expense_ratio"`
	CoreRequireIndex     bool    `yaml:"core_require_index"`
	PreferBenchmarkMatch bool    `yaml:"prefer_benchmark_match"`
}

type TurnoverConfig struct {
	Mode                     string  `yaml:"mode"`
	MinSwapScore             int     `yaml:"min_swap_score"`
	MaxProtectedReduceWeight float64 `yaml:"max_protected_reduce_weight"`
	MonthlyDCAAmount         float64 `yaml:"monthly_dca_amount"`
	PreferDCA                bool    `yaml:"prefer_dca"`
}

type FundConfig struct {
	Code             string   `yaml:"code"`
	Name             string   `yaml:"name"`
	Category         string   `yaml:"category"`
	AccountValue     float64  `yaml:"account_value,omitempty"`
	TargetWeight     float64  `yaml:"target_weight,omitempty"`
	Benchmark        string   `yaml:"benchmark"`
	Role             string   `yaml:"role"`
	Status           string   `yaml:"status,omitempty"`
	Protected        bool     `yaml:"protected,omitempty"`
	DCAEnabled       bool     `yaml:"dca_enabled,omitempty"`
	ExpenseRatio     float64  `yaml:"expense_ratio,omitempty"`
	FundSizeYi       float64  `yaml:"fund_size_yi,omitempty"`
	EstablishedYears float64  `yaml:"established_years,omitempty"`
	IsIndex          bool     `yaml:"is_index,omitempty"`
	Tags             []string `yaml:"tags,omitempty"`
}

func Load(path string) (*Config, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(buf, &cfg); err != nil {
		return nil, err
	}
	cfg.configPath = path
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) Validate() error {
	if c.Portfolio.Name == "" {
		return errors.New("portfolio.name is required")
	}
	if c.Storage.Driver == "" {
		return errors.New("storage.driver is required")
	}
	if c.Storage.DSN == "" {
		return errors.New("storage.dsn is required")
	}
	if len(c.Funds) == 0 {
		return errors.New("at least one fund is required")
	}
	seen := make(map[string]struct{}, len(c.Funds)+len(c.Candidates))
	var totalTarget float64
	for _, fund := range c.Funds {
		if err := validateConfiguredFund(fund, true); err != nil {
			return err
		}
		if _, ok := seen[fund.Code]; ok {
			return fmt.Errorf("duplicate fund code: %s", fund.Code)
		}
		seen[fund.Code] = struct{}{}
		totalTarget += fund.TargetWeight
	}
	for _, candidate := range c.Candidates {
		if err := validateConfiguredFund(candidate, false); err != nil {
			return err
		}
		if _, ok := seen[candidate.Code]; ok {
			return fmt.Errorf("duplicate fund code: %s", candidate.Code)
		}
		seen[candidate.Code] = struct{}{}
	}
	if totalTarget < 0.99 || totalTarget > 1.01 {
		return fmt.Errorf("target weights must sum to 1.0, got %.4f", totalTarget)
	}
	if c.Strategy.CandidatePool.MinFundSizeYi < 0 || c.Strategy.CandidatePool.MinEstablishedYears < 0 || c.Strategy.CandidatePool.MaxExpenseRatio < 0 {
		return errors.New("candidate_pool thresholds must be non-negative")
	}
	if c.Strategy.Turnover.MinSwapScore < 0 || c.Strategy.Turnover.MaxProtectedReduceWeight < 0 || c.Strategy.Turnover.MonthlyDCAAmount < 0 {
		return errors.New("turnover thresholds must be non-negative")
	}
	return nil
}

func validateConfiguredFund(fund FundConfig, requireWeights bool) error {
	if fund.Code == "" {
		return errors.New("fund code is required")
	}
	if fund.Name == "" {
		return fmt.Errorf("fund %s is missing name", fund.Code)
	}
	if requireWeights && fund.TargetWeight < 0 {
		return fmt.Errorf("fund %s has negative target weight", fund.Code)
	}
	if requireWeights && fund.AccountValue < 0 {
		return fmt.Errorf("fund %s has negative account value", fund.Code)
	}
	if fund.ExpenseRatio < 0 {
		return fmt.Errorf("fund %s has negative expense ratio", fund.Code)
	}
	if fund.FundSizeYi < 0 {
		return fmt.Errorf("fund %s has negative fund size", fund.Code)
	}
	if fund.EstablishedYears < 0 {
		return fmt.Errorf("fund %s has negative established years", fund.Code)
	}
	return nil
}

func (c *Config) ConfigDir() string {
	if c.configPath == "" {
		return "."
	}
	return filepath.Dir(c.configPath)
}

func (c *Config) ResolveStorageDSN() string {
	if filepath.IsAbs(c.Storage.DSN) {
		return c.Storage.DSN
	}
	return filepath.Join(c.ConfigDir(), c.Storage.DSN)
}

func Default() *Config {
	return &Config{
		Portfolio:  PortfolioConfig{Name: "dhw-fund-portfolio", Currency: "CNY", Benchmark: "custom-mix"},
		Storage:    StorageConfig{Driver: "sqlite", DSN: "../data/fundcli.db"},
		DataSource: DataSourceConfig{Provider: "eastmoney", TushareTokenEnv: "TUSHARE_TOKEN", RequestTimeoutSeconds: 15},
		Strategy: StrategyConfig{
			Rebalance:     RebalanceConfig{RelativeDriftThreshold: 0.25, AbsoluteDriftThreshold: 0.05},
			HoldingHealth: HoldingHealthConfig{Underperform60DThreshold: -0.08, ReviewScoreThreshold: 2, ReplaceScoreThreshold: 3},
			BuySignal:     BuySignalConfig{MaxSinglePositionWeight: 0.18, MinGapToTarget: 0.20},
			SellSignal:    SellSignalConfig{OverweightRelativeThreshold: 0.35, OverweightAbsoluteThreshold: 0.08},
			CandidatePool: CandidatePoolConfig{MinFundSizeYi: 8, MinEstablishedYears: 1, MaxExpenseRatio: 0.008, CoreRequireIndex: true, PreferBenchmarkMatch: true},
			Turnover:      TurnoverConfig{Mode: "low_turnover", MinSwapScore: 7, MaxProtectedReduceWeight: 0.22, MonthlyDCAAmount: 5000, PreferDCA: true},
		},
		Funds: []FundConfig{
			{Code: "000979", Name: "景顺长城沪港深精选股票A", Category: "active_cn_equity", AccountValue: 68000, TargetWeight: 0.13, Benchmark: "hs300_hk_mix", Role: "satellite", Status: "active", Protected: true, DCAEnabled: true},
			{Code: "012060", Name: "富国全球消费精选混合(QDII)人民币A", Category: "active_qdii", AccountValue: 43000, TargetWeight: 0.07, Benchmark: "global_consumer", Role: "satellite", Status: "active"},
			{Code: "000628", Name: "大成高鑫股票A", Category: "active_cn_equity", AccountValue: 42000, TargetWeight: 0.10, Benchmark: "hs300", Role: "satellite", Status: "active"},
			{Code: "021457", Name: "易方达恒生红利低波ETF联接A", Category: "hk_dividend", AccountValue: 26000, TargetWeight: 0.10, Benchmark: "hsi_div_lowvol", Role: "core", Status: "active", DCAEnabled: true},
			{Code: "090013", Name: "大成竞争优势混合A", Category: "active_cn_equity", AccountValue: 23000, TargetWeight: 0.08, Benchmark: "hs300", Role: "satellite", Status: "active"},
			{Code: "000218", Name: "国泰黄金ETF联接A", Category: "gold", AccountValue: 20000, TargetWeight: 0.10, Benchmark: "au9999", Role: "hedge", Status: "active"},
			{Code: "050025", Name: "博时标普500ETF联接A", Category: "sp500", AccountValue: 14000, TargetWeight: 0.15, Benchmark: "sp500", Role: "core", Status: "active", DCAEnabled: true},
			{Code: "009052", Name: "易方达中证红利ETF联接发起式C", Category: "cn_dividend", AccountValue: 14000, TargetWeight: 0.15, Benchmark: "csi_dividend", Role: "core", Status: "active", DCAEnabled: true},
			{Code: "012920", Name: "易方达全球成长精选混合(QDII)人民币A", Category: "active_qdii", AccountValue: 12000, TargetWeight: 0.07, Benchmark: "global_growth", Role: "satellite", Status: "active"},
			{Code: "001194", Name: "景顺长城稳健回报混合A", Category: "balanced", AccountValue: 100, TargetWeight: 0.05, Benchmark: "balanced_mix", Role: "stabilizer", Status: "active"},
		},
		Candidates: []FundConfig{
			{Code: "022459", Name: "易方达中证A500ETF联接A", Category: "active_cn_equity", Benchmark: "a500", Role: "satellite", ExpenseRatio: 0.005, FundSizeYi: 32, EstablishedYears: 2.1, IsIndex: true, Tags: []string{"a500", "broad_market"}},
			{Code: "008969", Name: "易方达中证海外中国互联网50ETF联接(QDII)A", Category: "active_qdii", Benchmark: "china_internet", Role: "satellite", ExpenseRatio: 0.006, FundSizeYi: 45, EstablishedYears: 4.2, IsIndex: true, Tags: []string{"internet", "china"}},
			{Code: "019524", Name: "华泰柏瑞纳斯达克100ETF发起式联接(QDII)A", Category: "active_qdii", Benchmark: "nasdaq100", Role: "satellite", ExpenseRatio: 0.008, FundSizeYi: 28, EstablishedYears: 2.0, IsIndex: true, Tags: []string{"us_tech", "nasdaq100"}},
			{Code: "014519", Name: "博时恒生高股息ETF发起式联接A", Category: "hk_dividend", Benchmark: "hsi_high_dividend", Role: "core", ExpenseRatio: 0.005, FundSizeYi: 12, EstablishedYears: 2.3, IsIndex: true, Tags: []string{"hk_dividend", "high_yield"}},
			{Code: "024029", Name: "招商恒生港股通高股息低波动ETF发起式联接A", Category: "hk_dividend", Benchmark: "hsi_div_lowvol", Role: "core", ExpenseRatio: 0.006, FundSizeYi: 9, EstablishedYears: 1.1, IsIndex: true, Tags: []string{"hk_dividend", "low_vol"}},
			{Code: "007466", Name: "华泰柏瑞中证红利低波动ETF联接A", Category: "cn_dividend", Benchmark: "csi_dividend_lowvol", Role: "core", ExpenseRatio: 0.005, FundSizeYi: 18, EstablishedYears: 5.0, IsIndex: true, Tags: []string{"dividend", "low_vol"}},
			{Code: "018064", Name: "华夏标普500ETF发起式联接(QDII)A(人民币)", Category: "sp500", Benchmark: "sp500", Role: "core", ExpenseRatio: 0.006, FundSizeYi: 16, EstablishedYears: 2.2, IsIndex: true, Tags: []string{"sp500", "us_large_cap"}},
			{Code: "000307", Name: "易方达黄金ETF联接A", Category: "gold", Benchmark: "au9999", Role: "hedge", ExpenseRatio: 0.006, FundSizeYi: 25, EstablishedYears: 8.0, IsIndex: true, Tags: []string{"gold", "commodity"}},
			{Code: "006319", Name: "易方达安瑞短债债券A", Category: "balanced", Benchmark: "short_bond", Role: "stabilizer", ExpenseRatio: 0.003, FundSizeYi: 40, EstablishedYears: 6.3, Tags: []string{"short_bond", "low_vol"}},
			{Code: "006662", Name: "易方达安悦超短债A", Category: "balanced", Benchmark: "ultra_short_bond", Role: "stabilizer", ExpenseRatio: 0.003, FundSizeYi: 55, EstablishedYears: 5.1, Tags: []string{"ultra_short_bond", "cash_plus"}},
		},
	}
}

func WriteExample(path string, force bool) error {
	if !force {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("config already exists: %s", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	buf, err := yaml.Marshal(Default())
	if err != nil {
		return err
	}
	content := "# Generated by fundcli init\n" + strings.TrimSpace(string(buf)) + "\n"
	return os.WriteFile(path, []byte(content), 0o644)
}
