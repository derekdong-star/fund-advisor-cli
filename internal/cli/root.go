package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
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
	root.AddCommand(newRunCmd(&configPath))
	root.AddCommand(newBackfillCmd(&configPath))
	root.AddCommand(newBacktestCmd(&configPath))
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
