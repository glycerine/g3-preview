package numeral

import (
	"errors"
	"testing"
)

func TestFixedPointRetainedOrderingSignsIncrementAndSwap(t *testing.T) {
	m, err := NewFixedPointManager(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	a, _ := m.New()
	b, _ := m.New()
	c, _ := m.New()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	defer func() { must(a.Close()); must(b.Close()); must(c.Close()); must(m.Close()) }()
	want := func(v *FixedPoint, expected string) {
		t.Helper()
		r, err := m.Rational(v)
		must(err)
		if got := r.RatString(); got != expected {
			t.Fatalf("value = %s, want %s", got, expected)
		}
	}
	must(m.SetRatio(a, -1, 2))
	must(m.Reset(b))
	must(m.SetRatio(c, 1, 2))
	for _, tc := range []struct {
		v                        *FixedPoint
		pos, neg, nonpos, nonneg bool
	}{
		{a, false, true, true, false},
		{b, false, false, true, true},
		{c, true, false, false, true},
	} {
		gotPos, err := m.IsPos(tc.v)
		must(err)
		gotNeg, err := m.IsNeg(tc.v)
		must(err)
		gotNonPos, err := m.IsNonPos(tc.v)
		must(err)
		gotNonNeg, err := m.IsNonNeg(tc.v)
		must(err)
		if gotPos != tc.pos || gotNeg != tc.neg || gotNonPos != tc.nonpos || gotNonNeg != tc.nonneg {
			t.Fatalf("sign predicates = %t,%t,%t,%t", gotPos, gotNeg, gotNonPos, gotNonNeg)
		}
	}
	less, err := m.Less(a, b)
	must(err)
	greater, err := m.Greater(c, b)
	must(err)
	le, err := m.LessEqual(a, a)
	must(err)
	ge, err := m.GreaterEqual(c, c)
	must(err)
	eq, err := m.Equal(a, c)
	must(err)
	if !less || !greater || !le || !ge || eq {
		t.Fatalf("ordering predicates = %t,%t,%t,%t,%t", less, greater, le, ge, eq)
	}
	must(m.Inc(a))
	want(a, "1/2")
	eq, err = m.Equal(a, c)
	must(err)
	if !eq {
		t.Fatal("increment failed to reach equal half")
	}
	must(m.Dec(a))
	want(a, "-1/2")
	must(m.Swap(a, c))
	want(a, "1/2")
	want(c, "-1/2")
	must(m.Swap(a, a))
	want(a, "1/2")
	must(m.SetInt64(a, 1))
	one, err := m.IsOne(a)
	must(err)
	if !one {
		t.Fatal("one predicate failed")
	}
	must(m.Neg(a))
	minusOne, err := m.IsMinusOne(a)
	must(err)
	if !minusOne {
		t.Fatal("minus one predicate failed")
	}
	must(m.SetUint64(a, (1<<32)-1))
	must(m.SetRatio(b, 1, 2))
	must(m.Add(a, b, a))
	if err := m.Inc(a); !errors.Is(err, ErrFixedPointOverflow) {
		t.Fatalf("increment overflow = %v", err)
	}
	want(a, "8589934591/2")
	must(m.Neg(a))
	if err := m.Dec(a); !errors.Is(err, ErrFixedPointOverflow) {
		t.Fatalf("decrement overflow = %v", err)
	}
	want(a, "-8589934591/2")
	other, err := NewFixedPointManager(1, 1)
	must(err)
	foreign, err := other.New()
	must(err)
	if err := m.Swap(a, foreign); !errors.Is(err, ErrFixedPointOwner) {
		t.Fatalf("foreign swap = %v", err)
	}
	must(foreign.Close())
	must(other.Close())
}
