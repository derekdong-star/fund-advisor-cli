package docs

import (
	"path/filepath"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
)

func buildIndex(cfg *config.Config, input PublishInput, result *ExportResult) error {
	root := docsRoot(cfg)
	if err := writeDoc(gitbookConfigPath(root), renderGitBookConfig()); err != nil {
		return err
	}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		if err := buildArchiveIndexes(root); err != nil {
			return err
		}
	} else if err := removeDocIfExists(filepath.Join(root, archiveDirName, readmeName)); err != nil {
		return err
	}
	if cfg.Publishing.GitBook.GenerateHomepage {
		if err := writeDoc(readmePath(root), renderHomepage(cfg, input, result)); err != nil {
			return err
		}
	} else if err := removeDocIfExists(readmePath(root)); err != nil {
		return err
	}
	if cfg.Publishing.GitBook.GenerateSummary {
		if err := writeDoc(summaryPath(root), renderSummary(cfg, result)); err != nil {
			return err
		}
	} else if err := removeDocIfExists(summaryPath(root)); err != nil {
		return err
	}
	if err := writeDoc(filepath.Join(root, filepath.FromSlash(cfg.Publishing.GitBook.StrategyOverviewPath)), renderStrategyOverview()); err != nil {
		return err
	}
	if err := writeDoc(filepath.Join(root, filepath.FromSlash(cfg.Publishing.GitBook.RiskDisclosurePath)), renderRiskDisclosure()); err != nil {
		return err
	}
	return nil
}
