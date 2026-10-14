package subpaving

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingRationalIntervalXnEqYSourceBranches(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	a, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewRationalInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	lo, hi, precision := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	t.Cleanup(func() {
		for _, e := range []error{out.Close(), a.Close(), m.Del(&lo), m.Del(&hi), m.Del(&precision), m.CloseChecked()} {
			if e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("rational interval root retained budget", budget.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	set := func(lower, upper string, lowerOpen, upperOpen bool) {
		t.Helper()
		must(m.SetString(&lo, lower))
		must(m.SetString(&hi, upper))
		must(a.SetFinite(&lo, &hi, lowerOpen, upperOpen))
	}
	show := func(v *RationalInterval, want string) {
		t.Helper()
		var b bytes.Buffer
		must(v.Display(&b))
		if b.String() != want {
			t.Fatalf("inverse interval = %q, want %q", b.String(), want)
		}
		must(v.CheckInvariant())
	}
	must(m.SetFractionSigned(&precision, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	root := interval.RationalRoot{Numbers: m, Limit: limit}
	set("4", "9", false, false)
	must(out.XnEqY(a, 2, &precision, root))
	must(m.SetInt32(&hi, 3))
	cmp, err := m.Compare(&out.upper, &hi)
	if err != nil || cmp <= 0 || out.lowerInf || out.upperInf || out.lowerOpen || out.upperOpen {
		t.Fatal("irrational Newton upper enclosure", cmp, err)
	}
	must(m.Copy(&lo, &out.upper))
	must(m.Neg(&lo))
	equal, err := m.Equal(&lo, &out.lower)
	if err != nil || !equal {
		t.Fatal("even inverse is symmetric", equal, err)
	}
	set("1", "4", false, true)
	must(out.XnEqY(a, 2, &precision, root))
	show(out, "(-2, 2)")
	must(a.XnEqY(a, 2, &precision, root))
	show(a, "(-2, 2)")
	set("-8", "8", true, false)
	must(out.XnEqY(a, 3, &precision, root))
	show(out, "(-2, 2]")
	must(a.ResetLower())
	must(out.XnEqY(a, 3, &precision, root))
	show(out, "(-oo, 2]")
	set("0", "4", false, false)
	must(a.ResetUpper())
	must(out.XnEqY(a, 2, &precision, root))
	show(out, "(-oo, +oo)")
	set("-1", "4", false, false)
	if err := out.XnEqY(a, 2, &precision, root); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatal("negative lower even inverse", err)
	}
	set("2", "2", false, false)
	limit.Limit, limit.Steps = 1, 1
	if err := out.XnEqY(a, 2, &precision, root); !errors.Is(err, interval.ErrRationalRootCanceled) {
		t.Fatal("interval root checkpoint", err)
	}
}
