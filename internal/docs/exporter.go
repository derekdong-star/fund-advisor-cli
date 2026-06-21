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
	if input.MomentumPool != nil {
		rendered, err := report.RenderMomentumPool(*input.MomentumPool, "markdown")
		if err != nil {
			return nil, err
		}
		latestRel := filepath.Join(latestDirName, "momentum-pool.md")
		latestAbs := latestReportPath(root, "momentum-pool")
		if err := writeLatestDoc(latestAbs, rendered, cfg.Publishing.GitBook.OverwriteLatest); err != nil {
			return nil, err
		}
		result.Latest["momentum-pool"] = ReportArtifact{Label: "当前强势基金榜", Path: latestRel}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			archiveAbs := archiveReportPath(root, input.Analysis.Summary.RunDate, "momentum-pool")
			if err := writeDoc(archiveAbs, rendered); err != nil {
				return nil, err
			}
			result.Archive["momentum-pool"] = ReportArtifact{Label: "归档强势基金榜", Path: relativeDocPath(root, archiveAbs)}
		}
	}
	if cfg.Publishing.GitBook.IncludeMomentumSensitivity && input.MomentumSensitivity != nil {
		rendered, err := report.RenderMomentumBacktestSensitivity(*input.MomentumSensitivity, "markdown")
		if err != nil {
			return nil, err
		}
		latestRel := filepath.Join(latestDirName, "momentum-sensitivity.md")
		latestAbs := latestReportPath(root, "momentum-sensitivity")
		if err := writeLatestDoc(latestAbs, rendered, cfg.Publishing.GitBook.OverwriteLatest); err != nil {
			return nil, err
		}
		result.Latest["momentum-sensitivity"] = ReportArtifact{Label: "动量多样本回测", Path: latestRel}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			archiveAbs := archiveReportPath(root, input.Analysis.Summary.RunDate, "momentum-sensitivity")
			if err := writeDoc(archiveAbs, rendered); err != nil {
				return nil, err
			}
			result.Archive["momentum-sensitivity"] = ReportArtifact{Label: "归档动量多样本回测", Path: relativeDocPath(root, archiveAbs)}
		}
	} else if cfg.Publishing.GitBook.IncludeMomentumSensitivity && input.MomentumSensitivityError != "" {
		if err := removeDocIfExists(latestReportPath(root, "momentum-sensitivity")); err != nil {
			return nil, err
		}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			if err := removeDocIfExists(archiveReportPath(root, input.Analysis.Summary.RunDate, "momentum-sensitivity")); err != nil {
				return nil, err
			}
		}
	}
	if cfg.Publishing.GitBook.IncludeMomentumParameterGrid && input.MomentumParameterGrid != nil {
		rendered, err := report.RenderMomentumBacktestParameterGrid(*input.MomentumParameterGrid, "markdown")
		if err != nil {
			return nil, err
		}
		latestRel := filepath.Join(latestDirName, "momentum-parameter-grid.md")
		latestAbs := latestReportPath(root, "momentum-parameter-grid")
		if err := writeLatestDoc(latestAbs, rendered, cfg.Publishing.GitBook.OverwriteLatest); err != nil {
			return nil, err
		}
		result.Latest["momentum-parameter-grid"] = ReportArtifact{Label: "动量参数稳健性", Path: latestRel}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			archiveAbs := archiveReportPath(root, input.Analysis.Summary.RunDate, "momentum-parameter-grid")
			if err := writeDoc(archiveAbs, rendered); err != nil {
				return nil, err
			}
			result.Archive["momentum-parameter-grid"] = ReportArtifact{Label: "归档动量参数稳健性", Path: relativeDocPath(root, archiveAbs)}
		}
	} else if cfg.Publishing.GitBook.IncludeMomentumParameterGrid && input.MomentumParameterGridError != "" {
		if err := removeDocIfExists(latestReportPath(root, "momentum-parameter-grid")); err != nil {
			return nil, err
		}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			if err := removeDocIfExists(archiveReportPath(root, input.Analysis.Summary.RunDate, "momentum-parameter-grid")); err != nil {
				return nil, err
			}
		}
	}
	if cfg.Publishing.GitBook.IncludeMomentumStages && input.MomentumStages != nil {
		rendered, err := report.RenderMomentumBacktestStages(*input.MomentumStages, "markdown")
		if err != nil {
			return nil, err
		}
		latestRel := filepath.Join(latestDirName, "momentum-stages.md")
		latestAbs := latestReportPath(root, "momentum-stages")
		if err := writeLatestDoc(latestAbs, rendered, cfg.Publishing.GitBook.OverwriteLatest); err != nil {
			return nil, err
		}
		result.Latest["momentum-stages"] = ReportArtifact{Label: "动量跨阶段稳健性", Path: latestRel}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			archiveAbs := archiveReportPath(root, input.Analysis.Summary.RunDate, "momentum-stages")
			if err := writeDoc(archiveAbs, rendered); err != nil {
				return nil, err
			}
			result.Archive["momentum-stages"] = ReportArtifact{Label: "归档动量跨阶段稳健性", Path: relativeDocPath(root, archiveAbs)}
		}
	} else if cfg.Publishing.GitBook.IncludeMomentumStages && input.MomentumStagesError != "" {
		if err := removeDocIfExists(latestReportPath(root, "momentum-stages")); err != nil {
			return nil, err
		}
		if cfg.Publishing.GitBook.ArchiveByRunDate {
			if err := removeDocIfExists(archiveReportPath(root, input.Analysis.Summary.RunDate, "momentum-stages")); err != nil {
				return nil, err
			}
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
