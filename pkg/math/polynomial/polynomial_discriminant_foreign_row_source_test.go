// Source: src/math/polynomial/polynomial.cpp:5473-5500;
// src/util/obj_ref.h:polynomial_ref destruction.
// Licensed under the MIT license. Source author: Leonardo de Moura.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
)

func TestPolynomialDiscriminantYoungerRefCallbackForeignRowSurvivesExactAbort(t *testing.T) {
	m := polynomialDiscriminantFixture(t)
	c := m.core
	for i := 0; i < 2; i++ {
		_, e := c.monomials.MkVar()
		serviceMust(t, e)
	}
	p := polynomialICCPUnownedTerms(t, c, []int32{1, 1}, [][]MonomialPower{{{0, 2}, {1, 1}}, nil})
	p = polynomialCaller(t, c, p)
	out, e := NewPolynomialRef(c)
	serviceMust(t, e)
	calls := 0
	var foreign Monomial
	var stopped error
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, value Polynomial) error {
		s := &m.discriminant
		if calls != 0 || s.executing || s.depth != 3 || value != s.refs[1].value {
			return nil
		}
		calls++
		foreign, e = core.monomialCore(value, 0)
		if e != nil {
			return e
		}
		seven := util.NewInt(7)
		if e = core.cheap.add(&seven, foreign, false); e != nil {
			return e
		}
		core.integer.Budget().FailAfter(0)
		_, stopped = core.memory.Allocate(8)
		if stopped == nil {
			t.Fatal("missing actual callback refusal")
		}
		return stopped
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	returned := m.Discriminant(p, 0, out)
	if calls != 1 || !errors.Is(returned, stopped) || m.DiscriminantSourceCause() != stopped {
		t.Fatal("actual younger reference stop", calls, returned, stopped)
	}
	audit := func() {
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value, m.discriminant.refs[0].value, m.discriminant.refs[1].value}, []*util.Int{&m.discriminant.lc}, out.header+h.header)
	}
	audit()
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, m.CleanupDiscriminant())
	audit()
	as, e := c.cheap.as.size(c)
	serviceMust(t, e)
	ms, e := c.cheap.ms.size(c)
	serviceMust(t, e)
	if as != 1 || ms != 1 {
		t.Fatal("parent consumed callback-owned row", as, ms)
	}
	a, e := c.cheap.as.coefficient(c, 0)
	serviceMust(t, e)
	term, e := c.cheap.ms.monomial(c, 0)
	serviceMust(t, e)
	if a != util.NewInt(7) || term != foreign {
		t.Fatal("foreign row value/identity changed", a, term, foreign)
	}
	if m.DiscriminantSourceCause() != stopped || !errors.Is(returned, stopped) {
		t.Fatal("own acknowledgment changed source history")
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	serviceMust(t, c.cheap.reset())
	serviceMust(t, out.Close())
}
