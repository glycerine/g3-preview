package subpaving

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingRationalIntervalInverseSourceBranches(t *testing.T) {
	budget := arena.NewBudget(1 << 24)
	m, err := numeral.NewRationalManager(numeral.RationalConfig{Integers: numeral.IntegerConfig{
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
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
		for _, v := range []*RationalInterval{out, a} {
			if err := v.Close(); err != nil {
				t.Error(err)
			}
		}
		if err := m.Del(&lo); err != nil {
			t.Error(err)
		}
		if err := m.Del(&hi); err != nil {
			t.Error(err)
		}
		if err := m.CloseChecked(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("retained inverse budget", budget.Used())
		}
	})
	set := func(lower, upper string, lowerOpen, upperOpen bool) {
		t.Helper()
		must(m.SetString(&lo, lower))
		must(m.SetString(&hi, upper))
		must(a.SetFinite(&lo, &hi, lowerOpen, upperOpen))
	}
	show := func(want string) {
		t.Helper()
		var b bytes.Buffer
		must(out.Display(&b))
		if b.String() != want {
			t.Fatalf("inverse = %q, want %q", b.String(), want)
		}
		must(out.CheckInvariant())
	}
	set("2", "4", false, false)
	must(out.Inv(a))
	show("[1/4, 1/2]")
	must(out.SetFrom(a))
	must(out.Inv(out))
	show("[1/4, 1/2]")
	set("0", "4", true, false)
	must(out.Inv(a))
	show("[1/4, +oo)")
	set("2", "4", false, false)
	must(a.ResetUpper())
	must(out.Inv(a))
	show("(0, 1/2]")
	set("-4", "-2", false, false)
	must(out.Inv(a))
	show("[-1/2, -1/4]")
	set("-4", "0", false, true)
	must(out.Inv(a))
	show("(-oo, -1/4]")
	set("-4", "-2", false, false)
	must(a.ResetLower())
	must(out.Inv(a))
	show("[-1/2, 0)")
	set("-1", "1", false, false)
	if err := out.Inv(a); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatalf("zero-containing inverse: %v", err)
	}
}
