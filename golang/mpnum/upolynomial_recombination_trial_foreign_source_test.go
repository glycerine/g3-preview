// Source: src/math/polynomial/upolynomial_factorization.cpp:1213-1240;
// src/util/mpzzp.h:46-56; src/util/mpz.cpp:213-242.
// Licensed under the MIT license. Source author: Leonardo de Moura.
// Source author: Dejan (t-dejanj), 2011-11-15.
package mpnum

import (
	"fmt"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestUPolynomialRecombinationTrialExactOldLeafPreservesForeignReplacement(t *testing.T) {
	for ordinal := int64(0); ordinal < 512; ordinal++ {
		found := false
		t.Run(fmt.Sprint(ordinal), func(t *testing.T) {
			q, s := mignotteDirectFixture(t)
			m := s.manager
			var prime, foreign, large Int
			serviceMust(t, q.z.SetLimbs(&prime, []uint32{0xffffffff, 0xffffffff, 0xffffffff, 0x7fffffff}))
			words := make([]uint32, 25000)
			words[24999] = 1
			serviceMust(t, q.z.SetLimbs(&large, words))
			defer func() {
				serviceMust(t, q.z.cleanupRawNumeral(&foreign))
				serviceMust(t, q.z.Del(&prime))
				serviceMust(t, q.z.Del(&large))
			}()
			_, e := s.newSquareFreeScope()
			serviceMust(t, e)
			fs, e := m.NewFactors()
			serviceMust(t, e)
			defer func() { serviceMust(t, fs.Close()) }()
			h, e := s.beginRecombinationTrial(fs)
			serviceMust(t, e)
			r := h.receipt
			q.budget.FailAfter(ordinal / 16)
			body := m.modular.copyWithSourceContext(&h.tmp, &prime, m.modular.trialAllocationContext(h))
			h.executing = false
			primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &foreign, &large)
			old := h.numeric.cell
			if old.object != (memory.ObjectRef{}) {
				if q.z.pendingCell != old || !numericPhysicalReservationRefusal(body) {
					t.Fatal("actual direct pending leaf absent", ordinal, body, old, q.z.pendingCell)
				}
				_ = h.finish(&r, body)
				serviceMust(t, h.own(&r, false))
				if h.numeric.cell != old || q.z.pendingCell != old {
					t.Fatal("false phase retired unjoined raw leaf")
				}
				q.budget.FailAfter(-1)
				serviceMust(t, q.z.CleanupPendingAllocation())
				// Actual independent nil-context leaf starts after the old token is consumed.
				// Search its real reservation prefixes rather than minting a pending token.
				foreignFound := false
				for next := ordinal % 16; next < ordinal%16+1; next++ {
					q.budget.FailAfter(next)
					failure := q.z.Copy(&foreign, &large)
					primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &foreign, &large)
					laterRaw, x := q.z.allocator.PendingRoots()
					serviceMust(t, x)
					if q.z.pendingCell.object != (memory.ObjectRef{}) || laterRaw != ([3]memory.Block{}) {
						replacement := q.z.pendingCell
						if (replacement.object != (memory.ObjectRef{}) && replacement == old) || !numericPhysicalReservationRefusal(failure) {
							t.Fatal("not a genuine later generation", next, failure, replacement, old)
						}
						q.budget.FailAfter(-1)
						serviceMust(t, h.own(&r, true))
						afterRaw, x := q.z.allocator.PendingRoots()
						serviceMust(t, x)
						if q.z.pendingCell != replacement || afterRaw != laterRaw {
							t.Fatal("old exact abort consumed foreign replacement", replacement, q.z.pendingCell)
						}
						primeTrialPhysicalAuditOwners(t, q, s, []*UPolynomialFactors{fs}, &prime, &foreign, &large)
						serviceMust(t, h.own(&r, true))
						afterRaw, x = q.z.allocator.PendingRoots()
						serviceMust(t, x)
						if q.z.pendingCell != replacement || afterRaw != laterRaw {
							t.Fatal("stale old abort consumed later pending")
						}
						serviceMust(t, q.z.CleanupPendingAllocation())
						serviceMust(t, q.z.cleanupRawNumeral(&foreign))
						foreignFound = true
						found = true
						t.Log("actual old/foreign pending reservation prefixes", ordinal/16, next)
						break
					}
					q.budget.FailAfter(-1)
					serviceMust(t, q.z.CleanupPendingAllocation())
					serviceMust(t, q.z.cleanupRawNumeral(&foreign))
				}
				if !foreignFound {
					serviceMust(t, h.own(&r, true))
				}
			} else {
				q.budget.FailAfter(-1)
				_ = h.finish(&r, body)
				serviceMust(t, h.own(&r, false))
				serviceMust(t, h.own(&r, true))
			}
		})
		if found {
			return
		}
	}
	t.Fatal("no actual direct old descriptor prefix")
}
