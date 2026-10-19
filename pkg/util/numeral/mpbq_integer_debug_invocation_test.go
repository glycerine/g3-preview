//go:build g3debug

package numeral

import (
	"errors"
	"strconv"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestBinaryIntegerDebugReceiptRejectsLaterConversion(t *testing.T) {
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
			t.Cleanup(func() {
				serviceMust(t, b.CloseChecked())
				serviceMust(t, diagnostics.Close())
			})
			a := NewBinaryIntPower(1, 63)
			var out BinaryRational
			n := NewInt(1)
			var receipt BinaryIntegerDebugInvocationReceipt
			budget.FailAfter(prefix)
			body := b.AddIntegerWithDebugReceipt(&a, &n, &out, &receipt)
			budget.FailAfter(-1)
			scope := &b.toOwnedRationalScope
			if b.IntegerDebugInvocationPending(&receipt) && scope.pending && scope.cursor == 5 && scope.sourceCause == nil {
				found = true
				if body == nil || b.IntegerDebugInvocationSourceCause(&receipt) != nil {
					t.Fatal("RAW-only debug failure acquired a source destructor stop", body)
				}
				stale := receipt
				stale.invocation++
				serviceMust(t, b.CleanupIntegerDebugInvocation(&stale))
				copy := *b
				copy.self = &copy
				serviceMust(t, copy.CleanupIntegerDebugInvocation(&receipt))
				if !b.IntegerDebugInvocationPending(&receipt) || !scope.pending {
					t.Fatal("stale receipt or copied owner acquired actual debug child")
				}
				older := receipt
				if e = b.AddIntegerWithDebugReceipt(nil, nil, nil, &receipt); !errors.Is(e, ErrBinaryDebugSnapshotPending) || receipt != older {
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
						serviceMust(t, b.CleanupIntegerDebugInvocation(&older))
						if !scope.pending || b.toOwnedInvocation != invocation || scope.cursor != cursor || scope.callsPending != calls {
							t.Fatal("old scalar receipt consumed later actual conversion")
						}
					}
					serviceMust(t, b.CleanupToOwnedRational())
					serviceMust(t, diagnostics.Manager.Z.CleanupPendingAllocation())
				}
				if !foreignPending {
					t.Fatal("later actual conversion never retained a child")
				}
			}
			serviceMust(t, b.CleanupIntegerDebugInvocation(&receipt))
			serviceMust(t, qm.Z.CleanupPendingAllocation())
			if b.IntegerDebugInvocationPending(&receipt) || scope.pending || b.IntegerDebugInvocationSourceCause(&receipt) != nil {
				t.Fatal("owning debug acknowledgement retained RAW or changed source history")
			}
			serviceMust(t, b.Del(&out))
			serviceMust(t, b.Del(&a))
		}) {
			return
		}
	}
	if !found {
		t.Fatal("no actual RAW-only debug conversion reservation reached")
	}
}
