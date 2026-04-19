package fetcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type MarketRankQuery struct {
	FundType  string
	QDIIType  string
	SortField string
	SortOrder string
	StartDate time.Time
	EndDate   time.Time
	Page      int
	PageSize  int
}

type MarketRankEntry struct {
	Code            string
	Name            string
	TradeDate       time.Time
	Return1W        float64
	Return1M        float64
	Return3M        float64
	Return6M        float64
	Return1Y        float64
	Return2Y        float64
	Return3Y        float64
	ReturnYTD       float64
	ReturnSince     float64
	EstablishedDate time.Time
}

type marketRankPayload struct {
	Datas    []string `json:"datas"`
	AllPages int      `json:"allPages"`
	AllNum   int      `json:"allNum"`
}

type marketAssetAllocation struct {
	Series []struct {
		Name string    `json:"name"`
		Data []float64 `json:"data"`
	} `json:"series"`
}

func (f *EastmoneyFetcher) SearchAllFunds(ctx context.Context) ([]model.MarketSearchFund, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://fund.eastmoney.com/js/fundcode_search.js", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fundcli/0.1")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fund search returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	payload := stripBOM(string(body))
	start := strings.Index(payload, "[[")
	end := strings.LastIndex(payload, "];")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("unexpected fund search payload")
	}
	var raw [][]string
	if err := json.Unmarshal([]byte(payload[start:end+1]), &raw); err != nil {
		return nil, fmt.Errorf("decode fund search payload: %w", err)
	}
	funds := make([]model.MarketSearchFund, 0, len(raw))
	for _, item := range raw {
		if len(item) < 5 {
			continue
		}
		funds = append(funds, model.MarketSearchFund{
			Code:     strings.TrimSpace(item[0]),
			Name:     strings.TrimSpace(item[2]),
			FundType: strings.TrimSpace(item[3]),
			Spell:    strings.TrimSpace(item[4]),
		})
	}
	return funds, nil
}

func (f *EastmoneyFetcher) FetchMarketRankings(ctx context.Context, query MarketRankQuery) ([]MarketRankEntry, int, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 50
	}
	if strings.TrimSpace(query.SortField) == "" {
		query.SortField = "1nzf"
	}
	if strings.TrimSpace(query.SortOrder) == "" {
		query.SortOrder = "desc"
	}
	if query.StartDate.IsZero() {
		query.StartDate = time.Now().UTC().AddDate(-1, 0, 0)
	}
	if query.EndDate.IsZero() {
		query.EndDate = time.Now().UTC()
	}
	values := url.Values{
		"op":         {"ph"},
		"dt":         {"kf"},
		"ft":         {defaultString(query.FundType, "all")},
		"rs":         {""},
		"gs":         {"0"},
		"sc":         {query.SortField},
		"st":         {query.SortOrder},
		"sd":         {query.StartDate.Format("2006-01-02")},
		"ed":         {query.EndDate.Format("2006-01-02")},
		"qdii":       {query.QDIIType},
		"tabSubtype": {",,,,,"},
		"pi":         {strconv.Itoa(query.Page)},
		"pn":         {strconv.Itoa(query.PageSize)},
		"dx":         {"1"},
	}
	endpoint := "https://fund.eastmoney.com/data/rankhandler.aspx?" + values.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Referer", "https://fund.eastmoney.com/data/fundranking.html")
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("market rank returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	payload, err := decodeRankPayload(string(body))
	if err != nil {
		return nil, 0, err
	}
	entries := make([]MarketRankEntry, 0, len(payload.Datas))
	for _, row := range payload.Datas {
		entry, ok := parseMarketRankEntry(row)
		if !ok {
			continue
		}
		entries = append(entries, entry)
	}
	return entries, payload.AllNum, nil
}

func (f *EastmoneyFetcher) FetchMarketProfile(ctx context.Context, fund model.MarketSearchFund, establishedDate time.Time, days int) (*model.MarketFundProfile, error) {
	url := fmt.Sprintf("https://fund.eastmoney.com/pingzhongdata/%s.js?v=%d", fund.Code, time.Now().Unix())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "fundcli/0.1")
	req.Header.Set("Referer", fmt.Sprintf("https://fund.eastmoney.com/%s.html", fund.Code))
	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("market profile returned status %d for %s", resp.StatusCode, fund.Code)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	parsed, err := ParsePingzhongData(fund.Code, string(body), days)
	if err != nil {
		return nil, err
	}
	profile := &model.MarketFundProfile{
		Fund:    fund,
		History: parsed.Snapshots,
		IsIndex: isIndexFund(fund),
	}
	if len(parsed.Snapshots) > 0 {
		latest := parsed.Snapshots[len(parsed.Snapshots)-1]
		profile.Latest = &latest
		if establishedDate.IsZero() {
			establishedDate = parsed.Snapshots[0].TradeDate
		}
	}
	if !establishedDate.IsZero() {
		profile.EstablishedYears = time.Since(establishedDate).Hours() / 24 / 365
	}
	if value, err := parseLatestNetAsset(string(body)); err == nil {
		profile.FundSizeYi = value
	}
	return profile, nil
}

func decodeRankPayload(body string) (*marketRankPayload, error) {
	payload := strings.TrimSpace(stripBOM(body))
	payload = strings.TrimPrefix(payload, "var rankData =")
	payload = strings.TrimSpace(strings.TrimSuffix(payload, ";"))
	if strings.Contains(payload, "ErrCode:-999") {
		return nil, fmt.Errorf("market rank access denied")
	}
	quoted := regexp.MustCompile(`([,{])([A-Za-z_][A-Za-z0-9_]*)\s*:`).ReplaceAllString(payload, `${1}"${2}":`)
	var result marketRankPayload
	if err := json.Unmarshal([]byte(quoted), &result); err != nil {
		return nil, fmt.Errorf("decode market rank payload: %w", err)
	}
	return &result, nil
}

func parseMarketRankEntry(row string) (MarketRankEntry, bool) {
	fields := strings.Split(row, ",")
	if len(fields) < 17 {
		return MarketRankEntry{}, false
	}
	tradeDate, _ := time.Parse("2006-01-02", strings.TrimSpace(fields[3]))
	establishedDate, _ := time.Parse("2006-01-02", strings.TrimSpace(fields[16]))
	return MarketRankEntry{
		Code:            strings.TrimSpace(fields[0]),
		Name:            strings.TrimSpace(fields[1]),
		TradeDate:       tradeDate,
		Return1W:        parseRankPercent(fields, 7),
		Return1M:        parseRankPercent(fields, 8),
		Return3M:        parseRankPercent(fields, 9),
		Return6M:        parseRankPercent(fields, 10),
		Return1Y:        parseRankPercent(fields, 11),
		Return2Y:        parseRankPercent(fields, 12),
		Return3Y:        parseRankPercent(fields, 13),
		ReturnYTD:       parseRankPercent(fields, 14),
		ReturnSince:     parseRankPercent(fields, 15),
		EstablishedDate: establishedDate,
	}, true
}

func parseRankPercent(fields []string, index int) float64 {
	if index >= len(fields) {
		return 0
	}
	value, err := strconv.ParseFloat(strings.TrimSpace(fields[index]), 64)
	if err != nil {
		return 0
	}
	return value / 100
}

func parseLatestNetAsset(body string) (float64, error) {
	raw, err := extractValue(body, "Data_assetAllocation")
	if err != nil {
		return 0, err
	}
	var allocation marketAssetAllocation
	if err := json.Unmarshal([]byte(raw), &allocation); err != nil {
		return 0, err
	}
	for _, series := range allocation.Series {
		if !strings.Contains(series.Name, "净资产") {
			continue
		}
		for idx := len(series.Data) - 1; idx >= 0; idx-- {
			if series.Data[idx] > 0 {
				return series.Data[idx], nil
			}
		}
	}
	return 0, fmt.Errorf("net asset not found")
}

func isIndexFund(fund model.MarketSearchFund) bool {
	name := strings.ToUpper(strings.TrimSpace(fund.Name))
	fundType := strings.TrimSpace(fund.FundType)
	return strings.Contains(fundType, "指数") || strings.Contains(name, "ETF") || strings.Contains(name, "INDEX")
}

func stripBOM(value string) string {
	return strings.TrimPrefix(value, "\ufeff")
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
