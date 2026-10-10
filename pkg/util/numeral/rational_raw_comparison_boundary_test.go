package numeral

import (
	"fmt"
	"strings"
	"testing"
)

// A completed caller's saved generation cannot clean a real subsequent failed
// cross-product invocation. Each reservation fixture gets a fresh manager:
// allocation failure may leave source manager scratch holes, and this test does
// not invent a repair/retry policy for those separate source failure prefixes.
func TestRationalComparisonBoundaryGenerationIdentity(t *testing.T) {
	matched := false
	for ordinal := int64(0); ordinal < 30 && !matched; ordinal++ {
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			m, budget := rationalManagerWithLimit(t, true, 8<<20)
			a, b := NewRational(), NewRational()
			serviceMust(t, m.SetString(&a, strings.Repeat("1234567890", 100)+"/7"))
			serviceMust(t, m.SetString(&b, strings.Repeat("9876543210", 100)+"/13"))
			t.Cleanup(func() { budget.FailAfter(-1); serviceMust(t, m.Del(&a)); serviceMust(t, m.Del(&b)) })
			_, err := m.Compare(&a, &b)
			serviceMust(t, err)
			older := m.ComparisonInvocationState()
			if older.Active || older.Generation == 0 {
				t.Fatal("completed comparison", older)
			}
			before, err := CaptureNumericAllocationRoots(m.Z)
			serviceMust(t, err)
			budget.FailAfter(ordinal)
			_, err = m.Compare(&a, &b)
			budget.FailAfter(-1)
			var roots NumericAllocationRoots
			serviceMust(t, roots.After(m.Z, before))
			state := m.ComparisonInvocationState()
			if state.Active {
				if err == nil || state.Generation == older.Generation {
					t.Fatal("replacement identity", err, state, older)
				}
				obsolete := older.Generation
				serviceMust(t, m.CleanupComparisonInvocation(&obsolete))
				if obsolete != 0 || m.ComparisonInvocationState() != state {
					t.Fatal("older generation consumed replacement", m.ComparisonInvocationState(), state)
				}
				zero := uint64(0)
				serviceMust(t, m.CleanupComparisonInvocation(&zero))
				if m.ComparisonInvocationState() != state {
					t.Fatal("zero generation adopted comparison")
				}
				matched = true
			}
			generation := state.Generation
			serviceMust(t, m.CleanupComparisonInvocation(&generation))
			if generation != 0 || m.ComparisonInvocationState().Active {
				t.Fatal("matching comparison retained")
			}
			serviceMust(t, CleanupNumericAllocationRoots(m.Z, &roots))
			qopsAuditLiveOwnership(t, m, []*Int{&a.Num, &a.Den, &b.Num, &b.Den})
		})
	}
	if !matched {
		t.Fatal("no actual retained cross-product failure reached")
	}
}
