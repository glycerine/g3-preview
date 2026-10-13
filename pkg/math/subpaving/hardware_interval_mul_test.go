package subpaving

import (
	"math"
	"math/big"
	"testing"

	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingHardwareIntervalMulNineSignClassesAndDirectedBounds(t *testing.T) {
	m, err := numeral.NewHardwareFloatNumeralManager(numeral.NewHardwareFloatManager(), 11, 53)
	if err != nil {
		t.Fatal(err)
	}
	var intervals [3]*HardwareInterval
	for i := range intervals {
		intervals[i], err = NewHardwareInterval(m)
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
	})
	a, b, out := intervals[0], intervals[1], intervals[2]
	var values [2]numeral.HardwareFloat
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	set := func(v *HardwareInterval, lo, hi int32, loOpen, hiOpen bool) {
		t.Helper()
		must(m.SetInt32(&values[0], lo))
		must(m.SetInt32(&values[1], hi))
		must(v.SetFinite(&values[0], &values[1], loOpen, hiOpen))
	}
	check := func(v *HardwareInterval, lo, hi float64, loInf, hiInf, loOpen, hiOpen bool) {
		t.Helper()
		actualLo, actualHi, err := v.endpoints()
		must(err)
		if v.lowerInf != loInf || v.upperInf != hiInf || v.lowerOpen != loOpen || v.upperOpen != hiOpen ||
			(!loInf && m.ToFloat64(actualLo) != lo) || (!hiInf && m.ToFloat64(actualHi) != hi) {
			t.Fatalf("product: [%g,%g], inf=(%t,%t), open=(%t,%t)", m.ToFloat64(actualLo), m.ToFloat64(actualHi), v.lowerInf, v.upperInf, v.lowerOpen, v.upperOpen)
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
			check(out, float64(tc.lo), float64(tc.hi), false, false, false, false)
		})
	}
	set(a, -2, 2, true, true)
	set(b, -2, 2, true, true)
	must(a.Mul(a, b))
	check(a, -4, 4, false, false, true, true)
	set(a, -2, -1, false, false)
	must(a.ResetLower())
	set(b, 1, 2, false, false)
	must(out.Mul(a, b))
	check(out, 0, -1, true, false, true, false)
	set(a, 0, 0, false, false)
	must(out.Mul(a, b))
	check(out, 0, 0, false, false, false, false)
	set(a, -2, 0, true, false)
	set(b, 1, 2, true, true)
	must(out.Mul(a, b))
	check(out, -4, 0, false, false, true, false)
	must(m.SetFloat64(&values[0], math.Nextafter(1, math.Inf(1))))
	must(a.SetFinite(&values[0], &values[0], false, false))
	must(b.SetFinite(&values[0], &values[0], false, false))
	must(out.Mul(a, b))
	lo, hi, err := out.endpoints()
	must(err)
	exact := new(big.Rat).SetFloat64(m.ToFloat64(&values[0]))
	exact.Mul(exact, exact)
	if new(big.Rat).SetFloat64(m.ToFloat64(lo)).Cmp(exact) > 0 || new(big.Rat).SetFloat64(m.ToFloat64(hi)).Cmp(exact) < 0 {
		t.Fatal("directed product does not enclose exact result")
	}
}
