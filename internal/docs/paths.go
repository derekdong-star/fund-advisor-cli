package docs

import (
	"path/filepath"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
)

const (
	latestDirName  = "latest"
	archiveDirName = "archive"
	stateFileName  = ".fundcli-publish.json"
	gitbookConfig  = ".gitbook.yaml"
	readmeName     = "README.md"
	summaryName    = "SUMMARY.md"
)

func docsRoot(cfg *config.Config) string {
	root := cfg.Publishing.GitBook.DocsRoot
	if filepath.IsAbs(root) {
		return root
	}
	return filepath.Clean(filepath.Join(cfg.ConfigDir(), root))
}

func latestReportPath(root, slug string) string {
	return filepath.Join(root, latestDirName, slug+".md")
}

func archiveReportPath(root string, date time.Time, slug string) string {
	return filepath.Join(root, archiveDirName, date.Format("2006"), date.Format("01"), date.Format("02"), slug+".md")
}

func gitbookConfigPath(root string) string {
	return filepath.Join(root, gitbookConfig)
}

func readmePath(root string) string {
	return filepath.Join(root, readmeName)
}

func summaryPath(root string) string {
	return filepath.Join(root, summaryName)
}

func statePath(root string) string {
	return filepath.Join(root, stateFileName)
}
