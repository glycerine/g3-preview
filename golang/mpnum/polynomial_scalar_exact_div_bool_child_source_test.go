// Source: src/math/polynomial/polynomial.cpp:5094-5114;
// src/util/mpz.cpp:2054-2069; src/util/scoped_numeral.h:31-33.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"errors"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

func TestPolynomialScalarExactDivCompiledAssertionCapturesActualBoolNUM(t *testing.T) {
	full := polynomialExactDivDirectPageOneFixture(t)
	c := full.core
	before := c.integer.dividesNumeral.invocation
	one := NewInt(1)
	result, e := c.ExactDivScalar(c.unit, &one)
	serviceMust(t, e)
	serviceMust(t, c.IncRef(result))
	defer func() { serviceMust(t, c.DecRef(result)) }()
	child := &c.integer.dividesNumeral
	if !modularSourceDebug {
		if child.invocation != before {
			t.Fatal("erased scalar assertion constructed bool NUM")
		}
		return
	}
	parent := child.parentContext
	if parent.scalarExact != &c.exactDivision.scalarReceipt || parent.scalarExactInvocation != c.exactDivision.invocation {
		t.Fatal("actual scalar assertion NUM lost its exact captured parent", parent.scalarExact, parent.scalarExactInvocation, c.exactDivision.invocation)
	}
	if child.invocation == before || !child.constructed || !child.ended || child.active {
		t.Fatal("literal successful bool NUM lexical destructor frontier")
	}
}

// The compiled source assertion reaches Rem's actual BIG copy before rejecting
// the nondivisible coefficient. Native source erases this assertion entirely.
func TestPolynomialScalarExactDivCompiledAssertionBIGRefusalPrefixes(t *testing.T) {
	if !modularSourceDebug {
		return
	}
	saw := false
	for prefix := int64(0); prefix < 100; prefix++ {
		full := polynomialExactDivDirectPageOneFixture(t)
		c, z := full.core, full.core.integer
		var a, d Int
		words := make([]uint32, 80)
		words[0], words[79] = 1, 1
		serviceMust(t, z.SetLimbs(&a, words))
		words[79] = 2
		serviceMust(t, z.SetLimbs(&d, words))
		p, e := c.MkConstantTakeCopy(&a)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		z.budget.FailAfter(prefix)
		result, body := c.ExactDivScalar(p, &d)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p, result}, []*Int{&a, &d}, 0)
		if body == nil {
			t.Fatal("nondivisible source assertion accepted", prefix)
		}
		s, child := &c.exactDivision, &z.dividesNumeral
		if child.parentContext.scalarExact != &s.scalarReceipt || child.parentContext.scalarExactInvocation != s.invocation || s.boolNumeral.frame != child || s.boolNumeral.invocation != child.invocation {
			t.Fatal("actual BIG bool parent lost", prefix)
		}
		pending := z.pendingCell
		raw, e := z.allocator.PendingRoots()
		serviceMust(t, e)
		acquired := pending.object != (memory.ObjectRef{}) || raw != ([3]memory.Block{})

		if acquired || child.pending(z) {
			saw = true
			if !s.active || !z.matchesDividesNumeralInvocation(&s.boolNumeral) || pending.object != (memory.ObjectRef{}) && child.numeric.cell != pending || raw != ([3]memory.Block{}) && child.numeric.allocator != raw {
				t.Fatal("actual bool refused acquisition lacks active scalar child", prefix)
			}
		}

		for i := range child.buffers {
			r := &child.buffers[i]
			if r.pending != (memory.Block{}) {
				if !r.bound(false) || r.buffer.pending != r.pending || r.boolDivides != s.boolNumeral {
					t.Fatal("actual failed remainder buffer ownership", prefix, i)
				}
				copied := *r
				if copied.bound(false) {
					t.Fatal("copied buffer gained cleanup authority", prefix, i)
				}
			}
		}
		if !errors.Is(body, ErrPolynomialAssertion) && !numericPhysicalReservationRefusal(body) {
			t.Fatal("actual refusal or source assertion required", prefix, body)
		}
		savedBody, savedChild := s.bodyCause, child.bodyCause
		z.budget.FailAfter(-1)
		serviceMust(t, c.CleanupExactDivision())
		if s.active || child.active || s.bodyCause != savedBody || child.bodyCause != savedChild {
			t.Fatal("matched cleanup or immutable call history", prefix)
		}
		serviceMust(t, c.DecRef(p))
		serviceMust(t, z.Del(&a))
		serviceMust(t, z.Del(&d))
		if errors.Is(body, ErrPolynomialAssertion) {
			if !saw {
				t.Fatal("actual bool BIG acquisition refusal not exercised")
			}
			t.Log("compiled assertion first complete false result", prefix)
			return
		}
	}
	t.Fatal("actual bool assertion prefix campaign did not complete")
}
