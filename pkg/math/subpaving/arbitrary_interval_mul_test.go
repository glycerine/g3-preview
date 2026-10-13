package subpaving

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingMPFIntervalMulNineSignClassesAndDirectedBounds(t *testing.T) {
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
			t.Error("MPF product roots", budget.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	a, b, out := intervals[0], intervals[1], intervals[2]
	set := func(v *ArbitraryInterval, lo, hi int32, loOpen, hiOpen bool) {
		t.Helper()
		must(m.SetInt32(&values[0], lo))
		must(m.SetInt32(&values[1], hi))
		must(v.SetFinite(&values[0], &values[1], loOpen, hiOpen))
	}
	check := func(v *ArbitraryInterval, lo, hi int32, loOpen, hiOpen bool) {
		t.Helper()
		actualLo, actualHi, err := v.endpoints()
		must(err)
		must(m.SetInt32(&values[2], lo))
		must(m.SetInt32(&values[3], hi))
		le, err := m.Equal(actualLo, &values[2])
		must(err)
		ue, err := m.Equal(actualHi, &values[3])
		must(err)
		if v.lowerInf || v.upperInf || !le || !ue || v.lowerOpen != loOpen || v.upperOpen != hiOpen {
			t.Fatalf("product interval: lower=%t upper=%t open=(%t,%t)", le, ue, v.lowerOpen, v.upperOpen)
		}
		must(v.CheckInvariant())
	}
	for _, tc := range []struct {
		name                   string
		al, au, bl, bu, lo, hi int32
	}{
		{"NN", -3, -1, -3, -1, 1, 9}, {"NM", -3, -1, -2, 2, -6, 6}, {"NP", -3, -1, 1, 3, -9, -1},
		{"MN", -2, 2, -3, -1, -6, 6}, {"MM", -2, 2, -2, 2, -4, 4}, {"MP", -2, 2, 1, 3, -6, 6},
		{"PN", 1, 3, -3, -1, -9, -1}, {"PM", 1, 3, -2, 2, -6, 6}, {"PP", 1, 3, 1, 3, 1, 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set(a, tc.al, tc.au, false, false)
			set(b, tc.bl, tc.bu, false, false)
			must(out.Mul(a, b))
			check(out, tc.lo, tc.hi, false, false)
		})
	}
	set(a, -2, 2, true, true)
	set(b, -2, 2, true, true)
	must(a.Mul(a, b))
	check(a, -4, 4, true, true)
	set(a, -2, -1, false, false)
	must(a.ResetLower())
	set(b, 1, 2, false, false)
	must(out.Mul(a, b))
	if !out.lowerInf || out.upperInf {
		t.Fatal("one-sided infinite product", out.lowerInf, out.upperInf)
	}
	_, actualHi, err := out.endpoints()
	must(err)
	must(m.SetInt32(&values[2], -1))
	eq, err := m.Equal(actualHi, &values[2])
	must(err)
	if !eq || out.upperOpen {
		t.Fatal("one-sided product upper endpoint")
	}
	must(m.SetFractionSigned(&values[1], 1, 3))
	must(a.SetFinite(&values[1], &values[1], false, false))
	must(b.SetFinite(&values[1], &values[1], false, false))
	must(out.Mul(a, b))
	lo, hi, err := out.endpoints()
	must(err)
	less, err := m.Less(lo, hi)
	must(err)
	if !less {
		t.Fatal("directed MPF product bounds collapsed")
	}
}
