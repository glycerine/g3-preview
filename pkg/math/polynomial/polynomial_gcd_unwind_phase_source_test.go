// Source: src/math/polynomial/polynomial.cpp:5751-5810,4352-4506; src/util/obj_ref.h.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialGCDInnerModeBlockWaitsForActualYoungerSourceStop(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	parent, e := full.beginGCDFunction()
	serviceMust(t, e)
	parent.constructNumeral(0)
	serviceMust(t, z.Copy(&parent.nums[0], &z.Two64))
	older := parent.nums[0]
	prime := unumeral.NewInt(7)
	serviceMust(t, parent.zp.construct(full, &prime))
	parent.stack[parent.depth] = gcdZPId
	parent.depth++
	child, e := full.beginGCDFunction()
	serviceMust(t, e)
	child.constructNumeral(0)
	var large unumeral.Int
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
	polynomialImpPhysicalAuditNamed(t, full, nil, []*unumeral.Int{&large}, 0)
	z.Budget().FailAfter(-1)
	serviceMust(t, full.CleanupGCD())
	polynomialImpPhysicalAuditNamed(t, full, nil, []*unumeral.Int{&large}, 0)
}

// Guarded helper domain: source restore sees an actual Copy-produced dangling
// saved numeral. The younger monomial slice releases its manager references
// before that older class body is reached by explicit owning unwind.
func TestPolynomialGCDCompletedMonomialReferencesPrecedeOlderStop(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	z := c.integer
	prime := unumeral.NewInt(7)
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
	var large unumeral.Int
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
	polynomialImpPhysicalAuditNamed(t, full, nil, []*unumeral.Int{&large}, 0)
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
	polynomialImpPhysicalAuditNamed(t, full, nil, []*unumeral.Int{&large}, 0)
}
