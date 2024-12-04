package utils

import (
	"fmt"
	"math"
	"strings"

	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

func PrettyFloat(f float64) string {
	for _, unit := range []string{"", "K", "M", "G"} {
		if math.Abs(f) < 1000.0 {
			return fmt.Sprintf("%3.2f%s", f, unit)
		}
		f /= 1000.0
	}
	return fmt.Sprintf("%.2fT", f)
}

func AbbreviateDecimal(v decimal.Decimal) string {
	s := v.StringFixedBank(9)
	ss := strings.Split(s, ".")
	if len(ss) == 1 {
		return s
	}

	fraction := ss[1]
	cnt := 0
	for _, c := range fraction {
		if c == '0' {
			cnt++
		} else {
			break
		}
	}

	const zero rune = '\u2080'
	if cnt >= 9 {
		fraction = fraction[:3]
	} else if cnt > 2 {
		fraction = fmt.Sprintf("0%s%s", string(zero+rune(cnt)), fraction[cnt:lo.Min([]int{9, cnt + 3})])
	} else {
		fraction = fraction[:cnt+3]
	}
	return fmt.Sprintf("%s.%s", ss[0], fraction)
}

func GetTokenAddress(text string) string {
	s := strings.Index(text, "\n")
	if s == -1 {
		return ""
	}

	line := text[:s]
	ss := strings.Split(line, "|")
	if len(ss) != 3 {
		return ""
	}

	return strings.TrimSpace(ss[2])
}

func TrimSpace(s string) string {
	s = strings.TrimSpace(s)
	var m, n int

	for i := 0; i < len(s); i++ {
		if s[i] != 0 {
			m = i
			break
		}
	}

	for i := len(s) - 1; i > 0; i-- {
		if s[i] != 0 {
			n = i + 1
			break
		}
	}

	return s[m:n]
}
