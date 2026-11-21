package ui

import (
	"strconv"
	"strings"
)

// The bar column is display only: it never reaches the SQL result, an
// export or --freq.
const (
	barCol       = "bar"
	FreqBarWidth = 20
)

var barBlock = "\u2588"

// freqWithBars appends the bar cell to every row of a value counts page,
// scaled to the row percent.
func freqWithBars(rows [][]string) [][]string {
	out := make([][]string, len(rows))
	for i, r := range rows {
		out[i] = append(append([]string(nil), r...), freqBar(r[2]))
	}
	return out
}

func freqBar(percent string) string {
	p, err := strconv.ParseFloat(percent, 64)
	if err != nil || p <= 0 {
		return ""
	}
	n := int(p*FreqBarWidth/100 + 0.5)
	if n > FreqBarWidth {
		n = FreqBarWidth
	}
	if n <= 0 {
		return ""
	}
	return strings.Repeat(barBlock, n)
}
