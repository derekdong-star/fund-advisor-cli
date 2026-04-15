package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

func Render(report model.AnalysisReport, format string) (string, error) {
	switch format {
	case "table":
		return RenderTable(report), nil
	case "markdown", "md":
		return RenderMarkdown(report), nil
	case "json":
		buf, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported report format: %s", format)
	}
}

func RenderTable(report model.AnalysisReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Portfolio: %s\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "Run Date: %s\n", report.Summary.RunDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "Portfolio Value: %.2f\n", report.Summary.PortfolioValue)
	fmt.Fprintf(&buf, "Weighted Day Change: %.2f%%\n\n", report.Summary.WeightedDayChangePct*100)

	buf.WriteString("Action Counts:\n")
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ACTION\tCOUNT")
	for action, count := range report.Summary.ActionCounts {
		fmt.Fprintf(tw, "%s\t%d\n", action, count)
	}
	_ = tw.Flush()

	if len(report.Recommendations) > 0 {
		buf.WriteString("\nRecommended Actions:\n")
		tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "ACTION\tFROM\tTO\tWEIGHT\tAMOUNT\tREASON")
		for _, recommendation := range report.Recommendations {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%.2f%%\t%.0f\t%s\n",
				recommendation.Kind,
				displayOrDash(recommendation.SourceFund),
				displayOrDash(recommendation.TargetFund),
				recommendation.SuggestedWeight*100,
				recommendation.SuggestedAmount,
				recommendation.Reason,
			)
		}
		_ = tw.Flush()
	}

	if report.ExecutionPlan != nil && len(report.ExecutionPlan.Steps) > 0 {
		buf.WriteString("\nExecution Plan:\n")
		fmt.Fprintf(&buf, "Gross Sell: %.0f\n", report.ExecutionPlan.GrossSellAmount)
		fmt.Fprintf(&buf, "Gross Buy: %.0f\n", report.ExecutionPlan.GrossBuyAmount)
		fmt.Fprintf(&buf, "Swap Amount: %.0f\n", report.ExecutionPlan.SwapAmount)
		fmt.Fprintf(&buf, "Reduce Amount: %.0f\n", report.ExecutionPlan.ReduceAmount)
		fmt.Fprintf(&buf, "Buy Amount: %.0f\n", report.ExecutionPlan.BuyAmount)
		fmt.Fprintf(&buf, "Net Cash Change: %.0f\n\n", report.ExecutionPlan.NetCashChange)
		tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "STEP\tACTION\tFUND\tRELATED\tAMOUNT\tFUNDING\tREASON")
		for _, step := range report.ExecutionPlan.Steps {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%.0f\t%s\t%s\n",
				step.Order,
				step.Action,
				step.Fund,
				displayOrDash(step.RelatedFund),
				step.Amount,
				displayOrDash(step.FundingSource),
				step.Reason,
			)
		}
		_ = tw.Flush()
	}

	buf.WriteString("\nSignals:\n")
	tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ACTION\tFUND\tCURRENT\tTARGET\t20D\t60D\tREASON")
	for _, signal := range report.Signals {
		fmt.Fprintf(tw, "%s\t%s\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%s\n",
			signal.Action,
			signal.FundName,
			signal.CurrentWeight*100,
			signal.TargetWeight*100,
			signal.Return20D*100,
			signal.Return60D*100,
			signal.Reason,
		)
	}
	_ = tw.Flush()

	if len(report.Candidates) > 0 {
		buf.WriteString("\nCandidates:\n")
		tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "FUND\tSCORE\t20D\t60D\tREPLACE_FOR\tREASON")
		for _, candidate := range report.Candidates {
			fmt.Fprintf(tw, "%s\t%d\t%.2f%%\t%.2f%%\t%s\t%s\n",
				candidate.FundName,
				candidate.Score,
				candidate.Return20D*100,
				candidate.Return60D*100,
				stringsJoin(candidate.ReplaceFor),
				candidate.Reason,
			)
		}
		_ = tw.Flush()
	}
	return buf.String()
}

func RenderMarkdown(report model.AnalysisReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s Daily Fund Report\n\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "- Run Date: `%s`\n", report.Summary.RunDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "- Portfolio Value: `%.2f`\n", report.Summary.PortfolioValue)
	fmt.Fprintf(&buf, "- Weighted Day Change: `%.2f%%`\n", report.Summary.WeightedDayChangePct*100)
	if len(report.Summary.Notes) > 0 {
		for _, note := range report.Summary.Notes {
			fmt.Fprintf(&buf, "- Note: %s\n", note)
		}
	}
	buf.WriteString("\n## Action Counts\n\n")
	for action, count := range report.Summary.ActionCounts {
		fmt.Fprintf(&buf, "- `%s`: %d\n", action, count)
	}
	if len(report.Recommendations) > 0 {
		buf.WriteString("\n## Recommended Actions\n\n")
		buf.WriteString("| Action | From | To | Weight | Amount | Reason |\n")
		buf.WriteString("| --- | --- | --- | ---: | ---: | --- |\n")
		for _, recommendation := range report.Recommendations {
			fmt.Fprintf(&buf, "| %s | %s | %s | %.2f%% | %.0f | %s |\n",
				recommendation.Kind,
				displayOrDash(recommendation.SourceFund),
				displayOrDash(recommendation.TargetFund),
				recommendation.SuggestedWeight*100,
				recommendation.SuggestedAmount,
				recommendation.Reason,
			)
		}
	}
	if report.ExecutionPlan != nil && len(report.ExecutionPlan.Steps) > 0 {
		buf.WriteString("\n## Execution Plan\n\n")
		fmt.Fprintf(&buf, "- Gross Sell: `%.0f`\n", report.ExecutionPlan.GrossSellAmount)
		fmt.Fprintf(&buf, "- Gross Buy: `%.0f`\n", report.ExecutionPlan.GrossBuyAmount)
		fmt.Fprintf(&buf, "- Swap Amount: `%.0f`\n", report.ExecutionPlan.SwapAmount)
		fmt.Fprintf(&buf, "- Reduce Amount: `%.0f`\n", report.ExecutionPlan.ReduceAmount)
		fmt.Fprintf(&buf, "- Buy Amount: `%.0f`\n", report.ExecutionPlan.BuyAmount)
		fmt.Fprintf(&buf, "- Net Cash Change: `%.0f`\n\n", report.ExecutionPlan.NetCashChange)
		buf.WriteString("| Step | Action | Fund | Related | Amount | Funding | Reason |\n")
		buf.WriteString("| ---: | --- | --- | --- | ---: | --- | --- |\n")
		for _, step := range report.ExecutionPlan.Steps {
			fmt.Fprintf(&buf, "| %d | %s | %s | %s | %.0f | %s | %s |\n",
				step.Order,
				step.Action,
				step.Fund,
				displayOrDash(step.RelatedFund),
				step.Amount,
				displayOrDash(step.FundingSource),
				step.Reason,
			)
		}
	}
	buf.WriteString("\n## Signals\n\n")
	buf.WriteString("| Action | Fund | Current | Target | 20D | 60D | Reason |\n")
	buf.WriteString("| --- | --- | ---: | ---: | ---: | ---: | --- |\n")
	for _, signal := range report.Signals {
		fmt.Fprintf(&buf, "| %s | %s | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %s |\n",
			signal.Action,
			signal.FundName,
			signal.CurrentWeight*100,
			signal.TargetWeight*100,
			signal.Return20D*100,
			signal.Return60D*100,
			signal.Reason,
		)
	}
	if len(report.Candidates) > 0 {
		buf.WriteString("\n## Candidate Replacements\n\n")
		buf.WriteString("| Fund | Score | 20D | 60D | Replace For | Reason |\n")
		buf.WriteString("| --- | ---: | ---: | ---: | --- | --- |\n")
		for _, candidate := range report.Candidates {
			fmt.Fprintf(&buf, "| %s | %d | %.2f%% | %.2f%% | %s | %s |\n",
				candidate.FundName,
				candidate.Score,
				candidate.Return20D*100,
				candidate.Return60D*100,
				stringsJoin(candidate.ReplaceFor),
				candidate.Reason,
			)
		}
	}
	return buf.String()
}

func stringsJoin(items []string) string {
	if len(items) == 0 {
		return "-"
	}
	return strings.Join(items, " / ")
}

func displayOrDash(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func RenderBacktest(report model.BacktestReport, format string) (string, error) {
	switch format {
	case "table":
		return renderBacktestTable(report), nil
	case "markdown", "md":
		return renderBacktestMarkdown(report), nil
	case "json":
		buf, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported backtest format: %s", format)
	}
}

func renderBacktestTable(report model.BacktestReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Backtest: %s\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "Range: %s -> %s\n", report.Summary.StartDate.Format("2006-01-02"), report.Summary.EndDate.Format("2006-01-02"))
	fmt.Fprintf(&buf, "Trading Days: %d\n", report.Summary.TradingDays)
	fmt.Fprintf(&buf, "Rebalance Every: %d\n\n", report.Summary.RebalanceEvery)

	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "METRIC\tVALUE")
	fmt.Fprintf(tw, "Initial Value\t%.2f\n", report.Summary.InitialValue)
	fmt.Fprintf(tw, "Final Value\t%.2f\n", report.Summary.FinalValue)
	fmt.Fprintf(tw, "Benchmark Final\t%.2f\n", report.Summary.BenchmarkFinalValue)
	fmt.Fprintf(tw, "Total Return\t%.2f%%\n", report.Summary.TotalReturn*100)
	fmt.Fprintf(tw, "Benchmark Return\t%.2f%%\n", report.Summary.BenchmarkReturn*100)
	fmt.Fprintf(tw, "Excess Return\t%.2f%%\n", report.Summary.ExcessReturn*100)
	fmt.Fprintf(tw, "Annualized Return\t%.2f%%\n", report.Summary.AnnualizedReturn*100)
	fmt.Fprintf(tw, "Benchmark Annualized\t%.2f%%\n", report.Summary.BenchmarkAnnualizedReturn*100)
	fmt.Fprintf(tw, "Max Drawdown\t%.2f%%\n", report.Summary.MaxDrawdown*100)
	fmt.Fprintf(tw, "Benchmark Max Drawdown\t%.2f%%\n", report.Summary.BenchmarkMaxDrawdown*100)
	fmt.Fprintf(tw, "Rebalance Count\t%d\n", report.Summary.RebalanceCount)
	fmt.Fprintf(tw, "Trade Count\t%d\n", report.Summary.TradeCount)
	fmt.Fprintf(tw, "Final Cash\t%.2f\n", report.Summary.CashFinal)
	_ = tw.Flush()

	if len(report.Summary.Notes) > 0 {
		buf.WriteString("\nNotes:\n")
		for _, note := range report.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}

	if len(report.Trades) > 0 {
		buf.WriteString("\nRecent Trades:\n")
		tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "DATE\tACTION\tFUND\tRELATED\tAMOUNT\tPRICE\tUNITS")
		start := 0
		if len(report.Trades) > 10 {
			start = len(report.Trades) - 10
		}
		for _, trade := range report.Trades[start:] {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%.0f\t%.4f\t%.4f\n",
				trade.Date.Format("2006-01-02"),
				trade.Action,
				trade.Fund,
				displayOrDash(trade.RelatedFund),
				trade.Amount,
				trade.Price,
				trade.Units,
			)
		}
		_ = tw.Flush()
	}

	return buf.String()
}

func renderBacktestMarkdown(report model.BacktestReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s Backtest\n\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "- Range: `%s` -> `%s`\n", report.Summary.StartDate.Format("2006-01-02"), report.Summary.EndDate.Format("2006-01-02"))
	fmt.Fprintf(&buf, "- Trading Days: `%d`\n", report.Summary.TradingDays)
	fmt.Fprintf(&buf, "- Rebalance Every: `%d`\n", report.Summary.RebalanceEvery)
	fmt.Fprintf(&buf, "- Rebalance Count: `%d`\n", report.Summary.RebalanceCount)
	fmt.Fprintf(&buf, "- Trade Count: `%d`\n\n", report.Summary.TradeCount)
	buf.WriteString("| Metric | Value |\n")
	buf.WriteString("| --- | ---: |\n")
	fmt.Fprintf(&buf, "| Initial Value | %.2f |\n", report.Summary.InitialValue)
	fmt.Fprintf(&buf, "| Final Value | %.2f |\n", report.Summary.FinalValue)
	fmt.Fprintf(&buf, "| Benchmark Final | %.2f |\n", report.Summary.BenchmarkFinalValue)
	fmt.Fprintf(&buf, "| Total Return | %.2f%% |\n", report.Summary.TotalReturn*100)
	fmt.Fprintf(&buf, "| Benchmark Return | %.2f%% |\n", report.Summary.BenchmarkReturn*100)
	fmt.Fprintf(&buf, "| Excess Return | %.2f%% |\n", report.Summary.ExcessReturn*100)
	fmt.Fprintf(&buf, "| Annualized Return | %.2f%% |\n", report.Summary.AnnualizedReturn*100)
	fmt.Fprintf(&buf, "| Benchmark Annualized | %.2f%% |\n", report.Summary.BenchmarkAnnualizedReturn*100)
	fmt.Fprintf(&buf, "| Max Drawdown | %.2f%% |\n", report.Summary.MaxDrawdown*100)
	fmt.Fprintf(&buf, "| Benchmark Max Drawdown | %.2f%% |\n", report.Summary.BenchmarkMaxDrawdown*100)
	fmt.Fprintf(&buf, "| Final Cash | %.2f |\n", report.Summary.CashFinal)

	if len(report.Summary.Notes) > 0 {
		buf.WriteString("\n## Notes\n\n")
		for _, note := range report.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}

	if len(report.Trades) > 0 {
		buf.WriteString("\n## Recent Trades\n\n")
		buf.WriteString("| Date | Action | Fund | Related | Amount | Price | Units |\n")
		buf.WriteString("| --- | --- | --- | --- | ---: | ---: | ---: |\n")
		start := 0
		if len(report.Trades) > 10 {
			start = len(report.Trades) - 10
		}
		for _, trade := range report.Trades[start:] {
			fmt.Fprintf(&buf, "| %s | %s | %s | %s | %.0f | %.4f | %.4f |\n",
				trade.Date.Format("2006-01-02"),
				trade.Action,
				trade.Fund,
				displayOrDash(trade.RelatedFund),
				trade.Amount,
				trade.Price,
				trade.Units,
			)
		}
	}

	return buf.String()
}
