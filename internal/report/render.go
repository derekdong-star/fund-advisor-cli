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

func RenderDCAPlan(plan model.DCAPlanReport, format string) (string, error) {
	switch format {
	case "table":
		return renderDCAPlanTable(plan), nil
	case "markdown", "md":
		return renderDCAPlanMarkdown(plan), nil
	case "json":
		buf, err := json.MarshalIndent(plan, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported dca plan format: %s", format)
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
				recommendationDisplayReason(recommendation),
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
				stepDisplayReason(step),
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
				candidateDisplayReason(candidate),
			)
		}
		_ = tw.Flush()
	}
	return buf.String()
}

func RenderMarkdown(report model.AnalysisReport) string {
	var buf bytes.Buffer
	displayRecommendations := displayRecommendations(report)
	buyRecommendations := filterRecommendations(displayRecommendations, "BUY")
	executionPlan := buildDisplayExecutionPlan(displayRecommendations)

	fmt.Fprintf(&buf, "# %s Investor Playbook\n\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "- Run Date: `%s`\n", report.Summary.RunDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "- Portfolio Value: `%.2f`\n", report.Summary.PortfolioValue)
	fmt.Fprintf(&buf, "- Weighted Day Change: `%.2f%%`\n", report.Summary.WeightedDayChangePct*100)
	if len(report.Summary.Notes) > 0 {
		for _, note := range report.Summary.Notes {
			fmt.Fprintf(&buf, "- Note: %s\n", note)
		}
	}
	buf.WriteString("\n## This Period\n\n")
	for _, line := range renderPlaybookSummary(report, displayRecommendations) {
		fmt.Fprintf(&buf, "- %s\n", line)
	}

	if len(buyRecommendations) > 0 {
		buf.WriteString("\n## Continue DCA\n\n")
		buf.WriteString("| Fund | Amount | Weight | Reason |\n")
		buf.WriteString("| --- | ---: | ---: | --- |\n")
		for _, recommendation := range buyRecommendations {
			fmt.Fprintf(&buf, "| %s | %.0f | %.2f%% | %s |\n",
				recommendation.TargetFund,
				recommendation.SuggestedAmount,
				recommendation.SuggestedWeight*100,
				recommendationDisplayReason(recommendation),
			)
		}
		if report.DCAPlan != nil && report.DCAPlan.Summary.ReserveAmount > 0 {
			fmt.Fprintf(&buf, "\n- Monthly DCA Reserve: `%.0f`\n", report.DCAPlan.Summary.ReserveAmount)
		}
	}

	reduceRecommendations := filterRecommendations(displayRecommendations, "REDUCE")
	if len(reduceRecommendations) > 0 {
		buf.WriteString("\n## Trim Exposure\n\n")
		buf.WriteString("| Fund | Amount | Weight | Reason |\n")
		buf.WriteString("| --- | ---: | ---: | --- |\n")
		for _, recommendation := range reduceRecommendations {
			fmt.Fprintf(&buf, "| %s | %.0f | %.2f%% | %s |\n",
				recommendation.SourceFund,
				recommendation.SuggestedAmount,
				recommendation.SuggestedWeight*100,
				recommendationDisplayReason(recommendation),
			)
		}
	}

	swapRecommendations := filterRecommendations(displayRecommendations, "SWAP")
	if len(swapRecommendations) > 0 {
		buf.WriteString("\n## Replacement Watch\n\n")
		buf.WriteString("| From | To | Amount | Weight | Reason |\n")
		buf.WriteString("| --- | --- | ---: | ---: | --- |\n")
		for _, recommendation := range swapRecommendations {
			fmt.Fprintf(&buf, "| %s | %s | %.0f | %.2f%% | %s |\n",
				recommendation.SourceFund,
				recommendation.TargetFund,
				recommendation.SuggestedAmount,
				recommendation.SuggestedWeight*100,
				recommendationDisplayReason(recommendation),
			)
		}
	}

	holdSignals := filterSignals(report.Signals, model.ActionHold)
	if len(holdSignals) > 0 {
		buf.WriteString("\n## Continue Holding\n\n")
		buf.WriteString("| Fund | Current | Target | 20D | 60D | Reason |\n")
		buf.WriteString("| --- | ---: | ---: | ---: | ---: | --- |\n")
		for _, signal := range holdSignals {
			fmt.Fprintf(&buf, "| %s | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %s |\n",
				signal.FundName,
				signal.CurrentWeight*100,
				signal.TargetWeight*100,
				signal.Return20D*100,
				signal.Return60D*100,
				signal.Reason,
			)
		}
	}

	pauseSignals := filterSignals(report.Signals, model.ActionPauseBuy)
	if len(pauseSignals) > 0 {
		buf.WriteString("\n## Pause Adding\n\n")
		buf.WriteString("| Fund | Current | Target | 20D | 60D | Reason |\n")
		buf.WriteString("| --- | ---: | ---: | ---: | ---: | --- |\n")
		for _, signal := range pauseSignals {
			fmt.Fprintf(&buf, "| %s | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %s |\n",
				signal.FundName,
				signal.CurrentWeight*100,
				signal.TargetWeight*100,
				signal.Return20D*100,
				signal.Return60D*100,
				signal.Reason,
			)
		}
	}

	if executionPlan != nil && len(executionPlan.Steps) > 0 {
		buf.WriteString("\n## Execution Order\n\n")
		fmt.Fprintf(&buf, "- Gross Sell: `%.0f`\n", executionPlan.GrossSellAmount)
		fmt.Fprintf(&buf, "- Gross Buy: `%.0f`\n", executionPlan.GrossBuyAmount)
		fmt.Fprintf(&buf, "- Swap Amount: `%.0f`\n", executionPlan.SwapAmount)
		fmt.Fprintf(&buf, "- Reduce Amount: `%.0f`\n", executionPlan.ReduceAmount)
		fmt.Fprintf(&buf, "- Buy Amount: `%.0f`\n", executionPlan.BuyAmount)
		fmt.Fprintf(&buf, "- Net Cash Change: `%.0f`\n\n", executionPlan.NetCashChange)
		buf.WriteString("| Step | Action | Fund | Related | Amount | Funding | Reason |\n")
		buf.WriteString("| ---: | --- | --- | --- | ---: | --- | --- |\n")
		for _, step := range executionPlan.Steps {
			fmt.Fprintf(&buf, "| %d | %s | %s | %s | %.0f | %s | %s |\n",
				step.Order,
				step.Action,
				step.Fund,
				displayOrDash(step.RelatedFund),
				step.Amount,
				displayOrDash(step.FundingSource),
				stepDisplayReason(step),
			)
		}
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
				candidateDisplayReason(candidate),
			)
		}
	}
	return buf.String()
}

func renderPlaybookSummary(report model.AnalysisReport, recommendations []model.TradeRecommendation) []string {
	lines := make([]string, 0, 5)
	buyCount := recommendationCount(recommendations, "BUY")
	swapCount := recommendationCount(recommendations, "SWAP")
	reduceCount := recommendationCount(recommendations, "REDUCE")
	holdCount := report.Summary.ActionCounts[model.ActionHold]
	pauseCount := report.Summary.ActionCounts[model.ActionPauseBuy]

	switch {
	case buyCount > 0 && swapCount == 0 && reduceCount == 0:
		lines = append(lines, fmt.Sprintf("本期以持有和定投为主，继续执行 %d 笔加仓。", buyCount))
	case swapCount > 0 || reduceCount > 0:
		lines = append(lines, fmt.Sprintf("本期存在 %d 笔仓位调整动作，执行前应复核原因。", swapCount+reduceCount))
	default:
		lines = append(lines, "本期以持有观察为主，暂无明确交易动作。")
	}
	if holdCount > 0 {
		lines = append(lines, fmt.Sprintf("继续持有 %d 只基金，不做主动调出。", holdCount))
	}
	if pauseCount > 0 {
		lines = append(lines, fmt.Sprintf("有 %d 只基金处于暂停加仓状态，先观察，不追投。", pauseCount))
	}
	if buyCount > 0 {
		lines = append(lines, fmt.Sprintf("计划投入 %.0f 元新增资金。", recommendationAmount(recommendations, "BUY")))
	} else if report.DCAPlan != nil && report.DCAPlan.Summary.ReserveAmount > 0 {
		lines = append(lines, fmt.Sprintf("本月保留 %.0f 元定投资金，等待更合适的执行窗口。", report.DCAPlan.Summary.ReserveAmount))
	}
	return lines
}

func displayRecommendations(report model.AnalysisReport) []model.TradeRecommendation {
	recommendations := make([]model.TradeRecommendation, 0, len(report.Recommendations))
	for _, recommendation := range report.Recommendations {
		if recommendation.Kind == "BUY" {
			continue
		}
		recommendations = append(recommendations, recommendation)
	}
	return append(recommendations, displayBuyRecommendations(report)...)
}

func displayBuyRecommendations(report model.AnalysisReport) []model.TradeRecommendation {
	if report.DCAPlan == nil || len(report.DCAPlan.Items) == 0 {
		return filterRecommendations(report.Recommendations, "BUY")
	}
	recommendations := make([]model.TradeRecommendation, 0, len(report.DCAPlan.Items))
	for _, item := range report.DCAPlan.Items {
		weight := 0.0
		if report.Summary.PortfolioValue > 0 {
			weight = item.PlannedAmount / report.Summary.PortfolioValue
		}
		recommendations = append(recommendations, model.TradeRecommendation{
			Kind:            "BUY",
			TargetFund:      item.FundName,
			SuggestedWeight: weight,
			SuggestedAmount: item.PlannedAmount,
			Reason:          item.Reason,
			CreatedAt:       report.Summary.RunDate,
		})
	}
	return recommendations
}

func buildDisplayExecutionPlan(recommendations []model.TradeRecommendation) *model.ExecutionPlan {
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
					Reason:      recommendationDisplayReason(recommendation),
				},
				model.ExecutionStep{
					Order:         order + 1,
					Action:        "BUY",
					Fund:          recommendation.TargetFund,
					RelatedFund:   recommendation.SourceFund,
					Amount:        recommendation.SuggestedAmount,
					Weight:        recommendation.SuggestedWeight,
					FundingSource: fmt.Sprintf("卖出 %s", recommendation.SourceFund),
					Reason:        recommendationDisplayReason(recommendation),
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
				Reason:        recommendationDisplayReason(recommendation),
			})
			order++
		}
	}
	plan.NetCashChange = plan.GrossSellAmount - plan.GrossBuyAmount
	plan.Steps = steps
	return plan
}

func candidateDisplayReason(candidate model.CandidateSuggestion) string {
	return displayReason(candidate.Reason, candidate.EnhancedReason)
}

func recommendationDisplayReason(recommendation model.TradeRecommendation) string {
	return displayReason(recommendation.Reason, recommendation.EnhancedReason)
}

func stepDisplayReason(step model.ExecutionStep) string {
	return step.Reason
}

func displayReason(ruleReason, enhancedReason string) string {
	ruleReason = strings.TrimSpace(ruleReason)
	enhancedReason = strings.TrimSpace(enhancedReason)
	switch {
	case enhancedReason == "":
		return ruleReason
	case ruleReason == "":
		return enhancedReason
	default:
		return fmt.Sprintf("%s Rule basis: %s", enhancedReason, ruleReason)
	}
}

func filterRecommendations(recommendations []model.TradeRecommendation, kind string) []model.TradeRecommendation {
	filtered := make([]model.TradeRecommendation, 0)
	for _, recommendation := range recommendations {
		if recommendation.Kind == kind {
			filtered = append(filtered, recommendation)
		}
	}
	return filtered
}

func recommendationCount(recommendations []model.TradeRecommendation, kind string) int {
	return len(filterRecommendations(recommendations, kind))
}

func recommendationAmount(recommendations []model.TradeRecommendation, kind string) float64 {
	var total float64
	for _, recommendation := range recommendations {
		if recommendation.Kind == kind {
			total += recommendation.SuggestedAmount
		}
	}
	return total
}

func filterSignals(signals []model.FundSignal, action model.Action) []model.FundSignal {
	filtered := make([]model.FundSignal, 0)
	for _, signal := range signals {
		if signal.Action == action {
			filtered = append(filtered, signal)
		}
	}
	return filtered
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

func renderDCAPlanTable(plan model.DCAPlanReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "DCA Plan: %s\n", plan.Summary.PortfolioName)
	fmt.Fprintf(&buf, "Plan Date: %s\n", plan.Summary.PlanDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "Frequency: %s\n", plan.Summary.Frequency)
	fmt.Fprintf(&buf, "Budget: %.0f\n", plan.Summary.Budget)
	fmt.Fprintf(&buf, "Planned: %.0f\n", plan.Summary.PlannedAmount)
	fmt.Fprintf(&buf, "Reserve: %.0f\n", plan.Summary.ReserveAmount)
	fmt.Fprintf(&buf, "Eligible Funds: %d\n", plan.Summary.EligibleFundCount)
	fmt.Fprintf(&buf, "Selected Funds: %d\n\n", plan.Summary.SelectedFundCount)

	if len(plan.Items) > 0 {
		buf.WriteString("Invest This Period:\n")
		tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "PRIORITY\tFUND\tROLE\tCURRENT\tTARGET\tGAP\tAMOUNT\tREASON")
		for _, item := range plan.Items {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%.2f%%\t%.2f%%\t%.2f%%\t%.0f\t%s\n",
				item.Priority,
				item.FundName,
				item.Role,
				item.CurrentWeight*100,
				item.TargetWeight*100,
				item.GapWeight*100,
				item.PlannedAmount,
				item.Reason,
			)
		}
		_ = tw.Flush()
	}

	if len(plan.Skipped) > 0 {
		buf.WriteString("\nSkipped This Period:\n")
		tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "FUND\tACTION\tREASON")
		for _, item := range plan.Skipped {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", item.FundName, item.Action, item.Reason)
		}
		_ = tw.Flush()
	}

	if len(plan.Summary.Notes) > 0 {
		buf.WriteString("\nNotes:\n")
		for _, note := range plan.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}

	return buf.String()
}

func renderDCAPlanMarkdown(plan model.DCAPlanReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s DCA Plan\n\n", plan.Summary.PortfolioName)
	fmt.Fprintf(&buf, "- Plan Date: `%s`\n", plan.Summary.PlanDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "- Frequency: `%s`\n", plan.Summary.Frequency)
	fmt.Fprintf(&buf, "- Budget: `%.0f`\n", plan.Summary.Budget)
	fmt.Fprintf(&buf, "- Planned: `%.0f`\n", plan.Summary.PlannedAmount)
	fmt.Fprintf(&buf, "- Reserve: `%.0f`\n", plan.Summary.ReserveAmount)
	fmt.Fprintf(&buf, "- Eligible Funds: `%d`\n", plan.Summary.EligibleFundCount)
	fmt.Fprintf(&buf, "- Selected Funds: `%d`\n", plan.Summary.SelectedFundCount)

	if len(plan.Items) > 0 {
		buf.WriteString("\n## Invest This Period\n\n")
		buf.WriteString("| Priority | Fund | Role | Current | Target | Gap | Amount | Reason |\n")
		buf.WriteString("| ---: | --- | --- | ---: | ---: | ---: | ---: | --- |\n")
		for _, item := range plan.Items {
			fmt.Fprintf(&buf, "| %d | %s | %s | %.2f%% | %.2f%% | %.2f%% | %.0f | %s |\n",
				item.Priority,
				item.FundName,
				item.Role,
				item.CurrentWeight*100,
				item.TargetWeight*100,
				item.GapWeight*100,
				item.PlannedAmount,
				item.Reason,
			)
		}
	}

	if len(plan.Skipped) > 0 {
		buf.WriteString("\n## Skipped This Period\n\n")
		buf.WriteString("| Fund | Action | Reason |\n")
		buf.WriteString("| --- | --- | --- |\n")
		for _, item := range plan.Skipped {
			fmt.Fprintf(&buf, "| %s | %s | %s |\n", item.FundName, item.Action, item.Reason)
		}
	}

	if len(plan.Summary.Notes) > 0 {
		buf.WriteString("\n## Notes\n\n")
		for _, note := range plan.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}

	return buf.String()
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
