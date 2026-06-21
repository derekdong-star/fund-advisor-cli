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

func RenderMomentumPool(pool model.MomentumPoolReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMomentumPoolTable(pool), nil
	case "markdown", "md":
		return renderMomentumPoolMarkdown(pool), nil
	case "json":
		buf, err := json.MarshalIndent(pool, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported momentum pool format: %s", format)
	}
}

func RenderMomentumBacktest(result model.MomentumBacktestReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMomentumBacktestTable(result), nil
	case "markdown", "md":
		return renderMomentumBacktestMarkdown(result), nil
	case "json":
		buf, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported momentum backtest format: %s", format)
	}
}

func RenderMomentumBacktestSensitivity(result model.MomentumBacktestSensitivityReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMomentumBacktestSensitivityTable(result), nil
	case "markdown", "md":
		return renderMomentumBacktestSensitivityMarkdown(result), nil
	case "json":
		buf, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported momentum sensitivity format: %s", format)
	}
}

func RenderMomentumBacktestParameterGrid(result model.MomentumBacktestParameterGridReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMomentumBacktestParameterGridTable(result), nil
	case "markdown", "md":
		return renderMomentumBacktestParameterGridMarkdown(result), nil
	case "json":
		buf, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported momentum parameter grid format: %s", format)
	}
}

func RenderMomentumBacktestStages(result model.MomentumBacktestStageReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMomentumBacktestStagesTable(result), nil
	case "markdown", "md":
		return renderMomentumBacktestStagesMarkdown(result), nil
	case "json":
		buf, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported momentum stage format: %s", format)
	}
}

func RenderMomentumBacktestEntryGrid(result model.MomentumBacktestEntryGridReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMomentumBacktestEntryGridTable(result), nil
	case "markdown", "md":
		return renderMomentumBacktestEntryGridMarkdown(result), nil
	case "json":
		buf, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported momentum entry grid format: %s", format)
	}
}

func RenderMomentumBacktestCorrelationGrid(result model.MomentumBacktestCorrelationGridReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMomentumBacktestCorrelationGridTable(result), nil
	case "markdown", "md":
		return renderMomentumBacktestCorrelationGridMarkdown(result), nil
	case "json":
		buf, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported momentum correlation grid format: %s", format)
	}
}

func RenderMomentumBacktestWeightGrid(result model.MomentumBacktestWeightGridReport, format string) (string, error) {
	switch format {
	case "table":
		return renderMomentumBacktestWeightGridTable(result), nil
	case "markdown", "md":
		return renderMomentumBacktestWeightGridMarkdown(result), nil
	case "json":
		buf, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", err
		}
		return string(buf), nil
	default:
		return "", fmt.Errorf("unsupported momentum weight grid format: %s", format)
	}
}

func RenderTable(report model.AnalysisReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "组合：%s\n", report.Summary.PortfolioName)
	fmt.Fprintf(&buf, "运行时间：%s\n", formatDisplayTime(report.Summary.RunDate))
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
	reduceRecommendations := filterRecommendations(displayRecommendations, "REDUCE")
	swapRecommendations := filterRecommendations(displayRecommendations, "SWAP")
	holdSignals := filterSignals(report.Signals, model.ActionHold)
	pauseSignals := filterSignals(report.Signals, model.ActionPauseBuy)
	executionPlan := buildDisplayExecutionPlan(displayRecommendations)

	renderMarkdownHeader(&buf, report)
	renderBuyRecommendationsSection(&buf, report, buyRecommendations)
	renderOpportunitySection(&buf, report.Opportunity)
	renderReduceRecommendationsSection(&buf, reduceRecommendations)
	renderSwapRecommendationsSection(&buf, swapRecommendations)
	renderSignalSection(&buf, "继续持有", holdSignals)
	renderPositionSnapshotSection(&buf, report)
	renderSignalSection(&buf, "暂停加仓", pauseSignals)
	renderExecutionPlanSection(&buf, executionPlan)
	renderCandidateSection(&buf, report.Candidates)
	return buf.String()
}

func renderMarkdownHeader(buf *bytes.Buffer, report model.AnalysisReport) {
	displayRecommendations := displayRecommendations(report)
	fmt.Fprintf(buf, "# %s 投资行动手册\n\n", report.Summary.PortfolioName)
	fmt.Fprintf(buf, "- 运行时间：`%s`\n", formatDisplayTime(report.Summary.RunDate))
	fmt.Fprintf(buf, "- 组合市值：`%.2f`\n", report.Summary.PortfolioValue)
	fmt.Fprintf(buf, "- 当日加权涨跌：`%.2f%%`\n", report.Summary.WeightedDayChangePct*100)
	if len(report.Summary.Notes) > 0 {
		for _, note := range report.Summary.Notes {
			fmt.Fprintf(buf, "- 备注：%s\n", note)
		}
	}
	buf.WriteString("\n## 本期结论\n\n")
	for _, line := range renderPlaybookSummary(report, displayRecommendations) {
		fmt.Fprintf(buf, "- %s\n", line)
	}
}

func renderBuyRecommendationsSection(buf *bytes.Buffer, report model.AnalysisReport, buyRecommendations []model.TradeRecommendation) {
	if len(buyRecommendations) > 0 {
		buf.WriteString("\n## 继续定投\n\n")
		buf.WriteString("| 基金 | 金额 | 权重 | 原因 |\n")
		buf.WriteString("| --- | ---: | ---: | --- |\n")
		for _, recommendation := range buyRecommendations {
			fmt.Fprintf(buf, "| %s | %.0f | %.2f%% | %s |\n",
				recommendation.TargetFund,
				recommendation.SuggestedAmount,
				recommendation.SuggestedWeight*100,
				recommendationDisplayReason(recommendation),
			)
		}
		if report.DCAPlan != nil && report.DCAPlan.Summary.ReserveAmount > 0 {
			fmt.Fprintf(buf, "\n- 本月定投预留：`%.0f`\n", report.DCAPlan.Summary.ReserveAmount)
		}
	}
}

func renderOpportunitySection(buf *bytes.Buffer, opportunity *model.OpportunityReport) {
	if opportunity == nil {
		return
	}
	buf.WriteString("\n## 机会雷达\n\n")
	fmt.Fprintf(buf, "- 市场状态：`%s`\n", opportunity.Summary.Window)
	fmt.Fprintf(buf, "- 解读：%s\n", opportunity.Summary.Reason)
	fmt.Fprintf(buf, "- 组合内可继续加仓：`%d`\n", opportunity.Summary.HoldingOpportunities)
	fmt.Fprintf(buf, "- 场外稳定候选：`%d`\n", opportunity.Summary.CandidateCount)

	if len(opportunity.Holdings) > 0 {
		buf.WriteString("\n### 组合内优先加仓\n\n")
		buf.WriteString("| 优先级 | 基金 | 计划金额 | 当前 | 目标 | 20日 | 60日 | 原因 |\n")
		buf.WriteString("| ---: | --- | ---: | ---: | ---: | ---: | ---: | --- |\n")
		for _, item := range opportunity.Holdings {
			fmt.Fprintf(buf, "| %d | %s | %.0f | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %s |\n",
				item.Priority,
				item.FundName,
				item.PlannedAmount,
				item.CurrentWeight*100,
				item.TargetWeight*100,
				item.Return20D*100,
				item.Return60D*100,
				item.Reason,
			)
		}
	}

	if len(opportunity.Candidates) > 0 {
		buf.WriteString("\n### 场外稳定候选\n\n")
		buf.WriteString("| 排名 | 主题 | 基金 | 评分 | 20日 | 60日 | 120日 | 原因 |\n")
		buf.WriteString("| ---: | --- | --- | ---: | ---: | ---: | ---: | --- |\n")
		for _, item := range opportunity.Candidates {
			reason := item.Reason
			if item.Retained {
				reason = "延续保留；" + reason
			}
			fmt.Fprintf(buf, "| %d | %s | %s | %d | %.2f%% | %.2f%% | %.2f%% | %s |\n",
				item.Rank,
				item.ThemeLabel,
				item.FundName,
				item.Score,
				item.Return20D*100,
				item.Return60D*100,
				item.Return120D*100,
				reason,
			)
		}
	}
}

func renderReduceRecommendationsSection(buf *bytes.Buffer, reduceRecommendations []model.TradeRecommendation) {
	if len(reduceRecommendations) > 0 {
		buf.WriteString("\n## 减仓观察\n\n")
		buf.WriteString("| 基金 | 金额 | 权重 | 原因 |\n")
		buf.WriteString("| --- | ---: | ---: | --- |\n")
		for _, recommendation := range reduceRecommendations {
			fmt.Fprintf(buf, "| %s | %.0f | %.2f%% | %s |\n",
				recommendation.SourceFund,
				recommendation.SuggestedAmount,
				recommendation.SuggestedWeight*100,
				recommendationDisplayReason(recommendation),
			)
		}
	}
}

func renderSwapRecommendationsSection(buf *bytes.Buffer, swapRecommendations []model.TradeRecommendation) {
	if len(swapRecommendations) > 0 {
		buf.WriteString("\n## 替换观察\n\n")
		buf.WriteString("| 从 | 到 | 金额 | 权重 | 原因 |\n")
		buf.WriteString("| --- | --- | ---: | ---: | --- |\n")
		for _, recommendation := range swapRecommendations {
			fmt.Fprintf(buf, "| %s | %s | %.0f | %.2f%% | %s |\n",
				recommendation.SourceFund,
				recommendation.TargetFund,
				recommendation.SuggestedAmount,
				recommendation.SuggestedWeight*100,
				recommendationDisplayReason(recommendation),
			)
		}
	}
}

func renderSignalSection(buf *bytes.Buffer, title string, signals []model.FundSignal) {
	if len(signals) > 0 {
		fmt.Fprintf(buf, "\n## %s\n\n", title)
		buf.WriteString("| 基金 | 当前 | 目标 | 20日 | 60日 | 原因 |\n")
		buf.WriteString("| --- | ---: | ---: | ---: | ---: | --- |\n")
		for _, signal := range signals {
			fmt.Fprintf(buf, "| %s | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %s |\n",
				signal.FundName,
				signal.CurrentWeight*100,
				signal.TargetWeight*100,
				signal.Return20D*100,
				signal.Return60D*100,
				signal.Reason,
			)
		}
	}
}

func renderPositionSnapshotSection(buf *bytes.Buffer, report model.AnalysisReport) {
	if len(report.Position) > 0 {
		showLedgerColumns := reportHasLedger(report)
		buf.WriteString("\n## 持仓快照\n\n")
		if showLedgerColumns {
			buf.WriteString("| 基金 | 当前市值 | 当前权重 | 份额 | 持仓成本 | 浮盈亏 | 收益率 | 最近手工交易 |\n")
			buf.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
			for _, state := range report.Position {
				fmt.Fprintf(buf, "| %s | %.2f | %.2f%% | %.4f | %.2f | %s | %s | %s |\n",
					state.Position.FundName,
					state.CurrentValue,
					state.CurrentWeight*100,
					state.Position.EstimatedUnits,
					state.HoldingCost,
					displaySignedMoney(state.UnrealizedPnL),
					displaySignedPercent(state.UnrealizedPnLPct),
					displayLedgerTradeInfo(state),
				)
			}
		} else {
			buf.WriteString("| 基金 | 当前市值 | 当前权重 | 份额 |\n")
			buf.WriteString("| --- | ---: | ---: | ---: |\n")
			for _, state := range report.Position {
				fmt.Fprintf(buf, "| %s | %.2f | %.2f%% | %.4f |\n",
					state.Position.FundName,
					state.CurrentValue,
					state.CurrentWeight*100,
					state.Position.EstimatedUnits,
				)
			}
		}
	}
}

func renderExecutionPlanSection(buf *bytes.Buffer, executionPlan *model.ExecutionPlan) {
	if executionPlan != nil && len(executionPlan.Steps) > 0 {
		buf.WriteString("\n## 执行顺序\n\n")
		fmt.Fprintf(buf, "- 总卖出：`%.0f`\n", executionPlan.GrossSellAmount)
		fmt.Fprintf(buf, "- 总买入：`%.0f`\n", executionPlan.GrossBuyAmount)
		fmt.Fprintf(buf, "- 替换金额：`%.0f`\n", executionPlan.SwapAmount)
		fmt.Fprintf(buf, "- 减仓金额：`%.0f`\n", executionPlan.ReduceAmount)
		fmt.Fprintf(buf, "- 买入金额：`%.0f`\n", executionPlan.BuyAmount)
		fmt.Fprintf(buf, "- 净现金变化：`%.0f`\n\n", executionPlan.NetCashChange)
		buf.WriteString("| 步骤 | 动作 | 基金 | 关联 | 金额 | 资金来源 | 原因 |\n")
		buf.WriteString("| ---: | --- | --- | --- | ---: | --- | --- |\n")
		for _, step := range executionPlan.Steps {
			fmt.Fprintf(buf, "| %d | %s | %s | %s | %.0f | %s | %s |\n",
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
}

func renderCandidateSection(buf *bytes.Buffer, candidates []model.CandidateSuggestion) {
	if len(candidates) > 0 {
		buf.WriteString("\n## 候选替代\n\n")
		buf.WriteString("| 基金 | 评分 | 20日 | 60日 | 替代对象 | 原因 |\n")
		buf.WriteString("| --- | ---: | ---: | ---: | --- | --- |\n")
		for _, candidate := range candidates {
			fmt.Fprintf(buf, "| %s | %d | %.2f%% | %.2f%% | %s | %s |\n",
				candidate.FundName,
				candidate.Score,
				candidate.Return20D*100,
				candidate.Return60D*100,
				stringsJoin(candidate.ReplaceFor),
				candidateDisplayReason(candidate),
			)
		}
	}
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
	if report.Opportunity != nil {
		switch {
		case report.Opportunity.Summary.HoldingOpportunities > 0:
			lines = append(lines, fmt.Sprintf("当前处于%s，组合内有 %d 只基金可继续加仓。", report.Opportunity.Summary.Window, report.Opportunity.Summary.HoldingOpportunities))
		case report.Opportunity.Summary.CandidateCount > 0:
			lines = append(lines, fmt.Sprintf("当前处于%s，可同步跟踪 %d 只场外稳定候选。", report.Opportunity.Summary.Window, report.Opportunity.Summary.CandidateCount))
		}
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
	fmt.Fprintf(&buf, "计划日期：%s\n", formatDisplayTime(plan.Summary.PlanDate))
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
	fmt.Fprintf(&buf, "- 计划日期：`%s`\n", formatDisplayTime(plan.Summary.PlanDate))
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
	fmt.Fprintf(&buf, "候选池运行时间：%s\n", formatDisplayTime(pool.Summary.RunDate))
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
	fmt.Fprintf(&buf, "- 运行时间：`%s`\n", formatDisplayTime(pool.Summary.RunDate))
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

func renderMomentumPoolTable(pool model.MomentumPoolReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "运行时间：%s\n", formatDisplayTime(pool.Summary.RunDate))
	fmt.Fprintf(&buf, "市场状态：%s，执行状态：%s，市场广度：%.1f%%，120日中位趋势：%.2f%%\n", momentumRegimeLabel(pool.Summary.Regime), momentumReadinessLabel(pool.Summary.Readiness), pool.Summary.PositiveBreadth*100, pool.Summary.MarketMedianReturn120D*100)
	fmt.Fprintf(&buf, "前瞻验证：%d 批 / %d 只，平均收益 %.2f%%，候选胜率 %.1f%%，批次胜率 %.1f%%\n", pool.Summary.ForwardValidationBatches, pool.Summary.ForwardValidationCandidates, pool.Summary.ForwardAverageReturn*100, pool.Summary.ForwardWinRate*100, pool.Summary.ForwardBatchWinRate*100)
	if pool.Summary.MaxCorrelationPair != "" {
		fmt.Fprintf(&buf, "候选相关性：平均 %.2f，最高 %.2f（%s）\n", pool.Summary.AveragePairCorrelation, pool.Summary.MaxPairCorrelation, pool.Summary.MaxCorrelationPair)
	}
	fmt.Fprintf(&buf, "权益基金：%d，评估：%d，达标：%d，入选：%d\n\n", pool.Summary.EquityCount, pool.Summary.EvaluatedCount, pool.Summary.QualifiedCount, pool.Summary.SelectedCount)
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "排名\t基金代码\t基金\t动量分\t20日\t60日\t120日\t120日回撤\t最高同类相关性\t原因")
	for _, item := range pool.Items {
		fmt.Fprintf(tw, "%d\t%s\t%s\t%.1f\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%s\t%s\n", item.Rank, item.FundCode, item.FundName, item.MomentumScore, item.Return20D*100, item.Return60D*100, item.Return120D*100, item.MaxDrawdown120D*100, formatPeerCorrelation(item), item.Reason)
	}
	_ = tw.Flush()
	if len(pool.Challengers) > 0 {
		fmt.Fprintf(&buf, "\n新晋强势挑战者（仅观察，不自动替换）：\n")
		challengerWriter := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
		fmt.Fprintln(challengerWriter, "原始排名\t基金代码\t基金\t动量分\t20日\t60日\t120日\t原因")
		for _, item := range pool.Challengers {
			fmt.Fprintf(challengerWriter, "%d\t%s\t%s\t%.1f\t%.2f%%\t%.2f%%\t%.2f%%\t%s\n", item.Rank, item.FundCode, item.FundName, item.MomentumScore, item.Return20D*100, item.Return60D*100, item.Return120D*100, item.Reason)
		}
		_ = challengerWriter.Flush()
	}
	return buf.String()
}

func renderMomentumPoolMarkdown(pool model.MomentumPoolReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# 当前强势基金榜\n\n")
	fmt.Fprintf(&buf, "- 更新时间：`%s`\n", formatDisplayTime(pool.Summary.RunDate))
	fmt.Fprintf(&buf, "- 筛选范围：知名基金公司权益产品；同一公司可多只入榜\n")
	fmt.Fprintf(&buf, "- 榜单说明：展示当前最强的 `%d` 只，前 `%d` 只标记为重点关注\n", len(pool.Opportunities), pool.Summary.SelectedCount)
	fmt.Fprintf(&buf, "- 当前执行：`%s`；前瞻验证只决定是否试投，不影响基金进入榜单\n", momentumReadinessLabel(pool.Summary.Readiness))
	fmt.Fprintf(&buf, "- 试投进度：%s\n", momentumTrialProgress(pool.Summary))
	if len(pool.Opportunities) > 0 {
		buf.WriteString("\n| 排名 | 基金代码 | 基金 | 标记 | 动量分 | 入榜后 | 观察日 | 20日 | 60日 | 120日 | 250日 | 申购状态 | 当前执行 |\n")
		buf.WriteString("| ---: | --- | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- | --- |\n")
		for _, item := range pool.Opportunities {
			fmt.Fprintf(&buf, "| %d | %s | %s | %s | %.1f | %.2f%% | %d/%d | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %s | %s |\n", item.Rank, item.FundCode, item.FundName, item.Source, item.MomentumScore, item.ObservationReturn*100, item.ObservationDays, pool.Summary.ForwardValidationDays, item.Return20D*100, item.Return60D*100, item.Return120D*100, item.Return250D*100, displayPurchaseStatus(item.PurchaseStatus), item.Execution)
		}
	}
	return buf.String()
}

func momentumTrialProgress(summary model.MomentumPoolSummary) string {
	if summary.Readiness == "small-trial" {
		return fmt.Sprintf("已完成真实周期验证，允许总仓位不超过 %.1f%% 的小额试投", summary.SuggestedTotalWeight*100)
	}
	remaining := summary.ForwardValidationDays - summary.OldestPendingTradingDays
	if remaining < 0 {
		remaining = 0
	}
	if summary.PendingValidationBatches > 0 {
		return fmt.Sprintf("已观察约 %d/%d 个交易日，还需约 %d 个；周期收益达到 %.1f%% 后可小额试投", summary.OldestPendingTradingDays, summary.ForwardValidationDays, remaining, summary.TrialReturnThreshold*100)
	}
	return fmt.Sprintf("等待完成 %d 个交易日真实周期；周期收益达到 %.1f%% 后可小额试投", summary.ForwardValidationDays, summary.TrialReturnThreshold*100)
}

func momentumValidationByCode(validations []model.MomentumFundValidation) map[string]model.MomentumFundValidation {
	result := make(map[string]model.MomentumFundValidation, len(validations))
	for _, validation := range validations {
		result[validation.FundCode] = validation
	}
	return result
}

func momentumProductValidationDisplay(validation model.MomentumFundValidation) (string, string) {
	if !validation.Qualified {
		return fmt.Sprintf("%d批", validation.CompletedBatches), "持续跟踪，不买入"
	}
	return fmt.Sprintf("%d批 / 最近 %.2f%% / 均收 %.2f%%", validation.CompletedBatches, validation.LatestReturn*100, validation.AverageReturn*100), validation.Execution
}

func formatPeerCorrelation(item model.MomentumPoolItem) string {
	if item.MaxCorrelatedPeer == "" {
		return "-"
	}
	return fmt.Sprintf("%.2f / %s", item.MaxPeerCorrelation, item.MaxCorrelatedPeer)
}

func formatPurchaseLimit(limit float64) string {
	if limit <= 0 || limit >= 1000000000 {
		return "不限"
	}
	if limit >= 10000 {
		return fmt.Sprintf("%.1f万", limit/10000)
	}
	return fmt.Sprintf("%.0f元", limit)
}

func displayPurchaseStatus(status string) string {
	if strings.TrimSpace(status) == "" {
		return "未知"
	}
	return status
}

func momentumRegimeLabel(regime string) string {
	if regime == "risk-on" {
		return "顺势参与"
	}
	return "现金观察"
}

func momentumReadinessLabel(readiness string) string {
	if readiness == "small-trial" {
		return "允许小额试投"
	}
	return "仅观察，暂不买入"
}

func momentumCandidateHeading(readiness string) string {
	if readiness == "small-trial" {
		return "小额试投候选"
	}
	return "观察候选（暂不买入）"
}

func renderMomentumBacktestTable(result model.MomentumBacktestReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "动量诊断回测：%s 至 %s\n", result.Summary.StartDate.Format("2006-01-02"), result.Summary.EndDate.Format("2006-01-02"))
	fmt.Fprintf(&buf, "策略总收益：%.2f%%，当前候选宇宙等权压力基准：%.2f%%，超额：%.2f%%，策略/基准最大回撤：%.2f%%/%.2f%%\n", result.Summary.TotalReturn*100, result.Summary.BenchmarkTotalReturn*100, result.Summary.ExcessReturn*100, result.Summary.MaxDrawdown*100, result.Summary.BenchmarkMaxDrawdown*100)
	fmt.Fprintf(&buf, "策略年化：%.2f%%，基准年化：%.2f%%，超额周期胜率：%.1f%%，持仓/空仓周期：%d/%d，估算成本：%.2f%%\n", result.Summary.AnnualizedReturn*100, result.Summary.BenchmarkAnnualizedReturn*100, result.Summary.ExcessWinRate*100, result.Summary.InvestedPeriodCount, result.Summary.CashPeriodCount, result.Summary.EstimatedCost*100)
	fmt.Fprintf(&buf, "未来赢家捕获：前10%%命中 %.1f%%，前%d只捕获 %.1f%%，平均未来收益百分位 %.1f%%（%d 个持仓周期）\n", result.Summary.FutureTopDecileHitRate*100, result.Summary.SelectionCount, result.Summary.FutureTopSelectionCapture*100, result.Summary.AverageFuturePercentile*100, result.Summary.WinnerEvaluationPeriods)
	return buf.String()
}

func renderMomentumBacktestMarkdown(result model.MomentumBacktestReport) string {
	var buf bytes.Buffer
	buf.WriteString("# 动量策略诊断回测\n\n")
	fmt.Fprintf(&buf, "- 回测区间：`%s` 至 `%s`\n", result.Summary.StartDate.Format("2006-01-02"), result.Summary.EndDate.Format("2006-01-02"))
	fmt.Fprintf(&buf, "- 候选基金数：`%d`\n", result.Summary.CandidateCount)
	fmt.Fprintf(&buf, "- 候选宇宙样本种子：`%d`\n", result.Summary.UniverseSeed)
	fmt.Fprintf(&buf, "- 每期入选数：`%d`\n", result.Summary.SelectionCount)
	fmt.Fprintf(&buf, "- 调仓间隔：`%d` 个交易日\n", result.Summary.RebalanceEvery)
	fmt.Fprintf(&buf, "- 总收益：`%.2f%%`\n", result.Summary.TotalReturn*100)
	fmt.Fprintf(&buf, "- 年化收益：`%.2f%%`\n", result.Summary.AnnualizedReturn*100)
	fmt.Fprintf(&buf, "- 最大回撤：`%.2f%%`\n", result.Summary.MaxDrawdown*100)
	fmt.Fprintf(&buf, "- 当前候选宇宙等权压力基准总收益：`%.2f%%`\n", result.Summary.BenchmarkTotalReturn*100)
	fmt.Fprintf(&buf, "- 当前候选宇宙等权压力基准年化：`%.2f%%`\n", result.Summary.BenchmarkAnnualizedReturn*100)
	fmt.Fprintf(&buf, "- 当前候选宇宙等权压力基准最大回撤：`%.2f%%`\n", result.Summary.BenchmarkMaxDrawdown*100)
	fmt.Fprintf(&buf, "- 策略相对压力基准超额收益：`%.2f%%`\n", result.Summary.ExcessReturn*100)
	fmt.Fprintf(&buf, "- 超额周期胜率：`%.1f%%`\n", result.Summary.ExcessWinRate*100)
	fmt.Fprintf(&buf, "- 前半段超额收益：`%.2f%%`\n", result.Summary.EarlyExcessReturn*100)
	fmt.Fprintf(&buf, "- 后半段超额收益：`%.2f%%`\n", result.Summary.RecentExcessReturn*100)
	fmt.Fprintf(&buf, "- 持仓周期数：`%d`\n", result.Summary.InvestedPeriodCount)
	fmt.Fprintf(&buf, "- 空仓周期数：`%d`\n", result.Summary.CashPeriodCount)
	fmt.Fprintf(&buf, "- 持仓周期胜率：`%.1f%%`\n", result.Summary.WinRate*100)
	fmt.Fprintf(&buf, "- 平均周期收益：`%.2f%%`\n", result.Summary.AveragePeriodReturn*100)
	fmt.Fprintf(&buf, "- 平均换手：`%.1f%%`\n", result.Summary.AverageTurnover*100)
	fmt.Fprintf(&buf, "- 估算累计交易成本：`%.2f%%`\n", result.Summary.EstimatedCost*100)
	fmt.Fprintf(&buf, "- 下一周期前10%%赢家命中率：`%.1f%%`\n", result.Summary.FutureTopDecileHitRate*100)
	fmt.Fprintf(&buf, "- 下一周期前%d只赢家捕获率：`%.1f%%`\n", result.Summary.SelectionCount, result.Summary.FutureTopSelectionCapture*100)
	fmt.Fprintf(&buf, "- 入选基金下一周期平均收益百分位：`%.1f%%`\n", result.Summary.AverageFuturePercentile*100)
	fmt.Fprintf(&buf, "- 后半段下一周期前10%%赢家命中率：`%.1f%%`\n", result.Summary.RecentTopDecileHitRate*100)
	fmt.Fprintf(&buf, "- 强势榜赢家捕获评价周期：`%d`（后半段 `%d`）\n", result.Summary.WinnerEvaluationPeriods, result.Summary.RecentWinnerPeriods)
	if len(result.Summary.Notes) > 0 {
		buf.WriteString("\n## 限制\n\n")
		for _, note := range result.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}
	return buf.String()
}

func renderMomentumBacktestSensitivityTable(result model.MomentumBacktestSensitivityReport) string {
	var buf bytes.Buffer
	summary := result.Summary
	fmt.Fprintf(&buf, "动量多样本敏感性回测：完成 %d/%d 个样本\n", summary.CompletedSeeds, summary.RequestedSeeds)
	fmt.Fprintf(&buf, "最差总收益：%.2f%%，最差超额：%.2f%%，最差近期超额：%.2f%%，最差最大回撤：%.2f%%\n", summary.WorstTotalReturn*100, summary.WorstExcessReturn*100, summary.WorstRecentExcessReturn*100, summary.WorstMaxDrawdown*100)
	fmt.Fprintf(&buf, "正总收益/正超额/正近期超额样本：%d/%d/%d\n\n", summary.PositiveTotalReturnSeeds, summary.PositiveExcessReturnSeeds, summary.PositiveRecentExcessSeeds)
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "种子\t候选数\t总收益\t超额\t近期超额\t最大回撤\t状态")
	for _, item := range result.Items {
		status := "完成"
		if item.Error != "" {
			status = item.Error
		}
		fmt.Fprintf(tw, "%d\t%d\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%s\n", item.UniverseSeed, item.CandidateCount, item.TotalReturn*100, item.ExcessReturn*100, item.RecentExcessReturn*100, item.MaxDrawdown*100, status)
	}
	_ = tw.Flush()
	return buf.String()
}

func renderMomentumBacktestSensitivityMarkdown(result model.MomentumBacktestSensitivityReport) string {
	var buf bytes.Buffer
	summary := result.Summary
	buf.WriteString("# 动量策略多样本敏感性回测\n\n")
	fmt.Fprintf(&buf, "- 生成时间：`%s`\n", formatDisplayTime(summary.GeneratedAt))
	fmt.Fprintf(&buf, "- 完成样本：`%d/%d`\n", summary.CompletedSeeds, summary.RequestedSeeds)
	fmt.Fprintf(&buf, "- 每期入选数：`%d`\n", summary.SelectionCount)
	fmt.Fprintf(&buf, "- 调仓间隔：`%d` 个交易日\n", summary.RebalanceEvery)
	fmt.Fprintf(&buf, "- 平均总收益：`%.2f%%`\n", summary.AverageTotalReturn*100)
	fmt.Fprintf(&buf, "- 平均超额收益：`%.2f%%`\n", summary.AverageExcessReturn*100)
	fmt.Fprintf(&buf, "- 最差总收益：`%.2f%%`\n", summary.WorstTotalReturn*100)
	fmt.Fprintf(&buf, "- 最差超额收益：`%.2f%%`\n", summary.WorstExcessReturn*100)
	fmt.Fprintf(&buf, "- 最差后半段超额收益：`%.2f%%`\n", summary.WorstRecentExcessReturn*100)
	fmt.Fprintf(&buf, "- 最差最大回撤：`%.2f%%`\n", summary.WorstMaxDrawdown*100)
	fmt.Fprintf(&buf, "- 正总收益样本：`%d/%d`\n", summary.PositiveTotalReturnSeeds, summary.CompletedSeeds)
	fmt.Fprintf(&buf, "- 正超额收益样本：`%d/%d`\n", summary.PositiveExcessReturnSeeds, summary.CompletedSeeds)
	fmt.Fprintf(&buf, "- 正后半段超额样本：`%d/%d`\n", summary.PositiveRecentExcessSeeds, summary.CompletedSeeds)
	buf.WriteString("\n## 样本明细\n\n")
	buf.WriteString("| 种子 | 候选数 | 总收益 | 超额收益 | 后半段超额 | 最大回撤 | 状态 |\n")
	buf.WriteString("| ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	for _, item := range result.Items {
		status := "完成"
		if item.Error != "" {
			status = item.Error
		}
		fmt.Fprintf(&buf, "| %d | %d | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %s |\n", item.UniverseSeed, item.CandidateCount, item.TotalReturn*100, item.ExcessReturn*100, item.RecentExcessReturn*100, item.MaxDrawdown*100, status)
	}
	if len(summary.Notes) > 0 {
		buf.WriteString("\n## 说明\n\n")
		for _, note := range summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}
	return buf.String()
}

func renderMomentumBacktestParameterGridTable(result model.MomentumBacktestParameterGridReport) string {
	var buf bytes.Buffer
	summary := result.Summary
	fmt.Fprintf(&buf, "动量参数稳健性网格：完成 %d 组，每组 %d 个样本\n", summary.CombinationCount, summary.Seeds)
	fmt.Fprintf(&buf, "正式参数：%s，排名：%d；领先参数：%s\n\n", momentumParameterGridLabel(summary.CurrentSelectionCount, summary.CurrentRebalanceEvery), summary.CurrentRank, momentumParameterGridLabel(summary.LeadingSelectionCount, summary.LeadingRebalanceEvery))
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "排名\t参数\t近期前10%命中\t前10%覆盖\t未来百分位\t近期正超额样本\t最差近期超额\t状态")
	for _, item := range result.Items {
		status := "完成"
		if item.Error != "" {
			status = item.Error
		}
		fmt.Fprintf(tw, "%d\t%s\t%.1f%%\t%.1f%%\t%.1f%%\t%d/%d\t%.2f%%\t%s\n", item.Rank, momentumParameterGridLabel(item.SelectionCount, item.RebalanceEvery), item.AverageRecentTopDecileHit*100, item.AverageTopDecileCapture*100, item.AverageFuturePercentile*100, item.PositiveRecentExcessSeeds, item.CompletedSeeds, item.WorstRecentExcessReturn*100, status)
	}
	_ = tw.Flush()
	return buf.String()
}

func renderMomentumBacktestParameterGridMarkdown(result model.MomentumBacktestParameterGridReport) string {
	var buf bytes.Buffer
	summary := result.Summary
	buf.WriteString("# 动量策略参数稳健性网格\n\n")
	fmt.Fprintf(&buf, "- 生成时间：`%s`\n", formatDisplayTime(summary.GeneratedAt))
	fmt.Fprintf(&buf, "- 完成参数组合：`%d`\n", summary.CombinationCount)
	fmt.Fprintf(&buf, "- 每组候选宇宙样本：`%d`\n", summary.Seeds)
	fmt.Fprintf(&buf, "- 正式参数：`%s`，排名 `%d`\n", momentumParameterGridLabel(summary.CurrentSelectionCount, summary.CurrentRebalanceEvery), summary.CurrentRank)
	fmt.Fprintf(&buf, "- 当前领先参数：`%s`\n", momentumParameterGridLabel(summary.LeadingSelectionCount, summary.LeadingRebalanceEvery))
	buf.WriteString("\n## 参数明细\n\n")
	buf.WriteString("| 排名 | 参数 | 近期前10%命中 | 前10%覆盖 | 前K捕获 | 未来百分位 | 近期正超额样本 | 最差近期超额 | 平均总收益 | 状态 |\n")
	buf.WriteString("| ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |\n")
	for _, item := range result.Items {
		status := "完成"
		if item.Error != "" {
			status = item.Error
		}
		fmt.Fprintf(&buf, "| %d | %s | %.1f%% | %.1f%% | %.1f%% | %.1f%% | %d/%d | %.2f%% | %.2f%% | %s |\n", item.Rank, momentumParameterGridLabel(item.SelectionCount, item.RebalanceEvery), item.AverageRecentTopDecileHit*100, item.AverageTopDecileCapture*100, item.AverageTopCaptureRate*100, item.AverageFuturePercentile*100, item.PositiveRecentExcessSeeds, item.CompletedSeeds, item.WorstRecentExcessReturn*100, item.AverageTotalReturn*100, status)
	}
	if len(summary.Notes) > 0 {
		buf.WriteString("\n## 说明\n\n")
		for _, note := range summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}
	return buf.String()
}

func momentumParameterGridLabel(selectionCount, rebalanceEvery int) string {
	return fmt.Sprintf("%d只 / %d日", selectionCount, rebalanceEvery)
}

func renderMomentumBacktestStagesTable(result model.MomentumBacktestStageReport) string {
	var buf bytes.Buffer
	summary := result.Summary
	fmt.Fprintf(&buf, "动量跨阶段稳健性：完整年度 %d/%d，平均超额为正年度 %d，所有样本超额为正年度 %d\n", summary.CompleteStageCount, summary.StageCount, summary.PositiveAverageExcessStages, summary.PositiveAllSeedExcessStages)
	if summary.WorstCompleteStage != "" {
		fmt.Fprintf(&buf, "最弱完整年度：%s，平均超额 %.2f%%，最差样本超额 %.2f%%\n\n", summary.WorstCompleteStage, summary.WorstCompleteAverageExcess*100, summary.WorstCompleteSeedExcess*100)
	}
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "阶段\t完整\t正收益样本\t正超额样本\t平均收益\t平均超额\t平均广度\t60日中位数\t120日中位数\t最差超额")
	for _, item := range result.Items {
		fmt.Fprintf(tw, "%s\t%s\t%d/%d\t%d/%d\t%.2f%%\t%.2f%%\t%.1f%%\t%.2f%%\t%.2f%%\t%.2f%%\n", item.StageLabel, yesNo(item.Complete), item.PositiveReturnSeeds, item.SeedCount, item.PositiveExcessSeeds, item.SeedCount, item.AverageReturn*100, item.AverageExcessReturn*100, item.AveragePositiveBreadth*100, item.AverageMedianReturn60D*100, item.AverageMedianReturn120D*100, item.WorstExcessReturn*100)
	}
	_ = tw.Flush()
	return buf.String()
}

func renderMomentumBacktestStagesMarkdown(result model.MomentumBacktestStageReport) string {
	var buf bytes.Buffer
	summary := result.Summary
	buf.WriteString("# 动量策略跨阶段稳健性\n\n")
	fmt.Fprintf(&buf, "- 生成时间：`%s`\n", formatDisplayTime(summary.GeneratedAt))
	fmt.Fprintf(&buf, "- 策略参数：`%d只 / %d日`\n", summary.SelectionCount, summary.RebalanceEvery)
	fmt.Fprintf(&buf, "- 完整年度：`%d/%d`\n", summary.CompleteStageCount, summary.StageCount)
	fmt.Fprintf(&buf, "- 平均超额为正的完整年度：`%d/%d`\n", summary.PositiveAverageExcessStages, summary.CompleteStageCount)
	fmt.Fprintf(&buf, "- 所有样本超额均为正的完整年度：`%d/%d`\n", summary.PositiveAllSeedExcessStages, summary.CompleteStageCount)
	if summary.WorstCompleteStage != "" {
		fmt.Fprintf(&buf, "- 最弱完整年度：`%s`，平均超额 `%.2f%%`，最差样本超额 `%.2f%%`\n", summary.WorstCompleteStage, summary.WorstCompleteAverageExcess*100, summary.WorstCompleteSeedExcess*100)
	}
	buf.WriteString("\n## 阶段明细\n\n")
	buf.WriteString("| 阶段 | 完整年度 | 正收益样本 | 正超额样本 | 平均收益 | 平均超额 | 平均广度 | 60日中位数 | 120日中位数 | 最差超额 |\n")
	buf.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, item := range result.Items {
		fmt.Fprintf(&buf, "| %s | %s | %d/%d | %d/%d | %.2f%% | %.2f%% | %.1f%% | %.2f%% | %.2f%% | %.2f%% |\n", item.StageLabel, yesNo(item.Complete), item.PositiveReturnSeeds, item.SeedCount, item.PositiveExcessSeeds, item.SeedCount, item.AverageReturn*100, item.AverageExcessReturn*100, item.AveragePositiveBreadth*100, item.AverageMedianReturn60D*100, item.AverageMedianReturn120D*100, item.WorstExcessReturn*100)
	}
	if len(summary.Notes) > 0 {
		buf.WriteString("\n## 说明\n\n")
		for _, note := range summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}
	return buf.String()
}

func renderMomentumBacktestEntryGridTable(result model.MomentumBacktestEntryGridReport) string {
	var buf bytes.Buffer
	summary := result.Summary
	fmt.Fprintf(&buf, "动量入场门槛网格：完成 %d 组，每组 %d 个样本\n", summary.CombinationCount, summary.Seeds)
	fmt.Fprintf(&buf, "正式门槛：广度 %.0f%% / 20日 %.0f%% / 60日 %.0f%%，排名：%d\n", summary.CurrentMinPositiveBreadth*100, summary.CurrentMinReturn20D*100, summary.CurrentMinReturn60D*100, summary.CurrentRank)
	fmt.Fprintf(&buf, "领先门槛：广度 %.0f%% / 20日 %.0f%% / 60日 %.0f%%\n\n", summary.LeadingMinPositiveBreadth*100, summary.LeadingMinReturn20D*100, summary.LeadingMinReturn60D*100)
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "排名\t广度/20日/60日\t近期正超额\t最差近期超额\t最差超额\t最差总收益\t最大回撤")
	for _, item := range result.Items {
		fmt.Fprintf(tw, "%d\t%.0f%%/%.0f%%/%.0f%%\t%d/%d\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\n", item.Rank, item.MinPositiveBreadth*100, item.MinReturn20D*100, item.MinReturn60D*100, item.PositiveRecentExcessSeeds, item.CompletedSeeds, item.WorstRecentExcessReturn*100, item.WorstExcessReturn*100, item.WorstTotalReturn*100, item.WorstMaxDrawdown*100)
	}
	_ = tw.Flush()
	return buf.String()
}

func renderMomentumBacktestEntryGridMarkdown(result model.MomentumBacktestEntryGridReport) string {
	var buf bytes.Buffer
	summary := result.Summary
	buf.WriteString("# 动量策略入场门槛网格\n\n")
	fmt.Fprintf(&buf, "- 正式门槛：`广度 %.0f%% / 20日 %.0f%% / 60日 %.0f%%`，排名 `%d`\n", summary.CurrentMinPositiveBreadth*100, summary.CurrentMinReturn20D*100, summary.CurrentMinReturn60D*100, summary.CurrentRank)
	fmt.Fprintf(&buf, "- 领先门槛：`广度 %.0f%% / 20日 %.0f%% / 60日 %.0f%%`\n", summary.LeadingMinPositiveBreadth*100, summary.LeadingMinReturn20D*100, summary.LeadingMinReturn60D*100)
	buf.WriteString("\n| 排名 | 广度 / 20日 / 60日 | 近期正超额样本 | 最差近期超额 | 最差超额 | 最差总收益 | 最大回撤 |\n")
	buf.WriteString("| ---: | --- | ---: | ---: | ---: | ---: | ---: |\n")
	for _, item := range result.Items {
		fmt.Fprintf(&buf, "| %d | %.0f%% / %.0f%% / %.0f%% | %d/%d | %.2f%% | %.2f%% | %.2f%% | %.2f%% |\n", item.Rank, item.MinPositiveBreadth*100, item.MinReturn20D*100, item.MinReturn60D*100, item.PositiveRecentExcessSeeds, item.CompletedSeeds, item.WorstRecentExcessReturn*100, item.WorstExcessReturn*100, item.WorstTotalReturn*100, item.WorstMaxDrawdown*100)
	}
	return buf.String()
}

func renderMomentumBacktestCorrelationGridTable(result model.MomentumBacktestCorrelationGridReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "动量相关性去重网格：完成 %d 组，每组 %d 个样本\n", result.Summary.CombinationCount, result.Summary.Seeds)
	fmt.Fprintf(&buf, "正式策略不去重，排名：%d；领先阈值：%s\n\n", result.Summary.CurrentRank, correlationThresholdLabel(result.Summary.LeadingMaxCorrelation))
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "排名\t相关性阈值\t近期正超额\t最差近期超额\t最差超额\t最差总收益\t最大回撤\t2024平均/最差超额")
	for _, item := range result.Items {
		fmt.Fprintf(tw, "%d\t%s\t%d/%d\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%\t%.2f%%/%.2f%%\n", item.Rank, correlationThresholdLabel(item.MaxCorrelation), item.PositiveRecentExcessSeeds, item.CompletedSeeds, item.WorstRecentExcessReturn*100, item.WorstExcessReturn*100, item.WorstTotalReturn*100, item.WorstMaxDrawdown*100, item.Average2024ExcessReturn*100, item.Worst2024ExcessReturn*100)
	}
	_ = tw.Flush()
	return buf.String()
}

func renderMomentumBacktestCorrelationGridMarkdown(result model.MomentumBacktestCorrelationGridReport) string {
	var buf bytes.Buffer
	buf.WriteString("# 动量策略相关性去重诊断\n\n")
	fmt.Fprintf(&buf, "- 正式策略：`不自动去重`，排名 `%d`\n", result.Summary.CurrentRank)
	fmt.Fprintf(&buf, "- 领先阈值：`%s`\n", correlationThresholdLabel(result.Summary.LeadingMaxCorrelation))
	fmt.Fprintf(&buf, "- 相关性窗口：最近 `%d` 个共同交易日\n", result.Summary.CorrelationWindowDays)
	buf.WriteString("\n| 排名 | 相关性阈值 | 近期正超额样本 | 最差近期超额 | 最差超额 | 最差总收益 | 最大回撤 | 2024平均超额 | 2024最差超额 |\n")
	buf.WriteString("| ---: | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, item := range result.Items {
		fmt.Fprintf(&buf, "| %d | %s | %d/%d | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% | %.2f%% |\n", item.Rank, correlationThresholdLabel(item.MaxCorrelation), item.PositiveRecentExcessSeeds, item.CompletedSeeds, item.WorstRecentExcessReturn*100, item.WorstExcessReturn*100, item.WorstTotalReturn*100, item.WorstMaxDrawdown*100, item.Average2024ExcessReturn*100, item.Worst2024ExcessReturn*100)
	}
	if len(result.Summary.Notes) > 0 {
		buf.WriteString("\n## 说明\n\n")
		for _, note := range result.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}
	return buf.String()
}

func renderMomentumBacktestWeightGridTable(result model.MomentumBacktestWeightGridReport) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "动量权重赢家捕获网格：完成 %d 组，每组 %d 个样本\n", result.Summary.ProfileCount, result.Summary.Seeds)
	fmt.Fprintf(&buf, "正式权重排名：%d；领先权重：%s\n\n", result.Summary.CurrentRank, result.Summary.LeadingLabel)
	tw := tabwriter.NewWriter(&buf, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "排名\t权重方案\t20/60/120/250日\t近期前10%命中\t最差近期命中\t近期正超额\t最差近期超额\t最差总收益")
	for _, item := range result.Items {
		fmt.Fprintf(tw, "%d\t%s\t%.0f/%.0f/%.0f/%.0f\t%.1f%%\t%.1f%%\t%d/%d\t%.2f%%\t%.2f%%\n", item.Rank, item.Label, item.Weight20D*100, item.Weight60D*100, item.Weight120D*100, item.Weight250D*100, item.AverageRecentTopDecileHitRate*100, item.WorstRecentTopDecileHitRate*100, item.PositiveRecentExcessSeeds, item.CompletedSeeds, item.WorstRecentExcessReturn*100, item.WorstTotalReturn*100)
	}
	_ = tw.Flush()
	return buf.String()
}

func renderMomentumBacktestWeightGridMarkdown(result model.MomentumBacktestWeightGridReport) string {
	var buf bytes.Buffer
	buf.WriteString("# 动量权重未来赢家捕获诊断\n\n")
	fmt.Fprintf(&buf, "- 正式权重排名：`%d`\n", result.Summary.CurrentRank)
	fmt.Fprintf(&buf, "- 领先权重方案：`%s`\n", result.Summary.LeadingLabel)
	buf.WriteString("\n| 排名 | 权重方案 | 20日 / 60日 / 120日 / 250日 | 近期前10%命中 | 最差近期命中 | 全期前10%命中 | 平均未来收益百分位 | 近期正超额样本 | 最差近期超额 | 最差超额 | 最差总收益 | 最大回撤 |\n")
	buf.WriteString("| ---: | --- | --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, item := range result.Items {
		fmt.Fprintf(&buf, "| %d | %s | %.0f%% / %.0f%% / %.0f%% / %.0f%% | %.1f%% | %.1f%% | %.1f%% | %.1f%% | %d/%d | %.2f%% | %.2f%% | %.2f%% | %.2f%% |\n", item.Rank, item.Label, item.Weight20D*100, item.Weight60D*100, item.Weight120D*100, item.Weight250D*100, item.AverageRecentTopDecileHitRate*100, item.WorstRecentTopDecileHitRate*100, item.AverageTopDecileHitRate*100, item.AverageFuturePercentile*100, item.PositiveRecentExcessSeeds, item.CompletedSeeds, item.WorstRecentExcessReturn*100, item.WorstExcessReturn*100, item.WorstTotalReturn*100, item.WorstMaxDrawdown*100)
	}
	if len(result.Summary.Notes) > 0 {
		buf.WriteString("\n## 说明\n\n")
		for _, note := range result.Summary.Notes {
			fmt.Fprintf(&buf, "- %s\n", note)
		}
	}
	return buf.String()
}

func correlationThresholdLabel(value float64) string {
	if value <= 0 {
		return "不去重"
	}
	return fmt.Sprintf("%.2f", value)
}

var chinaTimezone = time.FixedZone("UTC+8", 8*3600)

func formatDisplayTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.In(chinaTimezone).Format(time.RFC3339)
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

func displaySignedMoney(value float64) string {
	if value > 0 {
		return fmt.Sprintf("+%.2f", value)
	}
	return fmt.Sprintf("%.2f", value)
}

func displaySignedPercent(value float64) string {
	if value > 0 {
		return fmt.Sprintf("+%.2f%%", value*100)
	}
	return fmt.Sprintf("%.2f%%", value*100)
}

func reportHasLedger(report model.AnalysisReport) bool {
	for _, state := range report.Position {
		if state.LedgerApplied {
			return true
		}
	}
	return false
}

func displayLedgerTradeInfo(state model.PositionState) string {
	if state.LedgerTradeCount <= 0 || state.LastLedgerTradeAt.IsZero() {
		return "-"
	}
	return fmt.Sprintf("%s / %d笔", state.LastLedgerTradeAt.In(chinaTimezone).Format("2006-01-02"), state.LedgerTradeCount)
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
