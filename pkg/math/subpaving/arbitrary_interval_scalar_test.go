package subpaving

import (
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingMPFIntervalScalarMulDiv(t *testing.T) {
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
	var values [5]numeral.ArbitraryFloat
	a, err := NewArbitraryInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	out, err := NewArbitraryInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := out.Close(); err != nil {
			t.Error(err)
		}
		if err := a.Close(); err != nil {
			t.Error(err)
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
			t.Error("MPF scalar roots", budget.Used())
		}
	})
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	k := &values[4]
	check := func(v *ArbitraryInterval, lo, hi int32, li, ui, lop, uop bool) {
		t.Helper()
		actualLo, actualHi, err := v.endpoints()
		must(err)
		if v.lowerInf != li || v.upperInf != ui || v.lowerOpen != lop || v.upperOpen != uop {
			t.Fatalf("scalar shape inf=(%t,%t) open=(%t,%t)", v.lowerInf, v.upperInf, v.lowerOpen, v.upperOpen)
		}
		if !li {
			must(m.SetInt32(&values[2], lo))
			eq, err := m.Equal(actualLo, &values[2])
			must(err)
			if !eq {
				t.Fatal("scalar lower")
			}
		}
		if !ui {
			must(m.SetInt32(&values[3], hi))
			eq, err := m.Equal(actualHi, &values[3])
			must(err)
			if !eq {
				t.Fatal("scalar upper")
			}
		}
		must(v.CheckInvariant())
	}
	must(m.SetInt32(&values[0], -1))
	must(m.SetInt32(&values[1], 2))
	must(a.SetFinite(&values[0], &values[1], true, false))
	must(m.SetInt32(k, -2))
	must(out.MulScalar(k, a))
	check(out, -4, 2, false, false, false, true)
	must(a.MulScalar(k, a))
	check(a, -4, 2, false, false, false, true)
	must(m.SetInt32(k, 2))
	must(out.DivScalar(a, k))
	check(out, -2, 1, false, false, false, true)
	must(a.ResetLower())
	must(out.MulScalar(k, a))
	check(out, 0, 4, true, false, true, true)
	must(m.SetInt32(k, -2))
	must(out.DivScalar(a, k))
	check(out, -1, 0, false, true, true, true)
	must(m.Reset(k))
	must(out.MulScalar(k, a))
	check(out, 0, 0, true, true, true, true)
	must(out.DivScalar(a, k))
	check(out, 0, 0, true, true, true, true)
	must(m.SetInt32(&values[0], 1))
	must(m.SetInt32(&values[1], 2))
	must(a.SetFinite(&values[0], &values[1], false, false))
	must(m.SetInt32(k, 3))
	must(out.DivScalar(a, k))
	lo, hi, err := out.endpoints()
	must(err)
	m.RoundToMinusInf()
	must(m.SetFractionSigned(&values[0], 1, 3))
	m.RoundToPlusInf()
	must(m.SetFractionSigned(&values[1], 2, 3))
	lowerBeyond, err := m.Less(&values[0], lo)
	must(err)
	upperShort, err := m.Less(hi, &values[1])
	must(err)
	if lowerBeyond || upperShort {
		t.Fatal("directed scalar quotient excludes exact rational endpoints")
	}
	must(out.CheckInvariant())
}
