package numeral

import (
	"strconv"
	"testing"
)

func TestRationalRootInvocationActualRAWAndLaterIntegerCall(t *testing.T) {
	found := false
	for allowance := int64(0); allowance < 256 && !found; allowance++ {
		if !t.Run(strconv.FormatInt(allowance, 10), func(t *testing.T) {
			m, budget, _ := rawArithmeticGuardOwner(t)
			a, out := NewRational(), NewRational()
			serviceMust(t, m.SetString(&a, "340282366920938463574714963092423127049/49"))
			var receipt RationalRootInvocation
			budget.FailAfter(allowance)
			_, body := m.RootWithInvocation(&a, 2, &out, &receipt)
			budget.FailAfter(-1)
			z := m.Z
			if z.MatchesRootInvocation(&receipt.roots[0]) || z.MatchesRootInvocation(&receipt.roots[1]) {
				found = true
				if body == nil || receipt.SourceCause() != nil || z.RootRaw.EndEntered {
					t.Fatal("actual RAW failure acquired a source end-label", body)
				}
				index := z.RootRaw.SourceIndex
				serviceMust(t, receipt.Cleanup(false))
				if !receipt.Pending() || z.RootRaw.SourceIndex != index {
					t.Fatal("completed-child acknowledgement consumed parent RAW")
				}
				serviceMust(t, m.Del(&out))
				serviceMust(t, receipt.Cleanup(true))
				if receipt.Pending() || z.RootRaw.SourceIndex != index {
					t.Fatal("owning root cleanup replayed source end-label")
				}
				var later integerRootInvocationReceipt
				budget.FailAfter(0)
				_, next := z.rootInPlaceWithReceipt(&a.Num, 2, &later)
				budget.FailAfter(-1)
				if next == nil || !z.matchesRootInvocation(&later) {
					t.Fatal("later actual integer root did not retain RAW", next)
				}
				values, pending, used := z.RootRaw.Values, z.PendingCell, budget.Used()
				serviceMust(t, receipt.Cleanup(false))
				serviceMust(t, receipt.Cleanup(true))
				if !z.matchesRootInvocation(&later) || z.RootRaw.Values != values || z.PendingCell != pending || budget.Used() != used {
					t.Fatal("old rational-root receipt acquired a later integer call")
				}
				serviceMust(t, z.CleanupRootInvocation(&later, true))
			} else {
				serviceMust(t, receipt.Cleanup(false))
				serviceMust(t, m.Del(&out))
				serviceMust(t, receipt.Cleanup(true))
			}
			serviceMust(t, z.CleanupPendingAllocation())
			serviceMust(t, m.Del(&a))
		}) {
			return
		}
	}
	if !found {
		t.Fatal("no actual rational-root integer RAW child reached")
	}
}
