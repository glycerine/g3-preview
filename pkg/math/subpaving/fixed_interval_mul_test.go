package subpaving

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedIntervalMulSignClasses(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	lo, err := m.New()
	must(err)
	hi, err := m.New()
	must(err)
	a, err := NewFixedInterval(m)
	must(err)
	b, err := NewFixedInterval(m)
	must(err)
	c, err := NewFixedInterval(m)
	must(err)
	defer func() {
		must(a.Close())
		must(b.Close())
		must(c.Close())
		must(lo.Close())
		must(hi.Close())
		must(m.Close())
	}()
	set := func(v *FixedInterval, l, u int64, lOpen, uOpen bool) {
		t.Helper()
		must(m.SetInt64(lo, l))
		must(m.SetInt64(hi, u))
		must(v.SetFinite(lo, hi, lOpen, uOpen))
	}
	show := func(v *FixedInterval) string {
		t.Helper()
		var out bytes.Buffer
		must(v.Display(&out))
		return out.String()
	}
	for _, tc := range []struct {
		name           string
		al, au, bl, bu int64
		want           string
	}{
		{"NN", -3, -1, -3, -1, "[1, 9]"},
		{"NM", -3, -1, -2, 2, "[-6, 6]"},
		{"NP", -3, -1, 1, 3, "[-9, -1]"},
		{"MN", -2, 2, -3, -1, "[-6, 6]"},
		{"MM", -2, 2, -2, 2, "[-4, 4]"},
		{"MP", -2, 2, 1, 3, "[-6, 6]"},
		{"PN", 1, 3, -3, -1, "[-9, -1]"},
		{"PM", 1, 3, -2, 2, "[-6, 6]"},
		{"PP", 1, 3, 1, 3, "[1, 9]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set(a, tc.al, tc.au, false, false)
			set(b, tc.bl, tc.bu, false, false)
			must(c.Mul(a, b))
			if got := show(c); got != tc.want {
				t.Fatalf("product = %q, want %q", got, tc.want)
			}
			must(c.CheckInvariant())
		})
	}
	set(a, -2, 0, false, false)
	set(b, 1, 3, false, false)
	must(c.Mul(a, b))
	if got := show(c); got != "[-6, 0]" {
		t.Fatalf("closed zero product = %q", got)
	}
	set(a, -2, 0, false, true)
	must(c.Mul(a, b))
	if got := show(c); got != "[-6, 0)" {
		t.Fatalf("open zero product = %q", got)
	}
	set(a, -2, 2, false, true)
	set(b, -2, 2, false, false)
	must(c.Mul(a, b))
	if got := show(c); got != "[-4, 4]" {
		t.Fatalf("mixed tie product = %q", got)
	}
	set(a, -2, 2, true, true)
	set(b, -2, 2, true, true)
	must(a.Mul(a, b))
	if got := show(a); got != "(-4, 4)" {
		t.Fatalf("aliased open mixed product = %q", got)
	}
	set(a, -2, -1, false, false)
	must(a.ResetLower())
	set(b, 1, 2, false, false)
	must(c.Mul(a, b))
	if got := show(c); got != "(-oo, -1]" {
		t.Fatalf("one-sided product = %q", got)
	}
	set(b, 0, 0, false, false)
	must(c.Mul(a, b))
	if got := show(c); got != "[0, 0]" {
		t.Fatalf("zero times infinity = %q", got)
	}
	must(m.SetRounding(false))
	must(m.SetRatio(lo, -1, 3))
	must(m.SetRounding(true))
	must(m.SetRatio(hi, 2, 3))
	must(a.SetFinite(lo, hi, false, false))
	must(m.SetRounding(false))
	must(m.SetRatio(lo, -2, 3))
	must(m.SetRounding(true))
	must(m.SetRatio(hi, 1, 3))
	must(b.SetFinite(lo, hi, false, false))
	must(c.Mul(a, b))
	resultLower, err := m.Rational(c.lower)
	must(err)
	resultUpper, err := m.Rational(c.upper)
	must(err)
	if resultLower.Cmp(big.NewRat(-4, 9)) > 0 || resultUpper.Cmp(big.NewRat(2, 9)) < 0 {
		t.Fatalf("directed mixed product [%s, %s] excludes [-4/9, 2/9]", resultLower, resultUpper)
	}
}
