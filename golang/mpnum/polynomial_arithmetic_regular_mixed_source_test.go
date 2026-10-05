// Source: src/math/polynomial/polynomial.cpp:2067-2092,1885-1898.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialRegularMatchedRowsPreserveLaterForeignPositions(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	_, e := c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	p, e := c.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	q, e := c.MkVariable(1, 1)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(q))
	pt, e := c.monomialCore(p, 0)
	serviceMust(t, e)
	qt, e := c.monomialCore(q, 0)
	serviceMust(t, e)
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &r))
	serviceMust(t, c.som.addPolynomial(q))
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
	serviceMust(t, c.cleanupArithmeticInvocation(&r, false))
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	n, e := c.som.ms.size(c)
	serviceMust(t, e)
	a, e := c.som.as.size(c)
	serviceMust(t, e)
	pp, e := c.som.positions.Get(pt)
	serviceMust(t, e)
	qp, e := c.som.positions.Get(qt)
	serviceMust(t, e)
	if n != 1 || a != 1 || pp != ^uint32(0) || qp != 0 {
		t.Fatal("foreign row/position lost", n, a, pp, qp)
	}
	term, e := c.som.ms.monomial(c, 0)
	serviceMust(t, e)
	value, e := c.som.as.coefficient(c, 0)
	serviceMust(t, e)
	if term != qt || value != NewInt(1) {
		t.Fatal("foreign payload", term, value)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, q}, nil, 0)
	serviceMust(t, c.som.reset())
	serviceMust(t, c.DecRef(p))
	serviceMust(t, c.DecRef(q))
}

func TestPolynomialRegularLaterNilAccumulatorWriteTransfersActualRowToMember(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	seven, five := NewInt(7), NewInt(5)
	p, e := c.MkConstantTakeCopy(&seven)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	var old polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&old))
	serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &old))
	serviceMust(t, c.som.add(&five, c.monomials.unit, false))
	beforeRC, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	before, e := c.som.as.coefficient(c, 0)
	serviceMust(t, e)
	if before != NewInt(12) {
		t.Fatal("literal later accumulator value", before)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
	serviceMust(t, c.cleanupArithmeticInvocation(&old, false))
	serviceMust(t, c.cleanupArithmeticInvocation(&old, true))
	n, e := c.som.as.size(c)
	serviceMust(t, e)
	if n != 1 {
		t.Fatal("old abort retired later actual source member row", n)
	}
	after, e := c.som.as.coefficient(c, 0)
	serviceMust(t, e)
	afterRC, e := c.monomialRefCount(c.monomials.unit)
	serviceMust(t, e)
	if after != before || afterRC != beforeRC {
		t.Fatal("foreign actual row value/INC changed", after, afterRC, beforeRC)
	}
	polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
	serviceMust(t, c.som.reset())
	serviceMust(t, c.DecRef(p))
}

func TestPolynomialRegularFailedNilSetterKeepsForeignPendingAndOldTags(t *testing.T) {
	saw := false
	replacementSaw := false
	for ordinal := int64(0); ordinal < 32; ordinal++ {
		full := polynomialExactDivDirectPageOneFixture(t)
		c := full.core
		var huge Int
		words := make([]uint32, 80)
		words[0], words[79] = 1, 1
		serviceMust(t, c.integer.SetLimbs(&huge, words))
		seven := NewInt(7)
		p, e := c.MkConstantTakeCopy(&seven)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		var old polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&old))
		serviceMust(t, c.som.addPolynomialWithArithmeticReceipt(p, &old))
		tags := [2]uint64{c.som.coefficientOwners[0], c.som.monomialOwners[0]}
		c.integer.budget.FailAfter(ordinal)
		body := c.som.add(&huge, c.monomials.unit, false)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*Int{&huge}, 0)
		if body != nil {
			if tags != ([2]uint64{c.som.coefficientOwners[0], c.som.monomialOwners[0]}) {
				t.Fatal("failed setter ceded old actual tokens", ordinal)
			}
			foreign := c.integer.pendingCell
			raw, e := c.integer.allocator.PendingRoots()
			serviceMust(t, e)
			if foreign.raw != (memory.Block{}) || raw != ([3]memory.Block{}) {
				saw = true
			}
			c.integer.budget.FailAfter(-1)
			serviceMust(t, c.cleanupArithmeticInvocation(&old, false))
			serviceMust(t, c.cleanupArithmeticInvocation(&old, true))
			after, e := c.integer.allocator.PendingRoots()
			serviceMust(t, e)
			if c.integer.pendingCell != foreign || after != raw {
				t.Fatal("old abort consumed foreign actual allocation", ordinal)
			}
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, []*Int{&huge}, 0)
			// The actual nil numeric call belongs to the Integer owner, not old SOM.
			serviceMust(t, c.integer.CleanupPendingAllocation())
			var fillers [8]Int
			fillerCount := 0
			public := []*Int{&huge}
			for c.integer.cells.Stats().Slots > c.integer.cells.Stats().Live+c.integer.cells.Stats().Retired {
				if fillerCount == len(fillers) {
					t.Fatal("bounded actual descriptor fill", ordinal)
				}
				serviceMust(t, c.integer.Copy(&fillers[fillerCount], &huge))
				public = append(public, &fillers[fillerCount])
				fillerCount++
			}
			var later Int
			public = append(public, &later)
			c.integer.budget.FailAfter(1)
			replacementBody := c.integer.Copy(&later, &huge)
			replacement := c.integer.pendingCell
			replacementRaw, x := c.integer.allocator.PendingRoots()
			serviceMust(t, x)
			polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, public, 0)
			if replacementBody != nil && (replacement.raw != (memory.Block{}) || replacementRaw != ([3]memory.Block{})) {
				replacementSaw = true
				serviceMust(t, c.cleanupArithmeticInvocation(&old, false))
				serviceMust(t, c.cleanupArithmeticInvocation(&old, true))
				now, x := c.integer.allocator.PendingRoots()
				serviceMust(t, x)
				if c.integer.pendingCell != replacement || now != replacementRaw {
					t.Fatal("stale old cohort consumed actual later replacement", ordinal)
				}
			}
			c.integer.budget.FailAfter(-1)
			serviceMust(t, c.integer.CleanupPendingAllocation())
			serviceMust(t, c.integer.Del(&later))
			for i := 0; i < fillerCount; i++ {
				serviceMust(t, c.integer.Del(&fillers[i]))
			}
		} else {
			c.integer.budget.FailAfter(-1)
			serviceMust(t, c.cleanupArithmeticInvocation(&old, true))
			serviceMust(t, c.som.reset())
		}
		serviceMust(t, c.DecRef(p))
		serviceMust(t, c.integer.Del(&huge))
		if body == nil {
			if !saw || !replacementSaw {
				t.Fatal("no real foreign pending leaf reached")
			}
			return
		}
	}
	t.Fatal("actual foreign setter horizon did not reach success")
}
