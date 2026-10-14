package subpaving

import (
	"bytes"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingRationalIntervalMulNineSignClassesAndTies(t *testing.T) {
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
			t.Error("rational interval product retained budget", budget.Used())
		}
	})
	set := func(v *RationalInterval, lower, upper string, lowerOpen, upperOpen bool) {
		t.Helper()
		must(m.SetString(&lo, lower))
		must(m.SetString(&hi, upper))
		must(v.SetFinite(&lo, &hi, lowerOpen, upperOpen))
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
	inputs := map[byte][2]string{'N': {"-3", "-1"}, 'M': {"-2", "4"}, 'P': {"1", "3"}}
	for _, tc := range []struct{ pair, want string }{
		{"NN", "[1, 9]"}, {"NM", "[-12, 6]"}, {"NP", "[-9, -1]"},
		{"MN", "[-12, 6]"}, {"MM", "[-8, 16]"}, {"MP", "[-6, 12]"},
		{"PN", "[-9, -1]"}, {"PM", "[-6, 12]"}, {"PP", "[1, 9]"},
	} {
		t.Run(tc.pair, func(t *testing.T) {
			ab, bb := inputs[tc.pair[0]], inputs[tc.pair[1]]
			set(a, ab[0], ab[1], false, false)
			set(b, bb[0], bb[1], false, false)
			must(out.Mul(a, b))
			show(out, tc.want)
		})
	}
	set(a, "-3", "0", false, false)
	set(b, "0", "2", false, false)
	must(out.Mul(a, b))
	show(out, "[-6, 0]")
	must(a.ResetLower())
	must(out.Mul(a, b))
	show(out, "(-oo, 0]")
	set(a, "-2", "4", true, false)
	set(b, "-3", "6", false, false)
	must(out.Mul(a, b))
	show(out, "[-12, 24]")
	must(out.SetFrom(b))
	must(out.Mul(a, out))
	show(out, "[-12, 24]")
	set(a, "-2", "4", false, true)
	must(out.Mul(a, b))
	show(out, "[-12, 24)")
	must(a.ResetLower())
	must(b.ResetUpper())
	must(out.Mul(a, b))
	show(out, "(-oo, +oo)")
	set(a, "0", "0", false, false)
	must(out.Mul(a, b))
	show(out, "[0, 0]")
}
