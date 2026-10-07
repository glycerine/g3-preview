// Source: src/util/resource_limit.h:49-164.
package util

import (
	"errors"
	"math"
	"testing"
)

func TestSearchResourceLimitCounterBoundaryAndReset(t *testing.T) {
	r := newUtilLinearResourceLimit(2, 10, 3, searchResourceDescription("conflicts\x00ignored"))
	if text, present := r.getDescription(); !present || text != "conflicts" {
		t.Fatal("borrowed CString prefix", text, present)
	}
	if r.getCounter() != 0 || r.getLimit() != 2 || r.exhausted() {
		t.Fatal("source constructor state", r)
	}
	for i := int32(1); i <= 3; i++ {
		ok, err := r.incCounter()
		if err != nil || ok != (i <= 2) || r.getCounter() != i || r.exhausted() != (i >= 2) {
			t.Fatal("source <= increment versus >= exhaustion", i, ok, err, r)
		}
	}
	r.setInitLimit(7)
	if r.getCounter() != 3 || r.getLimit() != 7 {
		t.Fatal("setter reset counter", r)
	}
	r.resetCounters()
	if r.getCounter() != 0 || r.getLimit() != 7 {
		t.Fatal("counter reset changed limit", r)
	}
	if ok, err := r.incLimit(); err != nil || !ok || r.getLimit() != 10 {
		t.Fatal(ok, err, r)
	}
	if ok, err := r.incLimit(); err != nil || ok || r.getLimit() != 10 {
		t.Fatal("rejected increment published", ok, err, r)
	}
	r.reset()
	if r.getCounter() != 0 || r.getLimit() != 7 {
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
		r := newUtilLinearResourceLimit(f.initial, f.maximum, f.increment, searchResourceDescription("steps"))
		ok, err := r.incLimit()
		if err != nil || ok != f.accepted || r.getLimit() != f.want {
			t.Fatal(f, ok, err, r)
		}
	}
	r := newUtilLinearResourceLimit(3, 4, 2, searchResourceDescription("steps"))
	r.setMaxLimit(-8)
	if ok, err := r.incLimit(); err != nil || !ok || r.getLimit() != 5 {
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
		r := newUtilExponentialResourceLimit(f.initial, f.maximum, f.ratio, searchResourceDescription("branches"))
		ok, err := r.incLimit()
		if err != nil || ok != f.accepted || r.getLimit() != f.want {
			t.Fatal(f, ok, err, r)
		}
	}
}

func TestSearchResourceLimitGuardedUndefinedArithmeticDomains(t *testing.T) {
	for _, f := range []struct{ initial, increment int32 }{{math.MaxInt32, 1}, {math.MinInt32, -1}} {
		r := newUtilLinearResourceLimit(f.initial, -1, f.increment, searchResourceDescription("overflow"))
		if ok, err := r.incLimit(); ok || !errors.Is(err, errSearchResourceLimitDomain) || r.getLimit() != f.initial {
			t.Fatal(f, ok, err, r)
		}
	}
	for _, ratio := range []float64{math.NaN(), math.Inf(1), math.Inf(-1), 1 << 31, -(1 << 31) - 1} {
		r := newUtilExponentialResourceLimit(1, -1, ratio, searchResourceDescription("conversion"))
		if ok, err := r.incLimit(); ok || !errors.Is(err, errSearchResourceLimitDomain) || r.getLimit() != 1 {
			t.Fatal(ratio, ok, err, r)
		}
	}
	r := newUtilLinearResourceLimit(1, -1, 1, searchResourceDescription("counter"))
	r.counter = math.MaxInt32
	if ok, err := r.incCounter(); ok || !errors.Is(err, errSearchResourceLimitDomain) || r.getCounter() != math.MaxInt32 {
		t.Fatal(ok, err, r)
	}
}

func TestSearchResourceLimitIndependentActionHistories(t *testing.T) {
	// An independent state table enumerates all 5^5 action sequences. Bounds
	// are deliberately small, so signed-overflow adaptations play no role.
	for code := 0; code < 3125; code++ {
		r := newUtilLinearResourceLimit(2, 6, 2, searchResourceDescription("history"))
		counter, limit, initial, maximum := int32(0), int32(2), int32(2), int32(6)
		v := code
		for step := 0; step < 5; step++ {
			switch v % 5 {
			case 0:
				counter++
				ok, err := r.incCounter()
				if err != nil || ok != (counter <= limit) {
					t.Fatal(code, step, ok, err)
				}
			case 1:
				accepted := maximum < 0 || limit+2 <= maximum
				if accepted {
					limit += 2
				}
				ok, err := r.incLimit()
				if err != nil || ok != accepted {
					t.Fatal(code, step, ok, err)
				}
			case 2:
				r.reset()
				counter, limit = 0, initial
			case 3:
				r.setInitLimit(-1)
				initial, limit = -1, -1
			case 4:
				r.setMaxLimit(0)
				maximum = 0
			}
			v /= 5
			if r.getCounter() != counter || r.getLimit() != limit || r.exhausted() != (counter >= limit) {
				t.Fatal(code, step, r, counter, limit)
			}
		}
	}
}

func searchResourceDescription(text string) *string { return &text }

func TestSearchResourceLimitBorrowedNullableDescriptions(t *testing.T) {
	r := newUtilLinearResourceLimit(1, -1, 1, nil)
	if text, present := r.getDescription(); present || text != "" {
		t.Fatal("source nullptr description", text, present)
	}
	empty := ""
	e := newUtilExponentialResourceLimit(1, -1, 2, &empty)
	if text, present := e.getDescription(); !present || text != "" {
		t.Fatal("present empty description", text, present)
	}
	original := "conflicts\x00suffix"
	b := newUtilLinearResourceLimit(1, -1, 1, &original)
	original = "replacement"
	if text, present := b.getDescription(); !present || text != "conflicts" {
		t.Fatal("borrow must preserve original spelling, not pointer variable", text, present)
	}
}

func TestSearchResourceLimitColdVirtualContract(t *testing.T) {
	linear := newUtilLinearResourceLimit(2, 8, 2, nil)
	exponential := newUtilExponentialResourceLimit(2, 8, 2, nil)
	for _, r := range []utilSearchResourceLimit{&linear, &exponential} {
		if r.exhausted() || r.getLimit() != 2 {
			t.Fatal("initial virtual state")
		}
		ok, err := r.incLimit()
		if err != nil || !ok || r.getLimit() != 4 {
			t.Fatal("virtual limit", ok, err)
		}
		r.resetCounters()
		r.reset()
		if r.getLimit() != 2 || r.exhausted() {
			t.Fatal("virtual reset")
		}
	}
}
