package ledger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
)

func TestReconcileBuildsHoldingsFromSnapshotAndTrades(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Storage.DSN = filepath.Join(dir, "data", "fundcli.db")

	paths := ResolvePaths(cfg)
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	snapshot := []byte("as_of: 2026-04-20\npositions:\n  - fund_code: \"000979\"\n    fund_name: \"景顺长城沪港深精选股票A\"\n    units: 100\n    total_cost: 1000\n")
	if err := os.WriteFile(paths.SnapshotPath, snapshot, 0o644); err != nil {
		t.Fatalf("WriteFile(snapshot) error = %v", err)
	}
	trades := []byte("trade_date,fund_code,side,amount,nav,units,fee,note\n2026-04-21,000979,BUY,220,2.0,110,0,追加\n2026-04-22,000979,SELL,100,2.0,50,0,减仓\n")
	if err := os.WriteFile(paths.TradesCSVPath, trades, 0o644); err != nil {
		t.Fatalf("WriteFile(trades) error = %v", err)
	}

	result, err := Reconcile(cfg)
	if err != nil {
		t.Fatalf("Reconcile() error = %v", err)
	}
	if !result.Applied {
		t.Fatalf("expected applied reconcile")
	}
	if got := result.TradeCount; got != 2 {
		t.Fatalf("TradeCount = %d, want 2", got)
	}
	if got := result.Positions[0].Units; got <= 0 {
		t.Fatalf("expected reconciled units > 0, got %.4f", got)
	}
	if got := result.Positions[0].Units; got < 159.9 || got > 160.1 {
		t.Fatalf("Units = %.4f, want about 160", got)
	}
	if got := result.Positions[0].TotalCost; got < 929.4 || got > 929.6 {
		t.Fatalf("TotalCost = %.4f, want about 929.52", got)
	}
}

func TestReconcileRejectsOversell(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Storage.DSN = filepath.Join(dir, "data", "fundcli.db")

	paths := ResolvePaths(cfg)
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	snapshot := []byte("as_of: 2026-04-20\npositions:\n  - fund_code: \"000979\"\n    fund_name: \"景顺长城沪港深精选股票A\"\n    units: 10\n    total_cost: 100\n")
	if err := os.WriteFile(paths.SnapshotPath, snapshot, 0o644); err != nil {
		t.Fatalf("WriteFile(snapshot) error = %v", err)
	}
	trades := []byte("trade_date,fund_code,side,amount,nav,units,fee,note\n2026-04-21,000979,SELL,220,2.0,110,0,错误\n")
	if err := os.WriteFile(paths.TradesCSVPath, trades, 0o644); err != nil {
		t.Fatalf("WriteFile(trades) error = %v", err)
	}

	if _, err := Reconcile(cfg); err == nil {
		t.Fatalf("expected oversell to fail")
	}
}
