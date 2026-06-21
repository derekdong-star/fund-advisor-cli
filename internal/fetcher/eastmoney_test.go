package fetcher

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParsePingzhongData(t *testing.T) {
	t.Parallel()
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..")
	buf, err := os.ReadFile(filepath.Join(root, "testdata", "sample_pingzhongdata.js"))
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	result, err := ParsePingzhongData("000001", string(buf), 3)
	if err != nil {
		t.Fatalf("ParsePingzhongData() error = %v", err)
	}
	if got, want := result.Name, "示例基金A"; got != want {
		t.Fatalf("name = %s, want %s", got, want)
	}
	if got, want := len(result.Snapshots), 3; got != want {
		t.Fatalf("snapshot count = %d, want %d", got, want)
	}
	if got, want := result.Snapshots[len(result.Snapshots)-1].NAV, 1.02; got != want {
		t.Fatalf("latest nav = %.2f, want %.2f", got, want)
	}
	if got, want := result.Snapshots[len(result.Snapshots)-1].AccNAV, 1.12; got != want {
		t.Fatalf("latest accumulated nav = %.2f, want %.2f", got, want)
	}
}

func TestParsePingzhongDataFallsBackToUnitNAVWithoutAccumulatedTrend(t *testing.T) {
	body := `var fS_name = "示例基金A"; var Data_netWorthTrend = [{"x":1704067200000,"y":1.2,"equityReturn":0}];`

	result, err := ParsePingzhongData("000001", body, 0)
	if err != nil {
		t.Fatalf("ParsePingzhongData() error = %v", err)
	}
	if got, want := result.Snapshots[0].AccNAV, 1.2; got != want {
		t.Fatalf("accumulated nav fallback = %.2f, want %.2f", got, want)
	}
}

func TestDecodePurchaseStatuses(t *testing.T) {
	body := `var reData={datas:[["000001","基金A","混合型","1.0","06-18","开放申购","开放赎回","","10.0","100000000000","1.0","1","0.15%"],["000002","基金B","混合型","1.0","06-18","限大额","开放赎回","","100.0","10000.0","1.0","1","0.15%"]],record:"2",pages:"1",curpage:"1"}`

	statuses, err := decodePurchaseStatuses(body)
	if err != nil {
		t.Fatalf("decodePurchaseStatuses() error = %v", err)
	}
	if statuses["000001"].PurchaseStatus != "开放申购" || statuses["000001"].MinimumPurchase != 10 || statuses["000002"].DailyLimit != 10000 {
		t.Fatalf("unexpected purchase statuses: %+v", statuses)
	}
}
