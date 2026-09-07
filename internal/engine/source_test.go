package engine

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDataExt(t *testing.T) {
	cases := map[string]string{
		"a.csv":        ".csv",
		"a.CSV":        ".csv",
		"a.csv.gz":     ".csv",
		"a.json.zst":   ".json",
		"a.parquet":    ".parquet",
		"dir/b.ndjson": ".ndjson",
		"noext":        "",
	}
	for in, want := range cases {
		if got := dataExt(in); got != want {
			t.Errorf("dataExt(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestScanExprByFormat(t *testing.T) {
	csv := filepath.Join("..", "..", "testdata", "tiny.csv")
	got, err := ScanExpr(csv)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "read_csv_auto(") {
		t.Errorf("csv should use read_csv_auto, got %q", got)
	}

	nd := filepath.Join("..", "..", "testdata", "tiny.ndjson")
	got, err = ScanExpr(nd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "read_json_auto(") {
		t.Errorf("ndjson should use read_json_auto, got %q", got)
	}
}

func TestScanExprMissingFile(t *testing.T) {
	if _, err := ScanExpr("does-not-exist.csv"); err == nil {
		t.Error("missing file must be an error")
	}
}
