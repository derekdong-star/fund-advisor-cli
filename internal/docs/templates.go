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
		"## How To Read",
		"",
		"- Start with the latest daily playbook for current portfolio actions.",
		"- Use the monthly DCA plan for fresh capital allocation.",
		"- Use the stable market pool as a low-churn candidate list, not a daily trading list.",
	)
	if _, ok := result.Latest["backtest"]; ok {
		lines = append(lines, "- Review the strategy backtest as a validation page, not a prediction tool.")
	} else if strings.TrimSpace(input.BacktestError) != "" {
		lines = append(lines, "- Strategy backtest is hidden for this run because historical data is unavailable.")
	}

	lines = append(lines,
		"",
		"## Dashboard",
		"",
		"| Daily | DCA | Market Pool |",
		"| --- | --- | --- |",
		fmt.Sprintf("| %s | %s | %s |", sanitizeMarkdownCell(renderDailyDashboardSummary(analysis)), sanitizeMarkdownCell(renderDCADashboardSummary(input.Plan)), sanitizeMarkdownCell(renderMarketPoolDashboardSummary(input))),
		fmt.Sprintf("| %s | %s | %s |", dashboardCellLink(result, "daily", "Daily not available"), dashboardCellLink(result, "dca-plan", "DCA not available"), dashboardCellLink(result, "market-pool", marketPoolLinkFallback(input))),
		fmt.Sprintf("| %s | %s | %s |", sanitizeMarkdownCell(renderDailyDashboardHighlight(analysis)), sanitizeMarkdownCell(renderDCADashboardHighlight(input.Plan)), sanitizeMarkdownCell(renderMarketPoolDashboardHighlight(input))),
	)

	lines = append(lines,
		"",
		"## Latest Snapshot",
		"",
		fmt.Sprintf("- Run Date: `%s`", analysis.Summary.RunDate.Format("2006-01-02")),
		fmt.Sprintf("- Portfolio Value: `%.2f`", analysis.Summary.PortfolioValue),
		fmt.Sprintf("- Weighted Day Change: `%.2f%%`", analysis.Summary.WeightedDayChangePct*100),
	)
	if input.Plan != nil {
		lines = append(lines, fmt.Sprintf("- Monthly DCA Planned: `%.0f`", input.Plan.Summary.PlannedAmount))
	}
	if input.MarketPool != nil {
		lines = append(lines,
			fmt.Sprintf("- Stable Candidate Count: `%d`", input.MarketPool.Summary.SelectedCount),
			fmt.Sprintf("- Retained Candidates: `%d`", input.MarketPool.Summary.RetainedCount),
		)
	} else if strings.TrimSpace(input.MarketPoolError) != "" {
		lines = append(lines, fmt.Sprintf("- Stable Market Pool: unavailable (`%s`)", input.MarketPoolError))
	}

	if input.MarketPool != nil && len(input.MarketPool.Items) > 0 {
		lines = append(lines, "", "## Theme Snapshot", "")
		for _, item := range input.MarketPool.Items {
			lines = append(lines, fmt.Sprintf("- `%s`: %s (score %d, 250D %.2f%%, retained %s)", item.ThemeLabel, item.FundName, item.Score, item.Return250D*100, yesNo(item.Retained)))
		}
	}

	lines = append(lines, "", "## Quick Links", "")
	hasLatest := false
	for _, key := range []string{"daily", "dca-plan", "market-pool", "backtest"} {
		if item, ok := result.Latest[key]; ok {
			lines = append(lines, fmt.Sprintf("- [%s](%s)", item.Label, filepath.ToSlash(item.Path)))
			hasLatest = true
		}
	}
	if !hasLatest {
		lines = append(lines, "- Latest report pages are not available yet.")
	}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		lines = append(lines, "", "## Archive", "")
		lines = append(lines, fmt.Sprintf("- [Browse Archive](%s)", filepath.ToSlash(filepath.Join(archiveDirName, readmeName))))
		if dayIndex := latestArchiveIndexPath(result); dayIndex != "" {
			lines = append(lines, fmt.Sprintf("- [Latest Snapshot Folder](%s)", filepath.ToSlash(dayIndex)))
		}
		for _, key := range []string{"daily", "market-pool", "dca-plan", "backtest"} {
			if item, ok := result.Archive[key]; ok {
				lines = append(lines, fmt.Sprintf("- [Latest Archived %s](%s)", archiveLabel(key), filepath.ToSlash(item.Path)))
			}
		}
	}
	lines = append(lines,
		"",
		"## Methodology",
		"",
		fmt.Sprintf("- [Strategy Overview](%s)", filepath.ToSlash(cfg.Publishing.GitBook.StrategyOverviewPath)),
		fmt.Sprintf("- [Risk Disclosure](%s)", filepath.ToSlash(cfg.Publishing.GitBook.RiskDisclosurePath)),
		"",
		"## Publishing Note",
		"",
		"- Generated automatically from the portfolio rule engine.",
		"- Market data and fund NAV updates may be delayed or incomplete.",
		"- This material is for tracking and review only, not investment advice.",
	)
	return strings.Join(lines, "\n") + "\n"
}

func renderSummary(cfg *config.Config, result *ExportResult) string {
	lines := []string{"# Summary", "", "- [Home](README.md)"}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		lines = append(lines, fmt.Sprintf("- [Archive](%s)", filepath.ToSlash(filepath.Join(archiveDirName, readmeName))))
	}
	for _, key := range []string{"daily", "dca-plan", "market-pool", "backtest"} {
		if item, ok := result.Latest[key]; ok {
			lines = append(lines, fmt.Sprintf("- [%s](%s)", item.Label, filepath.ToSlash(item.Path)))
		}
	}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		if dayIndex := latestArchiveIndexPath(result); dayIndex != "" {
			lines = append(lines, fmt.Sprintf("- [Latest Snapshot](%s)", filepath.ToSlash(dayIndex)))
		}
	}
	lines = append(lines,
		fmt.Sprintf("- [Strategy Overview](%s)", filepath.ToSlash(cfg.Publishing.GitBook.StrategyOverviewPath)),
		fmt.Sprintf("- [Risk Disclosure](%s)", filepath.ToSlash(cfg.Publishing.GitBook.RiskDisclosurePath)),
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
		"# Strategy Overview",
		"",
		"- Low turnover: avoid frequent swaps and routine trimming.",
		"- Long-term holding: keep conviction holdings through normal noise.",
		"- DCA-first: new capital is allocated through a constrained monthly DCA plan.",
		"- Protected holding: `000979 景顺长城沪港深精选股票A` remains a long-term conviction position.",
	}, "\n") + "\n"
}

func renderRiskDisclosure() string {
	return strings.Join([]string{
		"# Risk Disclosure",
		"",
		"- This material is for process tracking and portfolio review only.",
		"- It is not investment advice and does not guarantee returns.",
		"- Market data and fund NAV updates may be delayed or incomplete.",
		"- Rule-based outputs can be wrong and should be reviewed before execution.",
	}, "\n") + "\n"
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

func dashboardCellLink(result *ExportResult, key, fallback string) string {
	if item, ok := result.Latest[key]; ok {
		return fmt.Sprintf("[%s](%s)", item.Label, filepath.ToSlash(item.Path))
	}
	return fallback
}

func marketPoolLinkFallback(input PublishInput) string {
	if strings.TrimSpace(input.MarketPoolError) != "" {
		return "Market pool unavailable"
	}
	return "Market pool not available"
}

func renderDailyDashboardSummary(analysis *model.AnalysisReport) string {
	if analysis == nil {
		return "Not available"
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
	return fmt.Sprintf("Hold %d / Pause %d / Adjust %d", holdCount, pauseCount, adjustCount)
}

func renderDailyDashboardHighlight(analysis *model.AnalysisReport) string {
	if analysis == nil {
		return "No portfolio snapshot"
	}
	if len(analysis.Recommendations) > 0 {
		return fmt.Sprintf("%d recommendations across %d signals", len(analysis.Recommendations), len(analysis.Signals))
	}
	return fmt.Sprintf("%d tracked funds with no active trade recommendation", len(analysis.Signals))
}

func renderDCADashboardSummary(plan *model.DCAPlanReport) string {
	if plan == nil {
		return "Not available"
	}
	return fmt.Sprintf("Planned %.0f / %d funds", plan.Summary.PlannedAmount, plan.Summary.SelectedFundCount)
}

func renderDCADashboardHighlight(plan *model.DCAPlanReport) string {
	if plan == nil {
		return "No DCA snapshot"
	}
	if len(plan.Items) == 0 {
		return fmt.Sprintf("Reserve %.0f with no active allocation", plan.Summary.ReserveAmount)
	}
	item := plan.Items[0]
	return fmt.Sprintf("Top allocation: %s %.0f", item.FundName, item.PlannedAmount)
}

func renderMarketPoolDashboardSummary(input PublishInput) string {
	if input.MarketPool != nil {
		return fmt.Sprintf("%d candidates / retained %d", input.MarketPool.Summary.SelectedCount, input.MarketPool.Summary.RetainedCount)
	}
	if strings.TrimSpace(input.MarketPoolError) != "" {
		return "Unavailable"
	}
	return "Not available"
}

func renderMarketPoolDashboardHighlight(input PublishInput) string {
	if input.MarketPool == nil || len(input.MarketPool.Items) == 0 {
		if strings.TrimSpace(input.MarketPoolError) != "" {
			return "Market pool refresh failed"
		}
		return "No stable candidate snapshot"
	}
	highlights := make([]string, 0, minInt(3, len(input.MarketPool.Items)))
	for _, item := range input.MarketPool.Items[:minInt(3, len(input.MarketPool.Items))] {
		highlights = append(highlights, fmt.Sprintf("%s: %s", item.ThemeLabel, item.FundName))
	}
	return strings.Join(highlights, " ; ")
}

func archiveLabel(key string) string {
	switch key {
	case "daily":
		return "Daily"
	case "dca-plan":
		return "DCA Plan"
	case "market-pool":
		return "Market Pool"
	case "backtest":
		return "Backtest"
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
