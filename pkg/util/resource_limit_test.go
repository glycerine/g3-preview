// Source: src/util/resource_limit.h:49-164.
package util

import (
	"errors"
	"math"
	"testing"
)

func TestSearchResourceLimitCounterBoundaryAndReset(t *testing.T) {
	r := NewLinearResourceLimit(2, 10, 3, searchResourceDescription("conflicts\x00ignored"))
	if text, present := r.GetDescription(); !present || text != "conflicts" {
		t.Fatal("borrowed CString prefix", text, present)
	}
	if r.GetCounter() != 0 || r.GetLimit() != 2 || r.Exhausted() {
		t.Fatal("source constructor state", r)
	}
	for i := int32(1); i <= 3; i++ {
		ok, err := r.IncCounter()
		if err != nil || ok != (i <= 2) || r.GetCounter() != i || r.Exhausted() != (i >= 2) {
			t.Fatal("source <= increment versus >= exhaustion", i, ok, err, r)
		}
	}
	r.SetInitLimit(7)
	if r.GetCounter() != 3 || r.GetLimit() != 7 {
		t.Fatal("setter reset counter", r)
	}
	r.ResetCounters()
	if r.GetCounter() != 0 || r.GetLimit() != 7 {
		t.Fatal("counter reset changed limit", r)
	}
	if ok, err := r.IncLimit(); err != nil || !ok || r.GetLimit() != 10 {
		t.Fatal(ok, err, r)
	}
	if ok, err := r.IncLimit(); err != nil || ok || r.GetLimit() != 10 {
		t.Fatal("rejected increment published", ok, err, r)
	}
	r.Reset()
	if r.GetCounter() != 0 || r.GetLimit() != 7 {
		t.Fatal("full reset", r)
	}
}

func TestSearchResourceLimitLinearSignedPolicies(t *testing.T) {
	for _, f := range []struct {
		initial, maximum, increment, want int32
		accepted                          bool
	}{
		{3, -1, 4, 7, true}, {3, 7, 4, 7, true}, {3, 6, 4, 3, false},
		{3, 3, 0, 3, true}, {3, 1, -2, 1, true}, {-3, -1, -2, -5, true},
		{5, 0, 1, 5, false},
	} {
		r := NewLinearResourceLimit(f.initial, f.maximum, f.increment, searchResourceDescription("steps"))
		ok, err := r.IncLimit()
		if err != nil || ok != f.accepted || r.GetLimit() != f.want {
			t.Fatal(f, ok, err, r)
		}
	}
	r := NewLinearResourceLimit(3, 4, 2, searchResourceDescription("steps"))
	r.SetMaxLimit(-8)
	if ok, err := r.IncLimit(); err != nil || !ok || r.GetLimit() != 5 {
		t.Fatal("negative maximum is unbounded", ok, err)
	}
}

func TestSearchResourceLimitExponentialTruncatesBeforeMaximum(t *testing.T) {
	for _, f := range []struct {
		initial, maximum int32
		ratio            float64
		want             int32
		accepted         bool
	}{
		{3, 4, 1.5, 4, true}, {3, 3, 1.5, 3, false}, {-3, -1, 1.5, -4, true},
		{3, -1, 0.5, 1, true}, {3, -1, 0, 0, true}, {3, -1, -1.5, -4, true},
		{7, 7, 1, 7, true}, {math.MaxInt32, -1, 1, math.MaxInt32, true},
	} {
		r := NewExponentialResourceLimit(f.initial, f.maximum, f.ratio, searchResourceDescription("branches"))
		ok, err := r.IncLimit()
		if err != nil || ok != f.accepted || r.GetLimit() != f.want {
			t.Fatal(f, ok, err, r)
		}
	}
}

func TestSearchResourceLimitGuardedUndefinedArithmeticDomains(t *testing.T) {
	for _, f := range []struct{ initial, increment int32 }{{math.MaxInt32, 1}, {math.MinInt32, -1}} {
		r := NewLinearResourceLimit(f.initial, -1, f.increment, searchResourceDescription("overflow"))
		if ok, err := r.IncLimit(); ok || !errors.Is(err, ErrResourceLimitDomain) || r.GetLimit() != f.initial {
			t.Fatal(f, ok, err, r)
		}
	}
	for _, ratio := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 1 << 31, -(1 << 31) - 1} {
		r := NewExponentialResourceLimit(1, -1, ratio, searchResourceDescription("conversion"))
		if ok, err := r.IncLimit(); ok || !errors.Is(err, ErrResourceLimitDomain) || r.GetLimit() != 1 {
			t.Fatal(ratio, ok, err, r)
		}
	}
	r := NewLinearResourceLimit(1, -1, 1, searchResourceDescription("counter"))
	r.counter = math.MaxInt32
	if ok, err := r.IncCounter(); ok || !errors.Is(err, ErrResourceLimitDomain) || r.GetCounter() != math.MaxInt32 {
		t.Fatal(ok, err, r)
	}
}

func TestSearchResourceLimitIndependentActionHistories(t *testing.T) {
	// An independent state table enumerates all 5^5 action sequences. Bounds
	// are deliberately small, so signed-overflow adaptations play no role.
	for code := 0; code < 3125; code++ {
		r := NewLinearResourceLimit(2, 6, 2, searchResourceDescription("history"))
		counter, limit, initial, maximum := int32(0), int32(2), int32(2), int32(6)
		v := code
		for step := 0; step < 5; step++ {
			switch v % 5 {
			case 0:
				counter++
				ok, err := r.IncCounter()
				if err != nil || ok != (counter <= limit) {
					t.Fatal(code, step, ok, err)
				}
			case 1:
				accepted := maximum < 0 || limit+2 <= maximum
				if accepted {
					limit += 2
				}
				ok, err := r.IncLimit()
				if err != nil || ok != accepted {
					t.Fatal(code, step, ok, err)
				}
			case 2:
				r.Reset()
				counter, limit = 0, initial
			case 3:
				r.SetInitLimit(-1)
				initial, limit = -1, -1
			case 4:
				r.SetMaxLimit(0)
				maximum = 0
			}
			v /= 5
			if r.GetCounter() != counter || r.GetLimit() != limit || r.Exhausted() != (counter >= limit) {
				t.Fatal(code, step, r, counter, limit)
			}
		}
	}
}

func searchResourceDescription(text string) *string { return &text }

func TestSearchResourceLimitBorrowedNullableDescriptions(t *testing.T) {
	r := NewLinearResourceLimit(1, -1, 1, nil)
	if text, present := r.GetDescription(); present || text != "" {
		t.Fatal("source nullptr description", text, present)
	}
	empty := ""
	e := NewExponentialResourceLimit(1, -1, 2, &empty)
	if text, present := e.GetDescription(); !present || text != "" {
		t.Fatal("present empty description", text, present)
	}
	original := "conflicts\x00suffix"
	b := NewLinearResourceLimit(1, -1, 1, &original)
	original = "replacement"
	if text, present := b.GetDescription(); !present || text != "conflicts" {
		t.Fatal("borrow must preserve original spelling, not pointer variable", text, present)
	}
}

func TestSearchResourceLimitColdVirtualContract(t *testing.T) {
	linear := NewLinearResourceLimit(2, 8, 2, nil)
	exponential := NewExponentialResourceLimit(2, 8, 2, nil)
	for _, r := range []ResourceLimit{&linear, &exponential} {
		if r.Exhausted() || r.GetLimit() != 2 {
			t.Fatal("initial virtual state")
		}
		ok, err := r.IncLimit()
		if err != nil || !ok || r.GetLimit() != 4 {
			t.Fatal("virtual limit", ok, err)
		}
		r.ResetCounters()
		r.Reset()
		if r.GetLimit() != 2 || r.Exhausted() {
			t.Fatal("virtual reset")
		}
	}
}
