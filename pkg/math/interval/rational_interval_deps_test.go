package interval

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestRationalIntervalSourceDependencyMasks(t *testing.T) {
	budget := arena.NewBudget(1 << 30)
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
	b, err := NewRationalInterval(m)
	must(err)
	lo, hi, k := numeral.NewRational(), numeral.NewRational(), numeral.NewRational()
	defer func() {
		must(a.Close())
		must(b.Close())
		must(m.Del(&lo))
		must(m.Del(&hi))
		must(m.Del(&k))
		must(m.CloseChecked())
		if budget.Used() != 0 {
			t.Fatalf("dependency mask owner retained %d bytes", budget.Used())
		}
	}()
	set := func(dst *RationalInterval, l, u string, lowerOpen, upperOpen bool) {
		t.Helper()
		must(m.SetString(&lo, l))
		must(m.SetString(&hi, u))
		must(dst.SetFinite(&lo, &hi, lowerOpen, upperOpen))
	}
	if got, err := (DepLower1 | DepUpper1).FirstToSecond(); err != nil || got != DepLower2|DepUpper2 {
		t.Fatalf("dep shift = %d, %v", got, err)
	}
	if _, err := (DepLower1 | DepLower2).FirstToSecond(); err != ErrExtendedDomain {
		t.Fatalf("invalid dep shift = %v", err)
	}
	inputs := map[byte][2]string{'N': {"-3", "-1"}, 'M': {"-2", "4"}, 'P': {"1", "3"}}
	mulWant := map[string]IntervalDeps{
		"NN": {10, 7}, "NM": {11, 7}, "NP": {11, 6},
		"MN": {14, 13}, "MM": {15, 15}, "MP": {13, 14},
		"PN": {7, 9}, "PM": {7, 11}, "PP": {5, 11},
	}
	for pair, want := range mulWant {
		ab, bb := inputs[pair[0]], inputs[pair[1]]
		set(a, ab[0], ab[1], false, false)
		set(b, bb[0], bb[1], false, false)
		got, err := a.MulDeps(b)
		must(err)
		if got != want {
			t.Fatalf("mul deps %s = %+v, want %+v", pair, got, want)
		}
	}
	set(a, "0", "0", false, false)
	set(b, "1", "3", false, false)
	got, err := a.MulDeps(b)
	must(err)
	if got != (IntervalDeps{3, 3}) {
		t.Fatalf("first zero deps = %+v", got)
	}
	got, err = b.MulDeps(a)
	must(err)
	if got != (IntervalDeps{12, 12}) {
		t.Fatalf("second zero deps = %+v", got)
	}
	set(a, "-2", "3", true, false)
	set(b, "1", "4", false, true)
	if got, err := a.AddDeps(b); err != nil || got != (IntervalDeps{5, 10}) {
		t.Fatalf("add deps = %+v, %v", got, err)
	}
	if got, err := a.SubDeps(b); err != nil || got != (IntervalDeps{9, 6}) {
		t.Fatalf("sub deps = %+v, %v", got, err)
	}
	if got, err := a.NegDeps(); err != nil || got != (IntervalDeps{2, 1}) {
		t.Fatalf("neg deps = %+v, %v", got, err)
	}
	must(m.SetInt32(&k, 0))
	if got, err := a.MulScalarDeps(&k); err != nil || got != (IntervalDeps{}) {
		t.Fatalf("zero scalar deps = %+v, %v", got, err)
	}
	must(m.SetInt32(&k, -2))
	if got, err := a.MulScalarDeps(&k); err != nil || got != (IntervalDeps{2, 1}) {
		t.Fatalf("negative scalar deps = %+v, %v", got, err)
	}
	if got, err := a.PowerDeps(1); err != nil || got != (IntervalDeps{1, 2}) {
		t.Fatalf("unit power deps = %+v, %v", got, err)
	}
	if got, err := a.PowerDeps(2); err != nil || got != (IntervalDeps{0, 3}) {
		t.Fatalf("mixed even deps = %+v, %v", got, err)
	}
	if got, err := a.PowerDeps(3); err != nil || got != (IntervalDeps{1, 2}) {
		t.Fatalf("odd deps = %+v, %v", got, err)
	}
	if _, err := a.InvDeps(); err != ErrExtendedDomain {
		t.Fatalf("zero-containing inverse deps = %v", err)
	}
	set(a, "1", "3", false, false)
	if got, err := a.InvDeps(); err != nil || got != (IntervalDeps{3, 1}) {
		t.Fatalf("positive inverse deps = %+v, %v", got, err)
	}
	set(a, "-3", "-1", false, false)
	if got, err := a.InvDeps(); err != nil || got != (IntervalDeps{2, 3}) {
		t.Fatalf("negative inverse deps = %+v, %v", got, err)
	}
	must(a.ResetLower())
	if got, err := a.NegDeps(); err != nil || got != (IntervalDeps{2, 0}) {
		t.Fatalf("one-sided neg deps = %+v, %v", got, err)
	}
	if got, err := a.PowerDeps(2); err != nil || got != (IntervalDeps{2, 0}) {
		t.Fatalf("negative infinite even deps = %+v, %v", got, err)
	}
	if got, err := a.PowerDeps(3); err != nil || got != (IntervalDeps{0, 2}) {
		t.Fatalf("negative infinite odd deps = %+v, %v", got, err)
	}
	must(a.Reset())
	if got, err := a.NegDeps(); err != nil || got != (IntervalDeps{}) {
		t.Fatalf("unbounded neg deps = %+v, %v", got, err)
	}
	divInputs := map[byte][2]string{'N': {"-4", "-2"}, 'M': {"-2", "4"}, 'P': {"2", "4"}}
	divWant := map[string]IntervalDeps{
		"NN": {14, 9}, "NP": {5, 14}, "MN": {10, 9},
		"MP": {5, 6}, "PN": {10, 13}, "PP": {13, 6},
	}
	for pair, want := range divWant {
		ab, bb := divInputs[pair[0]], divInputs[pair[1]]
		set(a, ab[0], ab[1], false, false)
		set(b, bb[0], bb[1], false, false)
		got, err := a.DivDeps(b)
		must(err)
		if got != want {
			t.Fatalf("div deps %s = %+v, want %+v", pair, got, want)
		}
	}
	set(a, "0", "0", false, false)
	if got, err := a.DivDeps(b); err != nil || got != (IntervalDeps{5, 6}) {
		t.Fatalf("zero numerator positive deps = %+v, %v", got, err)
	}
	set(b, "-4", "-2", false, false)
	if got, err := a.DivDeps(b); err != nil || got != (IntervalDeps{10, 9}) {
		t.Fatalf("zero numerator negative deps = %+v, %v", got, err)
	}
	set(b, "-1", "1", false, false)
	if _, err := a.DivDeps(b); err != ErrExtendedDomain {
		t.Fatalf("zero-containing denominator deps = %v", err)
	}
}
