package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
)

var markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)

func validateTree(cfg *config.Config, root string) error {
	required := []string{
		gitbookConfigPath(root),
	}
	if cfg.Publishing.GitBook.ArchiveByRunDate {
		required = append(required, filepath.Join(root, archiveDirName, readmeName))
	}
	if cfg.Publishing.GitBook.GenerateHomepage {
		required = append(required, readmePath(root))
	}
	if cfg.Publishing.GitBook.GenerateSummary {
		required = append(required, summaryPath(root))
	}
	if cfg.Publishing.GitBook.IncludeDaily {
		required = append(required, latestReportPath(root, "daily"))
	}
	if cfg.Publishing.GitBook.IncludeDCAPlan {
		required = append(required, latestReportPath(root, "dca-plan"))
	}
	if _, err := os.Stat(latestReportPath(root, "market-pool")); err == nil {
		required = append(required, latestReportPath(root, "market-pool"))
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}

	for _, path := range required {
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("missing required docs artifact %s: %w", path, err)
		}
	}
	return validateAllNavigationLinks(root)
}

func validateAllNavigationLinks(root string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		if name != readmeName && name != summaryName {
			return nil
		}
		return validateMarkdownLinks(path)
	})
}

func validateMarkdownLinks(path string) error {
	buf, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	matches := markdownLinkPattern.FindAllStringSubmatch(string(buf), -1)
	for _, match := range matches {
		target := strings.TrimSpace(match[1])
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			continue
		}
		abs := filepath.Join(filepath.Dir(path), filepath.FromSlash(target))
		if _, err := os.Stat(abs); err != nil {
			return fmt.Errorf("navigation link target missing in %s: %s", path, target)
		}
	}
	return nil
}
