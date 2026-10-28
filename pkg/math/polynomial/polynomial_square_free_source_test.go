package polynomial

// Source: src/math/polynomial/polynomial.cpp:4529-4598,7232-7246;
// src/math/polynomial/polynomial.h; src/util/obj_ref.h.

import (
	"errors"
	"fmt"
	"testing"

	umemory "github.com/glycerine/g3/pkg/util/memory"
	unumeral "github.com/glycerine/g3/pkg/util/numeral"
)

func TestPolynomialSquareFreeLiteralMultiplicityAndAliases(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{2, 4, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() { serviceMust(t, m.CleanupGCD()); serviceMust(t, out.Close()) })
	serviceMust(t, m.SquareFree(p, out))
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 2, {}: 2})
	serviceMust(t, m.SquareFreeVariable(p, 0, out))
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
	yes, e := m.IsSquareFree(p)
	serviceMust(t, e)
	if yes {
		t.Fatal("repeated source factor")
	}
	q := polynomialQueryTerms(t, c, []int32{2, 2}, [][]MonomialPower{{{0, 1}}, nil})
	serviceMust(t, out.Assign(q))
	serviceMust(t, m.SquareFree(q, out))
	if out.root.value != q {
		t.Fatal("source unchanged square-free identity")
	}
	yes, e = m.IsSquareFreeVariable(q, 0)
	serviceMust(t, e)
	if !yes {
		t.Fatal("source variable square-free identity")
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, q, out.root.value}, nil, out.header)
}

func TestPolynomialSquareFreeEveryReservation(t *testing.T) {
	for op := 0; op < 4; op++ {
		success := false
		for ordinal := int64(0); ordinal < 512; ordinal++ {
			done := false
			t.Run(fmt.Sprint(op, "/", ordinal), func(t *testing.T) {
				m := polynomialPrimitiveFixture(t)
				c := m.core
				p := polynomialQueryTerms(t, c, []int32{2, 4, 2}, [][]MonomialPower{{{0, 2}}, {{0, 1}, {1, 1}}, {{1, 2}}})
				out, e := NewPolynomialRef(c)
				serviceMust(t, e)
				var got bool
				c.integer.Budget().FailAfter(ordinal)
				switch op {
				case 0:
					e = m.SquareFree(p, out)
				case 1:
					e = m.SquareFreeVariable(p, 0, out)
				case 2:
					got, e = m.IsSquareFree(p)
				case 3:
					got, e = m.IsSquareFreeVariable(p, 0)
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
				c.integer.Budget().FailAfter(-1)
				if e == nil {
					done = true
					switch op {
					case 0:
						polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 2, {0, 1, 0}: 2})
					case 1:
						polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{1, 0, 0}: 1, {0, 1, 0}: 1})
					default:
						if got {
							t.Fatal("independent repeated factor")
						}
					}
				} else if !numericPhysicalReservationRefusal(e) {
					t.Fatal("actual source reservation prefix", e)
				}
				serviceMust(t, m.CleanupGCD())
				serviceMust(t, out.Close())
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p}, nil, 0)
			})
			if done {
				t.Logf("op%d failures0..%d first success%d", op, ordinal-1, ordinal)
				success = true
				break
			}
		}
		if !success {
			t.Fatal("actual source horizon not reached", op)
		}
	}
}

func TestPolynomialSquareFreeLiteralGF2AndAbsentVariable(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 2}}})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CleanupGCD())
		serviceMust(t, m.CleanupDerivative())
		serviceMust(t, out.Close())
	})
	// Source derivative is zero in GF2, so GCD(p,0)=p and the literal
	// exact-division branch returns one. This is source behavior, rather
	// than a replacement characteristic-p square-free algorithm.
	prime := unumeral.NewInt(2)
	serviceMust(t, c.modular.SetZP(&prime))
	serviceMust(t, m.SquareFreeVariable(p, 0, out))
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{}: 1})
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value}, nil, out.header)
	serviceMust(t, m.CleanupGCD())
	serviceMust(t, m.SquareFreeVariable(p, 1, out))
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{}: 1})
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value}, nil, out.header)
}

func TestPolynomialSquareFreeEarlyRoutesPreserveOlderDerivative(t *testing.T) {
	m := polynomialPrimitiveFixtureWithProfile(t, umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true})
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{0, 2}}})
	constant := polynomialQueryTerms(t, c, []int32{7}, [][]MonomialPower{nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CleanupGCD())
		serviceMust(t, m.CleanupDerivative())
		serviceMust(t, out.Close())
	})
	var old polynomialDerivativeInvocationReceipt
	c.integer.Budget().FailAfter(0)
	_, e = m.derivativeWithReceipt(p, 0, &old)
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, constant}, nil, out.header)
	c.integer.Budget().FailAfter(-1)
	if !numericPhysicalReservationRefusal(e) || !m.matchesDerivativeInvocation(&old) {
		t.Fatal("actual retained derivative", e)
	}
	nums := [2]unumeral.Int{m.derivative.n, m.derivative.a}
	nonce := m.derivative.invocation
	for _, v := range []Polynomial{c.zero, constant} {
		serviceMust(t, m.SquareFreeVariable(v, ^uint32(0), out))
		if out.root.value != v {
			t.Fatal("source zero/constant identity", v, out.root.value)
		}
		if m.derivative.invocation != nonce || [2]unumeral.Int{m.derivative.n, m.derivative.a} != nums || !m.matchesDerivativeInvocation(&old) {
			t.Fatal("early route mutated older lexical context")
		}
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, constant, out.root.value}, nil, out.header)
	}
	serviceMust(t, m.cleanupDerivativeInvocation(&old, true))
}

// src/test/polynomial.cpp:894-916. The registered driver returns before
// tst_sqf; these seven original expressions are dormant adaptations.
func TestPolynomialSquareFreeDormantSourceSevenExpressions(t *testing.T) {
	for which := 0; which < 7; which++ {
		t.Run(fmt.Sprint(which), func(t *testing.T) {
			m := polynomialPrimitiveFixture(t)
			c := m.core
			for i := 0; i < 2; i++ {
				_, e := c.monomials.MkVar()
				serviceMust(t, e)
			}
			xs := [4]Polynomial{}
			for i := range xs {
				xs[i] = polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{{{uint32(i), 1}}})
			}
			one := polynomialQueryTerms(t, c, []int32{1}, [][]MonomialPower{nil})
			three := polynomialQueryTerms(t, c, []int32{3}, [][]MonomialPower{nil})
			add := func(a, b Polynomial) Polynomial {
				r, e := c.Add(a, b)
				serviceMust(t, e)
				return polynomialCaller(t, c, r)
			}
			sub := func(a, b Polynomial) Polynomial {
				r, e := c.Sub(a, b)
				serviceMust(t, e)
				return polynomialCaller(t, c, r)
			}
			mul := func(a, b Polynomial) Polynomial {
				r, e := c.Mul(a, b)
				serviceMust(t, e)
				return polynomialCaller(t, c, r)
			}
			pw := func(a Polynomial, k uint32) Polynomial {
				r, e := NewPolynomialRef(c)
				serviceMust(t, e)
				t.Cleanup(func() { serviceMust(t, r.Close()) })
				serviceMust(t, c.Power(a, k, r))
				return r.root.value
			}
			var p, expected Polynomial
			sum := add(xs[0], xs[1])
			diff := sub(xs[0], xs[1])
			switch which {
			case 0:
				b := sub(pw(xs[0], 2), three)
				d := add(add(mul(xs[2], xs[2]), xs[3]), one)
				p = mul(mul(pw(sum, 2), b), pw(d, 3))
				expected = mul(mul(sum, b), d)
			case 1:
				p = mul(sum, diff)
				expected = p
			case 2:
				p = mul(pw(sum, 3), diff)
				expected = mul(sum, diff)
			case 3:
				p = three
				expected = three
			case 4:
				p = c.zero
				expected = c.zero
			case 5:
				expected = add(add(sum, xs[2]), xs[3])
				p = pw(expected, 5)
			case 6:
				p = add(pw(add(add(sum, xs[2]), xs[3]), 5), one)
				expected = p
			}
			out, e := NewPolynomialRef(c)
			serviceMust(t, e)
			t.Cleanup(func() {
				serviceMust(t, m.CleanupGCD())
				serviceMust(t, m.CleanupDerivative())
				serviceMust(t, out.Close())
			})
			serviceMust(t, m.SquareFree(p, out))
			got, e := m.IsSquareFree(out.root.value)
			serviceMust(t, e)
			if !got {
				t.Fatal("source ENSURE square-free result")
			}
			equal, e := c.Equal(out.root.value, p)
			serviceMust(t, e)
			if equal {
				got, e = m.IsSquareFree(p)
				serviceMust(t, e)
				if !got {
					t.Fatal("source ENSURE unchanged implies square-free")
				}
			}
			equal, e = c.Equal(expected, out.root.value)
			serviceMust(t, e)
			if !equal {
				t.Fatal("source ENSURE expected expression", which)
			}
		})
	}
}

func TestPolynomialSquareFreeEveryCheckpointFrontier(t *testing.T) {
	for op := 0; op < 4; op++ {
		success := false
		for limit := uint32(0); limit < 1024; limit++ {
			done := false
			t.Run(fmt.Sprint(op, "/", limit), func(t *testing.T) {
				m, p, control := polynomialEvalFixture(t, []polynomialEvalTerm{{2, [3]uint32{2, 0, 0}}, {4, [3]uint32{1, 1, 0}}, {2, [3]uint32{0, 2, 0}}})
				c := m.core
				out, e := NewPolynomialRef(c)
				serviceMust(t, e)
				t.Cleanup(func() {
					serviceMust(t, m.CleanupGCD())
					serviceMust(t, m.CleanupDerivative())
					serviceMust(t, out.Close())
				})
				control.checkpoints = 0
				control.cooperation = 0
				control.limit = limit
				var got bool
				switch op {
				case 0:
					e = m.SquareFree(p, out)
				case 1:
					e = m.SquareFreeVariable(p, 1, out)
				case 2:
					got, e = m.IsSquareFree(p)
				case 3:
					got, e = m.IsSquareFreeVariable(p, 1)
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
				if e == nil {
					done = true
					if control.checkpoints != control.cooperation {
						t.Fatal("success control pairing", control)
					}
					if op >= 2 && got {
						t.Fatal("independent repeated factor")
					}
				} else {
					if !errors.Is(e, ErrUPolynomialCanceled) {
						t.Fatal("actual source checkpoint cause", e)
					}
					if control.checkpoints != limit+1 || control.cooperation != limit {
						t.Fatal("actual cancellation frontier", control, limit)
					}
				}
				// Explicit invocation cleanup runs after immediate named graph, not an
				// arithmetic retry or a fabricated resource reset inside the source body.
				control.limit = ^uint32(0)
				serviceMust(t, m.CleanupGCD())
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value, out.root.incoming}, nil, out.header)
			})
			if done {
				t.Logf("op%d actual canceled frontiers0..%d first success%d", op, int64(limit)-1, limit)
				success = true
				break
			}
		}
		if !success {
			t.Fatal("actual checkpoint horizon not reached", op)
		}
	}
}

func TestPolynomialSquareFreeCallerPublicationPreservesForeignCallbackRoot(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 2, 1}, [][]MonomialPower{{{0, 2}}, {{0, 1}}, nil})
	old := polynomialICCPUnownedTerms(t, c, []int32{1, 5}, [][]MonomialPower{{{1, 1}}, nil})
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	serviceMust(t, out.Assign(old))
	var foreign Monomial
	var refused umemory.Block
	calls := 0
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, c *MultivariatePolynomialCore, value Polynomial) error {
		if value != old {
			return nil
		}
		calls++
		if calls != 1 {
			t.Fatal("actual old caller callback replay")
		}
		var e error
		foreign, e = c.monomials.MkVariable(1, 7)
		if e != nil {
			return e
		}
		c.integer.Budget().FailAfter(0)
		refused, e = c.memory.Allocate(8)
		if !numericPhysicalReservationRefusal(e) {
			t.Fatal("actual callback reservation", e)
		}
		return e
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	t.Cleanup(func() {
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, m.CleanupGCD())
		serviceMust(t, m.CleanupDerivative())
		serviceMust(t, out.Close())
		serviceMust(t, c.monomials.CleanupValue(&foreign))
		serviceMust(t, c.RemoveDeletionHandler(h))
		serviceMust(t, h.CloseChecked())
	})
	e = m.SquareFreeVariable(p, 0, out)
	if !numericPhysicalReservationRefusal(e) || calls != 1 || foreign == (Monomial{}) || out.root.incoming == (Polynomial{}) {
		t.Fatal("actual source output publication before old caller stop", e, calls, foreign, out.root)
	}
	incoming := out.root.incoming
	if _, e = c.monomials.Info(foreign); e != nil {
		t.Fatal("named callback monomial before fault lift", e)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p, out.root.value, incoming}, nil, out.header+h.header, []umemory.Block{refused})
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, m.CleanupGCD())
	if out.root.incoming != incoming {
		t.Fatal("operation cleanup consumed caller incoming")
	}
	polynomialQueryExpect(t, c, incoming, map[[3]uint32]int64{{1, 0, 0}: 1, {}: 1})
	if _, e = c.monomials.Info(foreign); e != nil {
		t.Fatal("parent cleanup swept callback-owned monomial", e)
	}
	polynomialImpPhysicalAuditNamedRaw(t, m, []Polynomial{p, out.root.value, incoming}, nil, out.header+h.header, []umemory.Block{refused})
	serviceMust(t, out.Close())
	if calls != 1 {
		t.Fatal("caller owning abort replayed source callback")
	}
}

func TestPolynomialSquareFreeCalleeAbortPreservesEarlierDerivativeCohort(t *testing.T) {
	m := polynomialPrimitiveFixture(t)
	c := m.core
	p := polynomialQueryTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}}, {{1, 2}}})
	prime := unumeral.NewInt(2)
	serviceMust(t, c.modular.SetZP(&prime))
	earlier, e := m.Derivative(p, 0)
	serviceMust(t, e)
	polynomialCaller(t, c, earlier)
	if len(m.derivative.orphans) != 1 || m.derivative.orphans[0] == (Monomial{}) {
		t.Fatal("actual earlier GF2 created cohort")
	}
	older := m.derivative.orphans[0]
	olderTag := m.derivative.orphanInvocations[0]
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	t.Cleanup(func() {
		serviceMust(t, m.CleanupGCD())
		serviceMust(t, m.CleanupDerivative())
		serviceMust(t, out.Close())
	})
	serviceMust(t, m.SquareFreeVariable(p, 1, out))
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{}: 1})
	if len(m.derivative.orphans) != 2 || m.derivative.orphans[1] == (Monomial{}) || m.derivative.orphanInvocations[1] == olderTag {
		t.Fatal("actual separately acquired square-free child cohort")
	}
	younger := m.derivative.orphans[1]
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, earlier, out.root.value}, nil, out.header)
	serviceMust(t, m.CleanupGCD())
	if m.derivative.orphans[0] != older || m.derivative.orphanInvocations[0] != olderTag || m.derivative.orphans[1] != (Monomial{}) || m.derivative.orphanInvocations[1] != 0 {
		t.Fatal("parent receipt cleanup crossed cohort boundary", older, younger, m.derivative.orphans, m.derivative.orphanInvocations)
	}
	if _, e = c.monomials.Info(older); e != nil {
		t.Fatal("earlier actual root lost", e)
	}
	polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, earlier, out.root.value}, nil, out.header)
}
