package subpaving

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	umemory "github.com/glycerine/g3/pkg/util/memory"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingMPFIntervalXnEqYRetainedCases(t *testing.T) {
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
	var values [3]numeral.ArbitraryFloat
	a, err := NewArbitraryInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	b, err := NewArbitraryInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, e := range []error{b.Close(), a.Close(), m.Del(&values[2]), m.Del(&values[1]), m.Del(&values[0]), m.Close(), core.Close(), runtime.Close()} {
			if e != nil {
				t.Error(e)
			}
		}
		if budget.Used() != 0 {
			t.Error("MPF inverse owners", budget.Used())
		}
	})
	lo, hi, p := &values[0], &values[1], &values[2]
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(m.SetFractionSigned(p, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	roots := interval.ArbitraryRoot{Numbers: m, Limit: limit}
	set := func(l, u int32, lOpen, uOpen bool) {
		t.Helper()
		must(m.SetInt32(lo, l))
		must(m.SetInt32(hi, u))
		must(a.SetFinite(lo, hi, lOpen, uOpen))
	}
	checkEncloses := func(v *ArbitraryInterval, lower, upper int32) {
		t.Helper()
		gotLower, gotUpper, e := v.endpoints()
		must(e)
		must(m.SetInt32(lo, lower))
		must(m.SetInt32(hi, upper))
		tooHigh, e := m.Greater(gotLower, lo)
		must(e)
		tooLow, e := m.Less(gotUpper, hi)
		must(e)
		if tooHigh || tooLow {
			t.Fatalf("inverse misses [%d,%d]", lower, upper)
		}
	}
	set(4, 9, false, false)
	must(b.XnEqY(a, 2, p, roots))
	checkEncloses(b, -3, 3)
	set(4, 9, false, true)
	must(b.XnEqY(a, 2, p, roots))
	if b.lowerOpen || b.upperOpen {
		t.Fatal("inexact even inverse kept open bounds")
	}
	set(-8, 27, true, false)
	must(b.XnEqY(a, 3, p, roots))
	checkEncloses(b, -2, 3)
	must(a.XnEqY(a, 3, p, roots))
	checkEncloses(a, -2, 3)
	set(4, 9, false, false)
	must(a.ResetUpper())
	must(b.XnEqY(a, 2, p, roots))
	if !b.lowerInf || !b.upperInf {
		t.Fatal("unbounded even inverse")
	}
	set(-8, 8, false, false)
	must(a.ResetLower())
	must(b.XnEqY(a, 3, p, roots))
	if !b.lowerInf || b.upperInf {
		t.Fatal("one-sided odd inverse")
	}
	set(-1, 9, false, false)
	if e := b.XnEqY(a, 2, p, roots); !errors.Is(e, interval.ErrExtendedDomain) {
		t.Fatal("negative even input", e)
	}
	set(2, 4, false, false)
	limit.Limit, limit.Steps = 1, 0
	if e := b.XnEqY(a, 3, p, roots); !errors.Is(e, interval.ErrArbitraryRootCanceled) {
		t.Fatal("root checkpoint", e)
	}
}
