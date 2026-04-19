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

func RenderMarketPool(pool model.MarketPoolReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMarketPoolTable(pool), nil
	case "markdown", "md":
		return renderMarketPoolMarkdown(pool), nil
	case "json":
		buf, err := json.MarshalIndent(pool, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported market pool format: %s", format)
	}
}

func RenderTable(report model.AnalysisReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "组合：%s\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "运行时间：%s\n", report.Summary.RunDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "组合市值：%.2f\n", report.Summary.PortfolioValue)
	fmt.Fprintf(&buf, "当日加权涨跌：%.2f%%\n\n", report.Summary.WeightedDayChangePct*100)

	buf.WriteString("动作统计：\n")
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "动作\t数量")
	for action, count := range report.Summary.ActionCounts {
		fmt.Fprintf(tw, "%s\t%d\n", displayAction(action), count)
	}
	_ = tw.Flush()

	if len(report.Recommendations) > 0 {
		buf.WriteString("\n建议动作：\n")
		tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "动作\t从\t到\t权重\t金额\t原因")
		for _, recommendation := range report.Recommendations {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%.2f%%\t%.0f\t%s\n",
				displayRecommendationKind(recommendation.Kind),
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
		buf.WriteString("\n执行计划：\n")
		fmt.Fprintf(&buf, "总卖出：%.0f\n", report.ExecutionPlan.GrossSellAmount)
		fmt.Fprintf(&buf, "总买入：%.0f\n", report.ExecutionPlan.GrossBuyAmount)
		fmt.Fprintf(&buf, "替换金额：%.0f\n", report.ExecutionPlan.SwapAmount)
		fmt.Fprintf(&buf, "减仓金额：%.0f\n", report.ExecutionPlan.ReduceAmount)
		fmt.Fprintf(&buf, "买入金额：%.0f\n", report.ExecutionPlan.BuyAmount)
		fmt.Fprintf(&buf, "净现金变化：%.0f\n\n", report.ExecutionPlan.NetCashChange)
		tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "步骤\t动作\t基金\t关联\t金额\t资金来源\t原因")
		for _, step := range report.ExecutionPlan.Steps {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%s\t%.0f\t%s\t%s\n",
				step.Order,
				displayExecutionAction(step.Action),
				step.Fund,
				displayOrDash(step.RelatedFund),
				step.Amount,
				displayOrDash(step.FundingSource),
				stepDisplayReason(step),
			)
		}
		_ = tw.Flush()
	}

	buf.WriteString("\n信号列表：\n")
	tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "动作\t基金\t当前\t目标\t20日\t60日\t原因")
	for _, signal := range report.Signals {
		fmt.Fprintf(tw, "%s\t%s\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%s\n",
			displayAction(signal.Action),
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
		buf.WriteString("\n候选列表：\n")
		tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "基金\t评分\t20日\t60日\t替代对象\t原因")
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

	fmt.Fprintf(&buf, "# %s 投资行动手册\n\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "- 运行时间：`%s`\n", report.Summary.RunDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "- 组合市值：`%.2f`\n", report.Summary.PortfolioValue)
	fmt.Fprintf(&buf, "- 当日加权涨跌：`%.2f%%`\n", report.Summary.WeightedDayChangePct*100)
	if len(report.Summary.Notes) > 0 {
		for _, note := range report.Summary.Notes {
			fmt.Fprintf(&buf, "- 备注：%s\n", note)
		}
	}
	buf.WriteString("\n## 本期结论\n\n")
	for _, line := range renderPlaybookSummary(report, displayRecommendations) {
		fmt.Fprintf(&buf, "- %s\n", line)
	}

	if len(buyRecommendations) > 0 {
		buf.WriteString("\n## 继续定投\n\n")
		buf.WriteString("| 基金 | 金额 | 权重 | 原因 |\n")
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
			fmt.Fprintf(&buf, "\n- 本月定投预留：`%.0f`\n", report.DCAPlan.Summary.ReserveAmount)
		}
	}

	reduceRecommendations := filterRecommendations(displayRecommendations, "REDUCE")
	if len(reduceRecommendations) > 0 {
		buf.WriteString("\n## 减仓观察\n\n")
		buf.WriteString("| 基金 | 金额 | 权重 | 原因 |\n")
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
		buf.WriteString("\n## 替换观察\n\n")
		buf.WriteString("| 从 | 到 | 金额 | 权重 | 原因 |\n")
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
		buf.WriteString("\n## 继续持有\n\n")
		buf.WriteString("| 基金 | 当前 | 目标 | 20日 | 60日 | 原因 |\n")
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
		buf.WriteString("\n## 暂停加仓\n\n")
		buf.WriteString("| 基金 | 当前 | 目标 | 20日 | 60日 | 原因 |\n")
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
		buf.WriteString("\n## 执行顺序\n\n")
		fmt.Fprintf(&buf, "- 总卖出：`%.0f`\n", executionPlan.GrossSellAmount)
		fmt.Fprintf(&buf, "- 总买入：`%.0f`\n", executionPlan.GrossBuyAmount)
		fmt.Fprintf(&buf, "- 替换金额：`%.0f`\n", executionPlan.SwapAmount)
		fmt.Fprintf(&buf, "- 减仓金额：`%.0f`\n", executionPlan.ReduceAmount)
		fmt.Fprintf(&buf, "- 买入金额：`%.0f`\n", executionPlan.BuyAmount)
		fmt.Fprintf(&buf, "- 净现金变化：`%.0f`\n\n", executionPlan.NetCashChange)
		buf.WriteString("| 步骤 | 动作 | 基金 | 关联 | 金额 | 资金来源 | 原因 |\n")
		buf.WriteString("| ---: | --- | --- | --- | ---: | --- | --- |\n")
		for _, step := range executionPlan.Steps {
			fmt.Fprintf(&buf, "| %d | %s | %s | %s | %.0f | %s | %s |\n",
				step.Order,
				displayExecutionAction(step.Action),
				step.Fund,
				displayOrDash(step.RelatedFund),
				step.Amount,
				displayOrDash(step.FundingSource),
				stepDisplayReason(step),
			)
		}
	}
	if len(report.Candidates) > 0 {
		buf.WriteString("\n## 候选替代\n\n")
		buf.WriteString("| 基金 | 评分 | 20日 | 60日 | 替代对象 | 原因 |\n")
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
		return fmt.Sprintf("%s 规则依据：%s", enhancedReason, ruleReason)
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
	fmt.Fprintf(&buf, "定投计划：%s\n", plan.Summary.PortfolioName)
	fmt.Fprintf(&buf, "计划日期：%s\n", plan.Summary.PlanDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "频率：%s\n", displayDCAFrequency(plan.Summary.Frequency))
	fmt.Fprintf(&buf, "预算：%.0f\n", plan.Summary.Budget)
	fmt.Fprintf(&buf, "已计划：%.0f\n", plan.Summary.PlannedAmount)
	fmt.Fprintf(&buf, "预留：%.0f\n", plan.Summary.ReserveAmount)
	fmt.Fprintf(&buf, "可选基金数：%d\n", plan.Summary.EligibleFundCount)
	fmt.Fprintf(&buf, "入选基金数：%d\n\n", plan.Summary.SelectedFundCount)

	if len(plan.Items) > 0 {
		buf.WriteString("本期执行：\n")
		tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "优先级\t基金\t角色\t当前\t目标\t差距\t金额\t原因")
		for _, item := range plan.Items {
			fmt.Fprintf(tw, "%d\t%s\t%s\t%.2f%%\t%.2f%%\t%.2f%%\t%.0f\t%s\n",
				item.Priority,
				item.FundName,
				displayFundRole(item.Role),
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
		buf.WriteString("\n本期跳过：\n")
		tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "基金\t动作\t原因")
		for _, item := range plan.Skipped {
			fmt.Fprintf(tw, "%s\t%s\t%s\n", item.FundName, displayAction(item.Action), item.Reason)
		}
		_ = tw.Flush()
	}

	if len(plan.Summary.Notes) > 0 {
		buf.WriteString("\n备注：\n")
		for _, note := range plan.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}

	return buf.String()
}

func renderDCAPlanMarkdown(plan model.DCAPlanReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# %s 定投计划\n\n", plan.Summary.PortfolioName)
	fmt.Fprintf(&buf, "- 计划日期：`%s`\n", plan.Summary.PlanDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "- 频率：`%s`\n", displayDCAFrequency(plan.Summary.Frequency))
	fmt.Fprintf(&buf, "- 预算：`%.0f`\n", plan.Summary.Budget)
	fmt.Fprintf(&buf, "- 已计划：`%.0f`\n", plan.Summary.PlannedAmount)
	fmt.Fprintf(&buf, "- 预留：`%.0f`\n", plan.Summary.ReserveAmount)
	fmt.Fprintf(&buf, "- 可选基金数：`%d`\n", plan.Summary.EligibleFundCount)
	fmt.Fprintf(&buf, "- 入选基金数：`%d`\n", plan.Summary.SelectedFundCount)

	if len(plan.Items) > 0 {
		buf.WriteString("\n## 本期执行\n\n")
		buf.WriteString("| 优先级 | 基金 | 角色 | 当前 | 目标 | 差距 | 金额 | 原因 |\n")
		buf.WriteString("| ---: | --- | --- | ---: | ---: | ---: | ---: | --- |\n")
		for _, item := range plan.Items {
			fmt.Fprintf(&buf, "| %d | %s | %s | %.2f%% | %.2f%% | %.2f%% | %.0f | %s |\n",
				item.Priority,
				item.FundName,
				displayFundRole(item.Role),
				item.CurrentWeight*100,
				item.TargetWeight*100,
				item.GapWeight*100,
				item.PlannedAmount,
				item.Reason,
			)
		}
	}

	if len(plan.Skipped) > 0 {
		buf.WriteString("\n## 本期跳过\n\n")
		buf.WriteString("| 基金 | 动作 | 原因 |\n")
		buf.WriteString("| --- | --- | --- |\n")
		for _, item := range plan.Skipped {
			fmt.Fprintf(&buf, "| %s | %s | %s |\n", item.FundName, displayAction(item.Action), item.Reason)
		}
	}

	if len(plan.Summary.Notes) > 0 {
		buf.WriteString("\n## 备注\n\n")
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
	fmt.Fprintf(&buf, "回测：%s\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "区间：%s -> %s\n", report.Summary.StartDate.Format("2006-01-02"), report.Summary.EndDate.Format("2006-01-02"))
	fmt.Fprintf(&buf, "交易日：%d\n", report.Summary.TradingDays)
	fmt.Fprintf(&buf, "调仓间隔：%d\n\n", report.Summary.RebalanceEvery)

	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "指标\t数值")
	fmt.Fprintf(tw, "初始市值\t%.2f\n", report.Summary.InitialValue)
	fmt.Fprintf(tw, "最终市值\t%.2f\n", report.Summary.FinalValue)
	fmt.Fprintf(tw, "基准最终市值\t%.2f\n", report.Summary.BenchmarkFinalValue)
	fmt.Fprintf(tw, "总收益率\t%.2f%%\n", report.Summary.TotalReturn*100)
	fmt.Fprintf(tw, "基准收益率\t%.2f%%\n", report.Summary.BenchmarkReturn*100)
	fmt.Fprintf(tw, "超额收益\t%.2f%%\n", report.Summary.ExcessReturn*100)
	fmt.Fprintf(tw, "年化收益率\t%.2f%%\n", report.Summary.AnnualizedReturn*100)
	fmt.Fprintf(tw, "基准年化收益率\t%.2f%%\n", report.Summary.BenchmarkAnnualizedReturn*100)
	fmt.Fprintf(tw, "最大回撤\t%.2f%%\n", report.Summary.MaxDrawdown*100)
	fmt.Fprintf(tw, "基准最大回撤\t%.2f%%\n", report.Summary.BenchmarkMaxDrawdown*100)
	fmt.Fprintf(tw, "调仓次数\t%d\n", report.Summary.RebalanceCount)
	fmt.Fprintf(tw, "交易次数\t%d\n", report.Summary.TradeCount)
	fmt.Fprintf(tw, "期末现金\t%.2f\n", report.Summary.CashFinal)
	_ = tw.Flush()

	if len(report.Summary.Notes) > 0 {
		buf.WriteString("\n备注：\n")
		for _, note := range report.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}

	if len(report.Trades) > 0 {
		buf.WriteString("\n近期交易：\n")
		tw = tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "日期\t动作\t基金\t关联\t金额\t价格\t份额")
		start := 0
		if len(report.Trades) > 10 {
			start = len(report.Trades) - 10
		}
		for _, trade := range report.Trades[start:] {
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%.0f\t%.4f\t%.4f\n",
				trade.Date.Format("2006-01-02"),
				displayExecutionAction(trade.Action),
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
	fmt.Fprintf(&buf, "# %s 策略回测\n\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "- 区间：`%s` -> `%s`\n", report.Summary.StartDate.Format("2006-01-02"), report.Summary.EndDate.Format("2006-01-02"))
	fmt.Fprintf(&buf, "- 交易日：`%d`\n", report.Summary.TradingDays)
	fmt.Fprintf(&buf, "- 调仓间隔：`%d`\n", report.Summary.RebalanceEvery)
	fmt.Fprintf(&buf, "- 调仓次数：`%d`\n", report.Summary.RebalanceCount)
	fmt.Fprintf(&buf, "- 交易次数：`%d`\n\n", report.Summary.TradeCount)
	buf.WriteString("| 指标 | 数值 |\n")
	buf.WriteString("| --- | ---: |\n")
	fmt.Fprintf(&buf, "| 初始市值 | %.2f |\n", report.Summary.InitialValue)
	fmt.Fprintf(&buf, "| 最终市值 | %.2f |\n", report.Summary.FinalValue)
	fmt.Fprintf(&buf, "| 基准最终市值 | %.2f |\n", report.Summary.BenchmarkFinalValue)
	fmt.Fprintf(&buf, "| 总收益率 | %.2f%% |\n", report.Summary.TotalReturn*100)
	fmt.Fprintf(&buf, "| 基准收益率 | %.2f%% |\n", report.Summary.BenchmarkReturn*100)
	fmt.Fprintf(&buf, "| 超额收益 | %.2f%% |\n", report.Summary.ExcessReturn*100)
	fmt.Fprintf(&buf, "| 年化收益率 | %.2f%% |\n", report.Summary.AnnualizedReturn*100)
	fmt.Fprintf(&buf, "| 基准年化收益率 | %.2f%% |\n", report.Summary.BenchmarkAnnualizedReturn*100)
	fmt.Fprintf(&buf, "| 最大回撤 | %.2f%% |\n", report.Summary.MaxDrawdown*100)
	fmt.Fprintf(&buf, "| 基准最大回撤 | %.2f%% |\n", report.Summary.BenchmarkMaxDrawdown*100)
	fmt.Fprintf(&buf, "| 期末现金 | %.2f |\n", report.Summary.CashFinal)

	if len(report.Summary.Notes) > 0 {
		buf.WriteString("\n## 备注\n\n")
		for _, note := range report.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}

	if len(report.Trades) > 0 {
		buf.WriteString("\n## 近期交易\n\n")
		buf.WriteString("| 日期 | 动作 | 基金 | 关联 | 金额 | 价格 | 份额 |\n")
		buf.WriteString("| --- | --- | --- | --- | ---: | ---: | ---: |\n")
		start := 0
		if len(report.Trades) > 10 {
			start = len(report.Trades) - 10
		}
		for _, trade := range report.Trades[start:] {
			fmt.Fprintf(&buf, "| %s | %s | %s | %s | %.0f | %.4f | %.4f |\n",
				trade.Date.Format("2006-01-02"),
				displayExecutionAction(trade.Action),
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

func renderMarketPoolTable(pool model.MarketPoolReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "候选池运行时间：%s\n", pool.Summary.RunDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "全市场基金数：%d\n", pool.Summary.UniverseCount)
	fmt.Fprintf(&buf, "主题匹配数：%d\n", pool.Summary.MatchedCount)
	fmt.Fprintf(&buf, "满足阈值数：%d\n", pool.Summary.EligibleCount)
	fmt.Fprintf(&buf, "最终入选数：%d\n", pool.Summary.SelectedCount)
	fmt.Fprintf(&buf, "沿用上期数：%d\n", pool.Summary.RetainedCount)
	if len(pool.Summary.Notes) > 0 {
		buf.WriteString("\n备注：\n")
		for _, note := range pool.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}
	if len(pool.Items) == 0 {
		buf.WriteString("\n当前没有筛选出稳定候选。\n")
		return buf.String()
	}
	buf.WriteString("\n候选列表：\n")
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "排名	主题	基金	评分	120日	250日	120日最大回撤	规模	保留	原因")
	for _, item := range pool.Items {
		fmt.Fprintf(tw, "%d	%s	%s	%d	%.2f%%	%.2f%%	%.2f%%	%.1f亿	%s	%s\n",
			item.Rank,
			item.ThemeLabel,
			item.FundName,
			item.Score,
			item.Return120D*100,
			item.Return250D*100,
			item.MaxDrawdown120D*100,
			item.FundSizeYi,
			yesNo(item.Retained),
			item.Reason,
		)
	}
	_ = tw.Flush()
	return buf.String()
}

func renderMarketPoolMarkdown(pool model.MarketPoolReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# 稳定候选池\n\n")
	fmt.Fprintf(&buf, "- 运行时间：`%s`\n", pool.Summary.RunDate.Format(time.RFC3339))
	fmt.Fprintf(&buf, "- 全市场基金数：`%d`\n", pool.Summary.UniverseCount)
	fmt.Fprintf(&buf, "- 主题匹配数：`%d`\n", pool.Summary.MatchedCount)
	fmt.Fprintf(&buf, "- 满足阈值数：`%d`\n", pool.Summary.EligibleCount)
	fmt.Fprintf(&buf, "- 最终入选数：`%d`\n", pool.Summary.SelectedCount)
	fmt.Fprintf(&buf, "- 沿用上期数：`%d`\n", pool.Summary.RetainedCount)
	if len(pool.Summary.Notes) > 0 {
		buf.WriteString("\n## 备注\n\n")
		for _, note := range pool.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}
	if len(pool.Items) == 0 {
		buf.WriteString("\n## 候选列表\n\n- 当前没有筛选出稳定候选。\n")
		return buf.String()
	}
	buf.WriteString("\n## 候选列表\n\n")
	buf.WriteString("| 排名 | 主题 | 基金 | 评分 | 120日 | 250日 | 120日最大回撤 | 规模 | 保留 | 原因 |\n")
	buf.WriteString("| ---: | --- | --- | ---: | ---: | ---: | ---: | ---: | --- | --- |\n")
	for _, item := range pool.Items {
		fmt.Fprintf(&buf, "| %d | %s | %s | %d | %.2f%% | %.2f%% | %.2f%% | %.1f亿 | %s | %s |\n",
			item.Rank,
			item.ThemeLabel,
			item.FundName,
			item.Score,
			item.Return120D*100,
			item.Return250D*100,
			item.MaxDrawdown120D*100,
			item.FundSizeYi,
			yesNo(item.Retained),
			item.Reason,
		)
	}
	return buf.String()
}

func yesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func displayAction(action model.Action) string {
	switch action {
	case model.ActionBuy:
		return "买入"
	case model.ActionHold:
		return "持有"
	case model.ActionPauseBuy:
		return "暂停加仓"
	case model.ActionReduce:
		return "减仓"
	case model.ActionReplaceWatch:
		return "替换观察"
	default:
		return string(action)
	}
}

func displayRecommendationKind(kind string) string {
	switch strings.ToUpper(strings.TrimSpace(kind)) {
	case "BUY":
		return "买入"
	case "REDUCE":
		return "减仓"
	case "SWAP":
		return "替换"
	default:
		return kind
	}
}

func displayExecutionAction(action string) string {
	switch strings.ToUpper(strings.TrimSpace(action)) {
	case "BUY":
		return "买入"
	case "SELL":
		return "卖出"
	default:
		return action
	}
}

func displayDCAFrequency(frequency string) string {
	switch strings.ToLower(strings.TrimSpace(frequency)) {
	case "monthly":
		return "每月"
	case "weekly":
		return "每周"
	default:
		return frequency
	}
}

func displayFundRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "core":
		return "核心"
	case "satellite":
		return "卫星"
	case "hedge":
		return "对冲"
	case "stabilizer":
		return "稳健"
	default:
		return role
	}
}
