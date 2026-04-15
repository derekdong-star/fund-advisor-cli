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
				model.ActionReduce: 1,
			},
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
		ExecutionPlan: &model.ExecutionPlan{
			GrossSellAmount: 5000,
			GrossBuyAmount:  5000,
			SwapAmount:      5000,
			Steps: []model.ExecutionStep{
				{Order: 1, Action: "SELL", Fund: "Old Fund", RelatedFund: "New Fund", Amount: 5000, Reason: "replacement"},
				{Order: 2, Action: "BUY", Fund: "New Fund", RelatedFund: "Old Fund", Amount: 5000, FundingSource: "卖出 Old Fund", Reason: "replacement"},
			},
		},
	})
	if !strings.Contains(rendered, "## Recommended Actions") {
		t.Fatalf("expected recommendation section in markdown output")
	}
	if !strings.Contains(rendered, "| SWAP | Old Fund | New Fund | 5.00% | 5000 | replacement |") {
		t.Fatalf("expected rendered recommendation row, got %s", rendered)
	}
	if !strings.Contains(rendered, "## Execution Plan") {
		t.Fatalf("expected execution plan section, got %s", rendered)
	}
	if !strings.Contains(rendered, "| 2 | BUY | New Fund | Old Fund | 5000 | 卖出 Old Fund | replacement |") {
		t.Fatalf("expected execution plan row, got %s", rendered)
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
	if !strings.Contains(rendered, "# test Backtest") {
		t.Fatalf("expected backtest heading, got %s", rendered)
	}
	if !strings.Contains(rendered, "| Total Return | 8.00% |") {
		t.Fatalf("expected total return row, got %s", rendered)
	}
}
