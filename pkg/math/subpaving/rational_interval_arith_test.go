package subpaving

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingRationalIntervalNegAddSubAndAlias(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget},
		Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	a, err := NewRationalInterval(m)
	must(err)
	b, err := NewRationalInterval(m)
	must(err)
	out, err := NewRationalInterval(m)
	must(err)
	lo, hi := numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, x := range []*RationalInterval{out, b, a} {
			if err := x.Close(); err != nil {
				t.Error(err)
			}
		}
		for _, x := range []*numeral.Rational{&lo, &hi} {
			if err := m.Del(x); err != nil {
				t.Error(err)
			}
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("rational interval arithmetic retained budget", budget.Used())
		}
	})
	show := func(v *RationalInterval, want string) {
		t.Helper()
		var rendered bytes.Buffer
		must(v.Display(&rendered))
		if rendered.String() != want {
			t.Fatalf("interval = %q, want %q", rendered.String(), want)
		}
		must(v.CheckInvariant())
	}
	must(m.SetString(&lo, "-3/2"))
	must(m.SetInt32(&hi, 2))
	must(a.SetFinite(&lo, &hi, true, false))
	must(m.SetInt32(&lo, 2))
	must(m.SetString(&hi, "5/2"))
	must(b.SetFinite(&lo, &hi, false, true))
	must(out.Add(a, b))
	show(out, "(1/2, 9/2)")
	must(out.Sub(a, b))
	show(out, "(-4, 0]")
	must(out.Neg(a))
	show(out, "[-2, 3/2)")
	must(out.Neg(out))
	show(out, "(-3/2, 2]")
	must(out.Add(out, b))
	show(out, "(1/2, 9/2)")
	must(out.Sub(out, b))
	show(out, "(-2, 5/2)")
	must(a.ResetLower())
	must(out.Neg(a))
	show(out, "[-2, +oo)")
	must(out.Add(a, b))
	show(out, "(-oo, 9/2)")
	must(out.Sub(a, b))
	show(out, "(-oo, 0]")
	must(b.ResetUpper())
	must(out.Add(a, b))
	show(out, "(-oo, +oo)")
}
