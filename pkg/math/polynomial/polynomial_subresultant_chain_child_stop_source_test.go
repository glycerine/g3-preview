// Source: src/math/polynomial/polynomial.cpp:5499-5572,4600-4614;
// src/util/mpz.h:393-407.
package polynomial

import (
	"errors"
	"testing"

	"github.com/glycerine/g3/pkg/arena"
	"github.com/glycerine/g3/pkg/util"
	umemory "github.com/glycerine/g3/pkg/util/memory"
)

// A real signed-host accounting guard stops the reached Power AddMul RAW DEL.
// This is an owner-primitive boundary, not an ordinary mathematical OOM.
func TestPolynomialSubresultantChainYoungerPowerSourceStopPreservesThreeLocals(t *testing.T) {
	budget := arena.NewBudget(1 << 26)
	rt, e := umemory.NewRuntime(umemory.RuntimeConfig{Budget: budget, ThreadLocal: true, SizeTBits: 64})
	serviceMust(t, e)
	cfg := serviceConfig(budget).Integers
	cfg.Memory = rt
	cfg.Profile = umemory.SmallAllocatorProfile{PointerBits: 64, Debug: true}
	rational, e := util.NewRationalManager(util.RationalConfig{Integers: cfg})
	serviceMust(t, e)
	m, e := NewMultivariatePolynomialManager(MultivariatePolynomialManagerConfig{Core: MultivariatePolynomialConfig{Integer: rational.Integers(), Rationals: rational, Checkpoint: func() bool { return true }, Cooperation: func() error { return nil }}})
	serviceMust(t, e)
	c := m.core
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	_, e = c.monomials.MkVar()
	serviceMust(t, e)
	var large, modulus, warm, warmProduct util.Int
	serviceMust(t, c.integer.SetLimbs(&large, []uint32{17, 1 << 30}))
	digits := make([]uint32, 6)
	digits[0] = 251
	digits[5] = 1 << 31
	serviceMust(t, c.integer.SetLimbs(&modulus, digits))
	linear, e := c.monomials.MkVariable(0, 1)
	serviceMust(t, e)
	y, e := c.monomials.MkVariable(1, 3)
	serviceMust(t, e)
	xy, e := c.monomials.MkMonomial([]MonomialPower{{0, 1}, {1, 3}})
	serviceMust(t, e)
	base, e := c.MkTerms([]util.Int{large, util.NewInt(3)}, []Monomial{y, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(base))
	q, e := c.MkTerms([]util.Int{large, util.NewInt(3), util.NewInt(1)}, []Monomial{xy, linear, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(q))
	fourth, e := c.monomials.MkVariable(0, 4)
	serviceMust(t, e)
	p, e := c.MkTerms([]util.Int{util.NewInt(1), util.NewInt(1)}, []Monomial{fourth, c.monomials.unit})
	serviceMust(t, e)
	serviceMust(t, c.IncRef(p))
	serviceMust(t, c.modular.SetZP(&modulus))
	var warmed polynomialReference
	serviceMust(t, c.powerInto(base, 2, &warmed))
	three := util.NewInt(3)
	for i := 0; i < 2; i++ {
		serviceMust(t, c.integer.Mul(&large, &three, &warmProduct))
		serviceMust(t, c.integer.Rem(&warmProduct, &(*c.modular.BorrowModulus()), &warm))
	}
	out, e := NewPolynomialRefVector(c)
	serviceMust(t, e)
	var external umemory.Block
	defer func() {
		recovery, x := rt.Allocate(400000)
		serviceMust(t, x)
		contextTLSFinish(t, rt, m.CleanupSubresultantChain)
		contextTLSFinish(t, rt, out.Close)
		contextTLSFinish(t, rt, func() error { return warmed.closeValue(c, true) })
		contextTLSFinish(t, rt, func() error { return warmed.abort(c) })
		for _, token := range []*Polynomial{&p, &q, &base} {
			x := c.DecRef(*token)
			if x != nil {
				if !errors.Is(x, arena.ErrBounds) {
					t.Fatal(x)
				}
				contextTLSFinish(t, rt, func() error { return c.CleanupValue(token) })
			}
		}
		for _, a := range []*util.Int{&large, &modulus, &warm, &warmProduct} {
			contextTLSFinish(t, rt, func() error { return c.integer.Del(a) })
		}
		contextTLSFinish(t, rt, m.CloseChecked)
		contextTLSFinish(t, rt, rational.CloseChecked)
		x = rt.Deallocate(&recovery)
		if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (umemory.Block{})) {
			t.Fatal(x)
		}
		if external != (umemory.Block{}) {
			contextTLSFinish(t, rt, func() error { return rt.Deallocate(&external) })
		}
		contextTLSFinish(t, rt, rt.Close)
		if budget.Used() != 0 {
			t.Fatal("retained exact owners", budget.Used())
		}
	}()
	external, e = rt.Allocate(200000)
	serviceMust(t, e)
	visits := 0
	c.config.Checkpoint = func() bool {
		if c.powerLocal.active {
			visits++
			if visits == 2 {
				contextTLSNearMinimum(t, rt)
				x := rt.Deallocate(&external)
				if !errors.Is(x, arena.ErrBounds) || external != (umemory.Block{}) {
					t.Fatal("actual external release", x)
				}
			}
		}
		return true
	}
	returned := m.SubresultantChain(p, q, 0, out)
	s := &m.subresultant
	if returned == nil || s.sourceCause == nil || s.constructed != 3 || s.refs[0].root.value != c.unit || s.refs[2].root.value == (Polynomial{}) {
		t.Fatal("younger source stop did not protect older locals", returned, s.constructed, visits, s.power, s.refs[0].root.value, s.refs[2].root.value)
	}
	source, body := s.sourceCause, s.bodyCause
	power := s.power
	if c.powerInvocationSourceCause(&power) != source {
		t.Fatal("exact younger cause", source, c.powerInvocationSourceCause(&power))
	}
	subresultantNamedAudit(t, m, out, []Polynomial{p, q, base, warmed.value, warmed.incoming}, []*util.Int{&large, &modulus, &warm, &warmProduct})
	state := rt.State()
	if m.finishSubresultantChainSource() != source || rt.State() != state || s.constructed != 3 {
		t.Fatal("sticky source retry changed older suffix")
	}
	recovery, e := rt.Allocate(400000)
	serviceMust(t, e)
	contextTLSFinish(t, rt, m.CleanupSubresultantChain)
	if s.sourceCause != source || s.bodyCause != body || c.powerInvocationSourceCause(&power) != source {
		t.Fatal("own ACK erased immutable child/parent history")
	}
	x := rt.Deallocate(&recovery)
	if x != nil && (!errors.Is(x, arena.ErrBounds) || recovery != (umemory.Block{})) {
		t.Fatal("actual recovery release", x)
	}
	if !errors.Is(returned, source) {
		t.Fatal("returned cause lost identity", returned)
	}
}
