package docs

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

func renderGitBookConfig() string {
	return "root: ./\n\nstructure:\n  readme: README.md\n  summary: SUMMARY.md\n"
}

func renderHomepage(cfg *config.Config, input PublishInput, result *ExportResult) string {
	analysis := input.Analysis
	lines := []string{
		fmt.Sprintf("# %s", cfg.Publishing.GitBook.SiteTitle),
		"",
	}
	if desc := strings.TrimSpace(cfg.Publishing.GitBook.SiteDescription); desc != "" {
		lines = append(lines, desc, "")
	}
	lines = append(lines,
		"## 如何阅读",
		"",
		"- 先看最新日报，了解当前组合动作。",
		"- 再看月度定投计划，安排新增资金。",
		"- 稳定候选池用于补充观察名单，不用于高频切换。",
	)
	if _, ok := result.Latest["backtest"]; ok {
		lines = append(lines, "- 回测页只用于验证规则稳定性，不代表未来收益。")
	} else if strings.TrimSpace(input.BacktestError) != "" {
		lines = append(lines, "- 本次未展示回测页，因为历史数据暂不可用。")
	}

	lines = append(lines,
		"",
		"## 总览看板",
		"",
		"| 日报 | 定投 | 候选池 |",
		"| --- | --- | --- |",
		fmt.Sprintf("| %s | %s | %s |", sanitizeMarkdownCell(renderDailyDashboardSummary(analysis)), sanitizeMarkdownCell(renderDCADashboardSummary(input.Plan)), sanitizeMarkdownCell(renderMarketPoolDashboardSummary(input))),
		fmt.Sprintf("| %s | %s | %s |", dashboardCellLink(result, "daily", "日报暂不可用"), dashboardCellLink(result, "dca-plan", "定投计划暂不可用"), dashboardCellLink(result, "market-pool", marketPoolLinkFallback(input))),
		fmt.Sprintf("| %s | %s | %s |", sanitizeMarkdownCell(renderDailyDashboardHighlight(analysis)), sanitizeMarkdownCell(renderDCADashboardHighlight(input.Plan)), sanitizeMarkdownCell(renderMarketPoolDashboardHighlight(input))),
	)

	lines = append(lines,
		"",
		"## 最新快照",
		"",
		fmt.Sprintf("- 运行日期：`%s`", analysis.Summary.RunDate.Format("2006-01-02")),
		fmt.Sprintf("- 组合市值：`%.2f`", analysis.Summary.PortfolioValue),
		fmt.Sprintf("- 当日加权涨跌：`%.2f%%`", analysis.Summary.WeightedDayChangePct*100),
	)
	if input.Plan != nil {
		lines = append(lines, fmt.Sprintf("- 本月计划定投：`%.0f`", input.Plan.Summary.PlannedAmount))
	}
	if input.MarketPool != nil {
		lines = append(lines,
			fmt.Sprintf("- 稳定候选数：`%d`", input.MarketPool.Summary.SelectedCount),
			fmt.Sprintf("- 保留候选数：`%d`", input.MarketPool.Summary.RetainedCount),
		)
	} else if strings.TrimSpace(input.MarketPoolError) != "" {
		lines = append(lines, fmt.Sprintf("- 稳定候选池：暂不可用（`%s`）", input.MarketPoolError))
	}

	if input.MarketPool != nil && len(input.MarketPool.Items) > 0 {
		lines = append(lines, "", "## 主题快照", "")
		for _, item := range input.MarketPool.Items {
			lines = append(lines, fmt.Sprintf("- `%s`：%s（评分 %d，250日 %.2f%%，保留 %s）", item.ThemeLabel, item.FundName, item.Score, item.Return250D*100, yesNo(item.Retained)))
		}
	}

	lines = append(lines, "", "## 快速入口", "")
	hasLatest := false
	for _, key := range []string{"daily", "dca-plan", "market-pool", "backtest"} {
		if item, ok := result.Latest[key]; ok {
			lines = append(lines, fmt.Sprintf("- [%s](%s)", item.Label, filepath.ToSlash(item.Path)))
			hasLatest = true
		}
	}
	if !hasLatest {
		lines = append(lines, "- 最新页面暂未生成。")
	}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		lines = append(lines, "", "## 历史归档", "")
		lines = append(lines, fmt.Sprintf("- [浏览归档](%s)", filepath.ToSlash(filepath.Join(archiveDirName, readmeName))))
		if dayIndex := latestArchiveIndexPath(result); dayIndex != "" {
			lines = append(lines, fmt.Sprintf("- [最新归档目录](%s)", filepath.ToSlash(dayIndex)))
		}
		for _, key := range []string{"daily", "market-pool", "dca-plan", "backtest"} {
			if item, ok := result.Archive[key]; ok {
				lines = append(lines, fmt.Sprintf("- [最新归档%s](%s)", archiveLabel(key), filepath.ToSlash(item.Path)))
			}
		}
	}
	lines = append(lines,
		"",
		"## 方法说明",
		"",
		fmt.Sprintf("- [策略说明](%s)", filepath.ToSlash(cfg.Publishing.GitBook.StrategyOverviewPath)),
		fmt.Sprintf("- [风险提示](%s)", filepath.ToSlash(cfg.Publishing.GitBook.RiskDisclosurePath)),
		"",
		"## 发布说明",
		"",
		"- 页面由组合规则引擎自动生成。",
		"- 市场数据和基金净值可能存在延迟或缺失。",
		"- 内容仅用于跟踪与复盘，不构成投资建议。",
	)
	return strings.Join(lines, "\n") + "\n"
}

func renderSummary(cfg *config.Config, result *ExportResult) string {
	lines := []string{"# 目录", "", "- [首页](README.md)"}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		lines = append(lines, fmt.Sprintf("- [历史归档](%s)", filepath.ToSlash(filepath.Join(archiveDirName, readmeName))))
	}
	for _, key := range []string{"daily", "dca-plan", "market-pool", "backtest"} {
		if item, ok := result.Latest[key]; ok {
			lines = append(lines, fmt.Sprintf("- [%s](%s)", item.Label, filepath.ToSlash(item.Path)))
		}
	}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		if dayIndex := latestArchiveIndexPath(result); dayIndex != "" {
			lines = append(lines, fmt.Sprintf("- [最新归档](%s)", filepath.ToSlash(dayIndex)))
		}
	}
	lines = append(lines,
		fmt.Sprintf("- [策略说明](%s)", filepath.ToSlash(cfg.Publishing.GitBook.StrategyOverviewPath)),
		fmt.Sprintf("- [风险提示](%s)", filepath.ToSlash(cfg.Publishing.GitBook.RiskDisclosurePath)),
	)
	return strings.Join(lines, "\n") + "\n"
}

func latestArchiveIndexPath(result *ExportResult) string {
	for _, key := range []string{"daily", "dca-plan", "market-pool", "backtest"} {
		if item, ok := result.Archive[key]; ok {
			return filepath.ToSlash(filepath.Join(filepath.Dir(item.Path), readmeName))
		}
	}
	return ""
}

func renderStrategyOverview() string {
	return strings.Join([]string{
		"# 策略说明",
		"",
		"- 低换手：避免频繁调仓和日常性减仓。",
		"- 长期持有：对高确信度持仓，在正常波动中保持耐心。",
		"- 定投优先：新增资金优先通过受约束的月度定投计划投入。",
		"- 重点持仓：`000979 景顺长城沪港深精选股票A` 继续视为长期高确信度仓位。",
	}, "\n") + "\n"
}

func renderRiskDisclosure() string {
	return strings.Join([]string{
		"# 风险提示",
		"",
		"- 内容仅用于流程跟踪和组合复盘。",
		"- 不构成投资建议，也不保证收益。",
		"- 市场数据和基金净值可能存在延迟或缺失。",
		"- 规则模型可能出错，执行前仍需人工复核。",
	}, "\n") + "\n"
}

func yesNo(value bool) string {
	if value {
		return "是"
	}
	return "否"
}

func dashboardCellLink(result *ExportResult, key, fallback string) string {
	if item, ok := result.Latest[key]; ok {
		return fmt.Sprintf("[%s](%s)", item.Label, filepath.ToSlash(item.Path))
	}
	return fallback
}

func marketPoolLinkFallback(input PublishInput) string {
	if strings.TrimSpace(input.MarketPoolError) != "" {
		return "候选池刷新失败"
	}
	return "候选池暂不可用"
}

func renderDailyDashboardSummary(analysis *model.AnalysisReport) string {
	if analysis == nil {
		return "暂无数据"
	}
	holdCount := analysis.Summary.ActionCounts[model.ActionHold]
	pauseCount := analysis.Summary.ActionCounts[model.ActionPauseBuy]
	adjustCount := 0
	for action, count := range analysis.Summary.ActionCounts {
		if action == model.ActionHold || action == model.ActionPauseBuy {
			continue
		}
		adjustCount += count
	}
	return fmt.Sprintf("持有 %d / 暂停 %d / 调整 %d", holdCount, pauseCount, adjustCount)
}

func renderDailyDashboardHighlight(analysis *model.AnalysisReport) string {
	if analysis == nil {
		return "暂无组合快照"
	}
	if len(analysis.Recommendations) > 0 {
		return fmt.Sprintf("%d 条建议，覆盖 %d 个信号", len(analysis.Recommendations), len(analysis.Signals))
	}
	return fmt.Sprintf("跟踪 %d 只基金，暂无主动交易建议", len(analysis.Signals))
}

func renderDCADashboardSummary(plan *model.DCAPlanReport) string {
	if plan == nil {
		return "暂无数据"
	}
	return fmt.Sprintf("计划 %.0f 元 / %d 只基金", plan.Summary.PlannedAmount, plan.Summary.SelectedFundCount)
}

func renderDCADashboardHighlight(plan *model.DCAPlanReport) string {
	if plan == nil {
		return "暂无定投快照"
	}
	if len(plan.Items) == 0 {
		return fmt.Sprintf("保留 %.0f 元，暂不分配", plan.Summary.ReserveAmount)
	}
	item := plan.Items[0]
	return fmt.Sprintf("优先定投：%s %.0f 元", item.FundName, item.PlannedAmount)
}

func renderMarketPoolDashboardSummary(input PublishInput) string {
	if input.MarketPool != nil {
		return fmt.Sprintf("%d 只候选 / 保留 %d 只", input.MarketPool.Summary.SelectedCount, input.MarketPool.Summary.RetainedCount)
	}
	if strings.TrimSpace(input.MarketPoolError) != "" {
		return "不可用"
	}
	return "暂无数据"
}

func renderMarketPoolDashboardHighlight(input PublishInput) string {
	if input.MarketPool == nil || len(input.MarketPool.Items) == 0 {
		if strings.TrimSpace(input.MarketPoolError) != "" {
			return "候选池刷新失败"
		}
		return "暂无稳定候选快照"
	}
	highlights := make([]string, 0, minInt(3, len(input.MarketPool.Items)))
	for _, item := range input.MarketPool.Items[:minInt(3, len(input.MarketPool.Items))] {
		highlights = append(highlights, fmt.Sprintf("%s：%s", item.ThemeLabel, item.FundName))
	}
	return strings.Join(highlights, "；")
}

func archiveLabel(key string) string {
	switch key {
	case "daily":
		return "日报"
	case "dca-plan":
		return "定投计划"
	case "market-pool":
		return "稳定候选池"
	case "backtest":
		return "策略回测"
	default:
		return key
	}
}

func sanitizeMarkdownCell(value string) string {
	value = strings.ReplaceAll(value, "|", "/")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
