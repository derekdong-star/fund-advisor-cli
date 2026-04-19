package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS fund_snapshots (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	fund_code TEXT NOT NULL,
	fund_name TEXT NOT NULL,
	trade_date TEXT NOT NULL,
	nav REAL NOT NULL,
	acc_nav REAL NOT NULL,
	day_change_pct REAL NOT NULL,
	estimate_nav REAL NOT NULL DEFAULT 0,
	estimate_change_pct REAL NOT NULL DEFAULT 0,
	source TEXT NOT NULL,
	created_at TEXT NOT NULL,
	UNIQUE(fund_code, trade_date)
);

CREATE TABLE IF NOT EXISTS portfolio_positions (
	fund_code TEXT PRIMARY KEY,
	fund_name TEXT NOT NULL,
	category TEXT NOT NULL,
	benchmark TEXT NOT NULL,
	role TEXT NOT NULL,
	status TEXT NOT NULL,
	protected INTEGER NOT NULL DEFAULT 0,
	dca_enabled INTEGER NOT NULL DEFAULT 0,
	account_value REAL NOT NULL,
	target_weight REAL NOT NULL,
	estimated_units REAL NOT NULL DEFAULT 0,
	updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS analysis_runs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_date TEXT NOT NULL,
	portfolio_value REAL NOT NULL,
	portfolio_drawdown_60d REAL NOT NULL DEFAULT 0,
	summary_json TEXT NOT NULL,
	report_json TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS market_pool_runs (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	run_date TEXT NOT NULL,
	summary_json TEXT NOT NULL,
	report_json TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS fund_signals (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	analysis_run_id INTEGER NOT NULL,
	fund_code TEXT NOT NULL,
	fund_name TEXT NOT NULL,
	action TEXT NOT NULL,
	score INTEGER NOT NULL,
	current_weight REAL NOT NULL,
	target_weight REAL NOT NULL,
	drift REAL NOT NULL,
	current_value REAL NOT NULL,
	return_20d REAL NOT NULL,
	return_60d REAL NOT NULL,
	return_120d REAL NOT NULL,
	latest_trade_date TEXT NOT NULL,
	reason TEXT NOT NULL,
	created_at TEXT NOT NULL,
	FOREIGN KEY(analysis_run_id) REFERENCES analysis_runs(id)
);
`

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{db: db}
	if err := store.Init(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Init() error {
	if err := s.configurePragmas(); err != nil {
		return err
	}
	if _, err := s.db.Exec(schema); err != nil {
		return err
	}
	if err := s.ensurePortfolioPositionColumn("protected", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensurePortfolioPositionColumn("dca_enabled", "INTEGER NOT NULL DEFAULT 0"); err != nil {
		return err
	}
	if err := s.ensureAnalysisRunColumn("report_json", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return s.ensureMarketPoolRunColumn("report_json", "TEXT NOT NULL DEFAULT ''")
}

func (s *Store) configurePragmas() error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
	}
	for _, pragma := range pragmas {
		if _, err := s.db.Exec(pragma); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ensurePortfolioPositionColumn(name, definition string) error {
	rows, err := s.db.Query(`PRAGMA table_info(portfolio_positions)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var columnName string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if columnName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(fmt.Sprintf("ALTER TABLE portfolio_positions ADD COLUMN %s %s", name, definition))
	return err
}

func (s *Store) ensureAnalysisRunColumn(name, definition string) error {
	rows, err := s.db.Query(`PRAGMA table_info(analysis_runs)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var columnName string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if columnName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(fmt.Sprintf("ALTER TABLE analysis_runs ADD COLUMN %s %s", name, definition))
	return err
}

func (s *Store) ensureMarketPoolRunColumn(name, definition string) error {
	rows, err := s.db.Query(`PRAGMA table_info(market_pool_runs)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var columnName string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &columnName, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if columnName == name {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = s.db.Exec(fmt.Sprintf("ALTER TABLE market_pool_runs ADD COLUMN %s %s", name, definition))
	return err
}

func (s *Store) UpsertPosition(position model.Position) error {
	_, err := s.db.Exec(`
INSERT INTO portfolio_positions (
	fund_code, fund_name, category, benchmark, role, status, protected, dca_enabled, account_value, target_weight, estimated_units, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(fund_code) DO UPDATE SET
	fund_name = excluded.fund_name,
	category = excluded.category,
	benchmark = excluded.benchmark,
	role = excluded.role,
	status = excluded.status,
	protected = excluded.protected,
	dca_enabled = excluded.dca_enabled,
	account_value = excluded.account_value,
	target_weight = excluded.target_weight,
	estimated_units = CASE WHEN excluded.estimated_units > 0 THEN excluded.estimated_units ELSE portfolio_positions.estimated_units END,
	updated_at = excluded.updated_at
`, position.FundCode, position.FundName, position.Category, position.Benchmark, position.Role, position.Status, boolToInt(position.Protected), boolToInt(position.DCAEnabled), position.AccountValue, position.TargetWeight, position.EstimatedUnits, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Store) SetEstimatedUnits(code string, units float64) error {
	_, err := s.db.Exec(`UPDATE portfolio_positions SET estimated_units = ?, updated_at = ? WHERE fund_code = ?`, units, time.Now().UTC().Format(time.RFC3339), code)
	return err
}

func (s *Store) ListPositions() ([]model.Position, error) {
	rows, err := s.db.Query(`SELECT fund_code, fund_name, category, benchmark, role, status, protected, dca_enabled, account_value, target_weight, estimated_units FROM portfolio_positions ORDER BY fund_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	positions := make([]model.Position, 0)
	for rows.Next() {
		var position model.Position
		var protected int
		var dcaEnabled int
		if err := rows.Scan(&position.FundCode, &position.FundName, &position.Category, &position.Benchmark, &position.Role, &position.Status, &protected, &dcaEnabled, &position.AccountValue, &position.TargetWeight, &position.EstimatedUnits); err != nil {
			return nil, err
		}
		position.Protected = protected == 1
		position.DCAEnabled = dcaEnabled == 1
		positions = append(positions, position)
	}
	return positions, rows.Err()
}

func (s *Store) SaveSnapshots(snapshots []model.FundSnapshot) error {
	if len(snapshots) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`
INSERT INTO fund_snapshots (
	fund_code, fund_name, trade_date, nav, acc_nav, day_change_pct, estimate_nav, estimate_change_pct, source, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(fund_code, trade_date) DO UPDATE SET
	fund_name = excluded.fund_name,
	nav = excluded.nav,
	acc_nav = excluded.acc_nav,
	day_change_pct = excluded.day_change_pct,
	estimate_nav = excluded.estimate_nav,
	estimate_change_pct = excluded.estimate_change_pct,
	source = excluded.source,
	created_at = excluded.created_at
`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, snapshot := range snapshots {
		_, err := stmt.Exec(
			snapshot.FundCode,
			snapshot.FundName,
			snapshot.TradeDate.Format("2006-01-02"),
			snapshot.NAV,
			snapshot.AccNAV,
			snapshot.DayChangePct,
			snapshot.EstimateNAV,
			snapshot.EstimateChangePct,
			snapshot.Source,
			time.Now().UTC().Format(time.RFC3339),
		)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (s *Store) LatestSnapshot(code string) (*model.FundSnapshot, error) {
	row := s.db.QueryRow(`
SELECT fund_code, fund_name, trade_date, nav, acc_nav, day_change_pct, estimate_nav, estimate_change_pct, source, created_at
FROM fund_snapshots WHERE fund_code = ? ORDER BY trade_date DESC LIMIT 1
`, code)
	return scanSnapshot(row)
}

func (s *Store) SnapshotHistory(code string, limit int) ([]model.FundSnapshot, error) {
	query := `
SELECT fund_code, fund_name, trade_date, nav, acc_nav, day_change_pct, estimate_nav, estimate_change_pct, source, created_at
FROM fund_snapshots WHERE fund_code = ? ORDER BY trade_date ASC`
	args := []any{code}
	if limit > 0 {
		query = `
SELECT fund_code, fund_name, trade_date, nav, acc_nav, day_change_pct, estimate_nav, estimate_change_pct, source, created_at
FROM (
  SELECT fund_code, fund_name, trade_date, nav, acc_nav, day_change_pct, estimate_nav, estimate_change_pct, source, created_at
  FROM fund_snapshots WHERE fund_code = ? ORDER BY trade_date DESC LIMIT ?
) ORDER BY trade_date ASC`
		args = append(args, limit)
	}
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := make([]model.FundSnapshot, 0)
	for rows.Next() {
		snapshot, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		history = append(history, *snapshot)
	}
	return history, rows.Err()
}

func (s *Store) SnapshotHistoryRange(code string, start, end time.Time) ([]model.FundSnapshot, error) {
	rows, err := s.db.Query(`
SELECT fund_code, fund_name, trade_date, nav, acc_nav, day_change_pct, estimate_nav, estimate_change_pct, source, created_at
FROM fund_snapshots
WHERE fund_code = ? AND trade_date >= ? AND trade_date <= ?
ORDER BY trade_date ASC
`, code, start.Format("2006-01-02"), end.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	history := make([]model.FundSnapshot, 0)
	for rows.Next() {
		snapshot, err := scanSnapshot(rows)
		if err != nil {
			return nil, err
		}
		history = append(history, *snapshot)
	}
	return history, rows.Err()
}

func (s *Store) SaveAnalysis(report model.AnalysisReport) (int64, error) {
	summaryJSON, err := json.Marshal(report.Summary)
	if err != nil {
		return 0, err
	}
	reportCopy := report
	reportCopy.RunID = 0
	reportJSON, err := json.Marshal(reportCopy)
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()
	result, err := tx.Exec(`
INSERT INTO analysis_runs (run_date, portfolio_value, portfolio_drawdown_60d, summary_json, report_json, created_at)
VALUES (?, ?, 0, ?, ?, ?)
`, report.Summary.RunDate.Format(time.RFC3339), report.Summary.PortfolioValue, string(summaryJSON), string(reportJSON), now)
	if err != nil {
		return 0, err
	}
	runID, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := s.saveSignalsTx(tx, runID, report.Signals); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return runID, nil
}

func (s *Store) SaveSignals(runID int64, signals []model.FundSignal) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := s.saveSignalsTx(tx, runID, signals); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) saveSignalsTx(tx *sql.Tx, runID int64, signals []model.FundSignal) error {
	stmt, err := tx.Prepare(`
INSERT INTO fund_signals (
	analysis_run_id, fund_code, fund_name, action, score, current_weight, target_weight, drift, current_value, return_20d, return_60d, return_120d, latest_trade_date, reason, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, signal := range signals {
		_, err := stmt.Exec(
			runID,
			signal.FundCode,
			signal.FundName,
			string(signal.Action),
			signal.Score,
			signal.CurrentWeight,
			signal.TargetWeight,
			signal.Drift,
			signal.CurrentValue,
			signal.Return20D,
			signal.Return60D,
			signal.Return120D,
			signal.LatestTradeDate.Format("2006-01-02"),
			signal.Reason,
			time.Now().UTC().Format(time.RFC3339),
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) LatestAnalysis() (*model.AnalysisReport, error) {
	row := s.db.QueryRow(`SELECT id, summary_json, report_json FROM analysis_runs ORDER BY id DESC LIMIT 1`)
	var runID int64
	var summaryJSON string
	var reportJSON string
	if err := row.Scan(&runID, &summaryJSON, &reportJSON); err != nil {
		return nil, err
	}
	if strings.TrimSpace(reportJSON) != "" {
		var report model.AnalysisReport
		if err := json.Unmarshal([]byte(reportJSON), &report); err == nil {
			report.RunID = runID
			if report.Summary.PortfolioName == "" && strings.TrimSpace(summaryJSON) != "" {
				_ = json.Unmarshal([]byte(summaryJSON), &report.Summary)
			}
			return &report, nil
		}
	}
	return s.loadLegacyAnalysis(runID, summaryJSON)
}

func (s *Store) SaveMarketPool(report model.MarketPoolReport) (int64, error) {
	summaryJSON, err := json.Marshal(report.Summary)
	if err != nil {
		return 0, err
	}
	reportCopy := report
	reportCopy.RunID = 0
	reportJSON, err := json.Marshal(reportCopy)
	if err != nil {
		return 0, err
	}
	result, err := s.db.Exec(`
INSERT INTO market_pool_runs (run_date, summary_json, report_json, created_at)
VALUES (?, ?, ?, ?)
`, report.Summary.RunDate.Format(time.RFC3339), string(summaryJSON), string(reportJSON), time.Now().UTC().Format(time.RFC3339))
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

func (s *Store) LatestMarketPool() (*model.MarketPoolReport, error) {
	row := s.db.QueryRow(`SELECT id, summary_json, report_json FROM market_pool_runs ORDER BY id DESC LIMIT 1`)
	var runID int64
	var summaryJSON string
	var reportJSON string
	if err := row.Scan(&runID, &summaryJSON, &reportJSON); err != nil {
		return nil, err
	}
	if strings.TrimSpace(reportJSON) == "" {
		var summary model.MarketPoolSummary
		if err := json.Unmarshal([]byte(summaryJSON), &summary); err != nil {
			return nil, err
		}
		return &model.MarketPoolReport{RunID: runID, Summary: summary}, nil
	}
	var report model.MarketPoolReport
	if err := json.Unmarshal([]byte(reportJSON), &report); err != nil {
		return nil, err
	}
	report.RunID = runID
	if report.Summary.RunDate.IsZero() && strings.TrimSpace(summaryJSON) != "" {
		_ = json.Unmarshal([]byte(summaryJSON), &report.Summary)
	}
	return &report, nil
}

func (s *Store) loadLegacyAnalysis(runID int64, summaryJSON string) (*model.AnalysisReport, error) {
	var summary model.AnalysisSummary
	if err := json.Unmarshal([]byte(summaryJSON), &summary); err != nil {
		return nil, err
	}
	rows, err := s.db.Query(`
SELECT fund_code, fund_name, action, score, current_weight, target_weight, drift, current_value, return_20d, return_60d, return_120d, latest_trade_date, reason, created_at
FROM fund_signals WHERE analysis_run_id = ? ORDER BY 
CASE action
  WHEN 'REPLACE_WATCH' THEN 1
  WHEN 'REDUCE' THEN 2
  WHEN 'BUY' THEN 3
  WHEN 'PAUSE_BUY' THEN 4
  ELSE 5 END,
fund_code ASC
`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	signals := make([]model.FundSignal, 0)
	for rows.Next() {
		var signal model.FundSignal
		var action string
		var latestDate string
		var createdAt string
		if err := rows.Scan(&signal.FundCode, &signal.FundName, &action, &signal.Score, &signal.CurrentWeight, &signal.TargetWeight, &signal.Drift, &signal.CurrentValue, &signal.Return20D, &signal.Return60D, &signal.Return120D, &latestDate, &signal.Reason, &createdAt); err != nil {
			return nil, err
		}
		signal.Action = model.Action(action)
		signal.LatestTradeDate, _ = time.Parse("2006-01-02", latestDate)
		signal.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		signals = append(signals, signal)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return &model.AnalysisReport{RunID: runID, Summary: summary, Signals: signals}, nil
}

func scanSnapshot(scanner interface{ Scan(dest ...any) error }) (*model.FundSnapshot, error) {
	var snapshot model.FundSnapshot
	var tradeDate string
	var createdAt string
	if err := scanner.Scan(&snapshot.FundCode, &snapshot.FundName, &tradeDate, &snapshot.NAV, &snapshot.AccNAV, &snapshot.DayChangePct, &snapshot.EstimateNAV, &snapshot.EstimateChangePct, &snapshot.Source, &createdAt); err != nil {
		return nil, err
	}
	snapshot.TradeDate, _ = time.Parse("2006-01-02", tradeDate)
	snapshot.CreatedAt, _ = time.Parse(time.RFC3339, strings.TrimSpace(createdAt))
	return &snapshot, nil
}

func (s *Store) DebugCounts() (string, error) {
	var snapshots int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM fund_snapshots`).Scan(&snapshots); err != nil {
		return "", err
	}
	var positions int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM portfolio_positions`).Scan(&positions); err != nil {
		return "", err
	}
	return fmt.Sprintf("snapshots=%d positions=%d", snapshots, positions), nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
