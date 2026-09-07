package ui

const (
	MinColWidth = 4
	MaxColWidth = 40
	SepWidth    = 3
)

func Fit(maxLens []int, headers []string) []int {
	out := make([]int, len(headers))
	for i, h := range headers {
		w := len([]rune(h))
		if w < MinColWidth {
			w = MinColWidth
		}
		if i < len(maxLens) && maxLens[i] > w {
			w = maxLens[i]
		}
		if w > MaxColWidth {
			w = MaxColWidth
		}
		out[i] = w
	}
	return out
}

func Window(widths []int, start, term int) int {
	if start < 0 || start >= len(widths) {
		return 0
	}
	used, n := 0, 0
	for i := start; i < len(widths); i++ {
		next := used + widths[i]
		if n > 0 {
			next += SepWidth
		}
		if next > term && n > 0 {
			break
		}
		used = next
		n++
	}
	return n
}
