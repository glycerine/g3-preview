package subpaving

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingMPFIntervalNegAddSubDirectedAndAlias(t *testing.T) {
	budget := arena.NewBudget(1 << 27)
	runtime, err := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, SizeTBits: 64})
	if err != nil {
		t.Fatal(err)
	}
	core, err := numeral.NewArbitraryFloatManager(numeral.IntegerConfig{
		Memory: runtime, Profile: umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true},
		Cells: arena.Config{Budget: budget}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: budget}},
	})
	if err != nil {
		t.Fatal(err)
	}
	m, err := numeral.NewArbitraryFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var values [4]numeral.ArbitraryFloat
	var intervals [3]*ArbitraryInterval
	for i := range intervals {
		intervals[i], err = NewArbitraryInterval(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for i := len(intervals) - 1; i >= 0; i-- {
			if err := intervals[i].Close(); err != nil {
				t.Error(err)
			}
		}
		for i := len(values) - 1; i >= 0; i-- {
			if err := m.Del(&values[i]); err != nil {
				t.Error(err)
			}
		}
		if err := m.Close(); err != nil {
			t.Error(err)
		}
		if err := core.Close(); err != nil {
			t.Error(err)
		}
		if err := runtime.Close(); err != nil {
			t.Error(err)
		}
		if budget.Used() != 0 {
			t.Error("MPF interval roots", budget.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	for i, n := range []int32{-2, 3, 1, 4} {
		must(m.SetInt32(&values[i], n))
	}
	a, b, out := intervals[0], intervals[1], intervals[2]
	must(a.SetFinite(&values[0], &values[1], false, true))
	must(b.SetFinite(&values[2], &values[3], true, false))
	check := func(v *ArbitraryInterval, lower, upper int32, loOpen, hiOpen bool) {
		t.Helper()
		lo, hi, err := v.endpoints()
		must(err)
		must(m.SetInt32(&values[0], lower))
		must(m.SetInt32(&values[1], upper))
		le, err := m.Equal(lo, &values[0])
		must(err)
		ue, err := m.Equal(hi, &values[1])
		must(err)
		if v.lowerInf || v.upperInf || !le || !ue || v.lowerOpen != loOpen || v.upperOpen != hiOpen {
			t.Fatalf("interval endpoints lower=%t upper=%t open=(%t,%t)", le, ue, v.lowerOpen, v.upperOpen)
		}
	}
	must(out.Add(a, b))
	check(out, -1, 7, true, true)
	must(out.Sub(a, b))
	check(out, -6, 2, false, true)
	must(out.Neg(a))
	check(out, -3, 2, true, false)
	must(a.Add(a, b))
	check(a, -1, 7, true, true)
	must(b.Sub(b, b))
	check(b, -3, 3, true, true)
	must(a.Neg(a))
	check(a, -7, 1, true, true)
	must(out.Reset())
	must(out.Add(out, b))
	if !out.lowerInf || !out.upperInf {
		t.Fatal("unbounded plus finite")
	}
	must(out.Neg(out))
	if !out.lowerInf || !out.upperInf {
		t.Fatal("negated unbounded")
	}
	must(m.SetInt32(&values[0], 1))
	must(m.SetFractionSigned(&values[2], 1, 3))
	must(a.SetFinite(&values[0], &values[0], false, false))
	must(b.SetFinite(&values[2], &values[2], false, false))
	must(out.Add(a, b))
	lo, hi, err := out.endpoints()
	must(err)
	less, err := m.Less(lo, hi)
	must(err)
	if !less {
		t.Fatal("directed MPF bounds collapsed")
	}
	must(a.ResetLower())
	must(out.Neg(a))
	if out.lowerInf || !out.upperInf || out.lowerOpen || !out.upperOpen {
		t.Fatal("negated half-infinite bound", out.lowerInf, out.upperInf, out.lowerOpen, out.upperOpen)
	}
	lo, _, err = out.endpoints()
	must(err)
	must(m.SetInt32(&values[0], -1))
	equal, err := m.Equal(lo, &values[0])
	must(err)
	if !equal {
		t.Fatal("negated half-infinite finite endpoint")
	}
	for _, v := range intervals {
		must(v.CheckInvariant())
	}
}
