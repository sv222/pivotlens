package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ScanExpr(path string) (string, error) {
	if _, err := os.Stat(path); err != nil {
		return "", fmt.Errorf("cannot open %s: %w", path, err)
	}
	switch dataExt(path) {
	case ".parquet":
		return "read_parquet(" + ql(path) + ")", nil
	case ".csv", ".tsv", ".txt":
		return "read_csv_auto(" + ql(path) + ", header=true, sample_size=20480)", nil
	case ".json", ".ndjson", ".jsonl":
		return "read_json_auto(" + ql(path) + ")", nil
	default:
		return ql(path), nil
	}
}

func dataExt(path string) string {
	base := strings.ToLower(filepath.Base(path))
	for _, z := range []string{".gz", ".zst", ".bz2"} {
		base = strings.TrimSuffix(base, z)
	}
	return filepath.Ext(base)
}
