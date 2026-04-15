# Fund Advisor CLI

Go CLI for daily mutual fund monitoring, NAV collection, and rule-based portfolio advice.

## What it does

- stores your portfolio config in YAML
- fetches historical NAV data from Eastmoney
- stores fund snapshots locally in SQLite
- estimates portfolio drift from target weights
- emits daily actions: `BUY`, `HOLD`, `PAUSE_BUY`, `REDUCE`, `REPLACE_WATCH`
- evaluates candidate funds for replacement suggestions and stores them with each analysis run
- writes Markdown or JSON reports to files when needed
- emits a recommended action list with swap / reduce / buy amounts

## Commands

```bash
go run ./cmd/fundcli init --config ./configs/portfolio.yaml
go run ./cmd/fundcli validate --config ./configs/portfolio.yaml
go run ./cmd/fundcli fetch --config ./configs/portfolio.yaml --days 180
go run ./cmd/fundcli analyze --config ./configs/portfolio.yaml
go run ./cmd/fundcli report --config ./configs/portfolio.yaml --format table
go run ./cmd/fundcli report --config ./configs/portfolio.yaml --format markdown --output ./reports/latest.md
go run ./cmd/fundcli run --config ./configs/portfolio.yaml --format markdown --output ./reports/daily.md
go run ./cmd/fundcli backtest --config ./configs/portfolio.yaml --days 120 --rebalance-every 20
```

## Config

The example portfolio is at [configs/portfolio.example.yaml](/Users/dhw/Documents/derekdong-star/fund-advisor-cli/configs/portfolio.example.yaml).

Key fields:

- `account_value`: current capital assigned to the fund
- `target_weight`: desired long-term weight in the portfolio
- `category`: used for overlap and peer checks
- `role`: `core`, `satellite`, `hedge`, or `stabilizer`
- `candidates`: optional watchlist used for replacement suggestions
- candidate metadata: supports `expense_ratio`, `fund_size_yi`, `established_years`, `is_index`, `tags`

## Strategy model

Signals are rules-based, not predictive.

- `BUY`: underweight versus target and no major health warnings
- `REDUCE`: overweight versus target or concentration too high
- `PAUSE_BUY`: health score is elevated but not yet a replace case
- `REPLACE_WATCH`: repeated weakness, duplicate exposure, or stale data risk
- `HOLD`: within acceptable allocation and health ranges

Candidate suggestions are generated when a held fund enters `REPLACE_WATCH`, `PAUSE_BUY`, or a weak `REDUCE` state, and a candidate fund has the same category or role plus acceptable recent strength. The engine can also filter candidates by benchmark match, expense ratio, fund size, established years, and whether a core replacement is index-based.

When multiple weak holdings compete for the same candidate, the recommendation engine assigns that candidate deterministically: `REPLACE_WATCH` first, then `REDUCE`, then `PAUSE_BUY`, and within the same priority it prefers the larger suggested replacement amount.

`report` renders the latest saved analysis snapshot, so it stays consistent with the last `analyze` or `run` execution. Daily reports now include a recommended action list with suggested source fund, target fund, portfolio weight change, and amount.

## Notes

- SQLite now uses `WAL` mode plus a `busy_timeout`, which reduces transient `database is locked` errors during repeated CLI runs.
- This tool is designed for daily NAV products, not intraday trading.
- QDII and ETF feeder funds can have delayed NAV updates.
- The first fetch infers estimated units from `account_value / latest_nav`.


## Backtest

`backtest` replays the current rule set on overlapping historical trading days from the local SQLite snapshot store. It compares the strategy against a simple buy-and-hold benchmark built from the starting portfolio weights, only uses cash raised by prior sells, and currently ignores fees, slippage, and taxes.
