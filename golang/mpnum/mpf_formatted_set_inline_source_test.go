// Source: src/util/mpf.cpp:335-374; no lexical NUM/RAW declarations.
// Licensed under the MIT license. Source author: Christoph Wintersteiger.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestArbitraryFloatFormattedSetInlineZeroAfterRetainedUnpackLeaf(t *testing.T) {
	found := false
	for ordinal := int64(0); ordinal < 20; ordinal++ {
		b := arena.NewBudget(1 << 27)
		m, e := NewArbitraryFloatManager(IntegerConfig{Cells: arena.Config{Budget: b, PageSize: 1}, Words: arena.VectorConfig{Descriptors: arena.Config{Budget: b}}, Profile: memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}})
		serviceMust(t, e)
		var x, out, z, after ArbitraryFloat
		one := NewInt(1)
		serviceMust(t, m.SetComponents(&x, 8, 4000, false, 0, &one))
		serviceMust(t, m.MkZero(8, 24, true, &z))
		_, e = m.powerTwoWrapper(3999, false)
		serviceMust(t, e)
		b.FailAfter(ordinal)
		body := m.SetFormat(&out, 8, 24, ArbitraryFloatNearestEven, &x)
		old := m.formattedSet.history
		selected := body != nil && old != nil && old.numeric.cell.object != (memory.ObjectRef{})
		if selected {
			integerPhysicalAudit(t, m.integer, &x.significand, &out.significand, &z.significand)
			if b.Used() != m.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
				t.Fatal("before-lift exact cold graph")
			}
			oldContext := m.formattedSet.allocationContext()
			copiedScope := m.formattedSet
			if (integerAllocationInvocationContext{mpfFormattedSet: &copiedScope, mpfFormattedSetInvocation: copiedScope.invocation}).formattedSetFrame(m.integer, false) != nil || m.formattedRoundSourceCause(&copiedScope) != memory.ErrRuntimeOwner || m.formattedRoundCleanup(&copiedScope, true) != memory.ErrRuntimeOwner {
				t.Fatal("copied actual scope authority")
			}
			copiedHistory := *old
			forged := old.issued
			forged.frame = &copiedHistory
			beforeCell := old.numeric.cell
			beforeUsed := b.Used()
			if e = m.CleanupFormattedSetWithReceipt(&forged, true); e != memory.ErrRuntimeOwner || old.numeric.cell != beforeCell || b.Used() != beforeUsed {
				t.Fatal("copied history authority", e)
			}
			bodyBefore, sourceBefore, ownBefore := m.FormattedSetInvocationCauses(&old.issued)
			b.FailAfter(0)
			if e = m.SetFormat(&after, 8, 24, ArbitraryFloatNearestEven, &z); e != nil {
				t.Fatal("literal inline zero erased by retained unrelated allocation", ordinal, e)
			}
			b.FailAfter(-1)
			if oldContext.formattedSetFrame(m.integer, false) != nil || oldContext.roots(m.integer) != nil || oldContext.prepareFormattedSetHistory(m.integer) != memory.ErrRuntimeOwner {
				t.Fatal("saved mutable binding retargeted to next invocation")
			}
			if after.SBits() != 24 || after.EBits() != 8 || !after.Sign() {
				t.Fatal("literal zero fields")
			}
			bodyAfter, sourceAfter, ownAfter := m.FormattedSetInvocationCauses(&old.issued)
			if bodyAfter != bodyBefore || sourceAfter != sourceBefore || ownAfter != ownBefore || old.numeric.cell != old.originalCell || old.owned {
				t.Fatal("inline next call erased old exact record")
			}
			// A later real allocating Copy owns a distinct child. Old ACK may
			// consume only its own detached unpack cell, even while that child
			// has an issued descriptor/backing failure.
			var foreignSig Int
			limbs := make([]uint32, 80)
			limbs[0] = 1
			limbs[79] = 1
			serviceMust(t, m.integer.SetLimbs(&foreignSig, limbs))
			var foreign, failed ArbitraryFloat
			serviceMust(t, m.SetComponents(&foreign, 8, 260, false, 0, &foreignSig))
			foreignReached := false
			for prefix := int64(0); prefix < 12; prefix++ {
				var child ArbitraryFloatValueCopyReceipt
				b.FailAfter(prefix)
				foreignBody := m.CopyWithReceipt(&failed, &foreign, &child)
				integerPhysicalAudit(t, m.integer, &x.significand, &out.significand, &z.significand, &after.significand, &foreignSig, &foreign.significand, &failed.significand)
				if b.Used() != m.memory.Stats().TotalBytes+m.OwnedStorageBytes() {
					t.Fatal("foreign before-lift cold graph")
				}
				b.FailAfter(-1)
				var q *arbitraryFloatValueCopyCall
				if child != (ArbitraryFloatValueCopyReceipt{}) {
					q, e = m.valueCopyCall(&child)
					serviceMust(t, e)
				}
				if foreignBody != nil && q != nil && polynomialNegNumericPending(&q.childFrame.roots) {
					beforeRoots := q.childFrame.roots
					beforeChild := q.issued
					current := m.formattedSet
					serviceMust(t, m.CleanupFormattedSetWithReceipt(&old.issued, false))
					serviceMust(t, m.CleanupValue(&out))
					serviceMust(t, m.CleanupFormattedSetWithReceipt(&old.issued, true))
					if q.childFrame.roots != beforeRoots || q.issued != beforeChild || q.retired || m.formattedSet != current {
						t.Fatal("old parent adopted foreign Copy")
					}
					foreignReached = true
				}
				serviceMust(t, m.CleanupValueCopy(&child, false))
				serviceMust(t, m.CleanupValue(&failed))
				serviceMust(t, m.CleanupValueCopy(&child, true))
				if foreignReached {
					break
				}
			}
			if !foreignReached {
				t.Fatal("real later foreign allocating Copy prefix not reached")
			}
			serviceMust(t, m.Del(&foreign))
			serviceMust(t, m.integer.Del(&foreignSig))
			found = true
		} else {
			b.FailAfter(-1)
		}
		if old != nil {
			serviceMust(t, m.CleanupFormattedSetWithReceipt(&old.issued, false))
			serviceMust(t, m.CleanupValue(&out))
			serviceMust(t, m.CleanupFormattedSetWithReceipt(&old.issued, true))
		}
		serviceMust(t, m.CleanupFormattedSet())
		serviceMust(t, m.CleanupValue(&out))
		serviceMust(t, m.Del(&x))
		serviceMust(t, m.Del(&z))
		serviceMust(t, m.Del(&after))
		if e = m.SourceFinalize(); e != nil && !m.integer.sourceGuard {
			t.Fatal(e)
		}
		serviceMust(t, m.Close())
		if b.Used() != 0 {
			t.Fatal(b.Used())
		}
		if found {
			break
		}
	}
	if !found {
		t.Fatal("actual unpack descriptor refusal prefix not reached")
	}
}
