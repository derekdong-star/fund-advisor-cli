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
- generates a monthly DCA plan that separates continue-invest from pause-for-risk cases

## Commands

```bash
go run ./cmd/fundcli init --config ./configs/portfolio.yaml
go run ./cmd/fundcli validate --config ./configs/portfolio.yaml
go run ./cmd/fundcli fetch --config ./configs/portfolio.yaml --days 180
go run ./cmd/fundcli analyze --config ./configs/portfolio.yaml
go run ./cmd/fundcli report --config ./configs/portfolio.yaml --format table
go run ./cmd/fundcli report --config ./configs/portfolio.yaml --format markdown --output ./reports/latest.md
go run ./cmd/fundcli dca-plan --config ./configs/portfolio.yaml --format markdown --output ./reports/dca-plan.md
go run ./cmd/fundcli market-pool --config ./configs/portfolio.yaml --format markdown --output ./reports/market-pool.md
go run ./cmd/fundcli run --config ./configs/portfolio.yaml --format markdown --output ./reports/daily.md
go run ./cmd/fundcli docs publish --config ./configs/portfolio.yaml
go run ./cmd/fundcli docs publish --config ./configs/portfolio.local.yaml --refresh --days 180
go run ./cmd/fundcli docs publish --config ./configs/portfolio.yaml --refresh --days 180
go run ./cmd/fundcli backtest --config ./configs/portfolio.yaml --days 120 --rebalance-every 20
```

## Config

The example portfolio is at [configs/portfolio.example.yaml](/Users/dhw/Documents/derekdong-star/fund-advisor-cli/configs/portfolio.example.yaml).

Use [configs/portfolio.local.yaml](/Users/dhw/Documents/derekdong-star/fund-advisor-cli/configs/portfolio.local.yaml) for local GitBook preview runs. It writes generated docs to `tmp/gitbook-local`, which is ignored by git and avoids conflicts with tracked publish artifacts.

Key fields:

- `account_value`: current capital assigned to the fund
- `target_weight`: desired long-term weight in the portfolio
- `category`: used for overlap and peer checks
- `role`: `core`, `satellite`, `hedge`, or `stabilizer`
- `candidates`: optional watchlist used for replacement suggestions
- candidate metadata: supports `expense_ratio`, `fund_size_yi`, `established_years`, `is_index`, `tags`
- turnover DCA settings: `monthly_dca_amount`, `min_dca_fund_amount`, `dca_frequency`, `max_dca_funds`, `pause_dca_on_risk`
- stable market pool settings: `market_pool.selection_count`, `max_funds_per_theme`, `min_return_120d`, `min_return_250d`, `max_drawdown_120d`, `retention_score_gap`
- GitBook publishing: `publishing.gitbook.docs_root`, `project_directory`, `generate_homepage`, `generate_summary`, `include_backtest`, `hide_backtest_when_unavailable`, `backtest_days`, `backtest_rebalance_every`, `retain_days`
- GitBook sync IDs: `publishing.gitbook.organization_id`, `site_id`, `space_id`

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

`dca-plan` renders a current contribution plan from live portfolio state. It only considers `dca_enabled` funds, can cap the number of active DCA targets, skips allocations below `min_dca_fund_amount`, and by default pauses monthly contributions for funds currently marked `PAUSE_BUY`, `REDUCE`, or `REPLACE_WATCH`.

`market-pool` builds a separate stable buy-candidate pool from the broader market. It scans fixed themes, applies medium-term return and drawdown thresholds, and uses a retention rule so the previous winner for a theme stays in place when the score gap is small enough. The goal is to keep 5-10 solid candidates instead of rotating the list every day.

## Notes

- SQLite now uses `WAL` mode plus a `busy_timeout`, which reduces transient `database is locked` errors during repeated CLI runs.
- This tool is designed for daily NAV products, not intraday trading.
- QDII and ETF feeder funds can have delayed NAV updates.
- The first fetch infers estimated units from `account_value / latest_nav`.

## Backtest

`backtest` replays the current rule set on overlapping historical trading days from the local SQLite snapshot store. It compares the strategy against a simple buy-and-hold benchmark built from the starting portfolio weights, only uses cash raised by prior sells, and currently ignores fees, slippage, and taxes.

## GitBook Export

`docs publish` builds a GitBook-ready tree under `publishing.gitbook.docs_root`. When you add `--refresh`, it now refreshes NAV data, reruns analysis, and rebuilds the stable market pool before exporting docs.

Generated artifacts include:

- `.gitbook.yaml`
- `README.md`
- `SUMMARY.md`
- `latest/daily.md`
- `latest/dca-plan.md`
- `latest/market-pool.md` when a market pool snapshot is available
- `latest/backtest.md` when enabled and not hidden for unavailable data
- `archive/YYYY/MM/DD/...` plus year/month/day index pages
- `strategy/overview.md`
- `about/risk.md`

The intended workflow is to point GitBook Git Sync at the configured `project_directory` and let GitBook sync the generated Markdown.

Recommended publish flow:

1. Run `go run ./cmd/fundcli docs publish --config ./configs/portfolio.local.yaml --refresh --days 180` for local preview. This writes to `tmp/gitbook-local/` so local checks do not modify tracked GitBook files.
2. Let GitHub Actions run `go run ./cmd/fundcli docs publish --config ./configs/portfolio.yaml --refresh --days 180` for the real publish output under `docs/gitbook/`.
3. The workflow syncs that generated `docs/gitbook/` tree to the `gitbook-publish` branch instead of committing it back to `main`.
4. In GitBook, connect Git Sync to branch `gitbook-publish` and set the content root to `docs/gitbook`.
5. Keep `organization_id`, `site_id`, and `space_id` in config as deployment metadata for future API-based publishing, but the current implementation only needs Git Sync.

The repository includes a runnable workflow at `.github/workflows/publish-gitbook.yml`. On the first run it creates `gitbook-publish` as a docs-only orphan branch, then syncs only `docs/gitbook` into that branch.

It runs on manual trigger or at `14:00` Asia/Shanghai time on weekdays, intended to generate the same-day decision report after the trading session has enough same-day context. The workflow still publishes with `--refresh`, logs the Asia/Shanghai trigger time, and only pushes updated GitBook artifacts to `gitbook-publish` when the generated content changed.

If you set `publishing.gitbook.retain_days`, old archive day folders older than that rolling window are pruned during `docs publish`. `0` keeps the full archive.
