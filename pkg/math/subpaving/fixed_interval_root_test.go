package subpaving

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/control"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedIntervalXnEqYRetainedCases(t *testing.T) {
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	m, err := numeral.NewFixedPointManager(2, 1)
	must(err)
	lo, err := m.New()
	must(err)
	hi, err := m.New()
	must(err)
	p, err := m.New()
	must(err)
	a, err := NewFixedInterval(m)
	must(err)
	b, err := NewFixedInterval(m)
	must(err)
	defer func() {
		must(a.Close())
		must(b.Close())
		must(lo.Close())
		must(hi.Close())
		must(p.Close())
		must(m.Close())
	}()
	must(m.SetRatio(p, 1, 8192))
	limit := &control.Reslimit{Limit: 1000}
	roots := interval.FixedRoot{Numbers: m, Limit: limit}
	set := func(l, u int64, lOpen, uOpen bool) {
		t.Helper()
		must(m.SetInt64(lo, l))
		must(m.SetInt64(hi, u))
		must(a.SetFinite(lo, hi, lOpen, uOpen))
	}
	show := func(v *FixedInterval) string {
		t.Helper()
		var out bytes.Buffer
		must(v.Display(&out))
		return out.String()
	}
	set(4, 9, false, false)
	must(b.XnEqY(a, 2, p, roots))
	if got := show(b); got != "[-3, 3]" {
		t.Fatalf("even inverse = %q", got)
	}
	set(4, 9, false, true)
	must(b.XnEqY(a, 2, p, roots))
	if got := show(b); got != "(-3, 3)" {
		t.Fatalf("open even inverse = %q", got)
	}
	set(-8, 27, true, false)
	must(b.XnEqY(a, 3, p, roots))
	if got := show(b); got != "(-2, 3]" {
		t.Fatalf("odd inverse = %q", got)
	}
	must(a.XnEqY(a, 3, p, roots))
	if got := show(a); got != "(-2, 3]" {
		t.Fatalf("aliased odd inverse = %q", got)
	}
	set(0, 9, false, false)
	must(a.ResetUpper())
	must(b.XnEqY(a, 2, p, roots))
	if got := show(b); got != "(-oo, +oo)" {
		t.Fatalf("unbounded even inverse = %q", got)
	}
	set(-8, 8, false, false)
	must(a.ResetLower())
	must(b.XnEqY(a, 3, p, roots))
	if got := show(b); got != "(-oo, 2]" {
		t.Fatalf("one-sided odd inverse = %q", got)
	}
	set(0, 2, true, true)
	must(b.XnEqY(a, 2, p, roots))
	if b.lowerOpen || b.upperOpen {
		t.Fatalf("inexact even root retained open endpoints: %q", show(b))
	}
	set(-1, 9, false, false)
	if err := b.XnEqY(a, 2, p, roots); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatalf("negative even input error = %v", err)
	}
	set(2, 4, false, false)
	limit.Limit, limit.Steps = 1, 0
	if err := b.XnEqY(a, 3, p, roots); !errors.Is(err, interval.ErrFixedRootCanceled) {
		t.Fatalf("root checkpoint error = %v", err)
	}
}
