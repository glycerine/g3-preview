// Source: src/math/polynomial/polynomial.cpp:5751-5810,4352-4506; src/util/obj_ref.h.
// Licensed under the MIT license. Source author: Leonardo de Moura (leonardo).
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialPSCCompletedPhaseLeavesFailedConstructorForOuterUnwind(t *testing.T) {
	for ordinal := int64(0); ordinal < 256; ordinal++ {
		m := polynomialCellFixture(t)
		p := polynomialQueryTerms(t, m, []int32{1, 1, 1, 2}, [][]MonomialPower{{{0, 1}, {1, 2}}, {{0, 1}, {1, 1}}, {{1, 1}}, nil})
		q := polynomialQueryTerms(t, m, []int32{1, 3}, [][]MonomialPower{{{0, 1}, {1, 1}}, nil})
		older, e := NewPolynomialRef(m)
		serviceMust(t, e)
		serviceMust(t, older.root.assign(m, p))
		t.Cleanup(func() { serviceMust(t, older.Close()) })
		S, e := NewPolynomialRefVector(m)
		serviceMust(t, e)
		t.Cleanup(func() { serviceMust(t, S.Close()) })
		m.integer.budget.FailAfter(ordinal)
		cause := m.PSCChain(p, q, 1, S)
		polynomialPSCPhysicalVectors(t, m, []*PolynomialRef{older}, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		m.integer.budget.FailAfter(-1)
		g := m.pscReceipts.generation
		if g == 0 || m.pending.pscOwner != g || m.pending.raw == (memory.ObjectRef{}) || m.singletonReference || m.generatedActive || m.monomials.creation.stage != 0 {
			serviceMust(t, m.CleanupPSC())
			serviceMust(t, S.Close())
			continue
		}
		if !numericPhysicalReservationRefusal(cause) {
			t.Fatal("not real owned placement failure", ordinal, cause)
		}
		pending := m.pending
		_, e = m.MkVariable(0, 17)
		if !errors.Is(e, ErrPolynomialPending) || !m.singletonReference || m.pending != pending {
			t.Fatal("foreign singleton not actually reached", ordinal, e, m.singletonReference)
		}
		beforeOlder := older.root.value
		serviceMust(t, m.cleanupPSCCompleted(g+1))
		if m.pending != pending {
			t.Fatal("foreign generation cleanup changed source constructor")
		}
		serviceMust(t, m.cleanupPSCCompleted(g))
		if m.pending != pending || older.root.value != beforeOlder {
			t.Fatal("completed younger cleanup aborted constructor before older local", m.pending, pending, older)
		}
		polynomialPSCPhysicalVectors(t, m, []*PolynomialRef{older}, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		serviceMust(t, older.root.closeValue(m, false))
		foreign := m.singletonMonomial
		rc, e := m.monomialRefCount(foreign)
		serviceMust(t, e)
		polynomialPSCPhysicalVectors(t, m, []*PolynomialRef{older}, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		serviceMust(t, m.cleanupPSCAborts(g))
		if !m.singletonReference || m.singletonMonomial != foreign {
			t.Fatal("matching old PSC placement swept later caller singleton", ordinal, foreign, rc, m.singletonReference, m.singletonMonomial)
		}
		after, e := m.monomialRefCount(foreign)
		serviceMust(t, e)
		if after != rc {
			t.Fatal("caller INC consumed", rc, after)
		}
		serviceMust(t, m.CleanupConstruction())
		serviceMust(t, S.Close())
		return
	}
	t.Fatal("no actual PSC placement frontier reached")
}

// Guarded helper domain: the child has an actual Copy-produced dangling
// numeral. This does not claim ordinary mathematical GCD creates this shape.
func TestPolynomialGCDInnerModeBlockWaitsForActualYoungerSourceStop(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	parent, e := full.beginGCDFunction()
	serviceMust(t, e)
	parent.constructNumeral(0)
	serviceMust(t, z.Copy(&parent.nums[0], &z.two64))
	older := parent.nums[0]
	prime := NewInt(7)
	serviceMust(t, parent.zp.construct(full, &prime))
	parent.stack[parent.depth] = gcdZPId
	parent.depth++
	child, e := full.beginGCDFunction()
	serviceMust(t, e)
	child.constructNumeral(0)
	var large Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	serviceMust(t, z.Copy(&child.nums[0], &z.two64))
	t.Cleanup(func() { z.budget.FailAfter(-1); serviceMust(t, full.CleanupGCD()); serviceMust(t, z.Del(&large)) })
	z.budget.FailAfter(0)
	body := z.Copy(&child.nums[0], &large)
	if !numericPhysicalReservationRefusal(body) || child.nums[0].cell != z.failedRef {
		t.Fatal("actual stopped child", body)
	}
	source := full.finishGCDFunctionSource(child)
	if !errors.Is(source, arena.ErrInvalidSlot) {
		t.Fatal("actual child source destructor", source)
	}
	full.gcdDepth--
	c.pscLeave()
	first := full.finishGCDLocalsTo(parent, 0)
	full.gcdDepth--
	c.pscLeave()
	field, e := c.modular.Modular()
	serviceMust(t, e)
	if first != source || !field || parent.zp.stage != 0 || parent.depth != 2 || parent.nums[0] != older {
		t.Fatal("older mode restoration crossed child stop", first, source, field, parent.depth)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*Int{&large}, 0)
	z.budget.FailAfter(-1)
	serviceMust(t, full.CleanupGCD())
	polynomialImpPhysicalAuditNamed(t, full, nil, []*Int{&large}, 0)
}

// Guarded helper domain: source restore sees an actual Copy-produced dangling
// saved numeral. The younger completed ref_buffer still frees its published
// backing before that older class body is reached by explicit owning unwind.
func TestPolynomialGCDCompletedMonomialBufferFreePrecedesOlderStop(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	prime := NewInt(7)
	serviceMust(t, c.modular.SetZP(&prime))
	s, e := full.beginGCDFunction()
	serviceMust(t, e)
	serviceMust(t, s.zp.construct(full, &prime))
	s.stack[s.depth] = gcdZPId
	s.depth++
	s.monomialBuffer.construct()
	s.stack[s.depth] = gcdMonomialBufferID
	s.depth++
	unit, e := c.monomials.Unit()
	serviceMust(t, e)
	for i := 0; i < 17; i++ {
		serviceMust(t, s.monomialBuffer.push(c, unit))
	}
	if s.monomialBuffer.raw == (memory.Block{}) {
		t.Fatal("actual overflow backing missing")
	}
	var large Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	serviceMust(t, z.Copy(&s.zp.p, &z.two64))
	full.gcdDepth--
	c.pscLeave()
	t.Cleanup(func() { z.budget.FailAfter(-1); serviceMust(t, full.CleanupGCD()); serviceMust(t, z.Del(&large)) })
	z.budget.FailAfter(0)
	cause := z.Copy(&s.zp.p, &large)
	if !numericPhysicalReservationRefusal(cause) || s.zp.p.cell != z.failedRef {
		t.Fatal("actual dangling Copy fixture", cause)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*Int{&large}, 0)
	z.budget.FailAfter(-1)
	stop := full.CleanupGCD()
	if !errors.Is(stop, arena.ErrInvalidSlot) {
		t.Fatal("older literal restore guard", stop)
	}
	if s.depth != 1 || s.monomialBuffer.raw != (memory.Block{}) {
		t.Fatal("completed younger source buffer not fully destroyed before older stop", s.depth, s.monomialBuffer.raw)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*Int{&large}, 0)
}
