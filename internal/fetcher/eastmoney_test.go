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
}
