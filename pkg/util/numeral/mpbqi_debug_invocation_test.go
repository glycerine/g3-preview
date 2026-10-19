//go:build g3debug

package numeral

import (
	"errors"
	"strconv"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestBinaryIntervalDebugReceiptActualRAWAndCleanupReads(t *testing.T) {
	found := false
	for prefix := int64(0); prefix < 1024 && !found; prefix++ {
		if !t.Run(strconv.FormatInt(prefix, 10), func(t *testing.T) {
			qm, budget, runtime := rawArithmeticGuardOwner(t)
			config := serviceConfig(budget)
			config.Integers.Memory = runtime
			config.Integers.Profile = umemory.SmallAllocatorProfile{PointerBits: 32, Debug: true}
			diagnostics, e := NewRationalService(config)
			serviceMust(t, e)
			b, e := NewBinaryRationalManager(qm.Z, BinaryRationalConfig{Diagnostics: diagnostics})
			serviceMust(t, e)
			m, e := NewBinaryIntervalManager(b)
			serviceMust(t, e)
			t.Cleanup(func() {
				serviceMust(t, m.CloseChecked())
				serviceMust(t, b.CloseChecked())
				serviceMust(t, diagnostics.Close())
			})
			var a, other, out BinaryInterval
			intervalSet(t, m, &a, 1, 2)
			intervalSet(t, m, &other, 1, 2)
			n := NewInt(0)
			serviceMust(t, qm.Z.SetString(&n, "9223372036854775809"))
			serviceMust(t, b.SetIntegerPower(a.BorrowUpper(), &n, 63))
			serviceMust(t, qm.Z.Del(&n))
			var receipt BinaryIntervalDebugInvocationReceipt
			budget.FailAfter(prefix)
			body := m.AddWithDebugReceipt(&a, &other, &out, &receipt)
			budget.FailAfter(-1)
			scope := &b.toOwnedRationalScope
			if m.DebugInvocationPending(&receipt) && scope.pending && scope.cursor == 5 && scope.sourceCause == nil {
				found = true
				if body == nil || m.DebugInvocationSourceCause(&receipt) != nil {
					t.Fatal("RAW-only debug failure acquired a source destructor stop", body)
				}
				if _, e = b.IsNonNeg(a.BorrowLower()); !errors.Is(e, ErrBinaryOwnedRationalPending) {
					t.Fatal("ordinary reads crossed pending debug RAW", e)
				}
				yes, e := b.IsNonNegForCleanup(a.BorrowLower())
				serviceMust(t, e)
				if !yes {
					t.Fatal("source cleanup sign changed")
				}
				_, e = b.MagnitudeUpperForCleanup(a.BorrowUpper())
				serviceMust(t, e)
				cursor, calls := scope.cursor, scope.callsPending
				serviceMust(t, m.SwapForCleanup(&a, &other))
				serviceMust(t, m.SwapForCleanup(&a, &other))
				if scope.cursor != cursor || scope.callsPending != calls || !scope.pending {
					t.Fatal("source cleanup reads/swaps acknowledged RAW ownership")
				}
				stale := receipt
				stale.invocation++
				serviceMust(t, m.CleanupDebugInvocation(&stale))
				copy := *m
				copy.self = &copy
				serviceMust(t, copy.CleanupDebugInvocation(&receipt))
				if !m.DebugInvocationPending(&receipt) || !scope.pending {
					t.Fatal("stale receipt or copied owner acquired actual debug child")
				}
				older := receipt
				if e = m.AddWithDebugReceipt(nil, nil, nil, &receipt); !errors.Is(e, ErrBinaryDebugSnapshotPending) || receipt != older {
					t.Fatal("pending operation read invalid inputs or replaced receipt", e)
				}
				// The public numerical owner may acknowledge its child directly.
				// A later actual conversion must not be acquired by the old parent.
				serviceMust(t, b.CleanupToOwnedRational())
				serviceMust(t, b.CleanupDebugInteger())
				serviceMust(t, b.CleanupDebugSnapshots())
				foreignPending := false
				foreignInput := NewBinaryIntPower(1, 63)
				for allowance := int64(0); allowance < 256 && !foreignPending; allowance++ {
					budget.FailAfter(allowance)
					value, call := b.ToOwnedRational(diagnostics, &foreignInput)
					budget.FailAfter(-1)
					if call == nil {
						serviceMust(t, diagnostics.Del(&value))
					}
					if scope.pending {
						foreignPending = true
						invocation, cursor, calls := b.toOwnedInvocation, scope.cursor, scope.callsPending
						serviceMust(t, m.CleanupDebugInvocation(&older))
						if !scope.pending || b.toOwnedInvocation != invocation || scope.cursor != cursor || scope.callsPending != calls {
							t.Fatal("old interval receipt consumed later actual conversion")
						}
					}
					serviceMust(t, b.CleanupToOwnedRational())
					serviceMust(t, diagnostics.Manager.Z.CleanupPendingAllocation())
				}
				if !foreignPending {
					t.Fatal("later actual conversion never retained a child")
				}
			}
			serviceMust(t, m.CleanupDebugInvocation(&receipt))
			serviceMust(t, qm.Z.CleanupPendingAllocation())
			if m.DebugInvocationPending(&receipt) || scope.pending || m.DebugInvocationSourceCause(&receipt) != nil {
				t.Fatal("owning debug acknowledgement retained RAW or changed source history")
			}
			serviceMust(t, m.Del(&out))
			serviceMust(t, m.Del(&other))
			serviceMust(t, m.Del(&a))
		}) {
			return
		}
	}
	if !found {
		t.Fatal("no actual RAW-only debug conversion reservation reached")
	}
}
