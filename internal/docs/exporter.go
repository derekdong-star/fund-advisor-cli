package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/report"
)

func exportReports(cfg *config.Config, input PublishInput) (*ExportResult, error) {
	root := docsRoot(cfg)
	result := &ExportResult{
		GeneratedAt: time.Now().UTC(),
		DocsRoot:    root,
		Latest:      make(map[string]ReportArtifact),
		Archive:     make(map[string]ReportArtifact),
	}
	if input.Analysis == nil {
		return nil, fmt.Errorf("analysis report is required")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	if cfg.Publishing.GitBook.IncludeDaily {
		rendered, err := report.Render(*input.Analysis, "markdown")
		if err != nil {
			return nil, err
		}
		latestRel := filepath.Join(latestDirName, "daily.md")
		latestAbs := latestReportPath(root, "daily")
		if err := writeLatestDoc(latestAbs, rendered, cfg.Publishing.GitBook.OverwriteLatest); err != nil {
			return nil, err
		}
		result.Latest["daily"] = ReportArtifact{Label: "每日报告", Path: latestRel}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			archiveAbs := archiveReportPath(root, input.Analysis.Summary.RunDate, "daily")
			if err := writeDoc(archiveAbs, rendered); err != nil {
				return nil, err
			}
			result.Archive["daily"] = ReportArtifact{Label: "归档每日报告", Path: relativeDocPath(root, archiveAbs)}
		}
	}
	if cfg.Publishing.GitBook.IncludeDCAPlan && input.Plan != nil {
		rendered, err := report.RenderDCAPlan(*input.Plan, "markdown")
		if err != nil {
			return nil, err
		}
		latestRel := filepath.Join(latestDirName, "dca-plan.md")
		latestAbs := latestReportPath(root, "dca-plan")
		if err := writeLatestDoc(latestAbs, rendered, cfg.Publishing.GitBook.OverwriteLatest); err != nil {
			return nil, err
		}
		result.Latest["dca-plan"] = ReportArtifact{Label: "月度定投计划", Path: latestRel}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			archiveAbs := archiveReportPath(root, input.Analysis.Summary.RunDate, "dca-plan")
			if err := writeDoc(archiveAbs, rendered); err != nil {
				return nil, err
			}
			result.Archive["dca-plan"] = ReportArtifact{Label: "归档定投计划", Path: relativeDocPath(root, archiveAbs)}
		}
	}
	if input.MarketPool != nil {
		rendered, err := report.RenderMarketPool(*input.MarketPool, "markdown")
		if err != nil {
			return nil, err
		}
		latestRel := filepath.Join(latestDirName, "market-pool.md")
		latestAbs := latestReportPath(root, "market-pool")
		if err := writeLatestDoc(latestAbs, rendered, cfg.Publishing.GitBook.OverwriteLatest); err != nil {
			return nil, err
		}
		result.Latest["market-pool"] = ReportArtifact{Label: "稳定候选池", Path: latestRel}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			archiveAbs := archiveReportPath(root, input.Analysis.Summary.RunDate, "market-pool")
			if err := writeDoc(archiveAbs, rendered); err != nil {
				return nil, err
			}
			result.Archive["market-pool"] = ReportArtifact{Label: "归档候选池", Path: relativeDocPath(root, archiveAbs)}
		}
	}
	if cfg.Publishing.GitBook.IncludeBacktest {
		backtestUnavailable := input.Backtest == nil && input.BacktestError != ""
		if cfg.Publishing.GitBook.HideBacktestWhenUnavailable && backtestUnavailable {
			if err := removeDocIfExists(latestReportPath(root, "backtest")); err != nil {
				return nil, err
			}
			if cfg.Publishing.GitBook.ArchiveByRunDate {
				if err := removeDocIfExists(archiveReportPath(root, input.Analysis.Summary.RunDate, "backtest")); err != nil {
					return nil, err
				}
			}
		} else {
			rendered, err := renderBacktestPage(input)
			if err != nil {
				return nil, err
			}
			latestRel := filepath.Join(latestDirName, "backtest.md")
			latestAbs := latestReportPath(root, "backtest")
			if err := writeLatestDoc(latestAbs, rendered, cfg.Publishing.GitBook.OverwriteLatest); err != nil {
				return nil, err
			}
			result.Latest["backtest"] = ReportArtifact{Label: "策略回测", Path: latestRel}
			if cfg.Publishing.GitBook.ArchiveByRunDate {
				archiveAbs := archiveReportPath(root, input.Analysis.Summary.RunDate, "backtest")
				if err := writeDoc(archiveAbs, rendered); err != nil {
					return nil, err
				}
				result.Archive["backtest"] = ReportArtifact{Label: "归档策略回测", Path: relativeDocPath(root, archiveAbs)}
			}
		}
	}
	if err := writeState(root, result, input.Analysis.Summary.RunDate); err != nil {
		return nil, err
	}
	return result, nil
}

func renderBacktestPage(input PublishInput) (string, error) {
	if input.Backtest != nil {
		return report.RenderBacktest(*input.Backtest, "markdown")
	}
	reason := input.BacktestError
	if reason == "" {
		reason = "历史回测数据暂不可用。"
	}
	return fmt.Sprintf("# 策略回测\n\n- 状态：不可用\n- 说明：%s\n", reason), nil
}

func writeState(root string, result *ExportResult, runDate time.Time) error {
	state := PublishState{
		LastRunDate: runDate.Format("2006-01-02"),
		Latest:      result.Latest,
		Archive:     result.Archive,
	}
	buf, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	return writeDoc(statePath(root), string(buf)+"\n")
}

func writeLatestDoc(path, content string, overwrite bool) error {
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
	}
	return writeDoc(path, content)
}

func writeDoc(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func removeDocIfExists(path string) error {
	err := os.Remove(path)
	if err == nil || os.IsNotExist(err) {
		return nil
	}
	return err
}

func relativeDocPath(root, absolute string) string {
	rel, err := filepath.Rel(root, absolute)
	if err != nil {
		return absolute
	}
	return rel
}
