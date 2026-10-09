// Source: src/math/polynomial/polynomial.cpp:454-481,1988-2039.
package polynomial

import (
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialRegularBorrowedPositionSurvivesOlderMonomialDestruction(t *testing.T) {
	for prefix := int64(0); prefix < 32; prefix++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		_, varErr := c.monomials.MkVar()
		serviceMust(t, varErr)
		_, varErr = c.monomials.MkVar()
		serviceMust(t, varErr)
		p, e := c.MkVariable(0, 1)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		var r polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&r))
		one := util.NewInt(1)
		c.integer.Budget().FailAfter(prefix)
		e = (c.som).addMulWithArithmeticReceipt(&one, c.monomials.unit, p, &r)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		if e != nil && r.frame.positionPublished {
			term := r.frame.positionTerm
			id, x := c.som.positions.id(term)
			serviceMust(t, x)
			ms, x := c.som.ms.size(c)
			serviceMust(t, x)
			if ms != 0 {
				t.Fatal("position before actual push", ms)
			}
			c.integer.Budget().FailAfter(-1)
			serviceMust(t, c.DecRef(p))
			if _, x = c.monomials.location(term); x == nil {
				t.Fatal("borrowed term remained physically live")
			}
			polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
			if x = c.cleanupArithmeticInvocation(&r, true); x != nil {
				t.Fatal("actual position publication must not reread destroyed borrowed monomial", prefix, e, x)
			}
			v, x := c.som.positions.read(id)
			serviceMust(t, x)
			if v != ^uint32(0) {
				t.Fatal("exact acquired marker not retired", v)
			}
			polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
			return
		}
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
		serviceMust(t, c.DecRef(p))
	}
	t.Fatal("did not reach real position Set then push refusal")
}

// Valid lower mapping operations exercise actual ID recycling, independent of
// any RC acquisition by the position map. This is not a whole ModD callback recipe.
func TestPolynomialRegularPositionMarkerPreservesRecycledForeignWrite(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	_, varErr := c.monomials.MkVar()
	serviceMust(t, varErr)
	_, varErr = c.monomials.MkVar()
	serviceMust(t, varErr)
	old, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(old))
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	r.frame.executing = true
	serviceMust(t, c.som.positions.setWithArithmeticReceipt(old, 7, &r))
	r.frame.executing = false
	id, e := c.som.positions.id(old)
	serviceMust(t, e)
	serviceMust(t, c.monomials.DecRef(old))
	newer, e := c.monomials.MkVariable(1, 1)
	serviceMust(t, e)
	serviceMust(t, c.monomials.IncRef(newer))
	newerID, e := c.som.positions.id(newer)
	serviceMust(t, e)
	if newerID != id || newer == old {
		t.Fatal("actual recycled ID / new identity", id, newerID, old, newer)
	}
	serviceMust(t, c.som.positions.Reset(newer))
	serviceMust(t, c.som.positions.Set(newer, 99))
	before := c.integer.Budget().Used()
	rc, e := c.monomialRefCount(newer)
	serviceMust(t, e)
	polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	value, e := c.som.positions.Get(newer)
	serviceMust(t, e)
	afterRC, e := c.monomialRefCount(newer)
	serviceMust(t, e)
	if value != 99 || afterRC != rc || c.integer.Budget().Used() != before {
		t.Fatal("old ACK changed foreign row", value, rc, afterRC, before, c.integer.Budget().Used())
	}
	polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
	serviceMust(t, c.som.positions.Reset(newer))
	serviceMust(t, c.monomials.DecRef(newer))
}

func TestPolynomialRegularPositionMarkerCallbackWritePreserved(t *testing.T) {
	for prefix := int64(0); prefix < 32; prefix++ {
		full := polynomialArithmeticDirectNumericFixture(t)
		c := full.core
		_, varErr := c.monomials.MkVar()
		serviceMust(t, varErr)
		_, varErr = c.monomials.MkVar()
		serviceMust(t, varErr)
		p, e := c.MkVariable(0, 1)
		serviceMust(t, e)
		serviceMust(t, c.IncRef(p))
		var r polynomialArithmeticInvocationReceipt
		serviceMust(t, c.som.acquireArithmeticInvocation(&r))
		one := util.NewInt(1)
		c.integer.Budget().FailAfter(prefix)
		e = (c.som).addMulWithArithmeticReceipt(&one, c.monomials.unit, p, &r)
		polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
		if e != nil && r.frame.positionPublished {
			term := r.frame.positionTerm
			id, x := c.som.positions.id(term)
			serviceMust(t, x)
			ms, x := c.som.ms.size(c)
			serviceMust(t, x)
			if ms != 0 {
				t.Fatal("position before actual push", ms)
			}
			c.integer.Budget().FailAfter(-1)
			calls := 0
			h, x := NewPolynomialDeletionHandler(c.integer.Budget(), func(h *PolynomialDeletionHandler, owner *MultivariatePolynomialCore, victim Polynomial) error {
				if victim != p {
					return nil
				}
				calls++
				term, x := owner.monomialCore(victim, 0)
				if x != nil {
					return x
				}
				if x = owner.som.positions.Reset(term); x != nil {
					return x
				}
				return owner.som.positions.Set(term, 99)
			})
			serviceMust(t, x)
			serviceMust(t, c.AddDeletionHandler(h))
			serviceMust(t, c.DecRef(p))
			serviceMust(t, c.RemoveDeletionHandler(h))
			serviceMust(t, h.CloseChecked())
			if calls != 1 {
				t.Fatal("actual deletion callback absent/replayed", calls)
			}
			if _, x = c.monomials.location(term); x == nil {
				t.Fatal("borrowed term remained physically live")
			}
			polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
			if x = c.cleanupArithmeticInvocation(&r, true); x != nil {
				t.Fatal("actual position publication must not reread destroyed borrowed monomial", prefix, e, x)
			}
			v, x := c.som.positions.read(id)
			serviceMust(t, x)
			if v != 99 {
				t.Fatal("later actual callback marker was retired", v)
			}
			polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
			return
		}
		c.integer.Budget().FailAfter(-1)
		serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
		serviceMust(t, c.DecRef(p))
	}
	t.Fatal("did not reach real position Set then push refusal")
}

func TestPolynomialRegularPositionMarkerRejectsCopiedAndStaleAcquisition(t *testing.T) {
	full := polynomialArithmeticDirectNumericFixture(t)
	c := full.core
	var r polynomialArithmeticInvocationReceipt
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	copied := *r.frame
	bad := r
	bad.frame = &copied
	r.frame.executing = true
	before := c.integer.Budget().Used()
	beforeLen := len(c.som.positions.values)
	if e := c.som.positions.setWithArithmeticReceipt(c.monomials.unit, 7, &bad); e != ErrPolynomialOwner {
		t.Fatal("copied frame accepted", e)
	}
	pcopy := c.som.positions
	if e := pcopy.setWithArithmeticReceipt(c.monomials.unit, 7, &r); e != ErrPolynomialOwner {
		t.Fatal("copied position owner accepted", e)
	}
	c.callbackDepth++
	if e := c.som.positions.setWithArithmeticReceipt(c.monomials.unit, 7, &r); e != ErrPolynomialOwner {
		t.Fatal("callback receipt acquired", e)
	}
	c.callbackDepth--
	if beforeLen != len(c.som.positions.values) || before != c.integer.Budget().Used() || r.frame.positionPublished {
		t.Fatal("rejected marker acquired storage/authority")
	}
	serviceMust(t, c.som.positions.setWithArithmeticReceipt(c.monomials.unit, 7, &r))
	r.frame.executing = false
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	old := r
	serviceMust(t, c.som.acquireArithmeticInvocation(&r))
	r.frame.executing = true
	if e := c.som.positions.setWithArithmeticReceipt(c.monomials.unit, 9, &old); e != ErrPolynomialOwner {
		t.Fatal("old actual nonce retargeted current frame", e)
	}
	value, e := c.som.positions.Get(c.monomials.unit)
	serviceMust(t, e)
	if value != ^uint32(0) || r.frame.positionPublished {
		t.Fatal("stale setter changed exact new mapping", value)
	}
	r.frame.executing = false
	serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
	polynomialImpPhysicalAuditNamed(t, full, nil, nil, 0)
}

func TestPolynomialRegularPositionMarkerActualMapDestructionConsumesToken(t *testing.T) {
	for _, own := range []bool{false, true} {
		t.Run(map[bool]string{false: "source", true: "owning"}[own], func(t *testing.T) {
			for prefix := int64(0); prefix < 32; prefix++ {
				full := polynomialArithmeticDirectNumericFixture(t)
				c := full.core
				_, e := c.monomials.MkVar()
				serviceMust(t, e)
				p, e := c.MkVariable(0, 1)
				serviceMust(t, e)
				serviceMust(t, c.IncRef(p))
				var r polynomialArithmeticInvocationReceipt
				serviceMust(t, c.som.acquireArithmeticInvocation(&r))
				one := util.NewInt(1)
				c.integer.Budget().FailAfter(prefix)
				body := (c.som).addMulWithArithmeticReceipt(&one, c.monomials.unit, p, &r)
				polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
				c.integer.Budget().FailAfter(-1)
				if body != nil && r.frame.positionPublished {
					if len(c.som.positions.values) == 0 {
						t.Fatal("published marker has no position storage")
					}
					if own {
						e = c.som.positions.CloseChecked()
					} else {
						e = c.som.positions.SourceFinalize()
					}
					serviceMust(t, e)
					if c.som.positions.values != nil {
						t.Fatal("native position storage was not consumed")
					}
					polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
					if e = c.cleanupArithmeticInvocation(&r, true); e != nil {
						t.Fatal("consumed actual map marker must be acknowledged", body, e)
					}
					if r.frame.positionPublished || c.som.positions.markerFrame != nil || r.frame.bodyCause != body {
						t.Fatal("actual consumption lost marker/history", r.frame)
					}
					serviceMust(t, c.DecRef(p))
					return
				}
				serviceMust(t, c.cleanupArithmeticInvocation(&r, true))
				serviceMust(t, c.DecRef(p))
			}
			t.Fatal("actual mapping acquisition frontier absent")
		})
	}
}
