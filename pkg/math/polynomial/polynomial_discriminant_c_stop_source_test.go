// Source: src/math/polynomial/polynomial.cpp:5473-5500; src/util/obj_ref.h.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

func TestPolynomialDiscriminantYoungerCoefficientRefStopBeforeLeadingAndPrime(t *testing.T) {
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
	var prime, coefficient Polynomial
	var leading util.Int
	var refused error
	h, e := NewPolynomialDeletionHandler(c.integer.Budget(), func(_ *PolynomialDeletionHandler, core *MultivariatePolynomialCore, value Polynomial) error {
		s := &m.discriminant
		if calls != 0 || s.executing || s.depth != 3 || s.stack[2] != 2 || value != s.refs[1].value {
			return nil
		}
		calls++
		prime, coefficient, leading = s.refs[0].value, s.refs[1].value, s.lc
		if prime == (Polynomial{}) || coefficient == (Polynomial{}) || out.root.value == (Polynomial{}) || out.root.assigning {
			t.Fatal("actual completed output before younger c destructor missing")
		}
		core.integer.Budget().FailAfter(0)
		raw, err := core.memory.Allocate(8)
		if raw != (umemory.Block{}) || err == nil {
			t.Fatal("real source callback allocation refusal missing", raw, err)
		}
		refused = err
		return err
	})
	serviceMust(t, e)
	serviceMust(t, c.AddDeletionHandler(h))
	returned := m.Discriminant(p, 0, out)
	s := &m.discriminant
	if calls != 1 || !errors.Is(returned, refused) || s.sourceCause != refused || s.depth != 3 || s.refs[0].value != prime || s.refs[1].value != coefficient || s.lc != leading {
		t.Fatal("younger c stop did not retain older declarations/first cause", calls, returned, s.depth)
	}
	audit := func() {
		polynomialImpPhysicalAuditNamed(t, m, []Polynomial{p, out.root.value, out.root.incoming, s.refs[0].value, s.refs[1].value}, []*util.Int{&s.lc}, out.header+h.header)
	}
	audit()
	used, state := c.integer.Budget().Used(), c.memory.State()
	if e := m.finishDiscriminantSource(); e != refused || used != c.integer.Budget().Used() || state != c.memory.State() || calls != 1 {
		t.Fatal("source retry replayed first c stop", e, calls)
	}
	c.integer.Budget().FailAfter(-1)
	serviceMust(t, m.CleanupDiscriminant())
	audit()
	if m.DiscriminantSourceCause() != refused || !errors.Is(returned, refused) || s.depth != 0 {
		t.Fatal("own acknowledgment erased returned/source history")
	}
	serviceMust(t, c.RemoveDeletionHandler(h))
	serviceMust(t, h.CloseChecked())
	polynomialQueryExpect(t, c, out.root.value, map[[3]uint32]int64{{0, 1, 0}: -4})
	serviceMust(t, m.Discriminant(c.unit, 0, out))
	if m.DiscriminantSourceCause() != nil || !errors.Is(returned, refused) {
		t.Fatal("next actual invocation retargeted old returned source error")
	}
	serviceMust(t, out.Close())
}
