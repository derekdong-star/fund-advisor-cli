package service

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/config"
	"github.com/derekdong-star/fund-advisor-cli/internal/model"
	"github.com/derekdong-star/fund-advisor-cli/internal/store"
)

func TestCompactMomentumPoolHistoryKeepsLatestRunPerDayAndRecentDays(t *testing.T) {
	start := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.UTC)
	history := []model.MomentumPoolReport{
		{Summary: model.MomentumPoolSummary{RunDate: start}, Items: []model.MomentumPoolItem{{FundCode: "old-day-1"}}},
		{Summary: model.MomentumPoolSummary{RunDate: start.Add(time.Hour)}, Items: []model.MomentumPoolItem{{FundCode: "latest-day-1"}}},
		{Summary: model.MomentumPoolSummary{RunDate: start.AddDate(0, 0, 1)}, Items: []model.MomentumPoolItem{{FundCode: "day-2"}}},
		{Summary: model.MomentumPoolSummary{RunDate: start.AddDate(0, 0, 2)}, Items: []model.MomentumPoolItem{{FundCode: "day-3"}}},
	}

	result := compactMomentumPoolHistory(history, 2)

	if len(result) != 2 || result[0].Items[0].FundCode != "day-2" || result[1].Items[0].FundCode != "day-3" {
		t.Fatalf("compactMomentumPoolHistory() = %+v", result)
	}
}

func TestFetchHistoryWithRetrySucceedsAfterTransientFailures(t *testing.T) {
	attempts := 0
	result, err := fetchHistoryWithRetry(context.Background(), 3, func() (*model.FetchResult, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("temporary")
		}
		return &model.FetchResult{Code: "A"}, nil
	})
	if err != nil || result.Code != "A" || attempts != 3 {
		t.Fatalf("fetchHistoryWithRetry() = %+v, %v, attempts=%d", result, err, attempts)
	}
}

func TestFetchHistoryWithRetryStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0
	_, err := fetchHistoryWithRetry(ctx, 3, func() (*model.FetchResult, error) {
		attempts++
		return nil, errors.New("temporary")
	})
	if !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("fetchHistoryWithRetry() error=%v attempts=%d", err, attempts)
	}
}

func TestFetchMarketProfileWithRetrySucceedsAfterTransientFailures(t *testing.T) {
	attempts := 0
	profile, err := fetchMarketProfileWithRetry(context.Background(), 3, func() (*model.MarketFundProfile, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("temporary")
		}
		return &model.MarketFundProfile{Fund: model.MarketSearchFund{Code: "A"}}, nil
	})
	if err != nil || profile.Fund.Code != "A" || attempts != 3 {
		t.Fatalf("fetchMarketProfileWithRetry() = %+v, %v, attempts=%d", profile, err, attempts)
	}
}

func TestFetchMarketProfileWithRetryStopsOnCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	attempts := 0
	_, err := fetchMarketProfileWithRetry(ctx, 3, func() (*model.MarketFundProfile, error) {
		attempts++
		return nil, errors.New("temporary")
	})
	if !errors.Is(err, context.Canceled) || attempts != 1 {
		t.Fatalf("fetchMarketProfileWithRetry() error=%v attempts=%d", err, attempts)
	}
}

func TestFetchSkipsFailedCandidateButKeepsWarning(t *testing.T) {
	cfg := config.Default()
	cfg.Funds = []config.FundConfig{{Code: "holding", Name: "Holding", Category: "equity", Benchmark: "benchmark", Role: "core"}}
	cfg.Candidates = []config.FundConfig{{Code: "candidate", Name: "Candidate", Category: "equity", Benchmark: "benchmark", Role: "watch"}}
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()
	service := &Service{config: cfg, store: store, fetchHistory: func(_ context.Context, code string, _ int) (*model.FetchResult, error) {
		if code == "candidate" {
			return nil, errors.New("timeout")
		}
		return &model.FetchResult{Code: code, Snapshots: []model.FundSnapshot{{FundCode: code, TradeDate: time.Now().UTC(), NAV: 1, AccNAV: 1}}}, nil
	}}

	if err := service.Fetch(context.Background(), 300); err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if len(service.FetchWarnings()) != 1 || !strings.Contains(service.FetchWarnings()[0], "candidate") {
		t.Fatalf("FetchWarnings() = %v", service.FetchWarnings())
	}
}

func TestFetchReturnsFailedHolding(t *testing.T) {
	cfg := config.Default()
	cfg.Funds = []config.FundConfig{{Code: "holding", Name: "Holding", Category: "equity", Benchmark: "benchmark", Role: "core"}}
	store, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("store.Open() error = %v", err)
	}
	defer store.Close()
	service := &Service{config: cfg, store: store, fetchHistory: func(context.Context, string, int) (*model.FetchResult, error) { return nil, errors.New("timeout") }}

	if err := service.Fetch(context.Background(), 300); err == nil || !strings.Contains(err.Error(), "fetch holding") {
		t.Fatalf("Fetch() error = %v", err)
	}
}
