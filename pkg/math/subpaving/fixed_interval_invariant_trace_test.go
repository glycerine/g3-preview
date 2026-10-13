package subpaving

import (
	"bytes"
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/math/interval"
	"github.com/glycerine/g3/pkg/util/diagnostic"
	"github.com/glycerine/g3/pkg/util/numeral"
)

func TestSubpavingFixedIntervalInvariantUsesExtendedLess(t *testing.T) {
	m, err := numeral.NewFixedPointManager(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	lo, _ := m.New()
	hi, _ := m.New()
	v, err := NewFixedInterval(m)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, close := range []func() error{v.Close, lo.Close, hi.Close, m.Close} {
			if err := close(); err != nil {
				t.Error(err)
			}
		}
	})
	var output bytes.Buffer
	tags := diagnostic.NewTraceTagsWithWriter(&output)
	if err := m.BindTrace(tags); err != nil {
		t.Fatal(err)
	}
	if err := v.CheckInvariant(); err != nil || output.Len() != 0 {
		t.Fatal("infinite invariant", err, output.String())
	}
	if err := m.SetInt64(lo, 1); err != nil {
		t.Fatal(err)
	}
	if err := m.SetInt64(hi, 2); err != nil {
		t.Fatal(err)
	}
	if err := v.SetFinite(lo, hi, false, false); err != nil {
		t.Fatal(err)
	}
	tags.Enable("mpfx_trace")
	check := func(want string) {
		t.Helper()
		if !tags.ShouldTrace("mpfx_trace") {
			want = ""
		}
		if got := output.String(); got != want {
			t.Fatalf("invariant trace = %q, want %q", got, want)
		}
		output.Reset()
	}
	if err := v.CheckInvariant(); err != nil {
		t.Fatal(err)
	}
	check("[mpfx] (1 < 2) == (1 == 1)\n")
	if err := m.SetInt64(v.upper, 1); err != nil {
		t.Fatal(err)
	}
	if err := v.CheckInvariant(); err != nil {
		t.Fatal(err)
	}
	check("") // equality returns before source less-than
	if err := m.SetInt64(v.lower, 3); err != nil {
		t.Fatal(err)
	}
	if err := v.CheckInvariant(); !errors.Is(err, interval.ErrExtendedDomain) {
		t.Fatalf("reversed interval = %v", err)
	}
	check("[mpfx] (3 < 1) == (0 == 1)\n")
}
