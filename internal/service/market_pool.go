package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/fetcher"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type marketTheme struct {
	Key             string
	Label           string
	FundType        string
	QDIIType        string
	SortField       string
	Keywords        []string
	ExcludeKeywords []string
	PreferIndex     bool
	MinReturn120D   float64
	MinReturn250D   float64
	MaxDrawdown120D float64
}

type marketRankCandidate struct {
	Fund            model.MarketSearchFund
	EstablishedDate time.Time
}

type marketPoolCandidate struct {
	Item model.MarketPoolItem
}

type marketPoolMetrics struct {
	Return20D       float64
	Return60D       float64
	Return120D      float64
	Return250D      float64
	MaxDrawdown120D float64
}

func (s *Service) BuildMarketPool(ctx context.Context, days int) (*model.MarketPoolReport, error) {
	if !s.config.MarketPool.Enabled {
		return nil, fmt.Errorf("market pool is disabled")
	}
	if days < 300 {
		days = 300
	}
	allFunds, err := s.fetcher.SearchAllFunds(ctx)
	if err != nil {
		return nil, err
	}
	previousByTheme := s.latestMarketPoolItemsByTheme()
	profileCache := make(map[string]*model.MarketFundProfile)
	report := newMarketPoolReport(len(allFunds), s.config.MarketPool.RetentionScoreGap)
	for _, theme := range defaultMarketThemes(s.config) {
		s.processMarketTheme(ctx, theme, allFunds, days, previousByTheme, profileCache, report)
	}
	finalizeMarketPoolReport(report, s.config.MarketPool.SelectionCount)
	runID, err := s.store.SaveMarketPool(*report)
	if err != nil {
		return nil, err
	}
	report.RunID = runID
	return report, nil
}

func (s *Service) latestMarketPoolItemsByTheme() map[string]model.MarketPoolItem {
	previous, err := s.store.LatestMarketPool()
	if err != nil || previous == nil {
		return nil
	}
	items := make(map[string]model.MarketPoolItem, len(previous.Items))
	for _, item := range previous.Items {
		items[item.ThemeKey] = item
	}
	return items
}

func newMarketPoolReport(universeCount, retentionScoreGap int) *model.MarketPoolReport {
	runDate := time.Now().UTC()
	return &model.MarketPoolReport{
		Summary: model.MarketPoolSummary{
			RunDate:       runDate,
			UniverseCount: universeCount,
			GeneratedAt:   runDate,
			Notes: []string{
				fmt.Sprintf("稳定候选池按固定主题筛选，旧候选若分数仅落后 %d 分以内则继续保留。", retentionScoreGap),
			},
		},
	}
}

func (s *Service) processMarketTheme(ctx context.Context, theme marketTheme, allFunds []model.MarketSearchFund, days int, previousByTheme map[string]model.MarketPoolItem, profileCache map[string]*model.MarketFundProfile, report *model.MarketPoolReport) {
	matched := filterThemeFunds(allFunds, theme)
	report.Summary.MatchedCount += len(matched)
	if len(matched) == 0 {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("%s 主题未匹配到基金名称。", theme.Label))
		return
	}
	ranked, err := s.collectThemeRankedCandidates(ctx, theme, matched)
	if err != nil {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("%s 主题抓取失败：%v", theme.Label, err))
		return
	}
	candidates := s.buildThemeMarketPoolCandidates(ctx, theme, ranked, days, profileCache)
	report.Summary.EligibleCount += len(candidates)
	if len(candidates) == 0 {
		report.Summary.Notes = append(report.Summary.Notes, fmt.Sprintf("%s 主题暂无满足阈值的稳定候选。", theme.Label))
		return
	}
	sort.Slice(candidates, func(i, j int) bool {
		return preferMarketPoolCandidate(candidates[i], candidates[j])
	})
	selected, retained := selectMarketPoolCandidate(theme, candidates, previousByTheme, s.config.MarketPool.RetentionScoreGap)
	if retained {
		report.Summary.RetainedCount++
	}
	report.Items = append(report.Items, selected)
}

func (s *Service) buildThemeMarketPoolCandidates(ctx context.Context, theme marketTheme, ranked []marketRankCandidate, days int, profileCache map[string]*model.MarketFundProfile) []marketPoolCandidate {
	candidates := make([]marketPoolCandidate, 0, len(ranked))
	for _, candidate := range ranked {
		profile, ok := profileCache[candidate.Fund.Code]
		if !ok {
			fetched, err := s.fetcher.FetchMarketProfile(ctx, candidate.Fund, candidate.EstablishedDate, days)
			if err != nil {
				continue
			}
			profile = fetched
			profileCache[candidate.Fund.Code] = profile
		}
		item, ok := buildMarketPoolItem(theme, s.config.MarketPool, profile)
		if !ok {
			continue
		}
		candidates = append(candidates, marketPoolCandidate{Item: item})
		if len(candidates) >= s.config.MarketPool.MaxFundsPerTheme {
			break
		}
	}
	return candidates
}

func selectMarketPoolCandidate(theme marketTheme, candidates []marketPoolCandidate, previousByTheme map[string]model.MarketPoolItem, retentionScoreGap int) (model.MarketPoolItem, bool) {
	selected := candidates[0].Item
	prev, ok := previousByTheme[theme.Key]
	if !ok {
		return selected, false
	}
	for _, candidate := range candidates {
		if candidate.Item.FundCode != prev.FundCode {
			continue
		}
		if candidate.Item.Score >= selected.Score-retentionScoreGap {
			candidate.Item.Retained = true
			return candidate.Item, true
		}
		break
	}
	return selected, false
}

func finalizeMarketPoolReport(report *model.MarketPoolReport, selectionCount int) {
	if len(report.Items) > selectionCount {
		report.Items = report.Items[:selectionCount]
	}
	for idx := range report.Items {
		report.Items[idx].Rank = idx + 1
	}
	report.Summary.SelectedCount = len(report.Items)
	if len(report.Items) == 0 {
		report.Summary.Notes = append(report.Summary.Notes, "当前没有筛选出稳定候选，建议检查主题关键词或放宽阈值。")
	}
}

func (s *Service) LatestMarketPool() (*model.MarketPoolReport, error) {
	return s.store.LatestMarketPool()
}

func (s *Service) collectThemeRankedCandidates(ctx context.Context, theme marketTheme, matched []model.MarketSearchFund) ([]marketRankCandidate, error) {
	matchedByCode := make(map[string]model.MarketSearchFund, len(matched))
	for _, fund := range matched {
		matchedByCode[fund.Code] = fund
	}
	candidates := make([]marketRankCandidate, 0, s.config.MarketPool.MaxFundsPerTheme*2)
	seen := make(map[string]struct{})
	endDate := time.Now().UTC()
	startDate := endDate.AddDate(-1, 0, 0)
	for page := 1; page <= 4; page++ {
		entries, total, err := s.fetcher.FetchMarketRankings(ctx, fetcher.MarketRankQuery{
			FundType:  theme.FundType,
			QDIIType:  theme.QDIIType,
			SortField: theme.SortField,
			SortOrder: "desc",
			StartDate: startDate,
			EndDate:   endDate,
			Page:      page,
			PageSize:  50,
		})
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			fund, ok := matchedByCode[entry.Code]
			if !ok {
				continue
			}
			if _, exists := seen[entry.Code]; exists {
				continue
			}
			seen[entry.Code] = struct{}{}
			candidates = append(candidates, marketRankCandidate{Fund: fund, EstablishedDate: entry.EstablishedDate})
			if len(candidates) >= s.config.MarketPool.MaxFundsPerTheme*2 {
				return candidates, nil
			}
		}
		if page*50 >= total {
			break
		}
	}
	return candidates, nil
}

func defaultMarketThemes(cfg *config.Config) []marketTheme {
	return []marketTheme{
		{
			Key:             "cn_broad",
			Label:           "A股宽基",
			FundType:        "000",
			SortField:       "1nzf",
			Keywords:        []string{"沪深300", "中证A500", "中证A50", "中证800", "中证500", "上证50"},
			ExcludeKeywords: []string{"增强", "分级"},
			PreferIndex:     true,
			MinReturn120D:   minThemeThreshold(cfg.MarketPool.MinReturn120D, 0.06),
			MinReturn250D:   minThemeThreshold(cfg.MarketPool.MinReturn250D, 0.10),
			MaxDrawdown120D: maxThemeThreshold(cfg.MarketPool.MaxDrawdown120D, 0.16),
		},
		{
			Key:             "cn_dividend",
			Label:           "A股红利",
			FundType:        "000",
			SortField:       "1nzf",
			Keywords:        []string{"红利", "红利低波"},
			ExcludeKeywords: []string{"港", "恒生", "美元"},
			PreferIndex:     true,
			MinReturn120D:   minThemeThreshold(cfg.MarketPool.MinReturn120D, 0.06),
			MinReturn250D:   minThemeThreshold(cfg.MarketPool.MinReturn250D, 0.10),
			MaxDrawdown120D: maxThemeThreshold(cfg.MarketPool.MaxDrawdown120D, 0.16),
		},
		{
			Key:             "hk_dividend",
			Label:           "港股红利",
			FundType:        "all",
			SortField:       "1nzf",
			Keywords:        []string{"恒生高股息", "港股通高股息", "恒生红利", "红利低波"},
			ExcludeKeywords: []string{"A股", "美元"},
			PreferIndex:     true,
			MinReturn120D:   minThemeThreshold(cfg.MarketPool.MinReturn120D, 0.06),
			MinReturn250D:   minThemeThreshold(cfg.MarketPool.MinReturn250D, 0.10),
			MaxDrawdown120D: maxThemeThreshold(cfg.MarketPool.MaxDrawdown120D, 0.18),
		},
		{
			Key:             "sp500",
			Label:           "美股标普500",
			FundType:        "all",
			SortField:       "1nzf",
			Keywords:        []string{"标普500"},
			ExcludeKeywords: []string{"美元", "现汇", "现钞"},
			PreferIndex:     true,
			MinReturn120D:   minThemeThreshold(cfg.MarketPool.MinReturn120D, 0.05),
			MinReturn250D:   minThemeThreshold(cfg.MarketPool.MinReturn250D, 0.08),
			MaxDrawdown120D: maxThemeThreshold(cfg.MarketPool.MaxDrawdown120D, 0.18),
		},
		{
			Key:             "nasdaq100",
			Label:           "美股纳指100",
			FundType:        "all",
			SortField:       "1nzf",
			Keywords:        []string{"纳斯达克100", "纳指100"},
			ExcludeKeywords: []string{"美元", "现汇", "现钞"},
			PreferIndex:     true,
			MinReturn120D:   minThemeThreshold(cfg.MarketPool.MinReturn120D, 0.05),
			MinReturn250D:   minThemeThreshold(cfg.MarketPool.MinReturn250D, 0.08),
			MaxDrawdown120D: maxThemeThreshold(cfg.MarketPool.MaxDrawdown120D, 0.22),
		},
		{
			Key:             "gold",
			Label:           "黄金",
			FundType:        "all",
			SortField:       "1nzf",
			Keywords:        []string{"黄金ETF联接", "黄金ETF", "黄金"},
			ExcludeKeywords: []string{"上海金", "美元"},
			PreferIndex:     true,
			MinReturn120D:   0.02,
			MinReturn250D:   0.05,
			MaxDrawdown120D: 0.15,
		},
	}
}

func buildMarketPoolItem(theme marketTheme, cfg config.MarketPoolConfig, profile *model.MarketFundProfile) (model.MarketPoolItem, bool) {
	if profile == nil || profile.Latest == nil || len(profile.History) == 0 {
		return model.MarketPoolItem{}, false
	}
	metrics := buildMarketPoolMetrics(profile.History)
	if !passesMarketPoolThresholds(theme, cfg, profile, metrics) {
		return model.MarketPoolItem{}, false
	}
	score, reasons := scoreMarketPoolProfile(theme, cfg, profile, metrics)
	if score < cfg.MinScore {
		return model.MarketPoolItem{}, false
	}
	return model.MarketPoolItem{
		ThemeKey:         theme.Key,
		ThemeLabel:       theme.Label,
		FundCode:         profile.Fund.Code,
		FundName:         profile.Fund.Name,
		FundType:         profile.Fund.FundType,
		Score:            score,
		Return20D:        metrics.Return20D,
		Return60D:        metrics.Return60D,
		Return120D:       metrics.Return120D,
		Return250D:       metrics.Return250D,
		MaxDrawdown120D:  metrics.MaxDrawdown120D,
		FundSizeYi:       profile.FundSizeYi,
		EstablishedYears: profile.EstablishedYears,
		LatestTradeDate:  profile.Latest.TradeDate,
		Reason:           strings.Join(reasons, "；"),
	}, true
}

func buildMarketPoolMetrics(history []model.FundSnapshot) marketPoolMetrics {
	return marketPoolMetrics{
		Return20D:       trailingReturn(history, 20),
		Return60D:       trailingReturn(history, 60),
		Return120D:      trailingReturn(history, 120),
		Return250D:      trailingReturn(history, 250),
		MaxDrawdown120D: trailingMaxDrawdown(history, 120),
	}
}

func passesMarketPoolThresholds(theme marketTheme, cfg config.MarketPoolConfig, profile *model.MarketFundProfile, metrics marketPoolMetrics) bool {
	if cfg.MinFundSizeYi > 0 && profile.FundSizeYi > 0 && profile.FundSizeYi < cfg.MinFundSizeYi {
		return false
	}
	if profile.EstablishedYears < cfg.MinEstablishedYears {
		return false
	}
	return metrics.Return120D >= theme.MinReturn120D &&
		metrics.Return250D >= theme.MinReturn250D &&
		metrics.MaxDrawdown120D <= theme.MaxDrawdown120D
}

func scoreMarketPoolProfile(theme marketTheme, cfg config.MarketPoolConfig, profile *model.MarketFundProfile, metrics marketPoolMetrics) (int, []string) {
	reasons := make([]string, 0, 8)
	score := 0
	if metrics.Return250D >= theme.MinReturn250D {
		score += 2
		reasons = append(reasons, fmt.Sprintf("250日收益 %.2f%%", metrics.Return250D*100))
	}
	if metrics.Return120D >= theme.MinReturn120D {
		score += 2
		reasons = append(reasons, fmt.Sprintf("120日收益 %.2f%%", metrics.Return120D*100))
	}
	if metrics.Return60D > 0 {
		score++
		reasons = append(reasons, fmt.Sprintf("60日收益 %.2f%%", metrics.Return60D*100))
	}
	if metrics.Return20D > -0.02 {
		score++
		reasons = append(reasons, fmt.Sprintf("20日回撤可控 %.2f%%", metrics.Return20D*100))
	}
	if metrics.MaxDrawdown120D <= theme.MaxDrawdown120D {
		score++
		reasons = append(reasons, fmt.Sprintf("120日最大回撤 %.2f%%", metrics.MaxDrawdown120D*100))
	}
	if profile.FundSizeYi >= maxFloat(profileThreshold(cfg.MinFundSizeYi), 20) {
		score++
		reasons = append(reasons, fmt.Sprintf("规模 %.1f 亿", profile.FundSizeYi))
	}
	if profile.EstablishedYears >= maxFloat(profileYears(cfg.MinEstablishedYears), 3) {
		score++
		reasons = append(reasons, fmt.Sprintf("成立 %.1f 年", profile.EstablishedYears))
	}
	if theme.PreferIndex && profile.IsIndex {
		score++
		reasons = append(reasons, "指数工具更稳定")
	}
	shareClassScore, shareClassReason := shareClassPreference(profile.Fund.Name)
	score += shareClassScore
	if shareClassReason != "" {
		reasons = append(reasons, shareClassReason)
	}
	return score, reasons
}

func filterThemeFunds(funds []model.MarketSearchFund, theme marketTheme) []model.MarketSearchFund {
	matched := make([]model.MarketSearchFund, 0)
	for _, fund := range funds {
		name := strings.TrimSpace(fund.Name)
		if name == "" || strings.Contains(name, "后端") {
			continue
		}
		if !matchesAny(name, theme.Keywords) {
			continue
		}
		if matchesAny(name, theme.ExcludeKeywords) {
			continue
		}
		matched = append(matched, fund)
	}
	return matched
}

func preferMarketPoolCandidate(left, right marketPoolCandidate) bool {
	if left.Item.Score != right.Item.Score {
		return left.Item.Score > right.Item.Score
	}
	if left.Item.Return250D != right.Item.Return250D {
		return left.Item.Return250D > right.Item.Return250D
	}
	if left.Item.MaxDrawdown120D != right.Item.MaxDrawdown120D {
		return left.Item.MaxDrawdown120D < right.Item.MaxDrawdown120D
	}
	if left.Item.FundSizeYi != right.Item.FundSizeYi {
		return left.Item.FundSizeYi > right.Item.FundSizeYi
	}
	return left.Item.FundCode < right.Item.FundCode
}

func trailingReturn(history []model.FundSnapshot, periods int) float64 {
	if len(history) < 2 {
		return 0
	}
	last := history[len(history)-1].NAV
	if last <= 0 {
		return 0
	}
	idx := len(history) - 1 - periods
	if idx < 0 {
		idx = 0
	}
	base := history[idx].NAV
	if base <= 0 {
		return 0
	}
	return last/base - 1
}

func trailingMaxDrawdown(history []model.FundSnapshot, periods int) float64 {
	if len(history) == 0 {
		return 0
	}
	start := len(history) - periods
	if start < 0 {
		start = 0
	}
	peak := history[start].NAV
	maxDrawdown := 0.0
	for _, point := range history[start:] {
		if point.NAV > peak {
			peak = point.NAV
		}
		if peak <= 0 {
			continue
		}
		drawdown := 1 - point.NAV/peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}
	return maxDrawdown
}

func matchesAny(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}
	return false
}

func minThemeThreshold(global, fallback float64) float64 {
	if global <= 0 {
		return fallback
	}
	if global < fallback {
		return global
	}
	return fallback
}

func maxThemeThreshold(global, fallback float64) float64 {
	if global <= 0 {
		return fallback
	}
	if global > fallback {
		return global
	}
	return fallback
}

func shareClassPreference(name string) (int, string) {
	trimmed := strings.TrimSpace(name)
	switch {
	case strings.Contains(trimmed, "美元") || strings.Contains(trimmed, "现汇") || strings.Contains(trimmed, "现钞"):
		return -2, "剔除外币份额偏好"
	case strings.HasSuffix(trimmed, "A") || strings.HasSuffix(trimmed, "A(人民币)"):
		return 1, "优先长期持有 A 类份额"
	case strings.HasSuffix(trimmed, "C"):
		return 0, "可替代为 C 类份额"
	default:
		return 0, ""
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func profileThreshold(value float64) float64 {
	return value
}

func profileYears(value float64) float64 {
	return value
}
