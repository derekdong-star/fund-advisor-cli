package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/derekdong-star/fund-advisor-cli/internal/model"
)

func TestSaveAndLoadMomentumPool(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	report := model.MomentumPoolReport{
		Summary:     model.MomentumPoolSummary{RunDate: time.Now().UTC(), SelectedCount: 1},
		Items:       []model.MomentumPoolItem{{Rank: 1, FundCode: "A", FundName: "Leader", Score: 80, LatestNAV: 2}},
		Challengers: []model.MomentumPoolItem{{Rank: 2, FundCode: "B", FundName: "Challenger", Score: 79, LatestNAV: 1.5}},
	}

	if _, err := store.SaveMomentumPool(report); err != nil {
		t.Fatalf("SaveMomentumPool() error = %v", err)
	}
	loaded, err := store.LatestMomentumPool()
	if err != nil {
		t.Fatalf("LatestMomentumPool() error = %v", err)
	}
	if len(loaded.Items) != 1 || loaded.Items[0].FundCode != "A" || loaded.Items[0].LatestNAV != 2 {
		t.Fatalf("loaded momentum pool = %+v", loaded)
	}
	if len(loaded.Challengers) != 1 || loaded.Challengers[0].FundCode != "B" {
		t.Fatalf("loaded challengers = %+v", loaded.Challengers)
	}
}

func TestLatestSelectedMomentumPoolSkipsCashRuns(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	selected := model.MomentumPoolReport{
		Summary: model.MomentumPoolSummary{RunDate: time.Now().UTC(), SelectedCount: 1, Regime: "risk-on"},
		Items:   []model.MomentumPoolItem{{Rank: 1, FundCode: "A", FundName: "Leader", LatestNAV: 2}},
	}
	cash := model.MomentumPoolReport{Summary: model.MomentumPoolSummary{RunDate: time.Now().UTC().Add(time.Hour), Regime: "cash"}}
	if _, err := store.SaveMomentumPool(selected); err != nil {
		t.Fatalf("SaveMomentumPool(selected) error = %v", err)
	}
	if _, err := store.SaveMomentumPool(cash); err != nil {
		t.Fatalf("SaveMomentumPool(cash) error = %v", err)
	}

	loaded, err := store.LatestSelectedMomentumPool()
	if err != nil {
		t.Fatalf("LatestSelectedMomentumPool() error = %v", err)
	}
	if len(loaded.Items) != 1 || loaded.Items[0].FundCode != "A" {
		t.Fatalf("loaded selected momentum pool = %+v", loaded)
	}
}

func TestMomentumPoolHistoryReturnsOldestFirst(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()
	first := model.MomentumPoolReport{Summary: model.MomentumPoolSummary{RunDate: time.Now().UTC()}, Items: []model.MomentumPoolItem{{FundCode: "A"}}}
	second := model.MomentumPoolReport{Summary: model.MomentumPoolSummary{RunDate: time.Now().UTC().Add(time.Hour)}, Items: []model.MomentumPoolItem{{FundCode: "B"}}}
	if _, err := store.SaveMomentumPool(first); err != nil {
		t.Fatalf("SaveMomentumPool(first) error = %v", err)
	}
	if _, err := store.SaveMomentumPool(second); err != nil {
		t.Fatalf("SaveMomentumPool(second) error = %v", err)
	}

	history, err := store.MomentumPoolHistory()
	if err != nil {
		t.Fatalf("MomentumPoolHistory() error = %v", err)
	}
	if len(history) != 2 || history[0].Items[0].FundCode != "A" || history[1].Items[0].FundCode != "B" {
		t.Fatalf("unexpected momentum history: %+v", history)
	}
}

func TestRestoreMomentumPoolHistoryOnlyRestoresEmptyStore(t *testing.T) {
	reports := []model.MomentumPoolReport{
		{Summary: model.MomentumPoolSummary{RunDate: time.Now().UTC()}, Items: []model.MomentumPoolItem{{FundCode: "A"}}},
		{Summary: model.MomentumPoolSummary{RunDate: time.Now().UTC().Add(time.Hour)}, Items: []model.MomentumPoolItem{{FundCode: "B"}}},
	}
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer store.Close()

	restored, err := store.RestoreMomentumPoolHistory(reports)
	if err != nil || restored != 2 {
		t.Fatalf("RestoreMomentumPoolHistory() = %d, %v", restored, err)
	}
	restored, err = store.RestoreMomentumPoolHistory([]model.MomentumPoolReport{{Items: []model.MomentumPoolItem{{FundCode: "C"}}}})
	if err != nil || restored != 0 {
		t.Fatalf("second RestoreMomentumPoolHistory() = %d, %v", restored, err)
	}
	history, err := store.MomentumPoolHistory()
	if err != nil || len(history) != 2 || history[0].Items[0].FundCode != "A" || history[1].Items[0].FundCode != "B" {
		t.Fatalf("unexpected restored history: %+v, %v", history, err)
	}
}
