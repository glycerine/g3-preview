package subpaving

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingRationalIntervalPowerSourceCases(t *testing.T) {
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
	lo, hi := numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, x := range []*RationalInterval{out, a} {
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
			t.Error("rational interval power retained budget", budget.Used())
		}
	})
	set := func(lower, upper string, lowerOpen, upperOpen bool) {
		t.Helper()
		must(m.SetString(&lo, lower))
		must(m.SetString(&hi, upper))
		must(a.SetFinite(&lo, &hi, lowerOpen, upperOpen))
	}
	show := func(v *RationalInterval, want string) {
		t.Helper()
		var rendered bytes.Buffer
		must(v.Display(&rendered))
		if rendered.String() != want {
			t.Fatalf("interval = %q, want %q", rendered.String(), want)
		}
		must(v.CheckInvariant())
	}
	check := func(n uint32, want string) {
		t.Helper()
		must(out.Power(a, n))
		show(out, want)
	}
	set("2", "3", false, false)
	check(2, "[4, 9]")
	must(out.Power(out, 2))
	show(out, "[16, 81]")
	check(0, "[1, 1]")
	set("-3", "-2", false, false)
	check(2, "[4, 9]")
	must(a.Power(a, 2))
	show(a, "[4, 9]")
	set("-3", "-2", false, false)
	must(a.ResetLower())
	check(2, "[4, +oo)")
	set("-2", "3", false, false)
	check(2, "[0, 9]")
	check(3, "[-8, 27]")
	check(0, "[0, 1]")
	must(a.ResetLower())
	check(3, "(-oo, 27]")
	set("-2", "2", true, false)
	check(2, "[0, 4]")
	set("-2", "2", false, true)
	check(2, "[0, 4]")
	set("-2", "2", true, true)
	check(2, "[0, 4)")
	must(a.ResetLower())
	check(2, "[0, +oo)")
	set("1/3", "2/3", false, false)
	check(2, "[1/9, 4/9]")
}
