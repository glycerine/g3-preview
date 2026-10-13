package subpaving

import (
	"bytes"
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingHardwareIntervalRetainedCoreAndDirectedArithmetic(t *testing.T) {
	core := numeral.NewHardwareFloatManager()
	m, err := numeral.NewHardwareFloatNumeralManager(core, 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var values [3]numeral.HardwareFloat
	var intervals [3]*HardwareInterval
	for i := range intervals {
		intervals[i], err = NewHardwareInterval(m)
		if err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for i := len(intervals) - 1; i >= 0; i-- {
			if e := intervals[i].Close(); e != nil {
				t.Error(e)
			}
		}
	})
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	set := func(v *HardwareInterval, l, u int32, lop, uop bool) {
		t.Helper()
		must(m.SetInt32(&values[0], l))
		must(m.SetInt32(&values[1], u))
		must(v.SetFinite(&values[0], &values[1], lop, uop))
	}
	check := func(v *HardwareInterval, l, u float64, li, ui, lop, uop bool) {
		t.Helper()
		lo, hi, e := v.endpoints()
		must(e)
		if m.ToFloat64(lo) != l || m.ToFloat64(hi) != u || v.lowerInf != li || v.upperInf != ui || v.lowerOpen != lop || v.upperOpen != uop {
			t.Fatalf("interval [%v,%v], flags (%t,%t,%t,%t)", m.ToFloat64(lo), m.ToFloat64(hi), v.lowerInf, v.upperInf, v.lowerOpen, v.upperOpen)
		}
	}
	a, b, out := intervals[0], intervals[1], intervals[2]
	set(a, -2, 3, false, true)
	set(b, 1, 4, true, false)
	must(out.Add(a, b))
	check(out, -1, 7, false, false, true, true)
	must(out.Sub(a, b))
	check(out, -6, 2, false, false, false, true)
	must(out.Neg(a))
	check(out, -3, 2, false, false, true, false)
	must(a.Add(a, b))
	check(a, -1, 7, false, false, true, true)
	must(b.Sub(b, b))
	check(b, -3, 3, false, false, true, true)
	must(a.Neg(a))
	check(a, -7, 1, false, false, true, true)
	must(out.SetFrom(a))
	equal, e := out.Equal(a)
	must(e)
	if !equal {
		t.Fatal("source copy equality")
	}
	must(out.SetFrom(out))
	set(a, -2, 2, false, false)
	zero, e := a.ContainsZero()
	must(e)
	if !zero {
		t.Fatal("crossing zero")
	}
	must(m.SetInt32(&values[2], 2))
	contains, e := a.Contains(&values[2])
	must(e)
	if !contains {
		t.Fatal("closed upper")
	}
	set(a, -2, 2, false, true)
	contains, e = a.Contains(&values[2])
	must(e)
	if contains {
		t.Fatal("open upper")
	}
	set(b, 2, 4, false, false)
	before, e := a.Before(b)
	must(e)
	if !before {
		t.Fatal("open touching intervals")
	}
	set(a, 2, 2, false, false)
	zero, e = a.ContainsZero()
	must(e)
	if zero {
		t.Fatal("positive singleton")
	}
	if e = a.SetFinite(&values[2], &values[2], true, false); !errors.Is(e, interval.ErrExtendedDomain) {
		t.Fatal("open singleton", e)
	}
	must(a.Reset())
	var display bytes.Buffer
	must(a.Display(&display))
	if display.String() != "(-oo, +oo)" {
		t.Fatal("reset display", display.String())
	}
	must(out.Add(a, b))
	if !out.lowerInf || !out.upperInf {
		t.Fatal("unbounded add")
	}
	must(out.Neg(b))
	check(out, -4, -2, false, false, false, false)
	must(b.ResetLower())
	must(out.Sub(b, a))
	if !out.lowerInf || !out.upperInf {
		t.Fatal("infinite difference")
	}
	// The source selects lower rounding for the lower endpoint and upper
	// rounding for the upper endpoint, independently of ambient mode.
	must(m.SetInt32(&values[0], 1))
	must(m.SetFloat64(&values[1], math.Ldexp(1, -53)))
	must(a.SetFinite(&values[0], &values[0], false, false))
	must(b.SetFinite(&values[1], &values[1], false, false))
	must(out.Add(a, b))
	lo, hi, e := out.endpoints()
	must(e)
	if m.ToFloat64(lo) != 1 || m.ToFloat64(hi) != math.Nextafter(1, math.Inf(1)) {
		t.Fatal("directed half-ulp sum", m.ToFloat64(lo), m.ToFloat64(hi))
	}
	copyOf := *a
	if e = copyOf.Reset(); !errors.Is(e, arena.ErrInvalidSlot) {
		t.Fatal("copied owner", e)
	}
	must(a.Close())
	if e = a.Reset(); !errors.Is(e, arena.ErrClosed) {
		t.Fatal("closed interval", e)
	}
}
