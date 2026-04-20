package report

import (
	"strings"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

func TestRenderMarkdownIncludesRecommendations(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	rendered := RenderMarkdown(model.AnalysisReport{
		Summary: model.AnalysisSummary{
			PortfolioName:  "test",
			RunDate:        now,
			PortfolioValue: 100000,
			ActionCounts: map[model.Action]int{
				model.ActionHold:     1,
				model.ActionPauseBuy: 1,
				model.ActionReduce:   1,
			},
		},
		Signals: []model.FundSignal{
			{FundName: "Hold Fund", Action: model.ActionHold, CurrentWeight: 0.10, TargetWeight: 0.10, Return20D: 0.02, Return60D: 0.04, Reason: "继续持有"},
			{FundName: "Pause Fund", Action: model.ActionPauseBuy, CurrentWeight: 0.15, TargetWeight: 0.10, Return20D: -0.01, Return60D: -0.03, Reason: "先暂停加仓"},
		},
		Position: []model.PositionState{{
			Position:          model.Position{FundName: "Hold Fund", EstimatedUnits: 123.4567},
			CurrentValue:      15000,
			CurrentWeight:     0.15,
			HoldingCost:       12000,
			UnrealizedPnL:     3000,
			UnrealizedPnLPct:  0.25,
			LedgerTradeCount:  2,
			LastLedgerTradeAt: now,
			LedgerApplied:     true,
		}},
		DCAPlan: &model.DCAPlanReport{
			Summary: model.DCAPlanSummary{ReserveAmount: 1000},
			Items: []model.DCAPlanItem{{
				FundName:      "Monthly Core",
				PlannedAmount: 4000,
				Priority:      1,
				Reason:        "按月定投",
			}},
		},
		Recommendations: []model.TradeRecommendation{{
			Kind:            "SWAP",
			SourceFund:      "Old Fund",
			TargetFund:      "New Fund",
			SuggestedWeight: 0.05,
			SuggestedAmount: 5000,
			Reason:          "replacement",
			CreatedAt:       now,
		}},
	})
	if !strings.Contains(rendered, "# test 投资行动手册") {
		t.Fatalf("expected playbook heading in markdown output")
	}
	if !strings.Contains(rendered, "## 替换观察") {
		t.Fatalf("expected replacement section in markdown output")
	}
	if !strings.Contains(rendered, "| Old Fund | New Fund | 5000 | 5.00% | replacement |") {
		t.Fatalf("expected rendered recommendation row, got %s", rendered)
	}
	if !strings.Contains(rendered, "## 继续持有") {
		t.Fatalf("expected hold section, got %s", rendered)
	}
	if !strings.Contains(rendered, "## 暂停加仓") {
		t.Fatalf("expected pause section, got %s", rendered)
	}
	if !strings.Contains(rendered, "## 继续定投") {
		t.Fatalf("expected continue dca section, got %s", rendered)
	}
	if !strings.Contains(rendered, "| Monthly Core | 4000 | 4.00% | 按月定投 |") {
		t.Fatalf("expected dca row from monthly plan, got %s", rendered)
	}
	if strings.Contains(rendered, "## 月度定投快照") {
		t.Fatalf("did not expect separate monthly dca snapshot, got %s", rendered)
	}
	if !strings.Contains(rendered, "## 持仓快照") {
		t.Fatalf("expected holdings snapshot section, got %s", rendered)
	}
	if !strings.Contains(rendered, "| Hold Fund | 15000.00 | 15.00% | 123.4567 | 12000.00 | +3000.00 | +25.00% | 2026-04-15 / 2笔 |") {
		t.Fatalf("expected holdings snapshot row, got %s", rendered)
	}
	if !strings.Contains(rendered, "## 执行顺序") {
		t.Fatalf("expected execution order section, got %s", rendered)
	}
	if !strings.Contains(rendered, "| 2 | 买入 | New Fund | Old Fund | 5000 | 卖出 Old Fund | replacement |") {
		t.Fatalf("expected swap execution row, got %s", rendered)
	}
	if !strings.Contains(rendered, "| 3 | 买入 | Monthly Core | - | 4000 | 组合卖出回笼资金 | 按月定投 |") {
		t.Fatalf("expected dca execution row, got %s", rendered)
	}
	if !strings.Contains(rendered, "- 本月定投预留：`1000`") {
		t.Fatalf("expected monthly dca reserve note, got %s", rendered)
	}
}

func TestRenderBacktestMarkdown(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 15, 0, 0, 0, 0, time.UTC)
	rendered, err := RenderBacktest(model.BacktestReport{
		Summary: model.BacktestSummary{
			PortfolioName:             "test",
			StartDate:                 now.AddDate(0, 0, -30),
			EndDate:                   now,
			TradingDays:               30,
			RebalanceEvery:            10,
			RebalanceCount:            2,
			TradeCount:                3,
			InitialValue:              100000,
			FinalValue:                108000,
			BenchmarkFinalValue:       104000,
			TotalReturn:               0.08,
			BenchmarkReturn:           0.04,
			ExcessReturn:              0.04,
			AnnualizedReturn:          0.12,
			BenchmarkAnnualizedReturn: 0.06,
			MaxDrawdown:               0.05,
			BenchmarkMaxDrawdown:      0.03,
			CashFinal:                 2000,
		},
		Trades: []model.BacktestTrade{{
			Date:   now,
			Action: "BUY",
			Fund:   "Fund A",
			Amount: 5000,
			Price:  1.2345,
			Units:  4050.0,
		}},
	}, "markdown")
	if err != nil {
		t.Fatalf("RenderBacktest() error = %v", err)
	}
	if !strings.Contains(rendered, "# test 策略回测") {
		t.Fatalf("expected backtest heading, got %s", rendered)
	}
	if !strings.Contains(rendered, "| 总收益率 | 8.00% |") {
		t.Fatalf("expected total return row, got %s", rendered)
	}
}

func TestRenderDCAPlanMarkdown(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	rendered, err := RenderDCAPlan(model.DCAPlanReport{
		Summary: model.DCAPlanSummary{
			PortfolioName:      "test",
			PlanDate:           now,
			Frequency:          "monthly",
			Budget:             5000,
			PlannedAmount:      4000,
			ReserveAmount:      1000,
			EligibleFundCount:  3,
			SelectedFundCount:  1,
			PauseOnRiskEnabled: true,
			Notes:              []string{"保留部分预算作为机动资金"},
		},
		Items: []model.DCAPlanItem{{
			Priority:      1,
			FundName:      "Core Fund",
			Role:          "core",
			CurrentWeight: 0.10,
			TargetWeight:  0.20,
			GapWeight:     0.10,
			PlannedAmount: 4000,
			Reason:        "继续定投",
		}},
		Skipped: []model.DCASkippedFund{{
			FundName: "Paused Fund",
			Action:   model.ActionPauseBuy,
			Reason:   "短期风险偏高",
		}},
	}, "markdown")
	if err != nil {
		t.Fatalf("RenderDCAPlan() error = %v", err)
	}
	if !strings.Contains(rendered, "# test 定投计划") {
		t.Fatalf("expected dca plan heading, got %s", rendered)
	}
	if !strings.Contains(rendered, "## 本期执行") {
		t.Fatalf("expected invest section, got %s", rendered)
	}
	if !strings.Contains(rendered, "| 1 | Core Fund | 核心 | 10.00% | 20.00% | 10.00% | 4000 | 继续定投 |") {
		t.Fatalf("expected item row, got %s", rendered)
	}
	if !strings.Contains(rendered, "| Paused Fund | 暂停加仓 | 短期风险偏高 |") {
		t.Fatalf("expected skipped row, got %s", rendered)
	}
}

func TestRenderMarkdownPrefersEnhancedReasons(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	rendered := RenderMarkdown(model.AnalysisReport{
		Summary: model.AnalysisSummary{PortfolioName: "test", RunDate: now, PortfolioValue: 100000},
		Recommendations: []model.TradeRecommendation{{
			Kind:            "SWAP",
			SourceFund:      "Old Fund",
			TargetFund:      "New Fund",
			SuggestedWeight: 0.05,
			SuggestedAmount: 5000,
			Reason:          "rule replacement",
			EnhancedReason:  "LLM prefers New Fund for stronger persistence.",
			CreatedAt:       now,
		}},
		Candidates: []model.CandidateSuggestion{{
			FundName:       "New Fund",
			Score:          7,
			Return20D:      0.03,
			Return60D:      0.08,
			ReplaceFor:     []string{"Old Fund"},
			Reason:         "rule candidate",
			EnhancedReason: "LLM keeps New Fund first because its medium-term trend is cleaner.",
		}},
	})
	if !strings.Contains(rendered, "LLM prefers New Fund for stronger persistence. 规则依据：rule replacement") {
		t.Fatalf("expected enhanced recommendation reason, got %s", rendered)
	}
	if !strings.Contains(rendered, "LLM keeps New Fund first because its medium-term trend is cleaner. 规则依据：rule candidate") {
		t.Fatalf("expected enhanced candidate reason, got %s", rendered)
	}
}

func TestRenderMarketPoolMarkdown(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 4, 19, 10, 0, 0, 0, time.UTC)
	rendered, err := RenderMarketPool(model.MarketPoolReport{
		Summary: model.MarketPoolSummary{
			RunDate:       now,
			UniverseCount: 19000,
			MatchedCount:  120,
			EligibleCount: 8,
			SelectedCount: 2,
			RetainedCount: 1,
			Notes:         []string{"old candidate retained due to score gap"},
		},
		Items: []model.MarketPoolItem{{
			Rank:            1,
			ThemeLabel:      "A股宽基",
			FundName:        "易方达中证A500ETF联接A",
			Score:           8,
			Return120D:      0.12,
			Return250D:      0.21,
			MaxDrawdown120D: 0.09,
			FundSizeYi:      32,
			Retained:        true,
			Reason:          "250日收益 21.00%；指数工具更稳定",
		}},
	}, "markdown")
	if err != nil {
		t.Fatalf("RenderMarketPool() error = %v", err)
	}
	if !strings.Contains(rendered, "# 稳定候选池") {
		t.Fatalf("expected market pool heading, got %s", rendered)
	}
	if !strings.Contains(rendered, "| 1 | A股宽基 | 易方达中证A500ETF联接A | 8 | 12.00% | 21.00% | 9.00% | 32.0亿 | 是 | 250日收益 21.00%；指数工具更稳定 |") {
		t.Fatalf("expected market pool row, got %s", rendered)
	}
}
