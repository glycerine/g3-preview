// Source: src/math/polynomial/polynomial.cpp:5154-5194;
// src/util/mpzzp.h:167-185; src/util/mpz.h:468-485.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package mpnum

import (
	"github.com/glycerine/g3/golang/arena"
	"github.com/glycerine/g3/golang/memory"
	"testing"
)

// These are explicit Go owner-metadata boundary fixtures. Source reservation
// and lexical-prefix campaigns belong to the actual Divides body integration.
func TestDividesNumericActualFrameAndImmutableContext(t *testing.T) {
	b := arena.NewBudget(1 << 26)
	q, e := NewRationalManager(RationalConfig{Integers: serviceConfig(b).Integers})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	s := &c.dividesLocal
	s.active = true
	s.constructed = 2
	s.executing = true
	s.invocation = 1
	r := polynomialDividesInvocationReceipt{core: c, frame: s, invocation: 1}
	s.constructed = 1
	if c.modular.dividesAllocationContext(&r).roots(c.integer) != nil {
		t.Fatal("monomial ref alone has no NUM authority")
	}
	s.constructed = 2
	context := c.modular.dividesAllocationContext(&r)
	if context.roots(c.integer) != &s.numeric {
		t.Fatal("actual exact frame rejected")
	}
	copied := *s
	foreign := r
	foreign.frame = &copied
	if c.integer.dividesAllocationRoots(&foreign) != nil {
		t.Fatal("copied frame acquired")
	}
	c.callbackDepth++
	if context.roots(c.integer) != nil {
		t.Fatal("callback acquired")
	}
	c.callbackDepth--

	foreignModular, e := NewIntegerModularManager(c.integer)
	serviceMust(t, e)
	if foreignModular.dividesAllocationContext(&r).roots(c.integer) != nil {
		t.Fatal("foreign modular receiver acquired")
	}
	serviceMust(t, foreignModular.Close())
	a, d, out := NewInt(6), NewInt(2), NewInt(0)

	var absent *IntegerModularManager
	if absent.divWithDividesReceipt(&a, &d, &out, &r) == nil || absent.negWithDividesReceipt(&out, &r) == nil {
		t.Fatal("absent receiver accepted")
	}
	serviceMust(t, c.modular.divWithDividesReceipt(&a, &d, &out, &r))
	serviceMust(t, c.modular.negWithDividesReceipt(&out, &r))
	if out.val != -3 {
		t.Fatal("same literal Div/Neg", out.val)
	}

	// A real negative Integer Div constructs and hands off its actual RAW child.
	a = NewInt(-7)
	serviceMust(t, c.modular.divWithDividesReceipt(&a, &d, &out, &r))
	if s.divisionRaw.manager != c.integer || s.divisionRaw.frame != &c.integer.divisionRaw || s.divisionRaw.invocation == 0 {
		t.Fatal("actual RAW child not acquired")
	}
	serviceMust(t, c.integer.cleanupRawInvocation(&s.divisionRaw, true))
	bound := c.integer.natural.divisionBufferReceipt(context, 0)
	if bound == nil || !bound.bound(true) {
		t.Fatal("actual Natural member binding absent")
	}
	copiedBuffer := *bound
	if copiedBuffer.bound(true) || copiedBuffer.bound(false) {
		t.Fatal("copied Natural receipt accepted")
	}
	// Embedded direct receipt may be rewritten only by a new actual invocation;
	// an earlier captured context must retain its old immutable generation.
	s.directReceipt = r
	old := c.modular.dividesAllocationContext(&s.directReceipt)
	s.invocation = 2
	s.directReceipt.invocation = 2
	if bound.bound(true) || bound.bound(false) {
		t.Fatal("old Natural receipt retargeted")
	}
	if old.roots(c.integer) != nil {
		t.Fatal("old context retargeted")
	}
	current := c.modular.dividesAllocationContext(&s.directReceipt)
	if current.roots(c.integer) != &s.numeric {
		t.Fatal("current context rejected")
	}
	s.active = false
	s.executing = false
	serviceMust(t, m.CloseChecked())
	serviceMust(t, q.CloseChecked())
	if b.Used() != 0 {
		t.Fatal("owner leak", b.Used())
	}
}

func TestDividesNumericActualBoolChildAndErasedFieldRoute(t *testing.T) {
	full := polynomialPSCFullFixture(t)
	c := full.core
	s := &c.dividesLocal
	*s = polynomialDividesScope{active: true, executing: true, constructed: 2, invocation: 1}
	r := polynomialDividesInvocationReceipt{core: c, frame: s, invocation: 1}
	zero := NewInt(0)
	for i := 0; i < 2; i++ {
		previous := s.boolNumeral.invocation
		result, e := c.modular.dividesWithDividesReceipt(&zero, &zero, &r)
		serviceMust(t, e)
		if !result || s.boolNumeral.frame != &c.integer.dividesNumeral || s.boolNumeral.manager != c.integer || s.boolNumeral.invocation <= previous || c.integer.dividesNumeral.parentContext.divides != &r || c.integer.dividesNumericPending(&r) {
			t.Fatal("actual repeated scoped bool child/handoff", result, s.boolNumeral)
		}
	}
	before := s.boolNumeral
	serviceMust(t, c.modular.SetZPUint64(5))
	one := NewInt(1)
	result, e := c.modular.dividesWithDividesReceipt(&one, &zero, &r)
	serviceMust(t, e)
	if !result || s.boolNumeral != before {
		t.Fatal("field nonzero shortcut acquired nonexistent scoped bool child")
	}
	s.executing = false
	serviceMust(t, c.integer.cleanupDividesNumeric(&r, false))
	serviceMust(t, c.integer.cleanupDividesNumeric(&r, true))
	s.active = false
}

// Actual lower Div/Inv reservation prefixes under an explicitly bound actual
// Divides frame. This does not replace the full-body source-order campaign.
func TestDividesNumericLowerActualPrefixes(t *testing.T) {
	for route := 0; route < 3; route++ {
		success, retained := false, false
		for prefix := int64(0); prefix < 256 && !success; prefix++ {
			var m *MultivariatePolynomialManager
			if route == 2 {
				m = dividesNumericPageOneFixture(t)
			} else {
				m = polynomialPSCFullFixture(t)
			}
			c := m.core
			s := &c.dividesLocal
			var a, divisor, modulus Int
			if route == 0 {
				ds := make([]uint32, 65)
				ds[64] = 1
				serviceMust(t, c.integer.SetLimbs(&a, ds))
				serviceMust(t, c.integer.Neg(&a))
			} else {
				prime := make([]uint32, 17)
				for i := range prime {
					prime[i] = ^uint32(0)
				}
				prime[16] = 511
				serviceMust(t, c.integer.SetLimbs(&modulus, prime))
				serviceMust(t, c.modular.SetZP(&modulus))
				a = NewInt(2)
			}
			divisor = NewInt(2)
			if route == 2 {
				divisor = NewInt(-2147483648)
			}
			*s = polynomialDividesScope{invocation: 1, constructed: 2, active: true, executing: true}
			r := polynomialDividesInvocationReceipt{core: c, frame: s, invocation: 1}
			c.integer.budget.FailAfter(prefix)
			callErr := c.modular.divWithDividesReceipt(&a, &divisor, &s.numeral, &r)
			s.executing = false
			if callErr != nil && !numericPhysicalReservationRefusal(callErr) {
				t.Fatal("actual lower Div refusal", route, prefix, callErr)
			}
			polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&a, &divisor, &modulus, &s.numeral}, 0)
			child := s.divisionRaw
			if route != 0 {
				child = s.extendedRaw
			}

			if route == 2 && c.integer.matchesRawInvocation(&child) && child.frame.count == 6 && child.frame.values[0].val == -2147483648 && c.integer.pendingCell.object != (sourceCellBacking{}).object {
				if child.frame.numeric.cell != c.integer.pendingCell {
					t.Fatal("actual ExtendedGCD Abs MIN leaf unowned", prefix, c.integer.pendingCell, child.frame.numeric)
				}
			}
			var before [6]Int
			if c.integer.matchesRawInvocation(&child) {
				before = child.frame.values
				for i := uint8(0); i < child.frame.count; i++ {
					retained = retained || callErr != nil && child.frame.values[i].cell != 0
				}
			}
			serviceMust(t, c.integer.cleanupDividesNumeric(&r, false))
			if c.integer.matchesRawInvocation(&child) && !child.frame.bodySucceeded && child.frame.values != before {
				t.Fatal("failed RAW body ran absent source end DEL")
			}
			polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&a, &divisor, &modulus, &s.numeral}, 0)
			// Older caller NUM completes before exact unjoined child owning abort.
			if callErr != nil {
				serviceMust(t, c.integer.Del(&s.numeral))
			}
			serviceMust(t, c.integer.cleanupDividesNumeric(&r, true))
			polynomialImpPhysicalAuditNamed(t, m, nil, []*Int{&a, &divisor, &modulus, &s.numeral}, 0)
			c.integer.budget.FailAfter(-1)
			if callErr == nil {
				var want Int
				if route == 0 {
					ds := make([]uint32, 64)
					ds[63] = 0x80000000
					serviceMust(t, c.integer.SetLimbs(&want, ds))
					serviceMust(t, c.integer.Neg(&want))
				} else if route == 1 {
					want = NewInt(1)
				} else {
					ds := make([]uint32, 16)
					ds[15] = 2048
					serviceMust(t, c.integer.SetLimbs(&want, ds))
					serviceMust(t, c.integer.Neg(&want))
				}
				cmp, e := c.integer.Compare(&s.numeral, &want)
				serviceMust(t, e)
				if cmp != 0 {
					t.Fatal("independent lower Div result", route, prefix)
				}
				serviceMust(t, c.integer.Del(&want))
				success = true
				t.Log("first actual lower Div success", route, prefix)
			}
			serviceMust(t, c.integer.Del(&s.numeral))
			s.active = false
			serviceMust(t, c.integer.Del(&a))
			serviceMust(t, c.integer.Del(&modulus))
		}
		if !success || route != 0 && !retained {
			t.Fatal("actual lower RAW horizon absent", route, success, retained)
		}
	}
}

func dividesNumericPageOneFixture(t *testing.T) *MultivariatePolynomialManager {
	b := arena.NewBudget(1 << 26)
	cfg := serviceConfig(b).Integers
	cfg.Cells.PageSize = 1
	cfg.Profile = memory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	q, e := NewRationalManager(RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: q.z, Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CloseChecked())
		serviceMust(t, q.CloseChecked())
		if b.Used() != 0 {
			t.Fatal("closed lower owner retains", b.Used())
		}
	})
	return m
}
