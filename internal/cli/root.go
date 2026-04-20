package cli

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/docs"
	"github.com/derekdong-star/fund-advisor-cli/internal/ledger"
	"github.com/derekdong-star/fund-advisor-cli/internal/llm"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
	"github.com/derekdong-star/fund-advisor-cli/internal/report"
	"github.com/derekdong-star/fund-advisor-cli/internal/service"
)

var fetchForDocsPublish = func(ctx context.Context, svc *service.Service, days int) error {
	return svc.Fetch(ctx, days)
}

var analyzeForDocsPublish = func(svc *service.Service) (*model.AnalysisReport, error) {
	return svc.Analyze()
}

var buildMarketPoolForDocsPublish = func(ctx context.Context, svc *service.Service, days int) (*model.MarketPoolReport, error) {
	return svc.BuildMarketPool(ctx, days)
}

func NewRootCmd() *cobra.Command {
	var configPath string
	root := &cobra.Command{
		Use:   "fundcli",
		Short: "Daily fund portfolio advisor",
	}
	defaultConfigPath := filepath.Join("configs", "portfolio.yaml")
	root.PersistentFlags().StringVar(&configPath, "config", defaultConfigPath, "path to portfolio config")
	root.AddCommand(newInitCmd(&configPath))
	root.AddCommand(newValidateCmd(&configPath))
	root.AddCommand(newFetchCmd(&configPath))
	root.AddCommand(newAnalyzeCmd(&configPath))
	root.AddCommand(newReportCmd(&configPath))
	root.AddCommand(newDCAPlanCmd(&configPath))
	root.AddCommand(newRunCmd(&configPath))
	root.AddCommand(newLedgerCmd(&configPath))
	root.AddCommand(newPositionsCmd(&configPath))
	root.AddCommand(newMarketPoolCmd(&configPath))
	root.AddCommand(newBackfillCmd(&configPath))
	root.AddCommand(newBacktestCmd(&configPath))
	root.AddCommand(newDocsCmd(&configPath))
	root.AddCommand(newLLMCmd(&configPath))
	return root
}

func newInitCmd(configPath *string) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Generate a starter portfolio config",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.WriteExample(*configPath, force); err != nil {
				return err
			}
			dataDir := filepath.Join(filepath.Dir(*configPath), "..", "data")
			if err := os.MkdirAll(dataDir, 0o755); err != nil {
				return err
			}
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "wrote example config to %s\n", *configPath)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing config")
	return cmd
}

func newValidateCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate config structure and portfolio weights",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			if err := svc.Validate(); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "config is valid: %s\n", *configPath)
			return err
		},
	}
}

func newFetchCmd(configPath *string) *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch latest fund NAV history into SQLite",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			if err := svc.Fetch(context.Background(), days); err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "fetch completed")
			return err
		},
	}
	cmd.Flags().IntVar(&days, "days", 180, "number of recent trading days to store")
	return cmd
}

func newAnalyzeCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "analyze",
		Short: "Analyze stored data and generate actions",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			reportData, err := svc.Analyze()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "analysis completed, run_id=%d\n", reportData.RunID)
			return err
		},
	}
}

func newReportCmd(configPath *string) *cobra.Command {
	var format string
	var output string
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Render a report from the latest saved analysis",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			reportData, err := svc.LatestReport()
			if err != nil {
				return err
			}
			rendered, err := report.Render(*reportData, format)
			if err != nil {
				return err
			}
			if output != "" {
				if err := writeOutput(output, rendered); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "report format: table|markdown|json")
	cmd.Flags().StringVar(&output, "output", "", "write rendered report to a file")
	return cmd
}

func newRunCmd(configPath *string) *cobra.Command {
	var format string
	var days int
	var output string
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Fetch, analyze, and print a report",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			var buffer writeBuffer
			if err := svc.Run(context.Background(), days, format, &buffer); err != nil {
				return err
			}
			if output != "" {
				if err := writeOutput(output, buffer.String()); err != nil {
					return err
				}
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), buffer.String())
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "report format: table|markdown|json")
	cmd.Flags().IntVar(&days, "days", 180, "number of recent trading days to store")
	cmd.Flags().StringVar(&output, "output", "", "write rendered report to a file")
	return cmd
}

func newDCAPlanCmd(configPath *string) *cobra.Command {
	var format string
	var output string
	cmd := &cobra.Command{
		Use:   "dca-plan",
		Short: "Generate the current DCA plan",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			plan, err := svc.DCAPlan()
			if err != nil {
				return err
			}
			rendered, err := report.RenderDCAPlan(*plan, format)
			if err != nil {
				return err
			}
			if output != "" {
				if err := writeOutput(output, rendered); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "dca plan format: table|markdown|json")
	cmd.Flags().StringVar(&output, "output", "", "write rendered dca plan to a file")
	return cmd
}

func newLedgerCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ledger",
		Short: "Manage manual holdings snapshot and trade ledger files",
	}
	cmd.AddCommand(newLedgerInitCmd(configPath))
	cmd.AddCommand(newLedgerCheckCmd(configPath))
	cmd.AddCommand(newLedgerAddTradeCmd(configPath))
	return cmd
}

func newLedgerInitCmd(configPath *string) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create holdings snapshot and trades csv templates",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			paths, err := ledger.WriteTemplate(svc.Config(), force)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "ledger templates created: snapshot=%s trades=%s\n", paths.SnapshotPath, paths.TradesCSVPath)
			return err
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing ledger template files")
	return cmd
}

func newLedgerAddTradeCmd(configPath *string) *cobra.Command {
	var fundCode string
	var side string
	var amount float64
	var nav float64
	var units float64
	var fee float64
	var note string
	var tradeDate string
	cmd := &cobra.Command{
		Use:   "add-trade",
		Short: "Append one manual buy/sell record to the ledger csv",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			parsedDate := time.Now().In(time.FixedZone("UTC+8", 8*3600))
			if strings.TrimSpace(tradeDate) != "" {
				parsedDate, err = time.Parse("2006-01-02", strings.TrimSpace(tradeDate))
				if err != nil {
					return fmt.Errorf("parse trade date: %w", err)
				}
			}
			paths, trade, err := ledger.AppendTrade(svc.Config(), ledger.TradeRecord{
				TradeDate: parsedDate,
				FundCode:  fundCode,
				Side:      side,
				Amount:    amount,
				NAV:       nav,
				Units:     units,
				Fee:       fee,
				Note:      note,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "ledger trade added: %s %s %s amount=%.2f nav=%.4f units=%.4f file=%s\n", trade.TradeDate.Format("2006-01-02"), trade.FundCode, trade.Side, trade.Amount, trade.NAV, trade.Units, paths.TradesCSVPath)
			return err
		},
	}
	cmd.Flags().StringVar(&fundCode, "fund-code", "", "fund code")
	cmd.Flags().StringVar(&side, "side", "BUY", "trade side: BUY or SELL")
	cmd.Flags().Float64Var(&amount, "amount", 0, "trade amount in CNY")
	cmd.Flags().Float64Var(&nav, "nav", 0, "trade NAV")
	cmd.Flags().Float64Var(&units, "units", 0, "trade units; optional when amount and nav are provided")
	cmd.Flags().Float64Var(&fee, "fee", 0, "trade fee in CNY")
	cmd.Flags().StringVar(&note, "note", "", "optional trade note")
	cmd.Flags().StringVar(&tradeDate, "date", "", "trade date in YYYY-MM-DD; defaults to today in Asia/Shanghai")
	_ = cmd.MarkFlagRequired("fund-code")
	_ = cmd.MarkFlagRequired("amount")
	_ = cmd.MarkFlagRequired("nav")
	return cmd
}

func newLedgerCheckCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "Validate the manual holdings snapshot and trade ledger",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			result, err := ledger.Check(svc.Config())
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "ledger check ok: snapshot_as_of=%s trades=%d funds=%d\n", result.SnapshotAsOf, result.TradeCount, len(result.Positions))
			return err
		},
	}
}

func newPositionsCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "positions",
		Short: "Reconcile derived positions from the manual ledger",
	}
	cmd.AddCommand(newPositionsReconcileCmd(configPath))
	return cmd
}

func newPositionsReconcileCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "reconcile",
		Short: "Rebuild estimated holdings from snapshot plus trade ledger",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			result, err := svc.ReconcileHoldings()
			if err != nil {
				return err
			}
			lines := make([]string, 0, len(result.Positions))
			for _, holding := range result.Positions {
				lines = append(lines, fmt.Sprintf("%s units=%.4f cost=%.2f", holding.FundCode, holding.Units, holding.TotalCost))
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "positions reconciled: snapshot_as_of=%s trades=%d\n%s\n", result.SnapshotAsOf, result.TradeCount, strings.Join(lines, "\n"))
			return err
		},
	}
}

func newMarketPoolCmd(configPath *string) *cobra.Command {
	var format string
	var output string
	var days int
	cmd := &cobra.Command{
		Use:   "market-pool",
		Short: "Build a stable market-wide candidate pool",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			pool, err := svc.BuildMarketPool(cmd.Context(), days)
			if err != nil {
				return err
			}
			rendered, err := report.RenderMarketPool(*pool, format)
			if err != nil {
				return err
			}
			if output != "" {
				if err := writeOutput(output, rendered); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return err
		},
	}
	cmd.Flags().StringVar(&format, "format", "table", "market pool format: table|markdown|json")
	cmd.Flags().StringVar(&output, "output", "", "write rendered market pool to a file")
	cmd.Flags().IntVar(&days, "days", 300, "number of recent trading days to evaluate")
	return cmd
}

func newBackfillCmd(configPath *string) *cobra.Command {
	var days int
	cmd := &cobra.Command{
		Use:   "backfill",
		Short: "Fetch recent historical NAV data",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			if err := svc.Fetch(context.Background(), days); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "backfill completed for %d days\n", days)
			return err
		},
	}
	cmd.Flags().IntVar(&days, "days", 365, "number of recent trading days to store")
	return cmd
}

type writeBuffer struct {
	data []byte
}

func (b *writeBuffer) Write(p []byte) (int, error) {
	b.data = append(b.data, p...)
	return len(p), nil
}

func (b *writeBuffer) String() string {
	return string(b.data)
}

func writeOutput(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}

func newBacktestCmd(configPath *string) *cobra.Command {
	var days int
	var rebalanceEvery int
	var format string
	var output string
	cmd := &cobra.Command{
		Use:   "backtest",
		Short: "Replay the strategy on historical NAV data",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			reportData, err := svc.Backtest(days, rebalanceEvery)
			if err != nil {
				return err
			}
			rendered, err := report.RenderBacktest(*reportData, format)
			if err != nil {
				return err
			}
			if output != "" {
				if err := writeOutput(output, rendered); err != nil {
					return err
				}
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), rendered)
			return err
		},
	}
	cmd.Flags().IntVar(&days, "days", 120, "number of overlapping trading days to replay")
	cmd.Flags().IntVar(&rebalanceEvery, "rebalance-every", 20, "re-run strategy every N trading days")
	cmd.Flags().StringVar(&format, "format", "table", "backtest format: table|markdown|json")
	cmd.Flags().StringVar(&output, "output", "", "write rendered report to a file")
	return cmd
}

func newLLMCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "llm",
		Short: "Run LLM connectivity and response checks",
	}
	cmd.AddCommand(newLLMPingCmd(configPath))
	return cmd
}

func newLLMPingCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "ping",
		Short: "Ping the configured OpenAI-compatible LLM provider",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(*configPath)
			if err != nil {
				return err
			}
			client := llm.NewClient(cfg.LLM)
			req := llm.CandidateRerankRequest{
				PortfolioName: cfg.Portfolio.Name,
				RunDate:       time.Now().UTC(),
				Candidates: []llm.CandidateRerankInput{
					{FundCode: "PING_A", FundName: "Ping Candidate A", Category: "test", Role: "core", Benchmark: "benchmark_a", Score: 6, Return20D: 0.02, Return60D: 0.04, Return120D: 0.05, ReplaceFor: []string{"Weak Holding"}, RuleReason: "ping candidate A"},
					{FundCode: "PING_B", FundName: "Ping Candidate B", Category: "test", Role: "core", Benchmark: "benchmark_b", Score: 7, Return20D: 0.03, Return60D: 0.06, Return120D: 0.08, ReplaceFor: []string{"Weak Holding"}, RuleReason: "ping candidate B"},
				},
			}
			resp, err := client.Ping(cmd.Context(), req)
			if err != nil {
				return err
			}
			if err := llm.ValidateCandidateRerankResponseForCLI(req, resp); err != nil {
				return err
			}
			topReason := ""
			if len(resp.Rankings) > 0 {
				topReason = resp.Rankings[0].Reason
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "llm ping ok: provider=%s base_url=%s model=%s rankings=%d top=%s top_reason=%q\n", cfg.LLM.Provider, cfg.LLM.BaseURL, cfg.LLM.Model, len(resp.Rankings), resp.Rankings[0].FundCode, topReason)
			return err
		},
	}
}

func newDocsCmd(configPath *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "docs",
		Short: "Export GitBook-ready documentation artifacts",
	}
	cmd.AddCommand(newDocsExportCmd(configPath))
	cmd.AddCommand(newDocsIndexCmd(configPath))
	cmd.AddCommand(newDocsValidateCmd(configPath))
	cmd.AddCommand(newDocsPublishCmd(configPath))
	return cmd
}

func newDocsExportCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "export",
		Short: "Export latest reports into the GitBook docs tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			input, err := buildDocsPublishInput(svc)
			if err != nil {
				return err
			}
			docSvc := docs.NewService(svc.Config())
			result, err := docSvc.Export(input)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "docs exported to %s\n", result.DocsRoot)
			return err
		},
	}
}

func newDocsIndexCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "index",
		Short: "Generate GitBook index files such as README and SUMMARY",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			input, err := buildDocsPublishInput(svc)
			if err != nil {
				return err
			}
			docSvc := docs.NewService(svc.Config())
			result, err := docSvc.Export(input)
			if err != nil {
				return err
			}
			if err := docSvc.BuildIndex(input, result); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "docs index generated at %s\n", result.DocsRoot)
			return err
		},
	}
}

func newDocsValidateCmd(configPath *string) *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate the generated GitBook docs tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()
			docSvc := docs.NewService(svc.Config())
			if err := docSvc.Validate(); err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "docs tree is valid: %s\n", svc.Config().Publishing.GitBook.DocsRoot)
			return err
		},
	}
}

func newDocsPublishCmd(configPath *string) *cobra.Command {
	var refresh bool
	var days int
	cmd := &cobra.Command{
		Use:   "publish",
		Short: "Export, index, and validate GitBook docs artifacts",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := service.New(*configPath)
			if err != nil {
				return err
			}
			defer svc.Close()

			var input docs.PublishInput
			if refresh {
				input, err = buildDocsPublishInputAfterRefresh(cmd.Context(), svc, days)
			} else {
				input, err = buildDocsPublishInput(svc)
			}
			if err != nil {
				return err
			}
			docSvc := docs.NewService(svc.Config())
			result, err := docSvc.Publish(input)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "docs published to %s\n", result.DocsRoot)
			return err
		},
	}
	cmd.Flags().BoolVar(&refresh, "refresh", false, "fetch fresh NAV data and analyze before publishing docs")
	cmd.Flags().IntVar(&days, "days", 180, "number of recent trading days to fetch when --refresh is enabled")
	return cmd
}

func buildDocsPublishInput(svc *service.Service) (docs.PublishInput, error) {
	analysis, plan, err := svc.CurrentAnalysisAndDCAPlan()
	if err != nil {
		return docs.PublishInput{}, err
	}
	return buildDocsPublishInputWithAnalysis(svc, analysis, plan), nil
}

func buildDocsPublishInputAfterRefresh(ctx context.Context, svc *service.Service, days int) (docs.PublishInput, error) {
	if err := fetchForDocsPublish(ctx, svc, days); err != nil {
		return docs.PublishInput{}, err
	}
	analysis, err := analyzeForDocsPublish(svc)
	if err != nil {
		return docs.PublishInput{}, err
	}
	input := buildDocsPublishInputWithAnalysis(svc, analysis, analysis.DCAPlan)
	marketPool, err := buildMarketPoolForDocsPublish(ctx, svc, days)
	if err == nil {
		input.MarketPool = marketPool
		input.MarketPoolError = ""
	} else {
		input.MarketPool = nil
		input.MarketPoolError = err.Error()
	}
	return input, nil
}

func buildDocsPublishInputWithAnalysis(svc *service.Service, analysis *model.AnalysisReport, plan *model.DCAPlanReport) docs.PublishInput {
	input := docs.PublishInput{Analysis: analysis, Plan: plan}
	marketPool, err := svc.LatestMarketPool()
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			input.MarketPoolError = err.Error()
		}
	} else {
		input.MarketPool = marketPool
	}
	if svc.Config().Publishing.GitBook.IncludeBacktest {
		backtest, err := svc.Backtest(
			svc.Config().Publishing.GitBook.BacktestDays,
			svc.Config().Publishing.GitBook.BacktestRebalanceEvery,
		)
		if err != nil {
			input.BacktestError = err.Error()
		} else {
			input.Backtest = backtest
		}
	}
	return input
}
