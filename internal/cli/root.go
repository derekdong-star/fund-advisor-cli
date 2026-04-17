package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/docs"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
	"github.com/derekdong-star/fund-advisor-cli/internal/report"
	"github.com/derekdong-star/fund-advisor-cli/internal/service"
)

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
	root.AddCommand(newBackfillCmd(&configPath))
	root.AddCommand(newBacktestCmd(&configPath))
	root.AddCommand(newDocsCmd(&configPath))
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
	if err := svc.Fetch(ctx, days); err != nil {
		return docs.PublishInput{}, err
	}
	analysis, err := svc.Analyze()
	if err != nil {
		return docs.PublishInput{}, err
	}
	return buildDocsPublishInputWithAnalysis(svc, analysis, analysis.DCAPlan), nil
}

func buildDocsPublishInputWithAnalysis(svc *service.Service, analysis *model.AnalysisReport, plan *model.DCAPlanReport) docs.PublishInput {
	input := docs.PublishInput{Analysis: analysis, Plan: plan}
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
