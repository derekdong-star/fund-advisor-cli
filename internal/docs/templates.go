package docs

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
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
		"## Latest Summary",
		"",
		fmt.Sprintf("- Latest Run Date: `%s`", analysis.Summary.RunDate.Format("2006-01-02")),
		fmt.Sprintf("- Portfolio Value: `%.2f`", analysis.Summary.PortfolioValue),
		fmt.Sprintf("- Weighted Day Change: `%.2f%%`", analysis.Summary.WeightedDayChangePct*100),
	)
	if input.Plan != nil {
		lines = append(lines, fmt.Sprintf("- Monthly DCA Planned: `%.0f`", input.Plan.Summary.PlannedAmount))
	}
	lines = append(lines, "", "## Latest Reports", "")
	for _, key := range []string{"daily", "dca-plan", "backtest"} {
		if item, ok := result.Latest[key]; ok {
			lines = append(lines, fmt.Sprintf("- [%s](%s)", item.Label, filepath.ToSlash(item.Path)))
		}
	}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		lines = append(lines, "", "## Archive", "")
		lines = append(lines, fmt.Sprintf("- [Browse Archive](%s)", filepath.ToSlash(filepath.Join(archiveDirName, readmeName))))
		if item, ok := result.Archive["daily"]; ok {
			lines = append(lines, fmt.Sprintf("- [Latest Archived Daily](%s)", filepath.ToSlash(item.Path)))
		}
		if dayIndex := latestArchiveIndexPath(result); dayIndex != "" {
			lines = append(lines, fmt.Sprintf("- [Latest Snapshot Folder](%s)", filepath.ToSlash(dayIndex)))
		}
	}
	lines = append(lines,
		"",
		"## Strategy",
		"",
		fmt.Sprintf("- [Overview](%s)", filepath.ToSlash(cfg.Publishing.GitBook.StrategyOverviewPath)),
		fmt.Sprintf("- [Risk Disclosure](%s)", filepath.ToSlash(cfg.Publishing.GitBook.RiskDisclosurePath)),
	)
	return strings.Join(lines, "\n") + "\n"
}

func renderSummary(cfg *config.Config, result *ExportResult) string {
	lines := []string{"# Summary", "", "- [Home](README.md)"}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		lines = append(lines, fmt.Sprintf("- [Archive](%s)", filepath.ToSlash(filepath.Join(archiveDirName, readmeName))))
	}
	for _, key := range []string{"daily", "dca-plan", "backtest"} {
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
	for _, key := range []string{"daily", "dca-plan", "backtest"} {
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
