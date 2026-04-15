---
name: fund-advisor-cli
description: Go-based CLI that tracks a fund portfolio daily, stores NAV history in SQLite, and generates rules-based buy/hold/reduce/replace-watch advice.
---

# Fund Advisor CLI

Use this skill when you want a local CLI for:

- tracking mutual fund / ETF feeder / QDII portfolios
- fetching daily NAV history
- generating rule-based investment suggestions
- evaluating candidate funds for replacement
- producing terminal or Markdown reports

## Workflow

1. Initialize a config

```bash
go run ./cmd/fundcli init --config ./configs/portfolio.yaml
```

2. Review and edit the generated portfolio config and candidate pool

```bash
sed -n '1,260p' ./configs/portfolio.yaml
```

3. Fetch data and generate advice

```bash
go run ./cmd/fundcli run --config ./configs/portfolio.yaml --format table
```

4. Save a Markdown daily report

```bash
go run ./cmd/fundcli report --config ./configs/portfolio.yaml --format markdown --output ./reports/daily.md
```

## Outputs

- SQLite database under `./data/`
- latest actions in terminal output
- optional Markdown or JSON report files

## Current assumptions

- Eastmoney provides the NAV history feed
- rules are allocation-driven rather than predictive
- replacement suggestions come from a manually curated candidate list
