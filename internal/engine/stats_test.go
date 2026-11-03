package engine

import (
	"context"
	"testing"
)

func statsSpec() QuerySpec {
	return NewSpec([]ColumnMeta{
		{Name: "region", Type: "VARCHAR"},
		{Name: "amount", Type: "DOUBLE"},
	})
}

func TestStatsSQLText(t *testing.T) {
	got, err := statsSpec().StatsSQL("region", false)
	if err != nil {
		t.Fatal(err)
	}
	want := `WITH base AS (SELECT "region" AS "region", "amount" AS "amount" FROM src), filtered AS (SELECT * FROM base) SELECT count(*), count(*) - count("region"), count(DISTINCT "region"), CAST(min("region") AS VARCHAR), CAST(max("region") AS VARCHAR) FROM filtered`
	if got != want {
		t.Errorf("\n got %s\nwant %s", got, want)
	}
}

func TestStatsSQLNumeric(t *testing.T) {
	got, err := statsSpec().StatsSQL("amount", true)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, `CAST(avg("amount") AS VARCHAR), CAST(sum("amount") AS VARCHAR)`) {
		t.Errorf("numeric stats missing avg/sum: %s", got)
	}
}

func TestStatsSQLRejectsPivot(t *testing.T) {
	q := statsSpec()
	q.Pivot = &PivotSpec{Rows: []string{"region"}, On: "status", Measures: []Measure{{Agg: "count"}}}
	if _, err := q.StatsSQL("region", false); err != ErrPivotStats {
		t.Fatalf("err = %v, want ErrPivotStats", err)
	}
}

func TestStatsSQLAppliesFilter(t *testing.T) {
	q := statsSpec()
	q.Filter = "EAST"
	got, err := q.StatsSQL("region", false)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(got, `ILIKE '%EAST%'`) {
		t.Errorf("filter not applied: %s", got)
	}
}

func TestNumericType(t *testing.T) {
	for _, n := range []string{"BIGINT", "INTEGER", "HUGEINT", "DOUBLE", "FLOAT", "DECIMAL(18,2)", "REAL"} {
		if !numericType(n) {
			t.Errorf("%q should be numeric", n)
		}
	}
	for _, s := range []string{"VARCHAR", "TEXT", "DATE", "TIMESTAMP", "BOOLEAN"} {
		if numericType(s) {
			t.Errorf("%q should not be numeric", s)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestStatsLive(t *testing.T) {
	s := openFixture(t, "tiny.csv")
	q := NewSpec(s.Columns())
	st, err := s.Stats(context.Background(), q, "status")
	if err != nil {
		t.Fatal(err)
	}
	if st.Rows != 5 || st.Distinct != 3 {
		t.Errorf("rows %d distinct %d, want 5 and 3", st.Rows, st.Distinct)
	}
	if st.Min != "COMPLETED" || st.Max != "PENDING" {
		t.Errorf("min %q max %q", st.Min, st.Max)
	}
	if st.Numeric {
		t.Error("status is not numeric")
	}
	am, err := s.Stats(context.Background(), q, "amount")
	if err != nil {
		t.Fatal(err)
	}
	if !am.Numeric {
		t.Error("amount should be numeric")
	}
	if am.Sum == "" || am.Avg == "" {
		t.Errorf("avg %q sum %q, want values", am.Avg, am.Sum)
	}
	if am.Nulls != 0 {
		t.Errorf("nulls = %d, want 0", am.Nulls)
	}
}
