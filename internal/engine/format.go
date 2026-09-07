package engine

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const NullText = "NULL"

func asString(v any) string {
	if v == nil {
		return ""
	}
	if b, ok := v.([]byte); ok {
		return string(b)
	}
	return fmt.Sprint(v)
}

func FormatValue(v any) string {
	if v == nil {
		return NullText
	}
	var s string
	switch x := v.(type) {
	case string:
		s = x
	case []byte:
		s = string(x)
	case bool:
		s = strconv.FormatBool(x)
	case float32:
		s = strconv.FormatFloat(float64(x), 'f', -1, 32)
	case float64:
		s = strconv.FormatFloat(x, 'f', -1, 64)
	case time.Time:
		if x.Hour() == 0 && x.Minute() == 0 && x.Second() == 0 && x.Nanosecond() == 0 {
			s = x.Format("2006-01-02")
		} else {
			s = x.Format("2006-01-02 15:04:05")
		}
	case *big.Int:
		s = x.String()
	case fmt.Stringer:
		s = x.String()
	default:
		s = fmt.Sprint(x)
	}
	return scrub(s)
}

func isCtrl(r rune) bool { return r < 0x20 || r == 0x7f }

func scrub(s string) string {
	if !strings.ContainsFunc(s, isCtrl) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isCtrl(r) {
			b.WriteRune('·')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}
