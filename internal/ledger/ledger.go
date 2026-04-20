package ledger

import (
	"encoding/csv"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
)

const (
	snapshotFileName = "holdings_snapshot.yaml"
	tradesFileName   = "trades.csv"
	dateLayout       = "2006-01-02"
)

type Paths struct {
	DataDir       string
	SnapshotPath  string
	TradesCSVPath string
}

type SnapshotFile struct {
	AsOf      string             `yaml:"as_of"`
	Positions []SnapshotPosition `yaml:"positions"`
}

type SnapshotPosition struct {
	FundCode  string  `yaml:"fund_code"`
	FundName  string  `yaml:"fund_name"`
	Units     float64 `yaml:"units"`
	TotalCost float64 `yaml:"total_cost"`
}

type TradeRecord struct {
	TradeDate time.Time
	FundCode  string
	Side      string
	Amount    float64
	NAV       float64
	Units     float64
	Fee       float64
	Note      string
}

type Holding struct {
	FundCode      string
	FundName      string
	Units         float64
	TotalCost     float64
	TradeCount    int
	LastTradeDate time.Time
}

type ReconcileResult struct {
	Paths        Paths
	SnapshotAsOf string
	TradeCount   int
	Positions    []Holding
	Note         string
	Applied      bool
}

func ResolvePaths(cfg *config.Config) Paths {
	dataDir := filepath.Dir(cfg.ResolveStorageDSN())
	return Paths{
		DataDir:       dataDir,
		SnapshotPath:  filepath.Join(dataDir, snapshotFileName),
		TradesCSVPath: filepath.Join(dataDir, tradesFileName),
	}
}

func WriteTemplate(cfg *config.Config, force bool) (Paths, error) {
	paths := ResolvePaths(cfg)
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		return paths, err
	}
	if !force {
		if _, err := os.Stat(paths.SnapshotPath); err == nil {
			return paths, fmt.Errorf("ledger snapshot already exists: %s", paths.SnapshotPath)
		}
		if _, err := os.Stat(paths.TradesCSVPath); err == nil {
			return paths, fmt.Errorf("ledger trades csv already exists: %s", paths.TradesCSVPath)
		}
	}
	snapshot := SnapshotFile{
		AsOf:      time.Now().In(time.FixedZone("CST", 8*3600)).Format(dateLayout),
		Positions: make([]SnapshotPosition, 0, len(cfg.Funds)),
	}
	for _, fund := range cfg.Funds {
		snapshot.Positions = append(snapshot.Positions, SnapshotPosition{
			FundCode:  fund.Code,
			FundName:  fund.Name,
			Units:     0,
			TotalCost: fund.AccountValue,
		})
	}
	buf, err := yaml.Marshal(snapshot)
	if err != nil {
		return paths, err
	}
	if err := os.WriteFile(paths.SnapshotPath, buf, 0o644); err != nil {
		return paths, err
	}
	csvContent := "trade_date,fund_code,side,amount,nav,units,fee,note\n"
	if err := os.WriteFile(paths.TradesCSVPath, []byte(csvContent), 0o644); err != nil {
		return paths, err
	}
	return paths, nil
}

func AppendTrade(cfg *config.Config, trade TradeRecord) (Paths, TradeRecord, error) {
	paths := ResolvePaths(cfg)
	trade.Side = strings.ToUpper(strings.TrimSpace(trade.Side))
	trade.FundCode = strings.TrimSpace(trade.FundCode)
	if trade.TradeDate.IsZero() {
		trade.TradeDate = time.Now().In(time.FixedZone("UTC+8", 8*3600))
	}
	if trade.Units <= 0 && trade.Amount > 0 && trade.NAV > 0 {
		trade.Units = trade.Amount / trade.NAV
	}
	if trade.Amount <= 0 && trade.Units > 0 && trade.NAV > 0 {
		trade.Amount = trade.Units * trade.NAV
	}
	if err := validateTrade(cfg, trade); err != nil {
		return paths, trade, err
	}
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		return paths, trade, err
	}
	writeHeader := false
	if info, err := os.Stat(paths.TradesCSVPath); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return paths, trade, err
		}
		writeHeader = true
	} else if info.Size() == 0 {
		writeHeader = true
	}
	file, err := os.OpenFile(paths.TradesCSVPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return paths, trade, err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	if writeHeader {
		if err := writer.Write([]string{"trade_date", "fund_code", "side", "amount", "nav", "units", "fee", "note"}); err != nil {
			return paths, trade, err
		}
	}
	if err := writer.Write([]string{
		trade.TradeDate.Format(dateLayout),
		trade.FundCode,
		trade.Side,
		fmt.Sprintf("%.2f", trade.Amount),
		fmt.Sprintf("%.4f", trade.NAV),
		fmt.Sprintf("%.4f", trade.Units),
		fmt.Sprintf("%.2f", trade.Fee),
		trade.Note,
	}); err != nil {
		return paths, trade, err
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return paths, trade, err
	}
	return paths, trade, nil
}

func ReconcileIfPresent(cfg *config.Config) (*ReconcileResult, error) {
	paths := ResolvePaths(cfg)
	if _, err := os.Stat(paths.SnapshotPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &ReconcileResult{Paths: paths, Applied: false}, nil
		}
		return nil, err
	}
	return Reconcile(cfg)
}

func Reconcile(cfg *config.Config) (*ReconcileResult, error) {
	paths := ResolvePaths(cfg)
	snapshot, err := LoadSnapshot(paths.SnapshotPath)
	if err != nil {
		return nil, err
	}
	trades, err := LoadTrades(paths.TradesCSVPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		trades = nil
	}
	result, err := reconcile(cfg, paths, snapshot, trades)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func Check(cfg *config.Config) (*ReconcileResult, error) {
	return Reconcile(cfg)
}

func LoadSnapshot(path string) (*SnapshotFile, error) {
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var snapshot SnapshotFile
	if err := yaml.Unmarshal(buf, &snapshot); err != nil {
		return nil, fmt.Errorf("parse ledger snapshot: %w", err)
	}
	if strings.TrimSpace(snapshot.AsOf) == "" {
		return nil, errors.New("ledger snapshot as_of is required")
	}
	if _, err := time.Parse(dateLayout, snapshot.AsOf); err != nil {
		return nil, fmt.Errorf("ledger snapshot as_of must use %s: %w", dateLayout, err)
	}
	return &snapshot, nil
}

func LoadTrades(path string) ([]TradeRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse ledger trades csv: %w", err)
	}
	if len(records) == 0 {
		return nil, nil
	}
	header := normalizeHeader(records[0])
	required := []string{"trade_date", "fund_code", "side", "amount", "nav", "units", "fee", "note"}
	index := make(map[string]int, len(header))
	for i, name := range header {
		index[name] = i
	}
	for _, name := range required {
		if _, ok := index[name]; !ok {
			return nil, fmt.Errorf("ledger trades csv missing column %s", name)
		}
	}
	trades := make([]TradeRecord, 0, maxInt(0, len(records)-1))
	for rowNum, row := range records[1:] {
		if isBlankCSVRow(row) {
			continue
		}
		get := func(name string) string {
			idx := index[name]
			if idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		tradeDate, err := time.Parse(dateLayout, get("trade_date"))
		if err != nil {
			return nil, fmt.Errorf("ledger trade row %d has invalid trade_date: %w", rowNum+2, err)
		}
		amount, err := parseCSVFloat(get("amount"))
		if err != nil {
			return nil, fmt.Errorf("ledger trade row %d has invalid amount: %w", rowNum+2, err)
		}
		nav, err := parseCSVFloat(get("nav"))
		if err != nil {
			return nil, fmt.Errorf("ledger trade row %d has invalid nav: %w", rowNum+2, err)
		}
		units, err := parseCSVFloat(get("units"))
		if err != nil {
			return nil, fmt.Errorf("ledger trade row %d has invalid units: %w", rowNum+2, err)
		}
		fee, err := parseCSVFloat(get("fee"))
		if err != nil {
			return nil, fmt.Errorf("ledger trade row %d has invalid fee: %w", rowNum+2, err)
		}
		if units <= 0 && amount > 0 && nav > 0 {
			units = amount / nav
		}
		if amount <= 0 && units > 0 && nav > 0 {
			amount = units * nav
		}
		trades = append(trades, TradeRecord{
			TradeDate: tradeDate,
			FundCode:  strings.TrimSpace(get("fund_code")),
			Side:      strings.ToUpper(strings.TrimSpace(get("side"))),
			Amount:    amount,
			NAV:       nav,
			Units:     units,
			Fee:       fee,
			Note:      get("note"),
		})
	}
	sort.SliceStable(trades, func(i, j int) bool {
		if trades[i].TradeDate.Equal(trades[j].TradeDate) {
			if trades[i].FundCode == trades[j].FundCode {
				return trades[i].Side < trades[j].Side
			}
			return trades[i].FundCode < trades[j].FundCode
		}
		return trades[i].TradeDate.Before(trades[j].TradeDate)
	})
	return trades, nil
}

func reconcile(cfg *config.Config, paths Paths, snapshot *SnapshotFile, trades []TradeRecord) (*ReconcileResult, error) {
	knownFunds := make(map[string]config.FundConfig, len(cfg.Funds))
	holdings := make(map[string]*Holding, len(cfg.Funds))
	for _, fund := range cfg.Funds {
		knownFunds[fund.Code] = fund
		holdings[fund.Code] = &Holding{FundCode: fund.Code, FundName: fund.Name}
	}
	seenSnapshot := make(map[string]struct{}, len(snapshot.Positions))
	for _, position := range snapshot.Positions {
		if strings.TrimSpace(position.FundCode) == "" {
			return nil, errors.New("ledger snapshot contains empty fund_code")
		}
		if _, ok := knownFunds[position.FundCode]; !ok {
			return nil, fmt.Errorf("ledger snapshot fund %s is not in configured funds", position.FundCode)
		}
		if _, exists := seenSnapshot[position.FundCode]; exists {
			return nil, fmt.Errorf("ledger snapshot fund %s appears multiple times", position.FundCode)
		}
		seenSnapshot[position.FundCode] = struct{}{}
		if position.Units < 0 {
			return nil, fmt.Errorf("ledger snapshot fund %s has negative units", position.FundCode)
		}
		if position.TotalCost < 0 {
			return nil, fmt.Errorf("ledger snapshot fund %s has negative total_cost", position.FundCode)
		}
		holding := holdings[position.FundCode]
		holding.Units = position.Units
		holding.TotalCost = position.TotalCost
		if strings.TrimSpace(position.FundName) != "" {
			holding.FundName = position.FundName
		}
	}
	for _, trade := range trades {
		if _, ok := knownFunds[trade.FundCode]; !ok {
			return nil, fmt.Errorf("ledger trade fund %s is not in configured funds", trade.FundCode)
		}
		if trade.FundCode == "" {
			return nil, errors.New("ledger trade fund_code is required")
		}
		if trade.Units <= 0 {
			return nil, fmt.Errorf("ledger trade %s %s requires positive units", trade.TradeDate.Format(dateLayout), trade.FundCode)
		}
		if trade.NAV <= 0 {
			return nil, fmt.Errorf("ledger trade %s %s requires positive nav", trade.TradeDate.Format(dateLayout), trade.FundCode)
		}
		if trade.Amount <= 0 {
			return nil, fmt.Errorf("ledger trade %s %s requires positive amount", trade.TradeDate.Format(dateLayout), trade.FundCode)
		}
		if trade.Fee < 0 {
			return nil, fmt.Errorf("ledger trade %s %s has negative fee", trade.TradeDate.Format(dateLayout), trade.FundCode)
		}
		holding := holdings[trade.FundCode]
		holding.TradeCount++
		holding.LastTradeDate = trade.TradeDate
		switch trade.Side {
		case "BUY":
			holding.Units += trade.Units
			holding.TotalCost += trade.Amount + trade.Fee
		case "SELL":
			if holding.Units+1e-9 < trade.Units {
				return nil, fmt.Errorf("ledger trade %s %s sells %.4f units but only %.4f are available", trade.TradeDate.Format(dateLayout), trade.FundCode, trade.Units, holding.Units)
			}
			costRemoved := 0.0
			if holding.Units > 0 && holding.TotalCost > 0 {
				costRemoved = holding.TotalCost * (trade.Units / holding.Units)
			}
			holding.Units -= trade.Units
			holding.TotalCost -= costRemoved
			if holding.Units < 1e-9 {
				holding.Units = 0
			}
			if holding.TotalCost < 1e-9 {
				holding.TotalCost = 0
			}
		default:
			return nil, fmt.Errorf("ledger trade %s %s has unsupported side %s", trade.TradeDate.Format(dateLayout), trade.FundCode, trade.Side)
		}
	}
	positions := make([]Holding, 0, len(cfg.Funds))
	for _, fund := range cfg.Funds {
		positions = append(positions, *holdings[fund.Code])
	}
	return &ReconcileResult{
		Paths:        paths,
		SnapshotAsOf: snapshot.AsOf,
		TradeCount:   len(trades),
		Positions:    positions,
		Note:         fmt.Sprintf("真实仓位已由 %s 与 %s 重建", filepath.Base(paths.SnapshotPath), filepath.Base(paths.TradesCSVPath)),
		Applied:      true,
	}, nil
}

func validateTrade(cfg *config.Config, trade TradeRecord) error {
	if trade.FundCode == "" {
		return errors.New("ledger trade fund_code is required")
	}
	known := false
	for _, fund := range cfg.Funds {
		if fund.Code == trade.FundCode {
			known = true
			break
		}
	}
	if !known {
		return fmt.Errorf("ledger trade fund %s is not in configured funds", trade.FundCode)
	}
	switch trade.Side {
	case "BUY", "SELL":
	default:
		return fmt.Errorf("ledger trade side must be BUY or SELL, got %s", trade.Side)
	}
	if trade.NAV <= 0 {
		return errors.New("ledger trade nav must be positive")
	}
	if trade.Units <= 0 {
		return errors.New("ledger trade units must be positive")
	}
	if trade.Amount <= 0 {
		return errors.New("ledger trade amount must be positive")
	}
	if trade.Fee < 0 {
		return errors.New("ledger trade fee must be non-negative")
	}
	return nil
}

func normalizeHeader(row []string) []string {
	normalized := make([]string, len(row))
	for i, value := range row {
		normalized[i] = strings.ToLower(strings.TrimSpace(value))
	}
	return normalized
}

func isBlankCSVRow(row []string) bool {
	for _, value := range row {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func parseCSVFloat(value string) (float64, error) {
	if strings.TrimSpace(value) == "" {
		return 0, nil
	}
	return strconv.ParseFloat(strings.TrimSpace(value), 64)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
