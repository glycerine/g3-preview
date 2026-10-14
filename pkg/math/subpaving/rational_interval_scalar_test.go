package subpaving

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingRationalIntervalScalarSourceBranches(t *testing.T) {
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
	out, err := NewRationalInterval(m)
	must(err)
	lo, hi, k := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, x := range []*RationalInterval{out, a} {
			if err := x.Close(); err != nil {
				t.Error(err)
			}
		}
		for _, x := range []*numeral.Rational{&lo, &hi, &k} {
			if err := m.Del(x); err != nil {
				t.Error(err)
			}
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("rational scalar interval retained budget", budget.Used())
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
	must(m.SetInt32(&lo, -1))
	must(m.SetInt32(&hi, 2))
	must(a.SetFinite(&lo, &hi, true, false))
	must(m.SetInt32(&k, -2))
	must(out.MulScalar(&k, a))
	show(out, "[-4, 2)")
	must(a.MulScalar(&k, a))
	show(a, "[-4, 2)")
	must(m.SetInt32(&k, 2))
	must(out.DivScalar(a, &k))
	show(out, "[-2, 1)")
	must(a.ResetLower())
	must(out.MulScalar(&k, a))
	show(out, "(-oo, 4)")
	must(m.SetInt32(&k, -2))
	must(out.DivScalar(a, &k))
	show(out, "(-1, +oo)")
	must(m.Reset(&k))
	must(out.MulScalar(&k, a))
	show(out, "(-oo, +oo)")
	must(out.DivScalar(a, &k))
	show(out, "(-oo, +oo)")
	must(m.SetString(&lo, "1/3"))
	must(m.SetString(&hi, "2/3"))
	must(a.SetFinite(&lo, &hi, false, false))
	must(m.SetInt32(&k, 3))
	must(out.DivScalar(a, &k))
	show(out, "[1/9, 2/9]")
}
