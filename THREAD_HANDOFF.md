# Thread Handoff

## Project
- Repo path: `/Users/dhw/Documents/derekdong-star/fund-advisor-cli`
- Git remote: `git@github.com:derekdong-star/fund-advisor-cli.git`
- Go module: `github.com/derekdong-star/fund-advisor-cli`
- Current branch: `main`
- Latest pushed commit: `9c284d7` (`Initial import of fund advisor CLI`)

## What This Project Does
A Go CLI for fund portfolio monitoring, NAV fetching, rule-based analysis, reporting, and backtesting.

Main commands already implemented:
- `init`
- `validate`
- `fetch`
- `analyze`
- `report`
- `run`
- `backfill`
- `backtest`

## User Preference That Changed The Strategy
The user does **not** like frequent rebalancing or frequent fund rotation.

Current preference is:
- prefer long-term holding
- prefer DCA into a few selected funds
- avoid routine sell / swap actions
- keep tracking and holding `景顺长城沪港深精选股票A`
- treat that fund as a long-term conviction position

The user explicitly said they have followed `景顺长城沪港深精选股票A` for years and have already earned about `3W` on it.

## Current Strategy Direction
The strategy has been adjusted toward `low_turnover` behavior.

Important behavior now:
- `protected` holdings are not routinely reduced
- low-turnover mode raises replacement thresholds
- many former `REDUCE` cases are now downgraded to `PAUSE_BUY`
- non-DCA funds that are under target are usually observed, not actively topped up
- buy recommendations are focused on `dca_enabled` funds
- DCA buy sizing is capped by `monthly_dca_amount`

## Important Current Config Decisions
In `configs/portfolio.yaml`:
- `turnover.mode = low_turnover`
- `turnover.min_swap_score = 7`
- `turnover.max_protected_reduce_weight = 0.22`
- `turnover.monthly_dca_amount = 5000`
- `turnover.prefer_dca = true`

Important fund flags:
- `000979` / `景顺长城沪港深精选股票A`
  - `protected: true`
  - `dca_enabled: true`
- `050025` / `博时标普500ETF联接A`
  - `dca_enabled: true`
- `009052` / `易方达中证红利ETF联接发起式C`
  - `dca_enabled: true`
- `021457` / `易方达恒生红利低波ETF联接A`
  - `dca_enabled: true`

## Current Verified Output
Latest meaningful saved analysis is `run_id=20` in `data/fundcli.db`.

Its output matches the user's current preference much better:
- `景顺长城沪港深精选股票A` => `HOLD`
- no swap recommendation
- no reduce recommendation
- mostly `HOLD` or `PAUSE_BUY`
- only two DCA buy suggestions remain:
  - `易方达中证红利ETF联接发起式C`: `5000`
  - `博时标普500ETF联接A`: `5000`

The generated `reports/daily.md` has already been refreshed to reflect this low-turnover result.

## Backtest Status
Backtest functionality exists and has already been fixed compared with earlier broken versions.

Historical conclusion so far:
- the ruleset shows only weak / unstable edge historically
- no strong evidence of robust alpha yet
- backtest should be treated as validation tooling, not proof that the strategy is strong

## Known Technical Work Already Done
Implemented and verified:
- module path migrated to `github.com/derekdong-star/fund-advisor-cli`
- internal imports updated to new module path
- SQLite persistence includes `protected` and `dca_enabled`
- low-turnover logic added in strategy engine
- candidate evaluation restricted more heavily in low-turnover mode
- tests updated and passing under the new module path

## Last Verification State
Verified in the new repo before handoff:
- `go test ./...` passes
- repo is pushed to GitHub on `main`
- README link paths were corrected for the new repo location
- `.gitignore` excludes SQLite temp files like `data/*.db-shm` and `data/*.db-wal`

## Good Next Steps
If continuing product work, the highest-value next tasks are:
1. Make DCA execution rules more explicit:
   - how often to invest
   - how to split fixed monthly budget across 2-4 core funds
   - whether to allow pause-on-risk rules
2. Improve report wording:
   - separate `hold / stop adding / continue DCA`
   - make the output read more like an investor playbook, less like a generic rebalance engine
3. Improve backtest realism:
   - periodic cash contributions
   - DCA schedule simulation
   - fees / slippage assumptions
   - turnover statistics
4. Decide whether `景顺长城沪港深精选股票A` should receive active DCA or just protected hold behavior
5. Consider adding a dedicated command for "monthly DCA plan"

## Important Caution
Do not regress the strategy back toward high-turnover swap-heavy behavior unless the user explicitly asks for that.
The current accepted direction is low-turnover, conviction-holding-friendly, and DCA-first.
