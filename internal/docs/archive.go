package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func pruneArchivedReports(root string, currentRunDate time.Time, retainDays int) error {
	if retainDays <= 0 {
		return nil
	}
	archiveRoot := filepath.Join(root, archiveDirName)
	entries, err := os.ReadDir(archiveRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	cutoff := normalizeArchiveDate(currentRunDate).AddDate(0, 0, -(retainDays - 1))
	for _, yearEntry := range entries {
		if !yearEntry.IsDir() {
			continue
		}
		yearRoot := filepath.Join(archiveRoot, yearEntry.Name())
		months, err := os.ReadDir(yearRoot)
		if err != nil {
			return err
		}
		for _, monthEntry := range months {
			if !monthEntry.IsDir() {
				continue
			}
			monthRoot := filepath.Join(yearRoot, monthEntry.Name())
			days, err := os.ReadDir(monthRoot)
			if err != nil {
				return err
			}
			for _, dayEntry := range days {
				if !dayEntry.IsDir() {
					continue
				}
				date, err := time.Parse("2006-01-02", strings.Join([]string{yearEntry.Name(), monthEntry.Name(), dayEntry.Name()}, "-"))
				if err != nil {
					continue
				}
				if date.Before(cutoff) {
					if err := os.RemoveAll(filepath.Join(monthRoot, dayEntry.Name())); err != nil {
						return err
					}
				}
			}
		}
	}
	return removeEmptyArchiveDirs(archiveRoot)
}

func removeEmptyArchiveDirs(archiveRoot string) error {
	years, err := childDirs(archiveRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, year := range years {
		yearRoot := filepath.Join(archiveRoot, year)
		months, err := childDirs(yearRoot)
		if err != nil {
			return err
		}
		for _, month := range months {
			monthRoot := filepath.Join(yearRoot, month)
			days, err := childDirs(monthRoot)
			if err != nil {
				return err
			}
			if len(days) == 0 {
				if err := os.Remove(monthRoot); err != nil && !os.IsNotExist(err) {
					return err
				}
			}
		}
		months, err = childDirs(yearRoot)
		if err != nil {
			return err
		}
		if len(months) == 0 {
			if err := os.Remove(yearRoot); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

func normalizeArchiveDate(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func buildArchiveIndexes(root string) error {
	archiveRoot := filepath.Join(root, archiveDirName)
	if _, err := os.Stat(archiveRoot); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if err := buildArchiveRootIndex(archiveRoot); err != nil {
		return err
	}
	years, err := childDirs(archiveRoot)
	if err != nil {
		return err
	}
	for _, year := range years {
		yearRoot := filepath.Join(archiveRoot, year)
		if err := buildYearIndex(root, yearRoot, year); err != nil {
			return err
		}
		months, err := childDirs(yearRoot)
		if err != nil {
			return err
		}
		for _, month := range months {
			monthRoot := filepath.Join(yearRoot, month)
			if err := buildMonthIndex(root, monthRoot, year, month); err != nil {
				return err
			}
			days, err := childDirs(monthRoot)
			if err != nil {
				return err
			}
			for _, day := range days {
				dayRoot := filepath.Join(monthRoot, day)
				if err := buildDayIndex(dayRoot, year, month, day); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func buildArchiveRootIndex(archiveRoot string) error {
	years, err := childDirs(archiveRoot)
	if err != nil {
		return err
	}
	lines := []string{"# Archive", "", "Historical report snapshots grouped by date.", ""}
	for _, year := range years {
		lines = append(lines, fmt.Sprintf("- [%s](%s/README.md)", year, filepath.ToSlash(year)))
	}
	return writeDoc(filepath.Join(archiveRoot, readmeName), strings.Join(lines, "\n")+"\n")
}

func buildYearIndex(root, yearRoot, year string) error {
	months, err := childDirs(yearRoot)
	if err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("# %s Archive", year),
		"",
		fmt.Sprintf("- [Back to archive](%s)", relativeFrom(yearRoot, filepath.Join(root, archiveDirName, readmeName))),
		"",
	}
	for _, month := range months {
		lines = append(lines, fmt.Sprintf("- [%s-%s](%s/README.md)", year, month, filepath.ToSlash(month)))
	}
	return writeDoc(filepath.Join(yearRoot, readmeName), strings.Join(lines, "\n")+"\n")
}

func buildMonthIndex(root, monthRoot, year, month string) error {
	days, err := childDirs(monthRoot)
	if err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("# %s-%s Archive", year, month),
		"",
		fmt.Sprintf("- [Back to %s](%s)", year, relativeFrom(monthRoot, filepath.Join(filepath.Dir(monthRoot), readmeName))),
		"",
	}
	for _, day := range days {
		lines = append(lines, fmt.Sprintf("- [%s-%s-%s](%s/README.md)", year, month, day, filepath.ToSlash(day)))
	}
	return writeDoc(filepath.Join(monthRoot, readmeName), strings.Join(lines, "\n")+"\n")
}

func buildDayIndex(dayRoot, year, month, day string) error {
	entries, err := os.ReadDir(dayRoot)
	if err != nil {
		return err
	}
	lines := []string{
		fmt.Sprintf("# %s-%s-%s Snapshot", year, month, day),
		"",
		fmt.Sprintf("- [Back to %s-%s](%s)", year, month, relativeFrom(dayRoot, filepath.Join(filepath.Dir(dayRoot), readmeName))),
		"",
		"## Reports",
		"",
	}
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == readmeName || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		label := strings.TrimSuffix(entry.Name(), ".md")
		label = strings.ReplaceAll(label, "-", " ")
		label = strings.Title(label)
		lines = append(lines, fmt.Sprintf("- [%s](%s)", label, filepath.ToSlash(entry.Name())))
	}
	return writeDoc(filepath.Join(dayRoot, readmeName), strings.Join(lines, "\n")+"\n")
}

func childDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	items := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			items = append(items, entry.Name())
		}
	}
	sort.Sort(sort.Reverse(sort.StringSlice(items)))
	return items, nil
}

func relativeFrom(fromDir, target string) string {
	rel, err := filepath.Rel(fromDir, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}
