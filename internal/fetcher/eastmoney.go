package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type EastmoneyFetcher struct {
	client *http.Client
}

type trendPoint struct {
	X            int64   `json:"x"`
	Y            float64 `json:"y"`
	EquityReturn float64 `json:"equityReturn"`
}

func NewEastmoneyFetcher(timeout time.Duration) *EastmoneyFetcher {
	return &EastmoneyFetcher{client: &http.Client{Timeout: timeout}}
}

func (f *EastmoneyFetcher) FetchHistory(ctx context.Context, code string, days int) (*model.FetchResult, error) {
	url := fmt.Sprintf("https://fund.eastmoney.com/pingzhongdata/%s.js?v=%d", code, time.Now().Unix())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fundcli/0.1")
	req.Header.Set("Referer", fmt.Sprintf("https://fund.eastmoney.com/%s.html", code))
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("eastmoney returned status %d for %s", resp.StatusCode, code)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return ParsePingzhongData(code, string(body), days)
}

func ParsePingzhongData(code, body string, days int) (*model.FetchResult, error) {
	name, err := extractStringVar(body, "fS_name")
	if err != nil {
		name = code
	}
	trendRaw, err := extractValue(body, "Data_netWorthTrend")
	if err != nil {
		return nil, err
	}
	var points []trendPoint
	if err := json.Unmarshal([]byte(trendRaw), &points); err != nil {
		return nil, fmt.Errorf("parse Data_netWorthTrend: %w", err)
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("no net worth trend found for %s", code)
	}
	snapshots := make([]model.FundSnapshot, 0, len(points))
	for _, point := range points {
		if point.Y == 0 {
			continue
		}
		tradeDate := time.UnixMilli(point.X).UTC()
		snapshots = append(snapshots, model.FundSnapshot{
			FundCode:     code,
			FundName:     name,
			TradeDate:    tradeDate,
			NAV:          point.Y,
			AccNAV:       point.Y,
			DayChangePct: point.EquityReturn / 100,
			Source:       "eastmoney-pingzhongdata",
			CreatedAt:    time.Now().UTC(),
		})
	}
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].TradeDate.Before(snapshots[j].TradeDate)
	})
	if days > 0 && len(snapshots) > days {
		snapshots = snapshots[len(snapshots)-days:]
	}
	return &model.FetchResult{Code: code, Name: name, Snapshots: snapshots}, nil
}

func extractStringVar(body, name string) (string, error) {
	value, err := extractValue(body, name)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "\"")
	return value, nil
}

func extractValue(body, name string) (string, error) {
	pattern := regexp.MustCompile(`(?s)var\s+` + regexp.QuoteMeta(name) + `\s*=\s*(.+?);`)
	matches := pattern.FindStringSubmatch(body)
	if len(matches) != 2 {
		return "", fmt.Errorf("variable %s not found", name)
	}
	value := strings.TrimSpace(matches[1])
	if unquoted, err := strconv.Unquote(value); err == nil {
		return unquoted, nil
	}
	return value, nil
}
