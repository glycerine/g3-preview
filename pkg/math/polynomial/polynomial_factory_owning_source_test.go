// Source: src/math/polynomial/polynomial.cpp:1392-1423,2481-2495.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"fmt"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialFactoryOwningCloseRetainedActualResult(t *testing.T) {
	for _, fullOwner := range []bool{false, true} {
		for _, cheap := range []bool{false, true} {
			t.Run(map[bool]string{false: "core", true: "full"}[fullOwner]+map[bool]string{false: "/placement", true: "/cheap"}[cheap], func(t *testing.T) {
				q, _ := rationalManager(t, false)
				var full *MultivariatePolynomialManager
				var c *MultivariatePolynomialCore
				var e error
				cfg := MultivariatePolynomialConfig{Integer: q.Integers(), Rationals: q, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}
				if fullOwner {
					full, e = NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: cfg})
					serviceMust(t, e)
					c = full.core
				} else {
					mm, x := NewMonomialManager(MonomialManagerConfig{Budget: q.Integers().BudgetField, Memory: q.Integers().RuntimeOwner()})
					serviceMust(t, x)
					serviceMust(t, mm.IncManagerRef())
					t.Cleanup(func() { serviceMust(t, mm.DecManagerRef()) })
					cfg.Monomials = mm
					c, e = NewMultivariatePolynomialCore(cfg)
					serviceMust(t, e)
				}
				var receipt polynomialFactoryInvocationReceipt
				closeOwner := func() error {
					if full != nil {
						return full.CloseChecked()
					}
					return c.CloseChecked()
				}
				t.Cleanup(func() {
					if !c.closed {
						serviceMust(t, c.cleanupFactoryInvocation(&receipt, true))
						serviceMust(t, closeOwner())
					}
				})
				var p Polynomial
				if cheap {
					a := util.NewInt(7)
					serviceMust(t, c.cheap.add(&a, c.monomials.unit, false))
					p, e = c.cheap.mkWithFactoryReceipt(false, &receipt)
				} else {
					p, e = c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
				}
				serviceMust(t, e)
				if !c.matchesFactoryInvocation(&receipt) || c.factoryInvocation.result != p {
					t.Fatal("actual constructor authority missing before owning close")
				}
				if full != nil {
					polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
				}
				// A raw constructor root owns no INC. Real caller ownership must
				// still refuse before any source or owning cleanup action.
				serviceMust(t, c.IncRef(p))
				before := c.factoryInvocation
				if e = closeOwner(); e != ErrPolynomialLive || c.factoryInvocation != before || c.closing {
					t.Fatalf("caller INC was not preserved: %v", e)
				}
				if e = c.CleanupConstruction(); e != ErrPolynomialLive || c.factoryInvocation != before {
					t.Fatalf("generic construction cleanup consumed caller INC: %v", e)
				}
				// Transfer returns authority to the caller before its actual DEC;
				// construct a fresh raw root for the untransferred-close proof.
				serviceMust(t, c.transferFactoryResult(&receipt, p))
				serviceMust(t, c.DecRef(p))
				if cheap {
					a := util.NewInt(11)
					serviceMust(t, c.cheap.add(&a, c.monomials.unit, false))
					p, e = c.cheap.mkWithFactoryReceipt(false, &receipt)
				} else {
					p, e = c.mkCellWithReceipt(0, nil, nil, nil, nil, &receipt)
				}
				serviceMust(t, e)
				if full != nil {
					polynomialImpPhysicalAuditNamed(t, full, []Polynomial{p}, nil, 0)
				}
				serviceMust(t, closeOwner())
				if !c.closed || c.factoryInvocation.active || c.factoryInvocation.result != (Polynomial{}) {
					t.Fatal("owning close retained an untransferred actual factory root")
				}
			})
		}
	}
}

func TestPolynomialFactoryOwningConstructionActualPrefixes(t *testing.T) {
	for _, large := range []bool{false, true} {
		success := false
		for ordinal := int64(0); ordinal < 64 && !success; ordinal++ {
			t.Run(fmt.Sprint(large, "/", ordinal), func(t *testing.T) {
				m := polynomialPSCFullFixture(t)
				c := m.core
				var appendReceipt polynomialCheapAppendInvocationReceipt
				a := util.NewInt(7)
				if large {
					limbs := make([]uint32, 8192)
					limbs[0], limbs[len(limbs)-1] = 1, 1
					serviceMust(t, c.integer.SetLimbs(&a, limbs))
					t.Cleanup(func() { serviceMust(t, c.integer.Del(&a)) })
				}
				serviceMust(t, c.cheap.addWithAppendReceipt(&a, c.monomials.unit, false, &appendReceipt))
				var factoryReceipt polynomialFactoryInvocationReceipt
				c.integer.Budget().FailAfter(ordinal)
				p, e := c.cheap.mkWithAppendReceipt(false, &appendReceipt, &factoryReceipt)
				if e != nil && !numericPhysicalReservationRefusal(e) {
					t.Fatal(e)
				}
				if !c.matchesFactoryInvocation(&factoryReceipt) {
					t.Fatal("real constructor authority missing")
				}
				polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, c.factoryInvocation.result}, []*util.Int{&a, &c.cheap.appendScope.movingCoefficient}, 0)
				c.integer.Budget().FailAfter(-1)
				// This is the actual named owner continuation, with no invocation
				// receipt supplied by the test and no registry-derived roots.
				serviceMust(t, c.CleanupConstruction())
				if c.factoryInvocation.active || c.factoryInvocation.result != (Polynomial{}) || c.cheap.appendScope.movingCoefficient != (util.Int{}) {
					t.Fatal("generic construction cleanup retained an owned factory/NUM")
				}
				polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&a}, 0)
				serviceMust(t, c.cheap.cleanupAppendInvocation(&appendReceipt, true))
				polynomialImpPhysicalAuditNamed(t, m, nil, []*util.Int{&a}, 0)
				if e == nil {
					success = true
					t.Log("first actual factory success", ordinal)
				}
			})
		}
		if !success {
			t.Fatal("actual constructor horizon not reached")
		}
	}
}
