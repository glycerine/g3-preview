package subpaving

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedIntervalScalarMulDiv(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	lower, err := m.New()
	must(err)
	upper, err := m.New()
	must(err)
	k, err := m.New()
	must(err)
	a, err := NewFixedInterval(m)
	must(err)
	b, err := NewFixedInterval(m)
	must(err)
	defer func() {
		must(a.Close())
		must(b.Close())
		must(lower.Close())
		must(upper.Close())
		must(k.Close())
		must(m.Close())
	}()
	show := func(v *FixedInterval) string {
		t.Helper()
		var out bytes.Buffer
		must(v.Display(&out))
		return out.String()
	}
	must(m.SetInt64(lower, -1))
	must(m.SetInt64(upper, 2))
	must(a.SetFinite(lower, upper, true, false))
	must(m.SetInt64(k, -2))
	must(b.MulScalar(k, a))
	if got := show(b); got != "[-4, 2)" {
		t.Fatalf("negative scalar product = %q", got)
	}
	must(a.MulScalar(k, a))
	if got := show(a); got != "[-4, 2)" {
		t.Fatalf("aliased scalar product = %q", got)
	}
	must(m.SetInt64(k, 2))
	must(b.DivScalar(a, k))
	if got := show(b); got != "[-2, 1)" {
		t.Fatalf("scalar quotient = %q", got)
	}
	must(a.ResetLower())
	must(b.MulScalar(k, a))
	if got := show(b); got != "(-oo, 4)" {
		t.Fatalf("one-sided scalar product = %q", got)
	}
	must(m.SetInt64(k, -2))
	must(b.DivScalar(a, k))
	if got := show(b); got != "(-1, +oo)" {
		t.Fatalf("negative one-sided quotient = %q", got)
	}
	must(m.SetInt64(k, 0))
	must(b.MulScalar(k, a))
	if got := show(b); got != "(-oo, +oo)" {
		t.Fatalf("zero scalar product = %q", got)
	}
	must(b.DivScalar(a, k))
	if got := show(b); got != "(-oo, +oo)" {
		t.Fatalf("zero scalar quotient = %q", got)
	}
	must(m.SetInt64(lower, 1))
	must(m.SetInt64(upper, 2))
	must(a.SetFinite(lower, upper, false, false))
	must(m.SetInt64(k, 3))
	must(b.DivScalar(a, k))
	lo, err := m.Rational(b.lower)
	must(err)
	hi, err := m.Rational(b.upper)
	must(err)
	if lo.Cmp(big.NewRat(1, 3)) > 0 || hi.Cmp(big.NewRat(2, 3)) < 0 {
		t.Fatalf("directed quotient [%s, %s] excludes [1/3, 2/3]", lo, hi)
	}
	must(b.CheckInvariant())
}
