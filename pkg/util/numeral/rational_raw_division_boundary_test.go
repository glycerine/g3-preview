package numeral

import (
	"fmt"
	"strings"
	"testing"
)

// A completed caller's saved generation cannot clean a real subsequent failed
// divisor-alias invocation. Each reservation fixture gets a fresh manager:
// allocation failure may leave source manager scratch holes, and this test does
// not invent a repair/retry policy for those separate source failure prefixes.
func TestRationalDivisionBoundaryGenerationIdentity(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 30 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, budget := rationalManagerWithLimit(t, true, 8<<20)
			a, b := NewRational(), NewRational()
			serviceMust(t, m.SetString(&a, strings.Repeat("1234567890", 100)+"/7"))
			serviceMust(t, m.SetString(&b, strings.Repeat("9876543210", 100)+"/13"))
			t.Cleanup(func() { budget.FailAfter(-1); serviceMust(t, m.Del(&a)); serviceMust(t, m.Del(&b)) })
			err := m.Div(&a, &b, &b)
			serviceMust(t, err)
			older := m.DivisionInvocationState()
			if older.Active || older.Generation == 0 {
				t.Fatal("completed division", older)
			}
			before, err := CaptureNumericAllocationRoots(m.Z)
			serviceMust(t, err)
			budget.FailAfter(ordinal)
			err = m.Div(&a, &b, &b)
			budget.FailAfter(-1)
			var roots NumericAllocationRoots
			serviceMust(t, roots.After(m.Z, before))
			state := m.DivisionInvocationState()
			if state.Active {
				if err == nil || state.Generation == older.Generation {
					t.Fatal("replacement identity", err, state, older)
				}
				obsolete := older.Generation
				serviceMust(t, m.CleanupDivisionInvocation(&obsolete))
				if obsolete != 0 || m.DivisionInvocationState() != state {
					t.Fatal("older generation consumed replacement", m.DivisionInvocationState(), state)
				}
				zero := uint64(0)
				serviceMust(t, m.CleanupDivisionInvocation(&zero))
				if m.DivisionInvocationState() != state {
					t.Fatal("zero generation adopted division")
				}
				matched = true
			}
			generation := state.Generation
			serviceMust(t, m.CleanupDivisionInvocation(&generation))
			if generation != 0 || m.DivisionInvocationState().Active {
				t.Fatal("matching division retained")
			}
			serviceMust(t, CleanupNumericAllocationRoots(m.Z, &roots))
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
		})
	}
	if !matched {
		t.Fatal("no actual retained divisor-alias failure reached")
	}
}

func TestRationalDivisionBoundaryGenerationExhaustion(t *testing.T) {
	m, _ := rationalManagerWithLimit(t, true, 8<<20)
	a, b := NewRational(), NewRational()
	serviceMust(t, m.SetInt64(&a, 3))
	serviceMust(t, m.SetInt64(&b, 2))
	m.RawGeneration = ^uint64(0)
	if err := m.Div(&a, &b, &b); err != ErrRationalRawArithmeticPending {
		t.Fatal(err)
	}
	if state := m.DivisionInvocationState(); state.Active || state.Generation != 0 {
		t.Fatal(state)
	}
	value, err := m.String(&b)
	serviceMust(t, err)
	if value != "2" {
		t.Fatal("exhaustion modified destination", value)
	}
	serviceMust(t, m.Del(&a))
	serviceMust(t, m.Del(&b))
}
