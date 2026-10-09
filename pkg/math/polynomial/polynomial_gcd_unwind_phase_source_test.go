// Source: src/math/polynomial/polynomial.cpp:5751-5810,4352-4506; src/util/obj_ref.h.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
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
		m.integer.Budget().FailAfter(ordinal)
		cause := m.PSCChain(p, q, 1, S)
		polynomialPSCPhysicalVectors(t, m, []*PolynomialRef{older}, []*PolynomialRefVector{S}, []Polynomial{p, q}, nil)
		m.integer.Budget().FailAfter(-1)
		g := m.pscReceipts.generation
		if g == 0 || m.pending.pscOwner != g || m.pending.raw == (util.ObjectRef{}) || m.singletonReference || m.generatedActive || m.monomials.creation.stage != 0 {
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
	serviceMust(t, z.Copy(&parent.nums[0], &z.Two64))
	older := parent.nums[0]
	prime := util.NewInt(7)
	serviceMust(t, parent.zp.construct(full, &prime))
	parent.stack[parent.depth] = gcdZPId
	parent.depth++
	child, e := full.beginGCDFunction()
	serviceMust(t, e)
	child.constructNumeral(0)
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	serviceMust(t, z.Copy(&child.nums[0], &z.Two64))
	t.Cleanup(func() { z.Budget().FailAfter(-1); serviceMust(t, full.CleanupGCD()); serviceMust(t, z.Del(&large)) })
	z.Budget().FailAfter(0)
	body := z.Copy(&child.nums[0], &large)
	if !numericPhysicalReservationRefusal(body) || (child.nums[0]).Cell != z.FailedRef {
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
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	z.Budget().FailAfter(-1)
	serviceMust(t, full.CleanupGCD())
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
}

// Guarded helper domain: source restore sees an actual Copy-produced dangling
// saved numeral. The younger monomial slice releases its manager references
// before that older class body is reached by explicit owning unwind.
func TestPolynomialGCDCompletedMonomialReferencesPrecedeOlderStop(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	prime := util.NewInt(7)
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
	before, e := c.monomialRefCount(unit)
	serviceMust(t, e)
	for i := 0; i < 17; i++ {
		serviceMust(t, s.monomialBuffer.push(c, unit))
	}
	var large util.Int
	limbs := make([]uint32, 8192)
	limbs[0], limbs[len(limbs)-1] = 17, 1
	serviceMust(t, z.SetLimbs(&large, limbs))
	serviceMust(t, z.Copy(&s.zp.p, &z.Two64))
	full.gcdDepth--
	c.pscLeave()
	t.Cleanup(func() { z.Budget().FailAfter(-1); serviceMust(t, full.CleanupGCD()); serviceMust(t, z.Del(&large)) })
	z.Budget().FailAfter(0)
	cause := z.Copy(&s.zp.p, &large)
	if !numericPhysicalReservationRefusal(cause) || (s.zp.p).Cell != z.FailedRef {
		t.Fatal("actual dangling Copy fixture", cause)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
	z.Budget().FailAfter(-1)
	stop := full.CleanupGCD()
	if !errors.Is(stop, arena.ErrInvalidSlot) {
		t.Fatal("older literal restore guard", stop)
	}
	after, e := c.monomialRefCount(unit)
	serviceMust(t, e)
	if s.depth != 1 || s.monomialBuffer.values != nil || after != before {
		t.Fatal("younger monomial references retained across older stop", s.depth, after, before)
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, []*util.Int{&large}, 0)
}
