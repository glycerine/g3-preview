package subpaving

import (
	"errors"
	"math"
	"testing"

	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingHardwareIntervalXnEqYRetainedCases(t *testing.T) {
	m, src, out, values := hardwareIntervalFixture(t)
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	set := func(lo, hi int32, loOpen, hiOpen bool) {
		t.Helper()
		must(m.SetInt32(&values[0], lo))
		must(m.SetInt32(&values[1], hi))
		must(src.SetFinite(&values[0], &values[1], loOpen, hiOpen))
	}
	check := func(v *HardwareInterval, lo, hi float64, li, ui, lop, uop bool) {
		t.Helper()
		a, b, err := v.endpoints()
		must(err)
		if v.lowerInf != li || v.upperInf != ui || v.lowerOpen != lop || v.upperOpen != uop ||
			(!li && math.Abs(m.ToFloat64(a)-lo) > 1e-10) || (!ui && math.Abs(m.ToFloat64(b)-hi) > 1e-10) {
			t.Fatalf("root [%g,%g] inf=(%t,%t) open=(%t,%t)", m.ToFloat64(a), m.ToFloat64(b), v.lowerInf, v.upperInf, v.lowerOpen, v.upperOpen)
		}
		must(v.CheckInvariant())
	}
	var precision numeral.HardwareFloat
	must(m.SetRatioInt32(&precision, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	roots := interval.HardwareRoot{Numbers: m, Limit: limit}
	set(1, 4, false, true)
	must(out.XnEqY(src, 2, &precision, roots))
	check(out, -2, 2, false, false, false, false)
	set(1, 4, false, false)
	must(src.XnEqY(src, 2, &precision, roots))
	check(src, -2, 2, false, false, false, false)
	set(0, 1, false, true)
	must(out.XnEqY(src, 2, &precision, roots))
	check(out, -1, 1, false, false, true, true)
	set(-8, 27, true, false)
	must(out.XnEqY(src, 3, &precision, roots))
	check(out, -2, 3, false, false, false, false)
	set(-8, 27, true, true)
	must(out.NthRoot(src, 3, &precision, roots))
	check(out, -2, 3, false, false, false, false)
	set(-8, 27, false, false)
	must(src.ResetLower())
	must(out.XnEqY(src, 3, &precision, roots))
	check(out, 0, 3, true, false, true, false)
	set(1, 4, false, false)
	must(src.ResetUpper())
	must(out.XnEqY(src, 2, &precision, roots))
	check(out, 0, 0, true, true, true, true)
	must(out.NthRoot(src, 2, &precision, roots))
	check(out, 1, 0, false, true, false, true)
	set(-1, 4, false, false)
	if err := out.XnEqY(src, 2, &precision, roots); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatal("negative even inverse", err)
	}
	set(1, 4, false, false)
	limit.Limit, limit.Steps = 1, 0
	if err := out.XnEqY(src, 2, &precision, roots); !errors.Is(err, interval.ErrHardwareRootCanceled) {
		t.Fatal("inverse root checkpoint", err)
	}
}
